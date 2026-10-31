package randomstate

import (
	"context"
	"fmt"
	"github.com/unixpickle/gocube"
)

const searchNodeLimit = 50000000

type search333 struct {
	ctx      context.Context
	t        *tables333
	nodes    int
	err      error
	solution []gocube.Move
}

func (s *search333) visit() bool {
	s.nodes++
	if s.nodes&1023 == 0 {
		if err := s.ctx.Err(); err != nil {
			s.err = err
			return false
		}
	}
	if s.nodes > searchNodeLimit {
		s.err = fmt.Errorf("random-state search exceeded %d nodes", searchNodeLimit)
		return false
	}
	return s.err == nil
}
func solve333(ctx context.Context, target gocube.CubieCube, t *tables333) ([]gocube.Move, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s := &search333{ctx: ctx, t: t}
	phase := target.Phase1Cube()
	for depth := t.h1.LowerBound(&phase); depth <= 12; depth++ {
		if s.phase1(target, phase, depth, 0, make([]gocube.Move, 0, 30)) {
			return s.solution, nil
		}
		if s.err != nil {
			return nil, s.err
		}
	}
	return nil, fmt.Errorf("random-state search found no solution within the two-phase bounds")
}
func (s *search333) phase1(cube gocube.CubieCube, phase gocube.Phase1Cube, depth, lastFace int, path []gocube.Move) bool {
	if !s.visit() {
		return false
	}
	if s.t.h1.LowerBound(&phase) > depth {
		return false
	}
	if depth == 0 {
		x, y, z := phase.Solved()
		for axis, solved := range []bool{x, y, z} {
			if !solved {
				continue
			}
			p2, err := gocube.NewPhase2Cube(cube, axis)
			if err != nil {
				continue
			}
			max := 30 - len(path)
			for d := s.t.h2.LowerBound(&p2); d <= max; d++ {
				tail, ok := s.phase2(p2, d, 0, axis, make([]gocube.Move, 0, 30))
				if ok {
					s.solution = append(append([]gocube.Move{}, path...), tail...)
					return true
				}
				if s.err != nil {
					return false
				}
			}
		}
		return false
	}
	for m := gocube.Move(0); m < 18; m++ {
		if m.Face() == lastFace {
			continue
		}
		nextCube, nextPhase := cube, phase
		nextCube.Move(m)
		nextPhase.Move(m, s.t.p1)
		if s.phase1(nextCube, nextPhase, depth-1, m.Face(), append(path, m)) {
			return true
		}
		if s.err != nil {
			return false
		}
	}
	return false
}
func (s *search333) phase2(cube gocube.Phase2Cube, depth, lastFace, axis int, path []gocube.Move) ([]gocube.Move, bool) {
	if !s.visit() {
		return nil, false
	}
	if s.t.h2.LowerBound(&cube) > depth {
		return nil, false
	}
	if depth == 0 {
		if cube.Solved() {
			return append([]gocube.Move{}, path...), true
		}
		return nil, false
	}
	for m := gocube.Phase2Move(0); m < 10; m++ {
		if m.Face() == lastFace {
			continue
		}
		next := cube
		next.Move(m, s.t.p2)
		if result, ok := s.phase2(next, depth-1, m.Face(), axis, append(path, m.Move(axis))); ok {
			return result, true
		}
		if s.err != nil {
			return nil, false
		}
	}
	return nil, false
}
