// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Francisco Rivas and contributors.
// Tests for the Go adaptation of Shuang Chen's threephase representation.
package tnoodle444

import (
	"context"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/random"
	"reflect"
	"strings"
	"testing"
	"time"
)

func faceletString(c *FullCube) string {
	f := c.facelets()
	var b strings.Builder
	for _, v := range f {
		b.WriteByte("URFDLB"[v])
	}
	return b.String()
}
func independentFacelets(t *testing.T, sequence string) string {
	t.Helper()
	moves, e := alg.Parse(sequence, 1000)
	if e != nil {
		t.Fatal(e)
	}
	states, e := cubestate.Replay(4, moves)
	if e != nil {
		t.Fatal(e)
	}
	return states[len(states)-1].Facelets()
}
func TestAllMoveFaceletsAgainstIndependentGeometry(t *testing.T) {
	for m := 0; m < 36; m++ {
		c := newFullCube()
		c.doMove(m)
		token := strings.TrimSpace(move2str[m])
		if got, want := faceletString(c), independentFacelets(t, token); got != want {
			t.Fatalf("%s\ngot  %s\nwant %s", token, got, want)
		}
		c.doMove(m/3*3 + 2 - m%3)
		if got := faceletString(c); got != faceletString(newFullCube()) {
			t.Fatal("inverse", token)
		}
	}
}
func TestMixedMovesAndLazyCopy(t *testing.T) {
	rng := random.Derive(42, "444", "representation", "1", 0)
	c := newFullCube()
	var sequence []string
	for i := 0; i < 100; i++ {
		m := int(random.Below(rng, 36))
		c.doMove(m)
		sequence = append(sequence, strings.TrimSpace(move2str[m]))
		if got, want := faceletString(c), independentFacelets(t, strings.Join(sequence, " ")); got != want {
			t.Fatalf("mixed prefix %d differs", i)
		}
	}
	lazy := newFullCube()
	for _, m := range []int{rx1, Ux2, Fx3, ux1} {
		lazy.move(m)
	}
	copy := newFullCubeCopy(lazy)
	if faceletString(lazy) != faceletString(copy) {
		t.Fatal("pending moves were lost on copy")
	}
	copy.doMove(Bx1)
	if faceletString(lazy) == faceletString(copy) {
		t.Fatal("copy aliases source")
	}
	if got := newFullCube().to333Facelet(); got != "UUUUUUUUURRRRRRRRRFFFFFFFFFDDDDDDDDDLLLLLLLLLBBBBBBBBB" {
		t.Fatal(got)
	}
}
func TestRandomStatesRoundTripAndConstraints(t *testing.T) {
	rng := random.Derive(7, "444", "representation", "1", 0)
	wingParities, cornerParities := map[int]bool{}, map[int]bool{}
	for n := 0; n < 200; n++ {
		c := newFullCubeRandom(rng)
		var counts [6]int
		for _, v := range c.center.ct {
			if v < 0 || v >= 6 {
				t.Fatal(v)
			}
			counts[v]++
		}
		for _, v := range counts {
			if v != 4 {
				t.Fatal(counts)
			}
		}
		seen := map[int]bool{}
		for _, v := range c.edge.ep {
			if v < 0 || v >= 24 || seen[v] {
				t.Fatal(c.edge.ep)
			}
			seen[v] = true
		}
		seen = map[int]bool{}
		twist := 0
		for i, v := range c.corner.cp {
			if v < 0 || v >= 8 || seen[v] {
				t.Fatal(c.corner.cp)
			}
			seen[v] = true
			twist += c.corner.co[i]
		}
		if twist%3 != 0 {
			t.Fatal("unreachable corner orientation")
		}
		wingParities[c.edge.getParity()] = true
		cornerParities[c.corner.getParity()] = true
		f := c.facelets()
		round := newFullCubeFacelets(f[:])
		if !reflect.DeepEqual(c.edge.ep, round.edge.ep) || !reflect.DeepEqual(c.center.ct, round.center.ct) || !reflect.DeepEqual(c.corner.cp, round.corner.cp) || !reflect.DeepEqual(c.corner.co, round.corner.co) {
			t.Fatal("96 facelet conversion lost state")
		}
	}
	if len(wingParities) != 2 || len(cornerParities) != 2 {
		t.Fatal("sampler restricts independent permutation parity")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := newFullCubeRandomContext(ctx, rng); e != context.Canceled {
		t.Fatal(e)
	}
}
func TestAllCornerCoordinates(t *testing.T) {
	seen := map[[8]int]bool{}
	for idx := 0; idx < 40320; idx++ {
		c := newCornerCubeCoords(idx, 0)
		if seen[c.cp] {
			t.Fatal("duplicate corner permutation", idx)
		}
		seen[c.cp] = true
	}
	seen = map[[8]int]bool{}
	for idx := 0; idx < 2187; idx++ {
		c := newCornerCubeCoords(0, idx)
		sum := 0
		for _, v := range c.co {
			if v < 0 || v >= 3 {
				t.Fatal(v)
			}
			sum += v
		}
		if sum%3 != 0 || seen[c.co] {
			t.Fatal("invalid twist coordinate", idx, c.co)
		}
		seen[c.co] = true
	}
}
func TestMoveNotationAndTables(t *testing.T) {
	for m := 0; m < 36; m++ {
		parsed := utilTomove(strings.TrimSpace(move2str[m]))
		if len(parsed) != 1 || parsed[0] != m {
			t.Fatal(m, parsed)
		}
	}
	for i, m := range move2std {
		if std2move[m] != i {
			t.Fatal(i, m)
		}
	}
	for i, m := range move3std {
		if std3move[m] != i {
			t.Fatal(i, m)
		}
	}
	if Cnk[24][8] != 735471 || fact[8] != 40320 {
		t.Fatal("combinatoric constants", Cnk[24][8], fact[8])
	}
}

type fullCubeZeroSource struct {
	started chan struct{}
	once    bool
}

func (s *fullCubeZeroSource) Uint64() uint64 {
	if !s.once {
		s.once = true
		close(s.started)
	}
	return 0
}
func TestRandomConstructorCancellationDuringRejectedDraw(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := &fullCubeZeroSource{started: make(chan struct{})}
	done := make(chan error, 1)
	go func() { _, e := newFullCubeRandomContext(ctx, source); done <- e }()
	select {
	case <-source.started:
	case <-time.After(time.Second):
		t.Fatal("sampler did not start")
	}
	cancel()
	select {
	case e := <-done:
		if e != context.Canceled {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("rejection sampling ignored cancellation")
	}
}
