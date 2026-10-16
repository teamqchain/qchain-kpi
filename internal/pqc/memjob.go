package pqc

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

// MemJob is one liboqs operation called through liboqs's C API (E9's "c"
// arm): the algorithm's own stack and C heap, without liboqs-go around it.
// The object, key and buffers are C memory, set up before any measurement;
// the job is the C function and its argument, run on memprobe's runner.
type MemJob struct {
	j          *C.kpi_pqc_job
	bufs       []unsafe.Pointer
	cap1, cap2 int // input buffer capacities
	len1, len2 int // fixed output lengths (sign: the signature's maximum)
	op         string
	pool1      []unsafe.Pointer // SetPool's C copies
	pool2      []unsafe.Pointer
	plen1      []int
	plen2      []int
}

// NewMemJob prepares op for algorithm a with the given key pair (keygen
// needs none).
func NewMemJob(a algs.Alg, op string, sk, pk []byte) (*MemJob, error) {
	if a.Classical() {
		return nil, fmt.Errorf("pqc: %s is not a liboqs algorithm", a.ID)
	}
	m := &MemJob{op: op}
	m.j = (*C.kpi_pqc_job)(m.alloc(int(unsafe.Sizeof(C.kpi_pqc_job{}))))
	name := C.CString(a.OQSName)
	defer C.free(unsafe.Pointer(name))
	fail := func(err error) (*MemJob, error) {
		m.Close()
		return nil, err
	}
	key := func(k []byte, n int, what string) error {
		if len(k) != n {
			return fmt.Errorf("pqc: %s %s is %d bytes, want %d", a.ID, what, len(k), n)
		}
		m.j.key = (*C.uint8_t)(m.copyIn(k))
		return nil
	}
	if a.Track == algs.Sig {
		if m.j.sig = C.OQS_SIG_new(name); m.j.sig == nil {
			return fail(fmt.Errorf("pqc: OQS_SIG_new(%s) failed", a.OQSName))
		}
		pkLen, skLen := int(m.j.sig.length_public_key), int(m.j.sig.length_secret_key)
		switch op {
		case "keygen":
			m.j.op, m.len1, m.len2 = C.KPI_MEM_KEYGEN, pkLen, skLen
		case "sign":
			m.j.op, m.len1 = C.KPI_MEM_SIGN, int(m.j.sig.length_signature)
			if err := key(sk, skLen, "secret key"); err != nil {
				return fail(err)
			}
		case "verify":
			m.j.op = C.KPI_MEM_VERIFY
			if err := key(pk, pkLen, "public key"); err != nil {
				return fail(err)
			}
		default:
			return fail(fmt.Errorf("pqc: %s has no operation %q", a.ID, op))
		}
	} else {
		if m.j.kem = C.OQS_KEM_new(name); m.j.kem == nil {
			return fail(fmt.Errorf("pqc: OQS_KEM_new(%s) failed", a.OQSName))
		}
		k := m.j.kem
		pkLen, skLen := int(k.length_public_key), int(k.length_secret_key)
		ctLen, ssLen := int(k.length_ciphertext), int(k.length_shared_secret)
		switch op {
		case "keygen":
			m.j.op, m.len1, m.len2 = C.KPI_MEM_KEYGEN, pkLen, skLen
		case "encaps":
			m.j.op, m.len1, m.len2 = C.KPI_MEM_ENCAPS, ctLen, ssLen
			if err := key(pk, pkLen, "public key"); err != nil {
				return fail(err)
			}
		case "decaps":
			m.j.op, m.len1 = C.KPI_MEM_DECAPS, ssLen
			if err := key(sk, skLen, "secret key"); err != nil {
				return fail(err)
			}
		default:
			return fail(fmt.Errorf("pqc: %s has no operation %q", a.ID, op))
		}
	}
	if m.len1 > 0 {
		m.j.out1 = (*C.uint8_t)(m.alloc(m.len1))
	}
	if m.len2 > 0 {
		m.j.out2 = (*C.uint8_t)(m.alloc(m.len2))
	}
	return m, nil
}

// Call returns the job: the C function and its argument.
func (m *MemJob) Call() (fn, arg unsafe.Pointer) {
	return unsafe.Pointer(C.kpi_pqc_job_run), unsafe.Pointer(m.j)
}

// SetInputs copies the next call's inputs into C memory: the message (sign,
// verify) or ciphertext (decaps), and the signature (verify).
func (m *MemJob) SetInputs(in1, in2 []byte) {
	if len(in1) > m.cap1 {
		m.j.in1, m.cap1 = (*C.uint8_t)(m.alloc(len(in1))), len(in1)
	}
	if len(in2) > m.cap2 {
		m.j.in2, m.cap2 = (*C.uint8_t)(m.alloc(len(in2))), len(in2)
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(m.j.in1)), m.cap1), in1)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(m.j.in2)), m.cap2), in2)
	m.j.in1len, m.j.in2len = C.size_t(len(in1)), C.size_t(len(in2))
	if m.op == "sign" {
		m.j.out1len = C.size_t(m.len1)
	}
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
		m.j.in1, m.j.in1len = (*C.uint8_t)(m.pool1[i%n]), C.size_t(m.plen1[i%n])
	}
	if n := len(m.pool2); n > 0 {
		m.j.in2, m.j.in2len = (*C.uint8_t)(m.pool2[i%n]), C.size_t(m.plen2[i%n])
	}
	if m.op == "sign" {
		m.j.out1len = C.size_t(m.len1)
	}
}

// Outputs returns copies of the last call's outputs (keygen: public and
// secret key; sign: the signature; encaps: ciphertext and shared secret;
// decaps: the shared secret).
func (m *MemJob) Outputs() (out1, out2 []byte) {
	n1 := m.len1
	if m.op == "sign" {
		n1 = int(m.j.out1len)
	}
	if n1 > 0 {
		out1 = C.GoBytes(unsafe.Pointer(m.j.out1), C.int(n1))
	}
	if m.len2 > 0 {
		out2 = C.GoBytes(unsafe.Pointer(m.j.out2), C.int(m.len2))
	}
	return out1, out2
}

// Close frees the liboqs object and every buffer.
func (m *MemJob) Close() {
	if m.j != nil {
		if m.j.sig != nil {
			C.OQS_SIG_free(m.j.sig)
		}
		if m.j.kem != nil {
			C.OQS_KEM_free(m.j.kem)
		}
	}
	for _, b := range m.bufs {
		C.free(b)
	}
	m.bufs, m.j = nil, nil
}

func (m *MemJob) alloc(n int) unsafe.Pointer {
	p := C.calloc(C.size_t(max(n, 1)), 1)
	if p == nil {
		panic("pqc: out of C memory")
	}
	m.bufs = append(m.bufs, p)
	return p
}

func (m *MemJob) copyIn(b []byte) unsafe.Pointer {
	p := m.alloc(len(b))
	copy(unsafe.Slice((*byte)(p), len(b)), b)
	return p
}
