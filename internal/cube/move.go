// Package cube contains size-aware cube moves, independent of generators.
package cube

import "fmt"

type Face uint8

const (
	U Face = iota
	D
	L
	R
	F
	B
)

type Turn uint8

const (
	Clockwise        Turn = 1
	Half             Turn = 2
	Counterclockwise Turn = 3
)

type Move struct {
	Face   Face
	Layers uint8
	Turns  Turn
}

func (f Face) Opposite() Face { return f ^ 1 }
func (m Move) Validate(size int) error {
	if size < 2 || m.Face > B || m.Layers == 0 || int(m.Layers) >= size || m.Turns < Clockwise || m.Turns > Counterclockwise {
		return fmt.Errorf("invalid move for cube size %d", size)
	}
	return nil
}
func (m Move) Notation(size int) (string, error) {
	if err := m.Validate(size); err != nil {
		return "", err
	}
	if m.Layers == 1 {
		tokens := [6][3]string{{"U", "U2", "U'"}, {"D", "D2", "D'"}, {"L", "L2", "L'"}, {"R", "R2", "R'"}, {"F", "F2", "F'"}, {"B", "B2", "B'"}}
		return tokens[m.Face][m.Turns-1], nil
	}
	face := string("UDLRFB"[m.Face])
	if m.Layers == 2 {
		face += "w"
	} else if m.Layers > 2 {
		face = fmt.Sprintf("%d%sw", m.Layers, face)
	}
	if m.Turns == Half {
		face += "2"
	} else if m.Turns == Counterclockwise {
		face += "'"
	}
	return face, nil
}
