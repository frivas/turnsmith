package cube444

import (
	"context"
	"errors"
	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"strings"
	"testing"
)

func TestWideMovesAndConstraints(t *testing.T) {
	length := 1000
	g := RandomMove{}
	got, e := g.Generate(context.Background(), random.Derive(42, "444", "random-move", "1", 0), puzzle.Options{Length: &length})
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Moves) != length {
		t.Fatal("length")
	}
	wide, outer := false, false
	faces := []cube.Face{}
	for i, token := range got.Moves {
		if strings.Contains(token, "w") {
			wide = true
		} else {
			outer = true
		}
		face := cube.Face(strings.IndexByte("UDLRFB", token[0]))
		if face > cube.B {
			t.Fatal(token)
		}
		faces = append(faces, face)
		if i > 0 && faces[i-1] == face {
			t.Fatal("same face")
		}
		if i > 1 && faces[i-2] == face && faces[i-1] == face.Opposite() {
			t.Fatal("opposite window")
		}
		if strings.TrimRight(strings.TrimSuffix(strings.TrimSuffix(token, "'"), "2"), "w") != string(token[0]) {
			t.Fatal("invalid token", token)
		}
	}
	if !wide || !outer {
		t.Fatal("missing move width")
	}
	again, e := g.Generate(context.Background(), random.Derive(42, "444", "random-move", "1", 0), puzzle.Options{Length: &length})
	if e != nil || strings.Join(got.Moves, " ") != strings.Join(again.Moves, " ") {
		t.Fatal("not deterministic")
	}
	if strings.Join(got.Moves[:12], " ") != "L2 B' L' Uw2 Lw D Rw Lw2 Fw Lw' D R'" {
		t.Fatal("algorithm v1 sequence changed", got.Moves[:12])
	}
}
func TestRegistrationAndErrors(t *testing.T) {
	r, e := puzzle.New(Registration())
	if e != nil {
		t.Fatal(e)
	}
	d, ok := r.Resolve("4x4x4")
	if !ok || d.ID != "444" || *d.Methods[0].DefaultLength != 40 {
		t.Fatal(d)
	}
	for _, n := range []int{0, 1001} {
		if _, e := (RandomMove{}).Generate(context.Background(), random.Derive(0, "a", "b", "c", 0), puzzle.Options{Length: &n}); e == nil {
			t.Fatal("accepted length", n)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n := 40
	if _, e := (RandomMove{}).Generate(ctx, random.Derive(0, "a", "b", "c", 0), puzzle.Options{Length: &n}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
