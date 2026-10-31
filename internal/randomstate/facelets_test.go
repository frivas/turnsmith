package randomstate

import (
	"context"
	"errors"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/random"
	"strings"
	"testing"
	"time"
)

func reducedState(t *testing.T, facelets string) cubestate.State {
	t.Helper()
	s, _ := cubestate.New(3)
	for f, face := range []cube.Face{cube.U, cube.R, cube.F, cube.D, cube.L, cube.B} {
		for i := 0; i < 9; i++ {
			s.Faces[face][i] = cube.Face(strings.IndexByte("UDLRFB", facelets[f*9+i]))
		}
	}
	return s
}
func applyReducedTokens(t *testing.T, s *cubestate.State, tokens []string) {
	t.Helper()
	for _, token := range tokens {
		moves, e := alg.Parse(token, 1)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Apply(moves[0]); e != nil {
			t.Fatal(e)
		}
	}
}
func verifyReducedSolution(t *testing.T, facelets string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	solution, e := Solve333Facelets(ctx, facelets)
	if e != nil {
		t.Fatal(e)
	}
	state := reducedState(t, facelets)
	applyReducedTokens(t, &state, solution)
	for _, face := range state.Faces {
		for _, color := range face {
			if color != face[4] {
				t.Fatalf("solution %q did not solve independently: %s", solution, state.Facelets())
			}
		}
	}
}
func TestSolve333FaceletsIndependentCanonicalCases(t *testing.T) {
	sequences := []string{""}
	for _, face := range "URFDLB" {
		for _, suffix := range []string{"", "2", "'"} {
			sequences = append(sequences, string(face)+suffix)
		}
	}
	rng := random.Derive(9182, "333", "facelets-tests", "1", 0)
	tokens := []string{"U", "U2", "U'", "R", "R2", "R'", "F", "F2", "F'", "D", "D2", "D'", "L", "L2", "L'", "B", "B2", "B'"}
	for i := 0; i < 20; i++ {
		parts := make([]string, 4+i%5)
		for j := range parts {
			parts[j] = tokens[random.Below(rng, uint64(len(tokens)))]
		}
		sequences = append(sequences, strings.Join(parts, " "))
	}
	for index, sequence := range sequences {
		t.Run(fmt.Sprintf("case-%02d", index), func(t *testing.T) {
			state, _ := cubestate.New(3)
			legacy := cube333.Solved()
			if sequence != "" {
				moves, e := alg.Parse(sequence, 100)
				if e != nil {
					t.Fatal(e)
				}
				for _, m := range moves {
					if e = state.Apply(m); e != nil {
						t.Fatal(e)
					}
					if e = legacy.Apply(cube.Move{Face: cube.Face(strings.IndexByte("UDLRFB", m.Face)), Layers: 1, Turns: cube.Turn(m.Turns)}); e != nil {
						t.Fatal(e)
					}
				}
			}
			solution, e := Solve333Facelets(context.Background(), state.Facelets())
			if e != nil {
				t.Fatal(e)
			}
			applyReducedTokens(t, &state, solution)
			if !state.Solved() {
				t.Fatal("cubestate independent replay failed", sequence, solution)
			}
			for _, token := range solution {
				m, e := alg.Parse(token, 1)
				if e != nil {
					t.Fatal(e)
				}
				if e = legacy.Apply(cube.Move{Face: cube.Face(strings.IndexByte("UDLRFB", m[0].Face)), Layers: 1, Turns: cube.Turn(m[0].Turns)}); e != nil {
					t.Fatal(e)
				}
			}
			if legacy != cube333.Solved() {
				t.Fatal("separate cube333 replay failed", sequence, solution)
			}
		})
	}
}
func TestSolve333FaceletsRotatedCentersRegression(t *testing.T) {
	// Actual reduced 4x4 target previously exceeded the search bound when the
	// callback treated its UBRDFL centers as fixed canonical URFDLB centers.
	verifyReducedSolution(t, "BLBUUBDDRFUUUBBUFDRBUBRRFLLDDFFDRUURLFBRFLBDLRFDRLDFLL")
	for _, sequence := range []string{"x R U F", "y U2 L B'", "z R2 F U'"} {
		moves, _ := alg.Parse(sequence, 20)
		states, e := cubestate.Replay(3, moves)
		if e != nil {
			t.Fatal(e)
		}
		verifyReducedSolution(t, states[len(states)-1].Facelets())
	}
}
func TestSolve333FaceletsRejectsInvalidAndCancellation(t *testing.T) {
	solved := "UUUUUUUUURRRRRRRRRFFFFFFFFFDDDDDDDDDLLLLLLLLLBBBBBBBBB"
	edgeFlip := []byte(solved)
	edgeFlip[5], edgeFlip[10] = edgeFlip[10], edgeFlip[5]
	cornerTwist := []byte(solved)
	cornerTwist[8], cornerTwist[9], cornerTwist[20] = cornerTwist[9], cornerTwist[20], cornerTwist[8]
	paritySwap := []byte(solved)
	paritySwap[10], paritySwap[19] = paritySwap[19], paritySwap[10]
	duplicateCenters := []byte(solved)
	duplicateCenters[13] = duplicateCenters[4]
	for name, input := range map[string]string{"empty": "", "length": solved[:53], "unknown": strings.Repeat("?", 54), "counts": strings.Repeat("U", 54), "duplicate-centers": string(duplicateCenters), "edge-flip": string(edgeFlip), "corner-twist": string(cornerTwist), "parity": string(paritySwap)} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			solution, e := Solve333Facelets(ctx, input)
			if e == nil || len(solution) > 0 {
				t.Fatal("invalid cube accepted", solution, e)
			}
			if errors.Is(e, context.DeadlineExceeded) {
				t.Fatal("invalid cube reached search instead of validation", e)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := Solve333Facelets(ctx, solved); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}
