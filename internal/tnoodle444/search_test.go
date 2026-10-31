// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Francisco Rivas and contributors.
package tnoodle444

import (
	"context"
	"errors"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/frivas/turnsmith/internal/randomstate"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNative444RandomState(t *testing.T) {
	if testing.Short() {
		t.Skip("full native table and random-state search")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	start := time.Now()
	for seed := uint64(0); seed < 100; seed++ {
		tokens, err := Generate(ctx, random.Derive(seed, "444", "wca", "1", 0), randomstate.Solve333Facelets)
		if err != nil {
			t.Fatalf("seed%d: %v", seed, err)
		}
		if seed == 0 {
			t.Logf("native444 first target %s: %v", time.Since(start), tokens)
		}
		if len(tokens) < 2 || len(tokens) > 128 {
			t.Fatal("invalid scramble length", seed, len(tokens))
		}
	}
	t.Logf("100 independently verified native random-state scrambles: %s", time.Since(start))
}

func TestEdge3CoordinateRoundTrips(t *testing.T) {
	e := newEdge3()
	for i := 0; i < edge3FactX[12]; i += 7919 {
		e.set(i)
		if got := e.get(10); got != i {
			t.Fatalf("%d -> %d", i, got)
		}
	}
}
func TestEdge3TransitionsMatchFullCube(t *testing.T) {
	c := newFullCube()
	for step := 0; step < 100; step++ {
		for m := 0; m < 20; m++ {
			e := newEdge3()
			e.setEdge(c.getEdge())
			e.move(m)
			next := newFullCubeCopy(c)
			next.doMove(movesMove3std[m])
			actual := newEdge3()
			actual.setEdge(next.getEdge())
			if e.get(10) != actual.get(10) {
				t.Fatalf("step%d move%d %v != %v", step, m, e.edge, actual.edge)
			}
		}
		c.doMove(movesMove3std[(step*17+7)%20])
	}
}

func TestEdge3PackedPruning(t *testing.T) {
	table := make([]uint32, 3)
	for i := range table {
		table[i] = ^uint32(0)
	}
	for i := 0; i < 48; i++ {
		edge3SetPruning(table, i, i%3)
	}
	for i := 0; i < 48; i++ {
		if got := edge3GetPruning(table, i); got != i%3 {
			t.Fatal(i, got)
		}
	}
}
func TestEdge3MoveRotationCoordinates(t *testing.T) {
	ctx := context.Background()
	if err := edge3InitMvrot(ctx); err != nil {
		t.Fatal(err)
	}
	e := newEdge3()
	for index := 0; index < edge3FactX[12]; index += 1012349 {
		e.set(index)
		for move := 0; move < 20; move++ {
			for rotation := 0; rotation < 8; rotation++ {
				next := newEdge3()
				next.copy(e)
				next.move(move)
				next.rotate(rotation)
				for _, end := range []int{4, 10} {
					expected := next.get(end)
					actual := edge3Getmvrot(e.edge[:], move<<3|rotation, end)
					if actual != expected {
						t.Fatal("move/rotation coordinate mismatch", index, move, rotation, end, actual, expected)
					}
				}
			}
		}
	}
}

func Test444ShallowThresholdModuloRotation(t *testing.T) {
	rotations := searchRotationPaths()
	if len(rotations) != 24 {
		t.Fatal("rotation group", len(rotations))
	}
	excluded := searchShallowSet(rotations)
	if len(excluded) != 28 {
		t.Fatal("one-move rotation classes", len(excluded))
	}
	solved, _ := cubestate.New(4)
	for _, face := range []byte("UDLRFB") {
		for width := 1; width <= 3; width++ {
			for turns := 1; turns <= 3; turns++ {
				state := solved.Clone()
				if err := state.Apply(alg.Move{Face: face, Width: width, Turns: turns}); err != nil {
					t.Fatal(err)
				}
				if !excluded[searchCanonical(state, rotations)] {
					t.Fatal("accepted a one-move state", face, width, turns)
				}
			}
		}
	}
}

type countSource struct {
	source random.Source
	count  int
}

func (s *countSource) Uint64() uint64 { s.count++; return s.source.Uint64() }
func Test444SolverFailureDoesNotReplaceTarget(t *testing.T) {
	ctx := context.Background()
	stream := &countSource{source: random.Derive(0, "444", "wca", "1", 0)}
	_, err := newFullCubeRandomContext(ctx, stream)
	if err != nil {
		t.Fatal(err)
	}
	expectedDraws := stream.count
	stream = &countSource{source: random.Derive(0, "444", "wca", "1", 0)}
	failure := errors.New("injected solver failure")
	calls := 0
	_, err = Generate(ctx, stream, func(context.Context, string) ([]string, error) { calls++; return nil, failure })
	if !errors.Is(err, failure) || calls != 1 || stream.count != expectedDraws {
		t.Fatal("solver failure replaced sample", err, calls, stream.count, expectedDraws)
	}
}
func Test444SearchBoundsAndCancellation(t *testing.T) {
	s := newSearch()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.ctx = ctx
	s.nodes = 1023
	if s.visit() || !errors.Is(s.err, context.Canceled) {
		t.Fatal("search ignored cancellation", s.err)
	}
	s = newSearch()
	s.ctx = context.Background()
	s.nodes = searchMaximumNodes
	if s.visit() || s.err == nil {
		t.Fatal("search ignored node limit")
	}
	if _, err := Generate(ctx, random.Derive(0, "444", "wca", "1", 0), randomstate.Solve333Facelets); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := Generate(context.Background(), nil, nil); err == nil {
		t.Fatal("accepted missing callback")
	}
}

func TestEdge3PruningCoverage(t *testing.T) {
	if err := initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	known := 0
	for i := 0; i < edge3NEprun; i++ {
		if edge3GetPruning(edge3Eprun, i) != 3 {
			known++
		}
	}
	if known != 2778197 {
		t.Fatal("edge pruning coverage differs from pinned TNoodle depth-nine count", known)
	}
	if edge3Getprun(0) != 0 {
		t.Fatal("solved pruning distance")
	}
	e := newEdge3()
	for coordinate := 0; coordinate < edge3FactX[12]; coordinate += 1234567 {
		e.set(coordinate)
		sym := e.getsym()
		distance := edge3Getprun(sym)
		if distance < 0 || distance > edge3MaxDepth {
			t.Fatal("invalid lower bound", sym, distance)
		}
	}
}

func TestNative444ConcurrentDeterminism(t *testing.T) {
	const workers = 4
	results := make(chan []string, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			tokens, err := Generate(context.Background(), random.Derive(1, "444", "wca", "1", 0), randomstate.Solve333Facelets)
			results <- tokens
			errs <- err
		}()
	}
	var expected []string
	for i := 0; i < workers; i++ {
		tokens := <-results
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		if expected == nil {
			expected = tokens
		} else if !reflect.DeepEqual(tokens, expected) {
			t.Fatal("concurrent generation changed sequence", tokens, expected)
		}
	}
}
func BenchmarkNative444Warm(b *testing.B) {
	if err := initialize(context.Background()); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Generate(context.Background(), random.Derive(uint64(i), "444", "wca", "1", 0), randomstate.Solve333Facelets); err != nil {
			b.Fatal(err)
		}
	}
}

func TestNative444Golden(t *testing.T) {
	tokens, err := Generate(context.Background(), random.Derive(0, "444", "wca", "1", 0), randomstate.Solve333Facelets)
	if err != nil {
		t.Fatal(err)
	}
	const want = "D2 L2 B2 U2 D2 R F2 R' B2 L' F2 D2 F' D U B' L F' B' D Uw2 Fw2 L2 U' Rw2 L' Uw2 Fw2 D' B2 L2 Fw2 F' U' Fw D2 B Uw R' U2 Rw U2 Fw F2 Rw"
	if strings.Join(tokens, " ") != want {
		t.Fatal("native444 algorithm1 changed", tokens)
	}
}
