package cubestate

import (
	"encoding/json"
	"github.com/frivas/turnsmith/internal/alg"
	"os"
	"testing"
)

func TestExistingReference(t *testing.T) {
	data, e := os.ReadFile("../cube333/testdata/states.json")
	if e != nil {
		t.Fatal(e)
	}
	var v struct {
		Cases []struct{ Moves, Facelets string }
	}
	if e = json.Unmarshal(data, &v); e != nil {
		t.Fatal(e)
	}
	for _, c := range v.Cases {
		s, _ := New(3)
		if c.Moves != "" {
			moves, e := alg.Parse(c.Moves, 1000)
			if e != nil {
				t.Fatal(e)
			}
			for _, m := range moves {
				if e := s.Apply(m); e != nil {
					t.Fatal(e)
				}
			}
		}
		if s.Facelets() != c.Facelets {
			t.Fatalf("reference %s\n%s\n%s", c.Moves, s.Facelets(), c.Facelets)
		}
	}
}
func TestEquivalentMoves(t *testing.T) {
	for _, tc := range []struct{ a, b string }{{"Rw", "R M'"}, {"Lw", "L M"}, {"Uw", "U E'"}, {"Fw", "F S"}, {"x", "R M' L'"}, {"y", "U E' D'"}, {"z", "F S B'"}} {
		a, _ := alg.Parse(tc.a, 100)
		b, _ := alg.Parse(tc.b, 100)
		sa, _ := Replay(3, a)
		sb, _ := Replay(3, b)
		if sa[len(sa)-1].Facelets() != sb[len(sb)-1].Facelets() {
			t.Fatalf("%s != %s", tc.a, tc.b)
		}
	}
}
func TestInverseAllSizes(t *testing.T) {
	for _, size := range []int{2, 3, 4} {
		s, _ := New(size)
		source := "R U F2 B' D L x y z"
		if size >= 3 {
			source += " Rw Uw Fw"
		}
		if size%2 == 1 {
			source += " M E S"
		}
		m, e := alg.Parse(source, 1000)
		if e != nil {
			t.Fatal(e)
		}
		for _, move := range append(m, alg.Inverse(m)...) {
			if e = s.Apply(move); e != nil {
				t.Fatal(e)
			}
		}
		if !s.Solved() {
			t.Fatalf("size %d failed", size)
		}
		copy := s.Clone()
		bad, _ := alg.Parse("7Rw", 10)
		if s.Apply(bad[0]) == nil || s.Facelets() != copy.Facelets() {
			t.Fatal("invalid mutated")
		}
	}
}
