package cube

import "testing"

func TestNotation(t *testing.T) {
	cases := []struct {
		m    Move
		size int
		want string
	}{{Move{R, 1, Clockwise}, 3, "R"}, {Move{U, 1, Half}, 3, "U2"}, {Move{B, 1, Counterclockwise}, 3, "B'"}, {Move{R, 2, Half}, 4, "Rw2"}, {Move{F, 3, Counterclockwise}, 6, "3Fw'"}}
	for _, tc := range cases {
		got, err := tc.m.Notation(tc.size)
		if err != nil || got != tc.want {
			t.Fatalf("%+v: %q %v", tc, got, err)
		}
	}
	for _, m := range []Move{{Face(6), 1, Clockwise}, {R, 0, Clockwise}, {R, 3, Half}, {R, 1, Turn(0)}, {R, 1, Turn(4)}} {
		if err := m.Validate(3); err == nil {
			t.Fatalf("accepted %+v", m)
		}
	}
	if err := (Move{R, 1, Clockwise}).Validate(1); err == nil {
		t.Fatal("accepted size one")
	}
	for f := U; f <= B; f++ {
		if f.Opposite() == f || f.Opposite().Opposite() != f {
			t.Fatal("bad opposite")
		}
	}
}
func FuzzMoveValidation(f *testing.F) {
	f.Add(uint8(0), uint8(1), uint8(1), 3)
	f.Fuzz(func(t *testing.T, face, layers, turns uint8, size int) {
		m := Move{Face(face), layers, Turn(turns)}
		token, err := m.Notation(size)
		if err == nil && (token == "" || m.Validate(size) != nil) {
			t.Fatal("inconsistent notation")
		}
	})
}
