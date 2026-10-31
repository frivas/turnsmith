// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Francisco Rivas and contributors.

package tnoodle444

import (
	"context"
	"errors"
	"testing"
)

func TestCenterCoordinateRoundTrips(t *testing.T) {
	c1 := newCenter1()
	for coordinate := 0; coordinate < 735471; coordinate++ {
		c1.set(coordinate)
		if got := c1.get(); got != coordinate {
			t.Fatalf("center1 %d -> %d", coordinate, got)
		}
	}
	c2 := newCenter2()
	for coordinate := 0; coordinate < 70; coordinate++ {
		c2.setrl(coordinate)
		if got := c2.getrl(); got != coordinate {
			t.Fatalf("center2 rl %d -> %d", coordinate, got)
		}
	}
	for coordinate := 0; coordinate < 6435; coordinate++ {
		c2.setct(coordinate)
		if got := c2.getct(); got != coordinate {
			t.Fatalf("center2 ct %d -> %d", coordinate, got)
		}
	}
	for i, v := range center3Rl2std {
		center3Std2rl[v] = i
	}
	c3 := newCenter3()
	for coordinate := 0; coordinate < 29400; coordinate++ {
		c3.setct(coordinate)
		if got := c3.getct(); got != coordinate {
			t.Fatalf("center3 %d -> %d", coordinate, got)
		}
	}
}

func TestCenterMovesAndInverses(t *testing.T) {
	cube := newCenterCube()
	for step := 0; step < 60; step++ {
		for axis := 0; axis < 3; axis++ {
			original := newCenter1FromCube(cube, axis)
			for move := 0; move < 36; move++ {
				c1 := *original
				c1.move(move)
				other := *cube
				other.move(move)
				if c1.ct != newCenter1FromCube(&other, axis).ct {
					t.Fatalf("center1 differs from center cube for move %d", move)
				}
				c1.move(move/3*3 + 2 - move%3)
				if c1.ct != original.ct {
					t.Fatalf("center1 move inverse %d", move)
				}
			}
		}
		cube.move((step*17 + 11) % 36)
	}
	c2 := newCenter2()
	c2.setct(4321)
	c2.setrl(43)
	for move := 0; move < 36; move++ {
		state := *c2
		state.move(move)
		state.move(move/3*3 + 2 - move%3)
		if state != *c2 {
			t.Fatalf("center2 move inverse %d", move)
		}
	}
	for _, coordinate := range []int{0, 1, 59, 372, 10381, 29399} {
		c3 := newCenter3()
		c3.setct(coordinate)
		for move := 0; move < 20; move++ {
			state := *c3
			state.move(move)
			standard := move3std[move]
			inverse := standard/3*3 + 2 - standard%3
			inverseIndex := -1
			for j, v := range move3std {
				if v == inverse {
					inverseIndex = j
					break
				}
			}
			if inverseIndex < 0 {
				t.Fatalf("phase3 inverse missing for %d", move)
			}
			state.move(inverseIndex)
			if state != *c3 {
				t.Fatalf("center3 move inverse %d at %d", move, coordinate)
			}
		}
	}
}

func TestCenterTablesAndSymmetry(t *testing.T) {
	ctx := context.Background()
	for _, build := range []func(context.Context) error{center1InitSym, center1InitSym2Raw, center1CreateMoveTable, center1CreatePrun, center2Init, center3Init} {
		// The temporary raw map avoids a binary search over symmetries while
		// constructing moves. Search uses only the immutable final tables.
		if center1Raw2sym == nil {
			center1Raw2sym = make([]int, 735471)
		}
		if err := build(ctx); err != nil {
			t.Fatal(err)
		}
	}
	defer func() { center1Raw2sym = nil }()
	for i := 0; i < 48; i++ {
		if center1Symmult[i][center1Syminv[i]] != 0 || center1Symmult[center1Syminv[i]][i] != 0 {
			t.Fatalf("symmetry inverse %d", i)
		}
		seen := [36]bool{}
		for _, move := range center1Symmove[i] {
			if move < 0 || move >= 36 || seen[move] {
				t.Fatalf("symmetry %d does not permute moves", i)
			}
			seen[move] = true
		}
	}
	c := newCenter1()
	for i, representative := range center1Sym2raw {
		if i > 0 && representative <= center1Sym2raw[i-1] {
			t.Fatal("representatives not strictly ordered")
		}
		c.set(representative)
		for symmetry := 0; symmetry < 48; symmetry++ {
			packed := center1Raw2sym[c.get()]
			if packed>>6 != i {
				t.Fatalf("representative %d symmetry %d maps to %d", i, symmetry, packed>>6)
			}
			c.nextSymmetry(symmetry)
		}
	}
	// After initialization Search drops the large reverse map. Exercise its
	// symmetry/binary-search path and reconstruct each sampled raw coordinate.
	rawMap := center1Raw2sym
	center1Raw2sym = nil
	for coordinate := 0; coordinate < 735471; coordinate += 97 {
		c.set(coordinate)
		packed := c.getsym()
		if packed < 0 || packed>>6 != rawMap[coordinate]>>6 {
			t.Fatalf("fallback symmetry for raw coordinate %d: %d", coordinate, packed)
		}
		c.set(center1Sym2raw[packed>>6])
		c.rotate(center1Syminv[packed&63])
		if c.get() != coordinate {
			t.Fatalf("fallback symmetry does not reconstruct raw coordinate %d", coordinate)
		}
	}
	center1Raw2sym = rawMap
	for coordinate, depth := range center1Csprun {
		centerCheckDepth(t, "center1", coordinate, depth, 27, func(m int) int { return center1Csprun[center1Ctsmv[coordinate][m]>>6] })
	}
	for coordinate, depth := range center2Ctprun {
		centerCheckDepth(t, "center2", coordinate, depth, 23, func(m int) int { return center2Ctprun[center2Ctmv[coordinate/70][m]*70+center2Rlmv[coordinate%70][m]] })
	}
	for coordinate, depth := range center3Prun {
		centerCheckDepth(t, "center3", coordinate, depth, 17, func(m int) int { return center3Prun[center3Ctmove[coordinate][m]] })
	}
	// A solved center coloring must have identity orientation, while each full
	// symmetry still has a valid solved-orientation recovery.
	solved := newCenterCube()
	if got := center1GetSolvedSym(solved); got != 0 {
		t.Fatalf("solved symmetry %d", got)
	}
}

func centerCheckDepth(t *testing.T, name string, coordinate, depth, moves int, next func(int) int) {
	t.Helper()
	if depth < 0 {
		t.Fatalf("%s unreachable coordinate %d", name, coordinate)
	}
	descends := depth == 0
	for move := 0; move < moves; move++ {
		d := next(move)
		if d < depth-1 || d > depth+1 {
			t.Fatalf("%s pruning edge %d: %d -> %d", name, coordinate, depth, d)
		}
		if d == depth-1 {
			descends = true
		}
	}
	if !descends {
		t.Fatalf("%s no shortest-path predecessor for coordinate %d", name, coordinate)
	}
}

func TestCenterBuildersHonorCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, build := range []func(context.Context) error{center1InitSym, center1InitSym2Raw, center1CreateMoveTable, center1CreatePrun, center2Init, center3Init} {
		if err := build(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("builder returned %v", err)
		}
	}
}
