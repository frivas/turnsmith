// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2014 Shuang Chen.
// Go adaptation of TNoodle threephase at d01a947d9f028f38085cda9b0507a9cf3d3f38a8
// by Francisco Rivas and contributors. See COPYING for the GNU GPL terms.
package tnoodle444

import (
	"context"
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cube"
	"github.com/frivas/turnsmith/internal/cubestate"
	"github.com/frivas/turnsmith/internal/random"
	"sort"
	"strings"
	"sync"
)

// Solve333 solves a reduced cube, represented by 54 URFDLB facelet characters.
// The caller supplies the native bounded solver; no subprocess is used here.
type Solve333 func(context.Context, string) ([]string, error)

const searchPhase1Solutions = 10000
const searchPhase2Attempts = 500
const searchPhase2Solutions = 100
const searchPhase3Attempts = 100
const searchMaximumNodes = 1000000000
const searchMaximumPhase1 = 12
const searchMaximumPhase2 = 20
const searchMaximumPhase3 = 20

type Search struct {
	ctx              context.Context
	solve333         Solve333
	nodes            int
	err              error
	c, c1, c2        *FullCube
	ct2              *Center2
	ct3              *Center3
	e12              *Edge3
	tempe            [21]*Edge3
	move1            [40]int
	move2            [40]int
	move3            [40]int
	length1, length2 int
	add1             bool
	p1SolsCnt        int
	p1sols           []*FullCube
	arr2             []*FullCube
}

func newSearch() *Search {
	s := &Search{c1: newFullCube(), c2: newFullCube(), ct2: newCenter2(), ct3: newCenter3(), e12: newEdge3()}
	for i := range s.tempe {
		s.tempe[i] = newEdge3()
	}
	return s
}
func (s *Search) visit() bool {
	s.nodes++
	if s.err != nil {
		return false
	}
	if s.nodes&1023 == 0 {
		if err := s.ctx.Err(); err != nil {
			s.err = err
			return false
		}
	}
	if s.nodes > searchMaximumNodes {
		s.err = fmt.Errorf("4x4 search exceeded %d nodes", searchMaximumNodes)
		return false
	}
	return true
}

var searchInitOnce sync.Once
var searchInitReady chan struct{}
var searchInitError error

func initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	searchInitOnce.Do(func() {
		searchInitReady = make(chan struct{})
		go func() { searchInitError = buildTables(context.Background()); close(searchInitReady) }()
	})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-searchInitReady:
		return searchInitError
	}
}
func buildTables(ctx context.Context) error {
	if err := center1InitSym(ctx); err != nil {
		return err
	}
	center1Raw2sym = make([]int, 735471)
	if err := center1InitSym2Raw(ctx); err != nil {
		return err
	}
	if err := center1CreateMoveTable(ctx); err != nil {
		return err
	}
	center1Raw2sym = nil
	for _, build := range []func(context.Context) error{center1CreatePrun, center2Init, center3Init, edge3InitMvrot, edge3InitRaw2Sym, edge3CreatePrun} {
		if err := build(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Generate samples a native random 4x4 target, rejects only states fewer than two
// face turns from solved, reduces it in three bounded phases, then inverts its
// solution. Solver failures are returned without replacing the sampled target.
func Generate(ctx context.Context, rng random.Source, solve333 Solve333) ([]string, error) {
	if solve333 == nil {
		return nil, fmt.Errorf("4x4 generation requires a 3x3 solver callback")
	}
	target, err := newFullCubeRandomContext(ctx, rng)
	if err != nil {
		return nil, err
	}
	// Compare rotation classes, since the 4x4 has no fixed single-face centers.
	rotations, shallow := searchGeometryTables()
	for rejected := 0; shallow[searchCanonical(searchGeometric(target), rotations)]; rejected++ {
		if rejected >= 1023 {
			return nil, fmt.Errorf("4x4 sampler rejected 1024 near-solved targets")
		}
		target, err = newFullCubeRandomContext(ctx, rng)
		if err != nil {
			return nil, err
		}
	}
	if err := initialize(ctx); err != nil {
		return nil, err
	}
	s := newSearch()
	s.ctx = ctx
	s.solve333 = solve333
	s.c = target
	result, err := s.doSearch()
	if err != nil {
		return nil, err
	}
	moves, err := alg.Parse(strings.Join(result, " "), 128)
	if err != nil {
		return nil, fmt.Errorf("4x4 solver returned invalid notation: %w", err)
	}
	states, err := cubestate.Replay(4, moves)
	if err != nil {
		return nil, err
	}
	if searchCanonical(states[len(states)-1], rotations) != searchCanonical(searchGeometric(target), rotations) {
		return nil, fmt.Errorf("4x4 scramble failed independent target verification")
	}
	return result, nil
}

func (s *Search) doSearch() ([]string, error) {
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	ud := newCenter1FromCube(s.c.getCenter(), 0).getsym()
	fb := newCenter1FromCube(s.c.getCenter(), 1).getsym()
	rl := newCenter1FromCube(s.c.getCenter(), 2).getsym()
	udprun, fbprun, rlprun := center1Csprun[ud>>6], center1Csprun[fb>>6], center1Csprun[rl>>6]
	found := false
	for s.length1 = min(udprun, fbprun, rlprun); s.length1 <= searchMaximumPhase1; s.length1++ {
		if (rlprun <= s.length1 && s.search1(rl>>6, rl&63, s.length1, -1, 0)) || (udprun <= s.length1 && s.search1(ud>>6, ud&63, s.length1, -1, 0)) || (fbprun <= s.length1 && s.search1(fb>>6, fb&63, s.length1, -1, 0)) {
			found = true
			break
		}
		if s.err != nil {
			return nil, s.err
		}
	}
	if !found || len(s.p1sols) == 0 {
		return nil, fmt.Errorf("4x4 phase one exhausted depth %d", searchMaximumPhase1)
	}
	sort.SliceStable(s.p1sols, func(i, j int) bool { return s.p1sols[i].value < s.p1sols[j].value })
	found = false
phase2:
	for maximum := 9; maximum <= searchMaximumPhase2; maximum++ {
		for total := s.p1sols[0].value; total <= searchMaximumPhase1+maximum; total++ {
			for _, candidate := range s.p1sols {
				if candidate.value > total {
					break
				}
				if total-candidate.length1 > maximum {
					continue
				}
				s.c1.copy(candidate)
				s.ct2.set(s.c1.getCenter(), s.c1.getEdge().getParity())
				ct, rl := s.ct2.getct(), s.ct2.getrl()
				s.length1 = candidate.length1
				s.length2 = total - candidate.length1
				if s.search2(ct, rl, s.length2, 28, 0) {
					found = true
					break phase2
				}
				if s.err != nil {
					return nil, s.err
				}
			}
		}
	}
	if !found || len(s.arr2) == 0 {
		return nil, fmt.Errorf("4x4 phase two exhausted depth %d", searchMaximumPhase2)
	}
	sort.SliceStable(s.arr2, func(i, j int) bool { return s.arr2[i].value < s.arr2[j].value })
	var selected *FullCube
	selectedLength := 0
phase3:
	for maximum := 13; maximum <= searchMaximumPhase3; maximum++ {
		for total := s.arr2[0].value; total <= searchMaximumPhase1+searchMaximumPhase2+maximum; total++ {
			for _, candidate := range s.arr2[:min(len(s.arr2), searchPhase3Attempts)] {
				if candidate.value > total {
					break
				}
				length := total - candidate.length1 - candidate.length2
				if length > maximum {
					continue
				}
				parity := s.e12.setEdge(candidate.getEdge())
				s.ct3.set(candidate.getCenter(), parity^candidate.getCorner().getParity())
				ct := s.ct3.getct()
				edge := s.e12.get(10)
				prun := edge3Getprun(s.e12.getsym())
				if prun <= length && s.search3(edge, ct, prun, length, 20, 0) {
					selected = candidate
					selectedLength = length
					break phase3
				}
				if s.err != nil {
					return nil, s.err
				}
			}
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("4x4 phase three exhausted depth %d", searchMaximumPhase3)
	}
	solutionCube := newFullCubeCopy(selected)
	for i := 0; i < selectedLength; i++ {
		solutionCube.move(movesMove3std[s.move3[i]])
	}
	facelets := solutionCube.to333Facelet()
	tail, err := s.solve333(s.ctx, facelets)
	if err != nil {
		return nil, fmt.Errorf("4x4 reduced 3x3 solve: %w", err)
	}
	if len(tail) > 30 {
		return nil, fmt.Errorf("4x4 reduced solver exceeded 30 moves")
	}
	for _, token := range tail {
		parsed, parseErr := alg.Parse(token, 1)
		if parseErr != nil || len(parsed) != 1 || parsed[0].Width != 1 || !strings.ContainsRune("UDLRFB", rune(parsed[0].Face)) {
			return nil, fmt.Errorf("invalid reduced 3x3 solution token %q", token)
		}
		moves := utilTomove(parsed[0].String())
		if len(moves) != 1 || moves[0] < 0 || moves[0] >= 18 {
			return nil, fmt.Errorf("invalid reduced 3x3 solution token %q", token)
		}
		solutionCube.move(moves[0])
	}
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	return strings.Fields(solutionCube.getMoveString(true, false)), nil
}

func (s *Search) search1(ct, sym, maxl, lm, depth int) bool {
	if !s.visit() {
		return false
	}
	if ct == 0 && maxl < 5 {
		return maxl == 0 && s.init2(sym, lm)
	}
	for axis := 0; axis < 27; axis += 3 {
		if axis == lm || axis == lm-9 || axis == lm-18 {
			continue
		}
		for power := 0; power < 3; power++ {
			m := axis + power
			next := center1Ctsmv[ct][center1Symmove[sym][m]]
			prun := center1Csprun[next>>6]
			if prun >= maxl {
				if prun > maxl {
					break
				}
				continue
			}
			nextSym := center1Symmult[sym][next&63]
			next >>= 6
			s.move1[depth] = m
			if s.search1(next, nextSym, maxl-1, axis, depth+1) {
				return true
			}
			if s.err != nil {
				return false
			}
		}
	}
	return false
}
func (s *Search) init2(sym, lm int) bool {
	s.c1.copy(s.c)
	for i := 0; i < s.length1; i++ {
		s.c1.move(s.move1[i])
	}
	switch center1Finish[sym] {
	case 0:
		s.c1.move(fx1)
		s.c1.move(bx3)
		s.move1[s.length1] = fx1
		s.move1[s.length1+1] = bx3
		s.add1 = true
		sym = 19
	case 12869:
		s.c1.move(ux1)
		s.c1.move(dx3)
		s.move1[s.length1] = ux1
		s.move1[s.length1+1] = dx3
		s.add1 = true
		sym = 34
	case 735470:
		s.add1 = false
		sym = 0
	default:
		s.err = fmt.Errorf("4x4 phase-one finish symmetry is invalid")
		return false
	}
	s.ct2.set(s.c1.getCenter(), s.c1.getEdge().getParity())
	ct, rl := s.ct2.getct(), s.ct2.getrl()
	prun := center2Ctprun[ct*70+rl]
	s.c1.value = prun + s.length1
	s.c1.length1 = s.length1
	s.c1.add1 = s.add1
	s.c1.sym = sym
	s.p1SolsCnt++
	if len(s.p1sols) < searchPhase2Attempts {
		s.p1sols = append(s.p1sols, newFullCubeCopy(s.c1))
	} else {
		worst := 0
		for i := 1; i < len(s.p1sols); i++ {
			if s.p1sols[i].value > s.p1sols[worst].value {
				worst = i
			}
		}
		if s.p1sols[worst].value > s.c1.value {
			s.p1sols[worst].copy(s.c1)
		}
	}
	return s.p1SolsCnt == searchPhase1Solutions
}
func (s *Search) search2(ct, rl, maxl, lm, depth int) bool {
	if !s.visit() {
		return false
	}
	if ct == 0 && center2Ctprun[rl] == 0 && maxl == 0 {
		return s.init3()
	}
	for m := 0; m < 23; m++ {
		if movesCkmv2[lm][m] {
			m = movesSkipAxis2[m]
			continue
		}
		next := center2Ctmv[ct][m]
		nextRl := center2Rlmv[rl][m]
		prun := center2Ctprun[next*70+nextRl]
		if prun >= maxl {
			if prun > maxl {
				m = movesSkipAxis2[m]
			}
			continue
		}
		s.move2[depth] = movesMove2std[m]
		if s.search2(next, nextRl, maxl-1, m, depth+1) {
			return true
		}
		if s.err != nil {
			return false
		}
	}
	return false
}
func (s *Search) init3() bool {
	s.c2.copy(s.c1)
	for i := 0; i < s.length2; i++ {
		s.c2.move(s.move2[i])
	}
	if !s.c2.checkEdge() {
		return false
	}
	parity := s.e12.setEdge(s.c2.getEdge())
	s.ct3.set(s.c2.getCenter(), parity^s.c2.getCorner().getParity())
	ct := s.ct3.getct()
	prun := edge3Getprun(s.e12.getsym())
	candidate := newFullCubeCopy(s.c2)
	candidate.value = s.length1 + s.length2 + max(prun, center3Prun[ct])
	candidate.length2 = s.length2
	s.arr2 = append(s.arr2, candidate)
	return len(s.arr2) == searchPhase2Solutions
}
func (s *Search) search3(edge, ct, prun, maxl, lm, depth int) bool {
	if !s.visit() {
		return false
	}
	if maxl == 0 {
		return edge == 0 && ct == 0
	}
	s.tempe[depth].set(edge)
	for m := 0; m < 17; m++ {
		if movesCkmv3[lm][m] {
			m = movesSkipAxis3[m]
			continue
		}
		next := center3Ctmove[ct][m]
		prun1 := center3Prun[next]
		if prun1 >= maxl {
			if prun1 > maxl && m < 14 {
				m = movesSkipAxis3[m]
			}
			continue
		}
		nextEdge := edge3Getmvrot(s.tempe[depth].edge[:], m<<3, 10)
		coord1 := nextEdge / edge3NRaw
		symcoord1 := edge3Raw2sym[coord1]
		sym := symcoord1 & 7
		symcoord1 >>= 3
		coord2 := edge3Getmvrot(s.tempe[depth].edge[:], m<<3|sym, 10) % edge3NRaw
		nextPrun := edge3Getprun(symcoord1*edge3NRaw+coord2, prun)
		if nextPrun >= maxl {
			if nextPrun > maxl && m < 14 {
				m = movesSkipAxis3[m]
			}
			continue
		}
		if s.search3(nextEdge, next, nextPrun, maxl-1, m, depth+1) {
			s.move3[depth] = m
			return true
		}
		if s.err != nil {
			return false
		}
	}
	return false
}

func searchGeometric(c *FullCube) cubestate.State {
	state, _ := cubestate.New(4)
	facelets := c.facelets()
	faces := [6]cube.Face{cube.U, cube.R, cube.F, cube.D, cube.L, cube.B}
	for f, face := range faces {
		for i := 0; i < 16; i++ {
			state.Faces[face][i] = faces[facelets[f*16+i]]
		}
	}
	return state
}
func searchRotationPaths() [][]alg.Move {
	solved, _ := cubestate.New(4)
	paths := [][]alg.Move{{}}
	states := []cubestate.State{solved}
	seen := map[string]bool{solved.Facelets(): true}
	for front := 0; front < len(states); front++ {
		for _, face := range []byte{'x', 'y', 'z'} {
			move := alg.Move{Face: face, Width: 1, Turns: 1}
			next := states[front].Clone()
			_ = next.Apply(move)
			key := next.Facelets()
			if !seen[key] {
				seen[key] = true
				states = append(states, next)
				paths = append(paths, append(append([]alg.Move{}, paths[front]...), move))
			}
		}
	}
	return paths
}
func searchCanonical(state cubestate.State, rotations [][]alg.Move) string {
	best := ""
	for _, path := range rotations {
		next := state.Clone()
		for _, move := range path {
			_ = next.Apply(move)
		}
		key := next.Facelets()
		if best == "" || key < best {
			best = key
		}
	}
	return best
}
func searchShallowSet(rotations [][]alg.Move) map[string]bool {
	solved, _ := cubestate.New(4)
	result := map[string]bool{searchCanonical(solved, rotations): true}
	for _, face := range []byte("UDLRFB") {
		for width := 1; width <= 2; width++ {
			for turns := 1; turns <= 3; turns++ {
				state := solved.Clone()
				_ = state.Apply(alg.Move{Face: face, Width: width, Turns: turns})
				result[searchCanonical(state, rotations)] = true
			}
		}
	}
	return result
}

var searchGeometryOnce sync.Once
var searchRotations [][]alg.Move
var searchShallow map[string]bool

func searchGeometryTables() ([][]alg.Move, map[string]bool) {
	searchGeometryOnce.Do(func() { searchRotations = searchRotationPaths(); searchShallow = searchShallowSet(searchRotations) })
	return searchRotations, searchShallow
}
