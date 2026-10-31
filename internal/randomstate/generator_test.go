package randomstate

import (
	"context"
	"errors"
	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/unixpickle/gocube"
	"strings"
	"testing"
	"time"
)

func TestPocketCoordinatesAndCoverage(t *testing.T) {
	for p := 0; p < 5040; p++ {
		if permutationCode(decodePermutation(p)) != p {
			t.Fatal("permutation", p)
		}
	}
	for o := 0; o < 729; o++ {
		if orientationCode(decodeOrientation(o)) != o {
			t.Fatal("orientation", o)
		}
	}
	start := time.Now()
	tables, err := load222(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	max := uint8(0)
	for i, d := range tables.distance {
		if d == 255 {
			t.Fatal("unreachable coordinate", i)
		}
		if d > max {
			max = d
		}
	}
	if max != 11 {
		t.Fatal("diameter", max)
	}
	t.Logf("222 first initialization %s, %d states, diameter %d", time.Since(start), len(tables.distance), max)
}

func TestSampledTargetsReached(t *testing.T) {
	ctx := context.Background()
	for _, id := range []string{"222", "333"} {
		count := 100
		if id == "333" {
			count = 20
		}
		for seed := uint64(0); seed < uint64(count); seed++ {
			rng := func() random.Source { return random.Derive(seed, id, "random-state", "1", 0) }
			target := gocube.SolvedCubieCube()
			var gen puzzle.Generator
			var err error
			if id == "333" {
				target, err = sample333(ctx, rng())
				gen = Cube333{}
			} else {
				target.Corners, err = sample222(ctx, rng())
				gen = Cube222{}
			}
			if err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			got, err := gen.Generate(ctx, rng(), puzzle.Options{})
			if err != nil {
				t.Fatalf("%s seed %d: %v", id, seed, err)
			}
			if seed == 0 {
				t.Logf("%s first generation %s: %s", id, time.Since(start), strings.Join(got.Moves, " "))
			}
			check := gocube.SolvedCubieCube()
			independent := cube333.Solved()
			for _, token := range got.Moves {
				move, err := gocube.ParseMove(token)
				if err != nil {
					t.Fatal(err)
				}
				check.Move(move)
				face := strings.IndexByte("UDLRFB", token[0])
				turns := cube.Clockwise
				if strings.HasSuffix(token, "2") {
					turns = cube.Half
				} else if strings.HasSuffix(token, "'") {
					turns = cube.Counterclockwise
				}
				if err := independent.Apply(cube.Move{Face: cube.Face(face), Layers: 1, Turns: turns}); err != nil {
					t.Fatal(err)
				}
			}
			if check.Corners != target.Corners || (id == "333" && check.Edges != target.Edges) {
				t.Fatalf("target not reached %s/%d", id, seed)
			}
			// Compare to the separate geometric facelet simulator. On 222 only corners exist.
			stickers := target.StickerCube()
			faces := [6]cube.Face{cube.U, cube.D, cube.F, cube.B, cube.R, cube.L}
			for f, face := range faces {
				for i := 0; i < 9; i++ {
					if id == "222" && i != 0 && i != 2 && i != 6 && i != 8 {
						continue
					}
					color := faces[stickers[f*9+i]-1]
					if independent[face][i] != color {
						t.Fatalf("independent facelets differ %s seed%d face%d sticker%d", id, seed, f, i)
					}
				}
			}
		}
	}
}

func TestCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, g := range []puzzle.Generator{Cube222{}, Cube333{}} {
		_, err := g.Generate(ctx, random.Derive(0, "x", "x", "x", 0), puzzle.Options{})
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
}

func Benchmark333(b *testing.B) {
	_, _ = load333(context.Background())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := (Cube333{}).Generate(context.Background(), random.Derive(uint64(i), "333", "random-state", "1", 0), puzzle.Options{})
		if err != nil {
			b.Fatal(err)
		}
	}
}
func Benchmark222(b *testing.B) {
	_, _ = load222(context.Background())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := (Cube222{}).Generate(context.Background(), random.Derive(uint64(i), "222", "random-state", "1", 0), puzzle.Options{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestGoldenSequences(t *testing.T) {
	for _, tc := range []struct {
		id, want string
		gen      puzzle.Generator
	}{
		{"222", "R U2 F' R U' R F' U F' U'", Cube222{}},
		{"333", "R2 U2 R2 D2 R2 B2 D2 B R2 U2 F D2 R2 D2 L' B' D R' D2 L F' L D2", Cube333{}},
	} {
		got, err := tc.gen.Generate(context.Background(), random.Derive(0, tc.id, "random-state", "1", 0), puzzle.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got.Moves, " ") != tc.want {
			t.Fatalf("%s sequence changed: %v", tc.id, got.Moves)
		}
	}
}

func TestConcurrentDeterminism(t *testing.T) {
	for _, gen := range []puzzle.Generator{Cube222{}, Cube333{}} {
		t.Run(gen.Descriptor().Description, func(t *testing.T) {
			for i := 0; i < 8; i++ {
				t.Run(string(rune('a'+i)), func(t *testing.T) {
					t.Parallel()
					a, err := gen.Generate(context.Background(), random.Derive(42, "test", "random-state", "1", 0), puzzle.Options{})
					if err != nil {
						t.Fatal(err)
					}
					b, err := gen.Generate(context.Background(), random.Derive(42, "test", "random-state", "1", 0), puzzle.Options{})
					if err != nil || strings.Join(a.Moves, " ") != strings.Join(b.Moves, " ") {
						t.Fatal("unstable result", err)
					}
				})
			}
		})
	}
}

func TestSearchCancellationAndLimit(t *testing.T) {
	tables, err := load333(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	search := search333{ctx: ctx, t: tables, nodes: 1023}
	if search.visit() || !errors.Is(search.err, context.Canceled) {
		t.Fatal("search ignored cancellation")
	}
	search = search333{ctx: context.Background(), t: tables, nodes: searchNodeLimit}
	if search.visit() || search.err == nil {
		t.Fatal("search ignored node limit")
	}
	for _, g := range []puzzle.Generator{Cube222{}, Cube333{}} {
		length := 20
		_, err := g.Generate(context.Background(), random.Derive(0, "a", "b", "c", 0), puzzle.Options{Length: &length})
		if err == nil {
			t.Fatal("accepted length")
		}
	}
	if strings.Join(inverseTokens(nil), " ") != "R R'" {
		t.Fatal("solved identity")
	}
}

func BenchmarkInitialize333(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tables := build333()
		if tables.h1 == nil {
			b.Fatal("missing tables")
		}
	}
}
func BenchmarkInitialize222(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tables := build222()
		if len(tables.distance) != pocketStates {
			b.Fatal("missing tables")
		}
	}
}
