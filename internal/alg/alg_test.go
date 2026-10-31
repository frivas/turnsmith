package alg

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct{ in, want string }{{"RU2R’U′", "R U2 R' U'"}, {"(R U)3", "R U R U R U"}, {"[R,U]", "R U R' U'"}, {"[R:U2]", "R U2 R'"}, {"([R,U])2'", "U R U' R' U R U' R'"}, {"r M E S x y z 2Rw2'", "Rw M E S x y z Rw2"}, {"R # note\n U // note\n F", "R U F"}} {
		m, e := Parse(tc.in, 1000)
		if e != nil || Format(m) != tc.want {
			t.Errorf("%q: %s %v", tc.in, Format(m), e)
		}
	}
}
func TestInvalid(t *testing.T) {
	for _, s := range []string{"", "()", "(R", "[R,U", "[R]", "[R,]", "R ?", "R3", "(R)0", "(R)100000000", strings.Repeat("(", 33) + "R" + strings.Repeat(")", 33), "[R,U]1000"} {
		if _, e := Parse(s, 1000); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
}
func TestInverse(t *testing.T) {
	m, e := Parse("([R,U] Fw M x2)3", 1000)
	if e != nil {
		t.Fatal(e)
	}
	if Format(Inverse(Inverse(m))) != Format(m) {
		t.Fatal("inverse roundtrip")
	}
}
