package classical

/*
#include <stdlib.h>
#include "memjob.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"qchain-kpi/internal/algs"
)

// MemJob is one classical operation called through this package's C
// functions with the key object kept, as in the reuse arm (E9's "c" arm). The
// key object, key bytes and buffers are set up before any measurement; the
// job is the C function and its argument, run on memprobe's runner.
type MemJob struct {
	j          *C.kpi_ossl_job
	bufs       []unsafe.Pointer
	cap1, cap2 int
	pool1      []unsafe.Pointer // SetPool's C copies
	pool2      []unsafe.Pointer
	plen1      []int
	plen2      []int
}

// NewMemJob prepares op for algorithm a with the given key pair (keygen
// needs none).
func NewMemJob(a algs.Alg, op string, sk, pk []byte) (*MemJob, error) {
	m := &MemJob{}
	m.j = (*C.kpi_ossl_job)(m.alloc(int(unsafe.Sizeof(C.kpi_ossl_job{}))))
	fail := func(err error) (*MemJob, error) {
		m.Close()
		return nil, err
	}
	out := func(n1, n2 int) {
		if n1 > 0 {
			m.j.out1, m.j.out1cap = (*C.uchar)(m.alloc(n1)), C.size_t(n1)
		}
		if n2 > 0 {
			m.j.out2, m.j.out2cap = (*C.uchar)(m.alloc(n2)), C.size_t(n2)
		}
	}
	if a.Track == algs.Sig {
		sc, err := NewSignature(a)
		if err != nil {
			return fail(err)
		}
		s := sc.(*sig)
		m.j.alg = s.code
		switch op {
		case "keygen":
			m.j.op = C.KPI_MEM_KEYGEN
			out(1024, 4096) // GenerateKey's buffers
		case "sign":
			m.j.op = C.KPI_MEM_SIGN
			if m.j.key, err = s.loadPrivate(sk, pk); err != nil {
				return fail(err)
			}
			out(s.sigCap, 0)
		case "verify":
			m.j.op = C.KPI_MEM_VERIFY
			if m.j.key, err = s.loadPublic(pk); err != nil {
				return fail(err)
			}
		default:
			return fail(fmt.Errorf("classical: %s has no operation %q", a.ID, op))
		}
		return m, nil
	}
	kc, err := NewKEM(a)
	if err != nil {
		return fail(err)
	}
	k := kc.(*kem)
	m.j.kem, m.j.alg = 1, k.code
	switch op {
	case "keygen":
		m.j.op = C.KPI_MEM_KEYGEN
		out(65, 32) // GenerateKey's buffers
	case "encaps":
		m.j.op = C.KPI_MEM_ENCAPS
		if m.j.key, err = k.loadPublic(pk); err != nil {
			return fail(err)
		}
		out(k.sizes.Ciphertext, 32)
	case "decaps":
		m.j.op = C.KPI_MEM_DECAPS
		if m.j.key, err = k.loadPrivate(sk, pk); err != nil {
			return fail(err)
		}
		out(32, 0)
	default:
		return fail(fmt.Errorf("classical: %s has no operation %q", a.ID, op))
	}
	if op != "keygen" {
		m.j.pkm, m.j.pkmlen = (*C.uchar)(m.copyIn(pk)), C.size_t(len(pk))
	}
	return m, nil
}

// Call returns the job: the C function and its argument.
func (m *MemJob) Call() (fn, arg unsafe.Pointer) {
	return unsafe.Pointer(C.kpi_ossl_job_run), unsafe.Pointer(m.j)
}

// SetInputs copies the next call's inputs into C memory: the message (sign,
// verify) or ciphertext (decaps), and the signature (verify).
func (m *MemJob) SetInputs(in1, in2 []byte) {
	if len(in1) > m.cap1 {
		m.j.in1, m.cap1 = (*C.uchar)(m.alloc(len(in1))), len(in1)
	}
	if len(in2) > m.cap2 {
		m.j.in2, m.cap2 = (*C.uchar)(m.alloc(len(in2))), len(in2)
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(m.j.in1)), m.cap1), in1)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(m.j.in2)), m.cap2), in2)
	m.j.in1len, m.j.in2len = C.size_t(len(in1)), C.size_t(len(in2))
}

// SetPool copies a whole input pool into C memory once (E3: no copying
// between timed calls); Select then points the job at entry i mod the pool
// size. in1 is the message (sign, verify) or ciphertext (decaps), in2 the
// signature (verify); either may be nil.
func (m *MemJob) SetPool(in1, in2 [][]byte) {
	m.pool1, m.plen1, m.pool2, m.plen2 = nil, nil, nil, nil
	for _, b := range in1 {
		m.pool1, m.plen1 = append(m.pool1, m.copyIn(b)), append(m.plen1, len(b))
	}
	for _, b := range in2 {
		m.pool2, m.plen2 = append(m.pool2, m.copyIn(b)), append(m.plen2, len(b))
	}
}

// Select points the job's inputs at pool entry i (no copy).
func (m *MemJob) Select(i int) {
	if n := len(m.pool1); n > 0 {
		m.j.in1, m.j.in1len = (*C.uchar)(m.pool1[i%n]), C.size_t(m.plen1[i%n])
	}
	if n := len(m.pool2); n > 0 {
		m.j.in2, m.j.in2len = (*C.uchar)(m.pool2[i%n]), C.size_t(m.plen2[i%n])
	}
}

// Outputs returns copies of the last call's outputs (keygen: public and
// secret key; sign: the signature; encaps: enc and the shared secret;
// decaps: the shared secret).
func (m *MemJob) Outputs() (out1, out2 []byte) {
	if m.j.out1 != nil {
		out1 = C.GoBytes(unsafe.Pointer(m.j.out1), C.int(m.j.out1len))
	}
	if m.j.out2 != nil {
		out2 = C.GoBytes(unsafe.Pointer(m.j.out2), C.int(m.j.out2len))
	}
	return out1, out2
}

// Close frees the key object and every buffer.
func (m *MemJob) Close() {
	if m.j != nil && m.j.key != nil {
		C.kpi_key_free(m.j.key)
	}
	for _, b := range m.bufs {
		C.free(b)
	}
	m.bufs, m.j = nil, nil
}

func (m *MemJob) alloc(n int) unsafe.Pointer {
	p := C.calloc(C.size_t(max(n, 1)), 1)
	if p == nil {
		panic("classical: out of C memory")
	}
	m.bufs = append(m.bufs, p)
	return p
}

func (m *MemJob) copyIn(b []byte) unsafe.Pointer {
	p := m.alloc(len(b))
	copy(unsafe.Slice((*byte)(p), len(b)), b)
	return p
}
