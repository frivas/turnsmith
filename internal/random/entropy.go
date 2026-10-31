package random

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// EntropySource reads fresh 64-bit words directly from its caller-owned reader.
// Production callers supply crypto/rand.Reader. It is local to one generator
// call and, like the deterministic sources, is not safe for concurrent use.
// A failed read is sticky. onError is called once so generators can be canceled
// despite Source's infallible Uint64 interface. Callers must inspect Err after
// generation and discard any generated output when it is non-nil.
type EntropySource struct {
	reader  io.Reader
	onError func(error)
	err     error
}

func NewEntropySource(reader io.Reader, onError func(error)) *EntropySource {
	return &EntropySource{reader: reader, onError: onError}
}
func (s *EntropySource) Err() error { return s.err }
func (s *EntropySource) Uint64() uint64 {
	if s.err != nil {
		return math.MaxUint64
	}
	var word [8]byte
	var err error
	if s.reader == nil {
		err = fmt.Errorf("entropy reader is required")
	} else {
		_, err = io.ReadFull(s.reader, word[:])
	}
	if err != nil {
		s.err = fmt.Errorf("read cryptographic randomness: %w", err)
		if s.onError != nil {
			s.onError(s.err)
		}
		// MaxUint64 is accepted by every positive-bound Below rejection threshold.
		// A zero fallback could loop forever before a generator checks cancellation.
		return math.MaxUint64
	}
	return binary.BigEndian.Uint64(word[:])
}
