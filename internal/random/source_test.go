package random

import (
	"bytes"
	"math"
	"strconv"
	"testing"
)

func TestSeeds(t *testing.T) {
	for _, s := range []string{"0", "42", "00042", "18446744073709551615"} {
		if _, err := ParseSeed(s); err != nil {
			t.Fatalf("%q: %v", s, err)
		}
	}
	for _, s := range []string{"", "-1", "+1", "1.0", "0x10", "18446744073709551616", " 42", "42\n", "１", "1e2"} {
		if _, err := ParseSeed(s); err == nil {
			t.Fatalf("accepted %q", s)
		}
	}
	n, err := ParseSeed("18446744073709551615")
	if err != nil || n != math.MaxUint64 {
		t.Fatal(n, err)
	}
	seed, err := Seed(bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 42}))
	if err != nil || seed != 42 {
		t.Fatal(seed, err)
	}
	if _, err := Seed(bytes.NewReader([]byte{1})); err == nil {
		t.Fatal("short entropy accepted")
	}
}
func FuzzParseSeed(f *testing.F) {
	for _, s := range []string{"0", "42", "18446744073709551615", "-1", "", "0x10"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		n, err := ParseSeed(s)
		if err != nil {
			return
		}
		if s == "" {
			t.Fatal("empty accepted")
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				t.Fatalf("accepted %q", s)
			}
		}
		again, err := ParseSeed(strconv.FormatUint(n, 10))
		if err != nil || again != n {
			t.Fatal("roundtrip")
		}
	})
}

func TestDerivedPCGVector(t *testing.T) {
	r := Derive(42, "333", "random-move", "1", 0)
	for i, want := range []uint64{5067026879568920168, 18189984400193564563, 728703022228311301, 3919567004198634860, 1278838995454122105} {
		if got := r.Uint64(); got != want {
			t.Fatalf("word %d: got %d, want %d", i, got, want)
		}
	}
	// Each field is a distinct part of the stream identity.
	baseline := Derive(42, "333", "random-move", "1", 0).Uint64()
	for _, r := range []Source{Derive(43, "333", "random-move", "1", 0), Derive(42, "444", "random-move", "1", 0), Derive(42, "333", "other", "1", 0), Derive(42, "333", "random-move", "2", 0), Derive(42, "333", "random-move", "1", 1)} {
		if r.Uint64() == baseline {
			t.Fatal("stream identities collided")
		}
	}
}

type scripted struct {
	values []uint64
	offset int
}

func (s *scripted) Uint64() uint64 { n := s.values[s.offset]; s.offset++; return n }
func TestBelowRejectsIncompleteInterval(t *testing.T) {
	r := &scripted{values: []uint64{0, 1, 2, 3, 4, 5}}
	// For n=6, [0,4) is discarded, then 4 maps to residue 4.
	if got := Below(r, 6); got != 4 || r.offset != 5 {
		t.Fatal(got, r.offset)
	}
	if got := Below(r, 1); got != 0 {
		t.Fatal(got)
	}
}
