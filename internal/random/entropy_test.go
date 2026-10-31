package random

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

func TestEntropyWordsAndStickyFailure(t *testing.T) {
	calls := 0
	s := NewEntropySource(bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 255, 254, 253, 252, 251, 250, 249, 248}), func(error) { calls++ })
	if got := s.Uint64(); got != 0x0102030405060708 {
		t.Fatalf("big endian word: %x", got)
	}
	if got := s.Uint64(); got != 0xfffefdfcfbfaf9f8 {
		t.Fatalf("second word: %x", got)
	}
	if s.Err() != nil {
		t.Fatal(s.Err())
	}
	if s.Uint64() != math.MaxUint64 || !errors.Is(s.Err(), io.EOF) {
		t.Fatal(s.Err())
	}
	if s.Uint64() != math.MaxUint64 || calls != 1 {
		t.Fatal("failure not sticky", calls)
	}
	// Bound 3 rejects a zero fallback. Every failed source draw must still let
	// Below return so that the generator reaches its next context check.
	if got := Below(s, 3); got != 0 {
		t.Fatal(got)
	}
}
func TestEntropyShortReadAndNil(t *testing.T) {
	s := NewEntropySource(bytes.NewReader([]byte{1, 2, 3}), nil)
	_ = s.Uint64()
	if !errors.Is(s.Err(), io.ErrUnexpectedEOF) {
		t.Fatal(s.Err())
	}
	s = NewEntropySource(nil, nil)
	_ = s.Uint64()
	if s.Err() == nil {
		t.Fatal("nil reader accepted")
	}
}
