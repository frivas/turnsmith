package cube333

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/frivas/turnsmith/internal/cube"
)

func TestReferenceStates(t *testing.T) {
	data, err := os.ReadFile("testdata/states.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		Cases []struct{ Moves, Facelets string }
	}
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.Cases {
		t.Run(fixture.Moves, func(t *testing.T) {
			state := Solved()
			for _, token := range strings.Fields(fixture.Moves) {
				move, err := parseMove(token)
				if err != nil {
					t.Fatal(err)
				}
				if err := state.Apply(move); err != nil {
					t.Fatal(err)
				}
			}
			var actual strings.Builder
			for _, face := range []cube.Face{cube.U, cube.R, cube.F, cube.D, cube.L, cube.B} {
				for _, value := range state[face] {
					actual.WriteByte("UDLRFB"[value])
				}
			}
			if actual.String() != fixture.Facelets {
				t.Fatalf("state disagrees with independent reference:\ngot  %s\nwant %s", actual.String(), fixture.Facelets)
			}
		})
	}
}

func TestMovePermutations(t *testing.T) {
	var initial State
	for face := range initial {
		for i := range initial[face] {
			initial[face][i] = cube.Face(face*9 + i)
		}
	}
	for face := cube.U; face <= cube.B; face++ {
		state := initial
		move := cube.Move{Face: face, Layers: 1, Turns: cube.Clockwise}
		if err := state.Apply(move); err != nil {
			t.Fatal(err)
		}
		for to, from := range []int{6, 3, 0, 7, 4, 1, 8, 5, 2} {
			if state[face][to] != initial[face][from] {
				t.Fatalf("face %d is not clockwise at cell %d", face, to)
			}
		}
		var count [54]int
		for f := range state {
			if state[f][4] != initial[f][4] {
				t.Fatal("center moved")
			}
			for _, sticker := range state[f] {
				count[sticker]++
			}
		}
		for _, n := range count {
			if n != 1 {
				t.Fatal("move duplicated or lost a sticker")
			}
		}
		for i := 0; i < 3; i++ {
			if err := state.Apply(move); err != nil {
				t.Fatal(err)
			}
		}
		if state != initial {
			t.Fatal("four quarter turns did not restore state")
		}
		for turn := cube.Clockwise; turn <= cube.Counterclockwise; turn++ {
			state = initial
			if err := state.Apply(cube.Move{Face: face, Layers: 1, Turns: turn}); err != nil {
				t.Fatal(err)
			}
			if err := state.Apply(cube.Move{Face: face, Layers: 1, Turns: 4 - turn}); err != nil {
				t.Fatal(err)
			}
			if state != initial {
				t.Fatal("move and inverse did not restore state")
			}
		}
	}
}

func TestInvalidDiagramMoves(t *testing.T) {
	for _, token := range []string{"", "X", "Rw", "r", "R3", "R2'", " R", "R U"} {
		if _, err := parseMove(token); err == nil {
			t.Fatalf("accepted %q", token)
		}
	}
	for _, move := range []cube.Move{{Face: cube.F, Layers: 2, Turns: cube.Clockwise}, {Face: 6, Layers: 1, Turns: cube.Clockwise}, {Face: cube.U, Layers: 0, Turns: cube.Half}} {
		state := Solved()
		if err := state.Apply(move); err == nil || state != Solved() {
			t.Fatal("invalid move accepted or mutated state")
		}
	}
}
