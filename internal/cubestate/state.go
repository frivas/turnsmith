// Package cubestate applies algorithms for display without influencing scrambles.
package cubestate

import (
	"fmt"
	"github.com/frivas/turnsmith/internal/alg"
	"github.com/frivas/turnsmith/internal/cube"
	"strings"
)

type State struct {
	Size  int
	Faces [6][]cube.Face
}

func New(size int) (State, error) {
	if size < 2 || size > 7 {
		return State{}, fmt.Errorf("cube size must be between 2 and 7")
	}
	s := State{Size: size}
	for f := range s.Faces {
		s.Faces[f] = make([]cube.Face, size*size)
		for i := range s.Faces[f] {
			s.Faces[f][i] = cube.Face(f)
		}
	}
	return s, nil
}
func (s State) Clone() State {
	for f := range s.Faces {
		s.Faces[f] = append([]cube.Face(nil), s.Faces[f]...)
	}
	return s
}
func (s State) Solved() bool {
	for f := range s.Faces {
		for _, c := range s.Faces[f] {
			if c != cube.Face(f) {
				return false
			}
		}
	}
	return true
}
func (s State) Facelets() string {
	var b strings.Builder
	for _, f := range []cube.Face{cube.U, cube.R, cube.F, cube.D, cube.L, cube.B} {
		for _, color := range s.Faces[f] {
			b.WriteByte("UDLRFB"[color])
		}
	}
	return b.String()
}
func (s State) point(f cube.Face, i int) (p, n [3]int) {
	r, c := i/s.Size, i%s.Size
	k := s.Size - 1
	x, y := 2*c-k, k-2*r
	switch f {
	case cube.U:
		return [3]int{x, k, 2*r - k}, [3]int{0, 1, 0}
	case cube.D:
		return [3]int{x, -k, k - 2*r}, [3]int{0, -1, 0}
	case cube.L:
		return [3]int{-k, y, 2*c - k}, [3]int{-1, 0, 0}
	case cube.R:
		return [3]int{k, y, k - 2*c}, [3]int{1, 0, 0}
	case cube.F:
		return [3]int{x, y, k}, [3]int{0, 0, 1}
	default:
		return [3]int{k - 2*c, y, -k}, [3]int{0, 0, -1}
	}
}
func (s State) locate(p, n [3]int) (cube.Face, int) {
	k := s.Size - 1
	var f cube.Face
	var r, c int
	switch {
	case n[1] == 1:
		f = cube.U
		r = (p[2] + k) / 2
		c = (p[0] + k) / 2
	case n[1] == -1:
		f = cube.D
		r = (k - p[2]) / 2
		c = (p[0] + k) / 2
	case n[0] == -1:
		f = cube.L
		r = (k - p[1]) / 2
		c = (p[2] + k) / 2
	case n[0] == 1:
		f = cube.R
		r = (k - p[1]) / 2
		c = (k - p[2]) / 2
	case n[2] == 1:
		f = cube.F
		r = (k - p[1]) / 2
		c = (p[0] + k) / 2
	default:
		f = cube.B
		r = (k - p[1]) / 2
		c = (k - p[0]) / 2
	}
	return f, r*s.Size + c
}
func rotate(p [3]int, axis int) [3]int {
	switch axis {
	case 0:
		return [3]int{p[0], p[2], -p[1]}
	case 1:
		return [3]int{-p[2], p[1], p[0]}
	default:
		return [3]int{p[1], -p[0], p[2]}
	}
}
func (s *State) Apply(m alg.Move) error {
	if m.Turns < 1 || m.Turns > 3 || m.Width < 1 {
		return fmt.Errorf("invalid move")
	}
	axis, side := 0, 1
	all, middle := false, false
	switch m.Face {
	case 'U':
		axis = 1
	case 'D':
		axis = 1
		side = -1
	case 'L':
		side = -1
	case 'R':
	case 'F':
		axis = 2
	case 'B':
		axis = 2
		side = -1
	case 'x':
		all = true
	case 'y':
		axis = 1
		all = true
	case 'z':
		axis = 2
		all = true
	case 'M':
		side = -1
		middle = true
	case 'E':
		axis = 1
		side = -1
		middle = true
	case 'S':
		axis = 2
		middle = true
	default:
		return fmt.Errorf("unsupported move %s", m.String())
	}
	if middle && s.Size%2 == 0 {
		return fmt.Errorf("slice move %s requires an odd cube size", m.String())
	}
	if !all && !middle && m.Width >= s.Size {
		return fmt.Errorf("move %s is too wide for %dx%d; use a cube rotation for all layers", m.String(), s.Size, s.Size)
	}
	next := s.Clone()
	turns := (m.Turns*side + 4) % 4
	threshold := s.Size - 1 - 2*(m.Width-1)
	for f := cube.U; f <= cube.B; f++ {
		for i, color := range s.Faces[f] {
			p, n := s.point(f, i)
			selected := all || (middle && p[axis] == 0) || (!middle && p[axis]*side >= threshold)
			if !selected {
				continue
			}
			for t := 0; t < turns; t++ {
				p, n = rotate(p, axis), rotate(n, axis)
			}
			to, j := s.locate(p, n)
			next.Faces[to][j] = color
		}
	}
	*s = next
	return nil
}
func Replay(size int, moves []alg.Move) ([]State, error) {
	s, e := New(size)
	if e != nil {
		return nil, e
	}
	states := make([]State, 1, len(moves)+1)
	states[0] = s.Clone()
	for i, m := range moves {
		if e = s.Apply(m); e != nil {
			return nil, fmt.Errorf("move %d (%s): %w", i+1, m.String(), e)
		}
		states = append(states, s.Clone())
	}
	return states, nil
}
func Size(puzzle string) (int, error) {
	switch puzzle {
	case "222", "2x2", "2x2x2":
		return 2, nil
	case "333", "3x3", "3x3x3", "":
		return 3, nil
	case "444", "4x4", "4x4x4":
		return 4, nil
	default:
		return 0, fmt.Errorf("no cube view for puzzle %q", puzzle)
	}
}
