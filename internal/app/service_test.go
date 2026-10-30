package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/frivas/turnsmith/internal/app"
	"github.com/frivas/turnsmith/internal/cube333"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
)

func service(t testing.TB) app.Service {
	t.Helper()
	r, err := puzzle.New(cube333.Registration())
	if err != nil {
		t.Fatal(err)
	}
	return app.Service{Registry: r, Entropy: bytes.NewReader(make([]byte, 8)), Version: "0.1.0"}
}
func ptr[T any](n T) *T { return &n }
func TestDefaultsAndPrefixes(t *testing.T) {
	s := service(t)
	req := app.Request{Count: 3, Seed: ptr(uint64(42))}
	a, err := s.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if a.Puzzle != "333" || a.Method != "random-move" || a.AlgorithmVersion != "1" || a.Seed != "42" || *a.Parameters.Length != 25 {
		t.Fatalf("metadata: %+v", a)
	}
	req.Count = 10
	b, err := s.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Scrambles, b.Scrambles[:3]) {
		t.Fatal("batch prefix changed")
	}
	for _, alias := range []string{"3x3", "3x3x3"} {
		req.Puzzle = alias
		c, err := s.Generate(context.Background(), req)
		if err != nil || !reflect.DeepEqual(b, c) {
			t.Fatal("alias changed results", err)
		}
	}
	req.Length = ptr(30)
	long, err := s.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	for i := range a.Scrambles {
		if !reflect.DeepEqual(a.Scrambles[i].Moves, long.Scrambles[i].Moves[:25]) {
			t.Fatal("length changed stream")
		}
	}
}
func TestLimits(t *testing.T) {
	s := service(t)
	for _, req := range []app.Request{
		{Count: 0}, {Count: -1}, {Count: 10001}, {Count: math.MaxInt},
		{Count: 1, Length: ptr(0)}, {Count: 1, Length: ptr(-1)}, {Count: 1, Length: ptr(1001)}, {Count: 1, Length: ptr(math.MaxInt)},
		{Count: 1001, Length: ptr(1000)}, {Count: 1, Puzzle: "444"}, {Count: 1, Method: "random-state"},
	} {
		_, err := s.Generate(context.Background(), req)
		var invalid *app.InvalidInput
		if !errors.As(err, &invalid) {
			t.Fatalf("%+v: %v", req, err)
		}
	}
	for _, req := range []app.Request{{Count: 10000, Length: ptr(1)}, {Count: 1000, Length: ptr(1000)}} {
		req.Seed = ptr(uint64(0))
		result, err := s.Generate(context.Background(), req)
		if err != nil || len(result.Scrambles) != req.Count {
			t.Fatal(req, err)
		}
	}
}
func TestEntropyAndCancellation(t *testing.T) {
	s := service(t)
	s.Entropy = strings.NewReader("")
	if _, err := s.Generate(context.Background(), app.Request{Count: 1}); err == nil || !strings.Contains(err.Error(), "acquire root seed") {
		t.Fatal(err)
	}
	if _, err := s.Generate(context.Background(), app.Request{Count: 1, Seed: ptr(uint64(0))}); err != nil {
		t.Fatal("explicit seed used entropy", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Generate(ctx, app.Request{Count: 1}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

type foreign struct {
	fail    bool
	cancel  context.CancelFunc
	invalid bool
}

func (f foreign) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "state", AlgorithmVersion: "test-1", Description: "test-only method with puzzle-owned tokens"}
}
func (f foreign) Generate(ctx context.Context, rng random.Source, o puzzle.Options) (puzzle.Generated, error) {
	if o.Length != nil {
		return puzzle.Generated{}, fmt.Errorf("unexpected length")
	}
	if f.fail {
		return puzzle.Generated{}, fmt.Errorf("test generation failure")
	}
	if f.cancel != nil {
		f.cancel()
	}
	if f.invalid {
		return puzzle.Generated{Moves: []string{"bad token"}}, nil
	}
	return puzzle.Generated{Moves: []string{"tip++", "corner-"}}, nil
}
func TestForeignCapabilities(t *testing.T) {
	registration := func(g foreign) puzzle.Registration {
		return puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "test-puzzle", Name: "Non-cube test fixture", DefaultMethod: "state"}, Generators: []puzzle.Generator{g}}
	}
	r, err := puzzle.New(registration(foreign{}))
	if err != nil {
		t.Fatal(err)
	}
	s := app.Service{Registry: r, Version: "test"}
	req := app.Request{Puzzle: "test-puzzle", Count: 1, Seed: ptr(uint64(0))}
	result, err := s.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result.Parameters.Length != nil || result.Scrambles[0].Sequence != "tip++ corner-" {
		t.Fatal(result)
	}
	req.Length = ptr(25)
	if _, err := s.Generate(context.Background(), req); err == nil {
		t.Fatal("explicit length accepted")
	}
	req.Length = nil
	for _, g := range []foreign{{fail: true}, {invalid: true}} {
		s.Registry, err = puzzle.New(registration(g))
		if err != nil {
			t.Fatal(err)
		}
		result, err = s.Generate(context.Background(), req)
		if err == nil || len(result.Scrambles) != 0 {
			t.Fatal("partial result", result, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.Registry, err = puzzle.New(registration(foreign{cancel: cancel}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Generate(ctx, req); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func BenchmarkDefault(b *testing.B) {
	s := service(b)
	req := app.Request{Count: 1, Seed: ptr(uint64(42))}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.Generate(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkBatch1000(b *testing.B) {
	s := service(b)
	req := app.Request{Count: 1000, Seed: ptr(uint64(42))}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := s.Generate(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

func TestConcurrentIndependentCalls(t *testing.T) {
	s := service(t)
	req := app.Request{Count: 3, Seed: ptr(uint64(42))}
	want, err := s.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			got, err := s.Generate(context.Background(), req)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatal("concurrent generation differs", err)
			}
		})
	}
}

// A method can change how many draws it consumes for one scramble without
// disturbing subsequent scrambles' streams.
type drawVariant struct {
	calls *int
	extra bool
}

func (drawVariant) Descriptor() puzzle.GeneratorDescriptor {
	return puzzle.GeneratorDescriptor{ID: "draw-test", AlgorithmVersion: "1"}
}
func (g drawVariant) Generate(_ context.Context, rng random.Source, _ puzzle.Options) (puzzle.Generated, error) {
	*g.calls++
	if g.extra && *g.calls == 1 {
		for i := 0; i < 100; i++ {
			rng.Uint64()
		}
	}
	return puzzle.Generated{Moves: []string{fmt.Sprint(rng.Uint64())}}, nil
}
func TestOneScramblesDrawsDoNotAffectAnother(t *testing.T) {
	generate := func(extra bool) app.Result {
		calls := 0
		r, err := puzzle.New(puzzle.Registration{Puzzle: puzzle.Descriptor{ID: "draw-test", Name: "test", DefaultMethod: "draw-test"}, Generators: []puzzle.Generator{drawVariant{&calls, extra}}})
		if err != nil {
			t.Fatal(err)
		}
		s := app.Service{Registry: r}
		result, err := s.Generate(context.Background(), app.Request{Puzzle: "draw-test", Count: 3, Seed: ptr(uint64(42))})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	a, b := generate(false), generate(true)
	if a.Scrambles[0].Sequence == b.Scrambles[0].Sequence || !reflect.DeepEqual(a.Scrambles[1:], b.Scrambles[1:]) {
		t.Fatal("streams are coupled")
	}
}
