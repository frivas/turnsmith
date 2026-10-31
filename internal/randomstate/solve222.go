package randomstate

import (
	"context"
	"fmt"
	"github.com/unixpickle/gocube"
	"sync"
)

var pocketMoves = [9]gocube.Move{0, 2, 4, 6, 8, 10, 12, 14, 16}

const pocketOrientations = 729
const pocketStates = 5040 * pocketOrientations
const solvedPocketOrientation = 364

type tables222 struct {
	permutations [5040][9]uint16
	orientations [729][9]uint16
	distance     []uint8
}

var initialize222 sync.Once
var ready222 chan struct{}
var cached222 *tables222

func load222(ctx context.Context) (*tables222, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	initialize222.Do(func() { ready222 = make(chan struct{}); go func() { cached222 = build222(); close(ready222) }() })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ready222:
		return cached222, nil
	}
}
func permutationCode(c gocube.CubieCorners) int {
	result := 0
	for i := 1; i < 8; i++ {
		lower := 0
		for j := i + 1; j < 8; j++ {
			if c[j].Piece < c[i].Piece {
				lower++
			}
		}
		result = result*(8-i) + lower
	}
	return result
}
func decodePermutation(code int) gocube.CubieCorners {
	c := gocube.SolvedCubieCorners()
	available := []int{1, 2, 3, 4, 5, 6, 7}
	factorials := [7]int{720, 120, 24, 6, 2, 1, 1}
	for i := 1; i < 8; i++ {
		n := code / factorials[i-1]
		code %= factorials[i-1]
		c[i].Piece = available[n]
		available = append(available[:n], available[n+1:]...)
	}
	return c
}
func orientationCode(c gocube.CubieCorners) int {
	result, base := 0, 1
	for i := 1; i < 7; i++ {
		result += c[i].Orientation * base
		base *= 3
	}
	return result
}
func decodeOrientation(code int) gocube.CubieCorners {
	c := gocube.SolvedCubieCorners()
	for i := 1; i < 7; i++ {
		c[i].Orientation = code % 3
		code /= 3
	}
	completeOrientation(&c)
	return c
}
func build222() *tables222 {
	t := &tables222{distance: make([]uint8, pocketStates)}
	for p := range t.permutations {
		c := decodePermutation(p)
		for m, move := range pocketMoves {
			next := c
			next.Move(move)
			t.permutations[p][m] = uint16(permutationCode(next))
		}
	}
	for o := range t.orientations {
		c := decodeOrientation(o)
		for m, move := range pocketMoves {
			next := c
			next.Move(move)
			t.orientations[o][m] = uint16(orientationCode(next))
		}
	}
	for i := range t.distance {
		t.distance[i] = 255
	}
	t.distance[solvedPocketOrientation] = 0
	queue := make([]uint32, 1, pocketStates)
	queue[0] = solvedPocketOrientation
	for front := 0; front < len(queue); front++ {
		hash := int(queue[front])
		p, o := hash/pocketOrientations, hash%pocketOrientations
		for m := range pocketMoves {
			next := int(t.permutations[p][m])*pocketOrientations + int(t.orientations[o][m])
			if t.distance[next] != 255 {
				continue
			}
			t.distance[next] = t.distance[hash] + 1
			queue = append(queue, uint32(next))
		}
	}
	return t
}
func solve222(ctx context.Context, corners gocube.CubieCorners, t *tables222) ([]gocube.Move, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p, o := permutationCode(corners), orientationCode(corners)
	hash := p*pocketOrientations + o
	if t.distance[hash] == 255 {
		return nil, fmt.Errorf("sampled 222 state is not reachable")
	}
	path := make([]gocube.Move, 0, t.distance[hash])
	for t.distance[hash] > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		found := false
		for m, move := range pocketMoves {
			np, no := int(t.permutations[p][m]), int(t.orientations[o][m])
			next := np*pocketOrientations + no
			if t.distance[next]+1 == t.distance[hash] {
				path = append(path, move)
				p, o, hash = np, no, next
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("invalid 222 distance table")
		}
	}
	return path, nil
}
