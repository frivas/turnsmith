package cube333

import (
	"context"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/frivas/turnsmith/internal/puzzle"
)

func TestInvariants(t *testing.T) {
	opposite := map[byte]byte{'U': 'D', 'D': 'U', 'L': 'R', 'R': 'L', 'F': 'B', 'B': 'F'}
	for seed := uint64(0); seed < 100; seed++ {
		for _, length := range []int{1, 2, 3, 25, 30, 1000} {
			result, err := (RandomMove{}).Generate(context.Background(), rand.NewPCG(seed, 0), puzzle.Options{Length: &length})
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Moves) != length {
				t.Fatalf("length %d: %d tokens", length, len(result.Moves))
			}
			for i, token := range result.Moves {
				if len(token) < 1 || len(token) > 2 || !strings.ContainsRune("UDLRFB", rune(token[0])) || (len(token) == 2 && token[1] != '2' && token[1] != '\'') {
					t.Fatalf("invalid token %q", token)
				}
				if i > 0 && token[0] == result.Moves[i-1][0] {
					t.Fatalf("same face: %v", result.Moves)
				}
				if i > 1 && token[0] == result.Moves[i-2][0] && result.Moves[i-1][0] == opposite[token[0]] {
					t.Fatalf("commuting redundancy: %v", result.Moves)
				}
			}
		}
	}
}
func TestIndependentCalls(t *testing.T) {
	n := 25
	g := RandomMove{}
	first, err := g.Generate(context.Background(), rand.NewPCG(42, 7), puzzle.Options{Length: &n})
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Generate(context.Background(), rand.NewPCG(99, 8), puzzle.Options{Length: &n})
	if err != nil {
		t.Fatal(err)
	}
	again, err := g.Generate(context.Background(), rand.NewPCG(42, 7), puzzle.Options{Length: &n})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, again) {
		t.Fatal("shared generation state")
	}
}
func TestInvalidAndCancelled(t *testing.T) {
	for _, n := range []int{0, -1, 1001} {
		if _, err := (RandomMove{}).Generate(context.Background(), rand.NewPCG(0, 0), puzzle.Options{Length: &n}); err == nil {
			t.Fatalf("accepted %d", n)
		}
	}
	if _, err := (RandomMove{}).Generate(context.Background(), rand.NewPCG(0, 0), puzzle.Options{}); err == nil {
		t.Fatal("accepted missing length")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n := 25
	if _, err := (RandomMove{}).Generate(ctx, rand.NewPCG(0, 0), puzzle.Options{Length: &n}); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}

type cancellingSource struct {
	cancel context.CancelFunc
	draws  int
}

func (r *cancellingSource) Uint64() uint64 {
	r.draws++
	if r.draws == 4 {
		r.cancel()
	}
	return 42
}
func TestCancellationDuringGeneration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n := 1000
	r := &cancellingSource{cancel: cancel}
	result, err := (RandomMove{}).Generate(ctx, r, puzzle.Options{Length: &n})
	if err != context.Canceled || len(result.Moves) != 0 || r.draws != 4 {
		t.Fatal(result, err, r.draws)
	}
}
