package main

// E2 (reuse vs reinit vs prod) and E2c (the pieces in between) arms. What each
// arm is, and why, is described in internal/proto/e2.go and docs/methodology.md.

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"qchain-kpi/internal/algs"
	"qchain-kpi/internal/inputs"
	"qchain-kpi/internal/proto"
	"qchain-kpi/internal/qchain"
	"qchain-kpi/internal/scheme"
	"qchain-kpi/internal/timing"
)

// slot is what SigObject and KEMObject have in common.
type slot interface {
	Open(kind scheme.ObjKind, sk, pk []byte) error
	Close()
}

// keyBufs hands out private copies of a secret key. liboqs-go keeps the slice
// given to Init and Clean zeroes it, so every Init gets its own copy; the
// copies are restored by refill, outside the timed region.
type keyBufs struct {
	src  []byte
	bufs [][]byte
	used int
}

func newKeyBufs(src []byte, n int) *keyBufs {
	k := &keyBufs{src: src}
	for i := 0; i < n; i++ {
		k.bufs = append(k.bufs, bytes.Clone(src))
	}
	return k
}

func (k *keyBufs) next() []byte {
	b := k.bufs[k.used]
	k.used++
	return b
}

func (k *keyBufs) refill() {
	for i := 0; i < k.used; i++ {
		copy(k.bufs[i], k.src)
	}
	k.used = 0
}

func batchFor(batch map[string]int, arm string) int {
	if n := batch[arm]; n > 1 {
		return n
	}
	return 1
}

// mergeSets joins arm sets into one block's arms, keeping their order.
func mergeSets(sets ...*armSet) *armSet {
	out := &armSet{}
	var afters []func() (int, error)
	for _, s := range sets {
		for k := range s.arms {
			var r func() error
			if k < len(s.resets) {
				r = s.resets[k]
			}
			out.resets = append(out.resets, r)
		}
		out.arms = append(out.arms, s.arms...)
		out.hasBytes = out.hasBytes || s.hasBytes
		out.closers = append(out.closers, s.closers...)
		if s.after != nil {
			afters = append(afters, s.after)
		}
	}
	out.after = func() (int, error) {
		n := 0
		for _, f := range afters {
			k, err := f()
			n += k
			if err != nil {
				return n, err
			}
		}
		return n, nil
	}
	return out
}

// checkArmOrder makes sure a builder produced exactly the arms (and order)
// that proto declares, which the orchestrator relies on for batch sizes.
func checkArmOrder(set *armSet, want []string) error {
	if len(set.arms) != len(want) {
		return fmt.Errorf("built %d arms, want %v", len(set.arms), want)
	}
	for k, a := range set.arms {
		if a.Name != want[k] {
			return fmt.Errorf("arm %d is %q, want %q (%v)", k, a.Name, want[k], want)
		}
	}
	return nil
}

// e2Inputs are the prepared inputs in the forms the arms need, all made
// before timing: production takes and returns hex strings, and signs the
// message string.
type e2Inputs struct {
	a              algs.Alg
	p              *inputs.Prep
	sizes          scheme.Sizes
	skHex, pkHex   string
	msgStr, sigHex []string // signatures
	ctHex, ssHex   []string // KEMs
}

func newE2Inputs(a algs.Alg, p *inputs.Prep, sizes scheme.Sizes) *e2Inputs {
	in := &e2Inputs{a: a, p: p, sizes: sizes, skHex: hex.EncodeToString(p.SK), pkHex: hex.EncodeToString(p.PK)}
	for i := 0; i < p.P; i++ {
		if a.Track == algs.Sig {
			in.msgStr = append(in.msgStr, string(p.Msgs[i]))
			in.sigHex = append(in.sigHex, hex.EncodeToString(p.Sigs[i]))
		} else {
			in.ctHex = append(in.ctHex, hex.EncodeToString(p.CTs[i]))
			in.ssHex = append(in.ssHex, hex.EncodeToString(p.SSs[i]))
		}
	}
	return in
}

// ------------------------------------------------------------------------ E2

// e2Arms builds the reuse, reinit and prod arms of one operation.
func e2Arms(a algs.Alg, op string, p *inputs.Prep, seed uint64, batch map[string]int) (*armSet, error) {
	reuse, err := e1Arms(a, op, p, seed)
	if err != nil {
		return nil, err
	}
	sets := []*armSet{reuse}
	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			reuse.close()
			return nil, err
		}
		in := newE2Inputs(a, p, s.Sizes())
		if !(a.Classical() && op == "keygen") {
			r, err := sigReinitArm(s, op, in, seed, batchFor(batch, proto.ArmReinit))
			if err != nil {
				mergeSets(sets...).close()
				return nil, err
			}
			sets = append(sets, r)
		}
		pr, err := sigProdArm(s, op, in, seed)
		if err != nil {
			mergeSets(sets...).close()
			return nil, err
		}
		sets = append(sets, pr)
	} else {
		k, err := newKEM(a)
		if err != nil {
			reuse.close()
			return nil, err
		}
		in := newE2Inputs(a, p, k.Sizes())
		if !(a.Classical() && op == "keygen") {
			r, err := kemReinitArm(k, op, in, seed, batchFor(batch, proto.ArmReinit))
			if err != nil {
				mergeSets(sets...).close()
				return nil, err
			}
			sets = append(sets, r)
		}
		pr, err := kemProdArm(k, op, in, seed)
		if err != nil {
			mergeSets(sets...).close()
			return nil, err
		}
		sets = append(sets, pr)
	}
	set := mergeSets(sets...)
	if err := checkArmOrder(set, proto.E2Arms(a, op)); err != nil {
		set.close()
		return nil, err
	}
	return set, nil
}

// exerciseSig fully checks a seeded subset of generated signature keys.
func exerciseSig(s scheme.Signature, sub *keySubset) func() (int, error) {
	return func() (int, error) {
		keys := sub.final()
		for n, k := range keys {
			sg, err := s.Sign(k.sk, k.pk, checkMessage)
			if err != nil {
				return n, fmt.Errorf("generated key %d cannot sign: %v", n, err)
			}
			if ok, err := s.Verify(k.pk, checkMessage, sg); err != nil || !ok {
				return n, fmt.Errorf("generated key %d: signature does not verify (%v)", n, err)
			}
		}
		return len(keys), nil
	}
}

// exerciseKEM fully checks a seeded subset of generated KEM keys.
func exerciseKEM(k scheme.KEM, sub *keySubset) func() (int, error) {
	return func() (int, error) {
		keys := sub.final()
		for n, kp := range keys {
			ct, ss, err := k.Encaps(kp.pk)
			if err != nil {
				return n, fmt.Errorf("generated key %d cannot encapsulate: %v", n, err)
			}
			ss2, err := k.Decaps(kp.sk, kp.pk, ct)
			if err != nil || !equalBytes(ss, ss2) {
				return n, fmt.Errorf("generated key %d: decapsulation mismatch (%v)", n, err)
			}
		}
		return len(keys), nil
	}
}

// lazyVerifier creates its verifier at the first check (after the first call).
type lazyVerifier struct {
	s scheme.Signature
	v scheme.VerifierReuse
}

func (l *lazyVerifier) verify(pk, msg, sg []byte) (bool, error) {
	if l.v == nil {
		var err error
		if l.v, err = l.s.NewVerifierReuse(pk); err != nil {
			return false, err
		}
	}
	return l.v.Verify(msg, sg)
}

func (l *lazyVerifier) close() {
	if l.v != nil {
		l.v.Close()
	}
}

type lazyDecaps struct {
	k scheme.KEM
	d scheme.DecapsulatorReuse
}

func (l *lazyDecaps) decaps(sk, pk, ct []byte) ([]byte, error) {
	if l.d == nil {
		var err error
		if l.d, err = l.k.NewDecapsulatorReuse(sk, pk); err != nil {
			return nil, err
		}
	}
	return l.d.Decaps(ct)
}

func (l *lazyDecaps) close() {
	if l.d != nil {
		l.d.Close()
	}
}

// sigReinitArm: a fresh library object for every call (Open, operation, Close).
func sigReinitArm(s scheme.Signature, op string, in *e2Inputs, seed uint64, b int) (*armSet, error) {
	so, ok := s.(scheme.SigObjects)
	if !ok {
		return nil, fmt.Errorf("%s has no E2 objects", in.a.ID)
	}
	x := so.NewSigObject()
	p := in.p
	set := &armSet{after: noAfter, closers: []func(){x.Close}}
	arm := timing.Arm{Name: proto.ArmReinit, Batch: b}
	var reset func() error
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		var pk, sk []byte
		set.hasBytes = in.sizes.SKVariable
		arm.Call = func(int) (int, error) {
			if err := x.Open(scheme.Keyless, nil, nil); err != nil {
				return 0, err
			}
			var skAlias []byte
			var err error
			pk, skAlias, err = x.Keygen()
			// liboqs-go's Clean zeroes the secret key it hands out, so a
			// caller must copy it first (production does so by hex-encoding
			// it before the deferred Clean).
			sk = bytes.Clone(skAlias)
			x.Close()
			return len(pk) + len(sk), err
		}
		arm.Check = func(i, _ int) error {
			if !in.sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("key lengths %d/%d, want %+v", len(pk), len(sk), in.sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		}
		set.after = exerciseSig(s, sub)
	case "sign":
		keys := newKeyBufs(p.SK, b)
		lv := &lazyVerifier{s: s}
		set.closers = append(set.closers, lv.close)
		set.hasBytes = in.sizes.SigVariable
		var last []byte
		arm.Call = func(i int) (int, error) {
			if err := x.Open(scheme.Private, keys.next(), p.PK); err != nil {
				return 0, err
			}
			var err error
			last, err = x.Sign(p.Msgs[i%p.P])
			x.Close()
			return len(last), err
		}
		reset = func() error { keys.refill(); return nil }
		arm.Check = func(i, _ int) error {
			keys.refill()
			if err := sigLengthOK(in.sizes, last); err != nil {
				return err
			}
			ok, err := lv.verify(p.PK, p.Msgs[i%p.P], last)
			if err != nil || !ok {
				return fmt.Errorf("signature %d does not verify (%v)", i, err)
			}
			return nil
		}
	case "verify":
		var ok bool
		arm.Call = func(i int) (int, error) {
			if err := x.Open(scheme.Public, nil, p.PK); err != nil {
				return 0, err
			}
			var err error
			ok, err = x.Verify(p.Msgs[i%p.P], p.Sigs[i%p.P])
			x.Close()
			return 0, err
		}
		arm.Check = func(i, _ int) error {
			if !ok {
				return fmt.Errorf("valid signature %d rejected", i%p.P)
			}
			return nil
		}
	default:
		return nil, fmt.Errorf("%s has no operation %q", in.a.ID, op)
	}
	set.arms = []timing.Arm{arm}
	set.resets = []func() error{reset}
	return set, nil
}

func kemReinitArm(k scheme.KEM, op string, in *e2Inputs, seed uint64, b int) (*armSet, error) {
	ko, ok := k.(scheme.KEMObjects)
	if !ok {
		return nil, fmt.Errorf("%s has no E2 objects", in.a.ID)
	}
	x := ko.NewKEMObject()
	p := in.p
	set := &armSet{after: noAfter, closers: []func(){x.Close}}
	arm := timing.Arm{Name: proto.ArmReinit, Batch: b}
	var reset func() error
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		var pk, sk []byte
		arm.Call = func(int) (int, error) {
			if err := x.Open(scheme.Keyless, nil, nil); err != nil {
				return 0, err
			}
			var skAlias []byte
			var err error
			pk, skAlias, err = x.Keygen()
			sk = bytes.Clone(skAlias) // see sigReinitArm
			x.Close()
			return len(pk) + len(sk), err
		}
		arm.Check = func(i, _ int) error {
			if !in.sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("key lengths %d/%d, want %+v", len(pk), len(sk), in.sizes)
			}
			sub.observe(i, pk, sk)
			return nil
		}
		set.after = exerciseKEM(k, sub)
	case "encaps":
		ld := &lazyDecaps{k: k}
		set.closers = append(set.closers, ld.close)
		var ct, ss []byte
		arm.Call = func(int) (int, error) {
			if err := x.Open(scheme.Public, nil, p.PK); err != nil {
				return 0, err
			}
			var err error
			ct, ss, err = x.Encaps()
			x.Close()
			return len(ct), err
		}
		arm.Check = func(i, _ int) error {
			if len(ct) != in.sizes.Ciphertext || len(ss) != in.sizes.SharedSecret {
				return fmt.Errorf("ciphertext/secret lengths %d/%d", len(ct), len(ss))
			}
			ss2, err := ld.decaps(p.SK, p.PK, ct)
			if err != nil || !equalBytes(ss, ss2) {
				return fmt.Errorf("encapsulation %d does not decapsulate to the same secret (%v)", i, err)
			}
			return nil
		}
	case "decaps":
		keys := newKeyBufs(p.SK, b)
		var ss []byte
		arm.Call = func(i int) (int, error) {
			if err := x.Open(scheme.Private, keys.next(), p.PK); err != nil {
				return 0, err
			}
			var err error
			ss, err = x.Decaps(p.CTs[i%p.P])
			x.Close()
			return len(ss), err
		}
		reset = func() error { keys.refill(); return nil }
		arm.Check = func(i, _ int) error {
			keys.refill()
			if !equalBytes(ss, p.SSs[i%p.P]) {
				return fmt.Errorf("decapsulation %d returned the wrong secret", i%p.P)
			}
			return nil
		}
	default:
		return nil, fmt.Errorf("%s has no operation %q", in.a.ID, op)
	}
	set.arms = []timing.Arm{arm}
	set.resets = []func() error{reset}
	return set, nil
}

// sigProdArm: production's own function for the post-quantum algorithms
// (copied verbatim), and the same pattern on OpenSSL for the classical ones.
func sigProdArm(s scheme.Signature, op string, in *e2Inputs, seed uint64) (*armSet, error) {
	p := in.p
	set := &armSet{after: noAfter}
	arm := timing.Arm{Name: proto.ArmProd}
	var call func(i int) (int, error)
	var x scheme.SigObject
	if in.a.Classical() {
		x = s.(scheme.SigObjects).NewSigObject()
		set.closers = append(set.closers, x.Close)
	} else {
		qchain.SetSignatureAlgorithm(in.a.OQSName)
	}
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		var pkHex, skHex string
		set.hasBytes = in.sizes.SKVariable
		if in.a.Classical() {
			call = func(int) (int, error) {
				pk, sk, err := x.Keygen()
				if err != nil {
					return 0, err
				}
				pkHex, skHex = hex.EncodeToString(pk), hex.EncodeToString(sk)
				return len(pkHex) + len(skHex), nil
			}
		} else {
			call = func(int) (int, error) {
				var err error
				pkHex, skHex, err = qchain.ProdSigKeygen()
				return len(pkHex) + len(skHex), err
			}
		}
		arm.Check = func(i, _ int) error {
			pk, err1 := hex.DecodeString(pkHex)
			sk, err2 := hex.DecodeString(skHex)
			if err1 != nil || err2 != nil || !in.sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("prod key pair: lengths %d/%d (%v %v)", len(pk), len(sk), err1, err2)
			}
			sub.observe(i, pk, sk)
			return nil
		}
		set.after = exerciseSig(s, sub)
	case "sign":
		lv := &lazyVerifier{s: s}
		set.closers = append(set.closers, lv.close)
		set.hasBytes = in.sizes.SigVariable
		var sigHex string
		if in.a.Classical() {
			ecdsa := in.a.ID == "ecdsa-p256"
			call = func(i int) (int, error) {
				sk, err := hex.DecodeString(in.skHex)
				if err != nil {
					return 0, err
				}
				var pk []byte
				if ecdsa {
					if pk, err = hex.DecodeString(in.pkHex); err != nil {
						return 0, err
					}
				}
				if err := x.Open(scheme.Private, sk, pk); err != nil {
					return 0, err
				}
				sg, err := x.Sign([]byte(in.msgStr[i%p.P]))
				x.Close()
				if err != nil {
					return 0, err
				}
				sigHex = hex.EncodeToString(sg)
				return len(sigHex), nil
			}
		} else {
			call = func(i int) (int, error) {
				var err error
				sigHex, err = qchain.ProdSign(in.msgStr[i%p.P], in.skHex)
				return len(sigHex), err
			}
		}
		arm.Check = func(i, _ int) error {
			sg, err := hex.DecodeString(sigHex)
			if err != nil {
				return err
			}
			if err := sigLengthOK(in.sizes, sg); err != nil {
				return err
			}
			ok, err := lv.verify(p.PK, p.Msgs[i%p.P], sg)
			if err != nil || !ok {
				return fmt.Errorf("prod signature %d does not verify (%v)", i, err)
			}
			return nil
		}
	case "verify":
		var ok bool
		if in.a.Classical() {
			call = func(i int) (int, error) {
				pk, err := hex.DecodeString(in.pkHex)
				if err != nil {
					return 0, err
				}
				sg, err := hex.DecodeString(in.sigHex[i%p.P])
				if err != nil {
					return 0, err
				}
				if err := x.Open(scheme.Public, nil, pk); err != nil {
					return 0, err
				}
				ok, err = x.Verify([]byte(in.msgStr[i%p.P]), sg)
				x.Close()
				return 0, err
			}
		} else {
			call = func(i int) (int, error) {
				var err error
				ok, err = qchain.ProdVerify(in.msgStr[i%p.P], in.sigHex[i%p.P], in.pkHex)
				return 0, err
			}
		}
		arm.Check = func(i, _ int) error {
			if !ok {
				return fmt.Errorf("prod: valid signature %d rejected", i%p.P)
			}
			return nil
		}
	default:
		return nil, fmt.Errorf("%s has no operation %q", in.a.ID, op)
	}
	arm.Call = call
	set.arms = []timing.Arm{arm}
	return set, nil
}

func kemProdArm(k scheme.KEM, op string, in *e2Inputs, seed uint64) (*armSet, error) {
	p := in.p
	set := &armSet{after: noAfter}
	arm := timing.Arm{Name: proto.ArmProd}
	var call func(i int) (int, error)
	var x scheme.KEMObject
	if in.a.Classical() {
		x = k.(scheme.KEMObjects).NewKEMObject()
		set.closers = append(set.closers, x.Close)
	} else {
		qchain.SetKEMAlgorithm(in.a.OQSName)
	}
	switch op {
	case "keygen":
		sub := newKeySubset(seed)
		var pkHex, skHex string
		if in.a.Classical() {
			call = func(int) (int, error) {
				pk, sk, err := x.Keygen()
				if err != nil {
					return 0, err
				}
				pkHex, skHex = hex.EncodeToString(pk), hex.EncodeToString(sk)
				return len(pkHex) + len(skHex), nil
			}
		} else {
			call = func(int) (int, error) {
				var err error
				pkHex, skHex, err = qchain.ProdKEMKeygen()
				return len(pkHex) + len(skHex), err
			}
		}
		arm.Check = func(i, _ int) error {
			pk, err1 := hex.DecodeString(pkHex)
			sk, err2 := hex.DecodeString(skHex)
			if err1 != nil || err2 != nil || !in.sizes.KeyLengthsOK(pk, sk) {
				return fmt.Errorf("prod key pair: lengths %d/%d (%v %v)", len(pk), len(sk), err1, err2)
			}
			sub.observe(i, pk, sk)
			return nil
		}
		set.after = exerciseKEM(k, sub)
	case "encaps":
		ld := &lazyDecaps{k: k}
		set.closers = append(set.closers, ld.close)
		var ctHex, ssHex string
		if in.a.Classical() {
			call = func(int) (int, error) {
				pk, err := hex.DecodeString(in.pkHex)
				if err != nil {
					return 0, err
				}
				if err := x.Open(scheme.Public, nil, pk); err != nil {
					return 0, err
				}
				ct, ss, err := x.Encaps()
				x.Close()
				if err != nil {
					return 0, err
				}
				ctHex, ssHex = hex.EncodeToString(ct), hex.EncodeToString(ss)
				return len(ctHex), nil
			}
		} else {
			call = func(int) (int, error) {
				var err error
				ctHex, ssHex, err = qchain.ProdKEMEncap(in.pkHex)
				return len(ctHex), err
			}
		}
		arm.Check = func(i, _ int) error {
			ct, err1 := hex.DecodeString(ctHex)
			ss, err2 := hex.DecodeString(ssHex)
			if err1 != nil || err2 != nil || len(ct) != in.sizes.Ciphertext || len(ss) != in.sizes.SharedSecret {
				return fmt.Errorf("prod encapsulation %d: lengths %d/%d (%v %v)", i, len(ct), len(ss), err1, err2)
			}
			ss2, err := ld.decaps(p.SK, p.PK, ct)
			if err != nil || !equalBytes(ss, ss2) {
				return fmt.Errorf("prod encapsulation %d does not decapsulate to the same secret (%v)", i, err)
			}
			return nil
		}
	case "decaps":
		var ssHex string
		if in.a.Classical() {
			call = func(i int) (int, error) {
				ct, err := hex.DecodeString(in.ctHex[i%p.P])
				if err != nil {
					return 0, err
				}
				sk, err := hex.DecodeString(in.skHex)
				if err != nil {
					return 0, err
				}
				pk, err := hex.DecodeString(in.pkHex)
				if err != nil {
					return 0, err
				}
				if err := x.Open(scheme.Private, sk, pk); err != nil {
					return 0, err
				}
				ss, err := x.Decaps(ct)
				x.Close()
				if err != nil {
					return 0, err
				}
				ssHex = hex.EncodeToString(ss)
				return len(ssHex), nil
			}
		} else {
			call = func(i int) (int, error) {
				var err error
				ssHex, err = qchain.ProdKEMDecap(in.ctHex[i%p.P], in.skHex)
				return len(ssHex), err
			}
		}
		arm.Check = func(i, _ int) error {
			if ssHex != in.ssHex[i%p.P] {
				return fmt.Errorf("prod decapsulation %d returned the wrong secret", i%p.P)
			}
			return nil
		}
	default:
		return nil, fmt.Errorf("%s has no operation %q", in.a.ID, op)
	}
	arm.Call = call
	set.arms = []timing.Arm{arm}
	return set, nil
}

// ----------------------------------------------------------------------- E2c

// e2cArms builds the component arms of one operation (proto.E2cArms order).
func e2cArms(a algs.Alg, op string, p *inputs.Prep, batch map[string]int) (*armSet, error) {
	var sizes scheme.Sizes
	var newSlot func() slot
	if a.Track == algs.Sig {
		s, err := newSignature(a)
		if err != nil {
			return nil, err
		}
		sizes = s.Sizes()
		so := s.(scheme.SigObjects)
		newSlot = func() slot { return so.NewSigObject() }
	} else {
		k, err := newKEM(a)
		if err != nil {
			return nil, err
		}
		sizes = k.Sizes()
		ko := k.(scheme.KEMObjects)
		newSlot = func() slot { return ko.NewKEMObject() }
	}
	in := newE2Inputs(a, p, sizes)
	kind := scheme.KindOf(op)
	set := &armSet{after: noAfter}
	for _, name := range proto.E2cArms(a, op) {
		b := batchFor(batch, name)
		var arm timing.Arm
		var reset func() error
		var err error
		switch name {
		case proto.ArmHexDecodeSK:
			arm = hexDecodeArm(name, []string{in.skHex}, [][]byte{p.SK})
		case proto.ArmHexDecodePK:
			arm = hexDecodeArm(name, []string{in.pkHex}, [][]byte{p.PK})
		case proto.ArmHexDecodeSig:
			arm = hexDecodeArm(name, in.sigHex, p.Sigs)
		case proto.ArmHexDecodeCT:
			arm = hexDecodeArm(name, in.ctHex, p.CTs)
		case proto.ArmHexEncodePK:
			arm = hexEncodeArm(name, [][]byte{p.PK})
		case proto.ArmHexEncodeSK:
			arm = hexEncodeArm(name, [][]byte{p.SK})
		case proto.ArmHexEncodeSig:
			arm = hexEncodeArm(name, p.Sigs)
		case proto.ArmHexEncodeCT:
			arm = hexEncodeArm(name, p.CTs)
		case proto.ArmHexEncodeSS:
			arm = hexEncodeArm(name, p.SSs)
		case proto.ArmInit:
			arm, reset, err = initArm(newSlot, kind, p, b, set)
		case proto.ArmClean:
			arm, reset, err = cleanArm(newSlot, kind, p, b, set)
		default:
			err = fmt.Errorf("unknown E2c arm %q", name)
		}
		if err != nil {
			set.close()
			return nil, err
		}
		arm.Batch = b
		if name == proto.ArmHexDecodeSig || name == proto.ArmHexEncodeSig {
			set.hasBytes = set.hasBytes || sizes.SigVariable
		}
		set.arms = append(set.arms, arm)
		set.resets = append(set.resets, reset)
	}
	return set, nil
}

func hexDecodeArm(name string, pool []string, want [][]byte) timing.Arm {
	var last []byte
	return timing.Arm{
		Name: name,
		Call: func(i int) (int, error) {
			var err error
			last, err = hex.DecodeString(pool[i%len(pool)])
			return len(last), err
		},
		Check: func(i, _ int) error {
			if !equalBytes(last, want[i%len(want)]) {
				return fmt.Errorf("%s %d: decoded bytes differ", name, i)
			}
			return nil
		},
	}
}

func hexEncodeArm(name string, pool [][]byte) timing.Arm {
	var last string
	return timing.Arm{
		Name: name,
		Call: func(i int) (int, error) {
			last = hex.EncodeToString(pool[i%len(pool)])
			return len(last), nil
		},
		Check: func(i, _ int) error {
			if len(last) != 2*len(pool[i%len(pool)]) {
				return fmt.Errorf("%s %d: %d hex characters", name, i, len(last))
			}
			return nil
		},
	}
}

// initArm times Open alone (liboqs Init, or loading the key into an OpenSSL
// key object). The b objects opened during one sample are closed, and their
// key copies restored, outside the timed region.
func initArm(newSlot func() slot, kind scheme.ObjKind, p *inputs.Prep, b int, set *armSet) (timing.Arm, func() error, error) {
	slots := make([]slot, b)
	for j := range slots {
		slots[j] = newSlot()
	}
	keys := newKeyBufs(p.SK, b)
	used := 0
	set.closers = append(set.closers, func() {
		for _, s := range slots {
			s.Close()
		}
	})
	reset := func() error {
		for j := 0; j < used; j++ {
			slots[j].Close()
		}
		used = 0
		keys.refill()
		return nil
	}
	arm := timing.Arm{
		Name: proto.ArmInit,
		Call: func(int) (int, error) {
			var sk []byte
			if kind == scheme.Private {
				sk = keys.next()
			}
			err := slots[used].Open(kind, sk, p.PK)
			used++
			return 0, err
		},
		Check: func(int, int) error { return reset() },
	}
	return arm, reset, nil
}

// cleanArm times Close alone (liboqs Clean, or freeing the OpenSSL key
// object). The objects are opened beforehand, outside the timed region.
func cleanArm(newSlot func() slot, kind scheme.ObjKind, p *inputs.Prep, b int, set *armSet) (timing.Arm, func() error, error) {
	slots := make([]slot, b)
	keys := newKeyBufs(p.SK, b)
	used := 0
	open := func(j int) error {
		var sk []byte
		if kind == scheme.Private {
			sk = keys.bufs[j]
		}
		return slots[j].Open(kind, sk, p.PK)
	}
	for j := range slots {
		slots[j] = newSlot()
		if err := open(j); err != nil {
			return timing.Arm{}, nil, err
		}
	}
	set.closers = append(set.closers, func() {
		for _, s := range slots {
			s.Close()
		}
	})
	reset := func() error {
		for j := 0; j < used; j++ {
			copy(keys.bufs[j], p.SK)
			if err := open(j); err != nil {
				return err
			}
		}
		used = 0
		return nil
	}
	arm := timing.Arm{
		Name: proto.ArmClean,
		Call: func(int) (int, error) {
			slots[used].Close()
			used++
			return 0, nil
		},
		Check: func(int, int) error { return reset() },
	}
	return arm, reset, nil
}
