package app_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

type entropyGenerator struct {
	sources   []random.Source
	mode      string
	cancelled bool
}

func (g *entropyGenerator) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "direct", AlgorithmVersion: "1", DirectEntropy: true}
}
func (g *entropyGenerator) Generate(ctx context.Context, source random.Source, _ puzzle.Options) (puzzle.Generated, error) {
	g.sources = append(g.sources, source)
	if g.mode == "wait" {
		<-ctx.Done()
		return puzzle.Generated{}, ctx.Err()
	}
	if g.mode == "loop" {
		for {
			if e := ctx.Err(); e != nil {
				g.cancelled = true
				return puzzle.Generated{}, e
			}
			_ = random.Below(source, 3)
		}
	}
	word := source.Uint64()
	if g.mode == "cancel-check" {
		g.cancelled = ctx.Err() != nil
		return puzzle.Generated{}, ctx.Err()
	}
	return puzzle.Generated{Moves: []string{fmt.Sprintf("R%d", word)}}, nil
}
func entropyService(t *testing.T, g *entropyGenerator, r io.Reader) app.Service {
	t.Helper()
	registry, e := puzzle.New(puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "test", Name: "Test", DefaultMethod: "direct"}, Generators: []puzzle.Generator{g}})
	if e != nil {
		t.Fatal(e)
	}
	return app.Service{Registry: registry, Entropy: r, Version: "test"}
}
func words(values ...uint64) []byte {
	var b bytes.Buffer
	for _, v := range values {
		_ = binary.Write(&b, binary.BigEndian, v)
	}
	return b.Bytes()
}
func TestDirectEntropyBatchAndMetadata(t *testing.T) {
	g := &entropyGenerator{}
	s := entropyService(t, g, bytes.NewReader(words(1, 2, 3)))
	result, e := s.Generate(context.Background(), app.Request{Puzzle: "test", Count: 3})
	if e != nil {
		t.Fatal(e)
	}
	if result.Seed != "" || result.Randomness != "cryptographic" {
		t.Fatal(result)
	}
	for i, v := range result.Scrambles {
		if v.Sequence != fmt.Sprintf("R%d", i+1) {
			t.Fatal(result.Scrambles)
		}
	}
	if len(g.sources) != 3 || g.sources[0] == g.sources[1] || g.sources[1] == g.sources[2] {
		t.Fatal("sources shared across generator calls")
	}
	for _, source := range g.sources {
		if _, ok := source.(*random.EntropySource); !ok {
			t.Fatal("not direct entropy")
		}
	}
	data, _ := json.Marshal(result)
	if !strings.Contains(string(data), `"randomness":"cryptographic"`) {
		t.Fatal(string(data))
	}
}
func TestDirectExplicitSeedRetainsDerivedStreams(t *testing.T) {
	g := &entropyGenerator{}
	s := entropyService(t, g, panicEntropy{})
	seed := uint64(42)
	result, e := s.Generate(context.Background(), app.Request{Puzzle: "test", Seed: &seed, Count: 3})
	if e != nil {
		t.Fatal(e)
	}
	if result.Randomness != "seeded-practice" || result.Seed != "42" {
		t.Fatal(result)
	}
	for i, scramble := range result.Scrambles {
		word := random.Derive(seed, "test", "direct", "1", uint64(i)).Uint64()
		if scramble.Sequence != fmt.Sprintf("R%d", word) {
			t.Fatal(scramble)
		}
	}
}

type panicEntropy struct{}

func (panicEntropy) Read([]byte) (int, error) {
	panic("explicit seed or pre-cancelled request read entropy")
}
func TestDirectEntropyFailureReturnsNoResultAndOriginalCause(t *testing.T) {
	cause := errors.New("entropy failure fixture")
	for _, mode := range []string{"", "loop", "cancel-check"} {
		g := &entropyGenerator{mode: mode}
		s := entropyService(t, g, failingEntropy{cause})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		result, e := s.Generate(ctx, app.Request{Puzzle: "test", Count: 1})
		cancel()
		if !errors.Is(e, cause) || !reflect.DeepEqual(result, app.Result{}) {
			t.Fatal(mode, result, e)
		}
		if mode != "" && !g.cancelled {
			t.Fatal("entropy error did not cancel generator", mode)
		}
	}
	// Failure after a successful first scramble still returns no partial batch.
	g := &entropyGenerator{}
	s := entropyService(t, g, bytes.NewReader(words(1)))
	result, e := s.Generate(context.Background(), app.Request{Puzzle: "test", Count: 2})
	if !errors.Is(e, io.EOF) || !reflect.DeepEqual(result, app.Result{}) {
		t.Fatal(result, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s = entropyService(t, &entropyGenerator{}, panicEntropy{})
	if _, e = s.Generate(ctx, app.Request{Puzzle: "test", Count: 1}); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
}

type failingEntropy struct{ err error }

func (r failingEntropy) Read([]byte) (int, error) { return 0, r.err }
func TestLegacyJSONDoesNotGainRandomness(t *testing.T) {
	s := service(t)
	seed := uint64(42)
	r, e := s.Generate(context.Background(), app.Request{Seed: &seed, Count: 1})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(r)
	if strings.Contains(string(b), "randomness") {
		t.Fatal(string(b))
	}
}

func TestDirectGenerationParentCancellation(t *testing.T) {
	g := &entropyGenerator{mode: "wait"}
	s := entropyService(t, g, panicEntropy{})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result, e := s.Generate(ctx, app.Request{Puzzle: "test", Count: 1})
	if !errors.Is(e, context.DeadlineExceeded) || !reflect.DeepEqual(result, app.Result{}) {
		t.Fatal(result, e)
	}
}
