package cube333

import (
	"fmt"
	"strings"

	"github.com/frivas/turnsmith/internal/cube"
)

// State stores facelets as seen looking directly at each face, in row-major
// order. Face values identify the original center color, not a moving position.
type State [6][9]cube.Face

func Solved() State {
	var state State
	for face := cube.U; face <= cube.B; face++ {
		for i := range state[face] {
			state[face][i] = face
		}
	}
	return state
}

func parseMove(token string) (cube.Move, error) {
	move := cube.Move{Layers: 1, Turns: cube.Clockwise}
	if len(token) < 1 || len(token) > 2 {
		return move, fmt.Errorf("unsupported diagram move %q", token)
	}
	face := strings.IndexByte("UDLRFB", token[0])
	if face < 0 {
		return move, fmt.Errorf("unsupported diagram move %q", token)
	}
	move.Face = cube.Face(face)
	if len(token) == 2 {
		switch token[1] {
		case '2':
			move.Turns = cube.Half
		case '\'':
			move.Turns = cube.Counterclockwise
		default:
			return move, fmt.Errorf("unsupported diagram move %q", token)
		}
	}
	return move, nil
}

// Face coordinates use x=right, y=up, z=front. Each face's row/column axes
// follow its outside viewpoint, including the reversed back-face columns.
func facelet(face cube.Face, index int) (position, normal [3]int) {
	r, c := index/3, index%3
	switch face {
	case cube.U:
		return [3]int{c - 1, 1, r - 1}, [3]int{0, 1, 0}
	case cube.D:
		return [3]int{c - 1, -1, 1 - r}, [3]int{0, -1, 0}
	case cube.L:
		return [3]int{-1, 1 - r, c - 1}, [3]int{-1, 0, 0}
	case cube.R:
		return [3]int{1, 1 - r, 1 - c}, [3]int{1, 0, 0}
	case cube.F:
		return [3]int{c - 1, 1 - r, 1}, [3]int{0, 0, 1}
	default: // B
		return [3]int{1 - c, 1 - r, -1}, [3]int{0, 0, -1}
	}
}

func locate(p, n [3]int) (cube.Face, int) {
	switch {
	case n[1] == 1:
		return cube.U, (p[2]+1)*3 + p[0] + 1
	case n[1] == -1:
		return cube.D, (1-p[2])*3 + p[0] + 1
	case n[0] == -1:
		return cube.L, (1-p[1])*3 + p[2] + 1
	case n[0] == 1:
		return cube.R, (1-p[1])*3 + 1 - p[2]
	case n[2] == 1:
		return cube.F, (1-p[1])*3 + p[0] + 1
	default: // B
		return cube.B, (1-p[1])*3 + 1 - p[0]
	}
}

// rotate is a clockwise quarter turn viewed from the positive end of an axis.
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

func (s *State) Apply(move cube.Move) error {
	if err := move.Validate(3); err != nil {
		return err
	}
	if move.Layers != 1 {
		return fmt.Errorf("step diagrams support outer-face moves only")
	}
	_, normal := facelet(move.Face, 4)
	axis := 0
	for normal[axis] == 0 {
		axis++
	}
	side := normal[axis]
	turns := (int(move.Turns)*side + 4) % 4
	next := *s
	for face := cube.U; face <= cube.B; face++ {
		for i, color := range s[face] {
			position, direction := facelet(face, i)
			if position[axis] != side {
				continue
			}
			for turn := 0; turn < turns; turn++ {
				position, direction = rotate(position, axis), rotate(direction, axis)
			}
			toFace, toIndex := locate(position, direction)
			next[toFace][toIndex] = color
		}
	}
	*s = next
	return nil
}
