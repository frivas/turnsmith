// Package randomstate implements native, deterministic reachable-state samplers
// and scramble construction. It is separate from the random-move generator.
package randomstate

import (
	"context"
	"github.com/frivas/turnsmith/internal/random"
	"github.com/unixpickle/gocube"
)

func draw(ctx context.Context, rng random.Source, n uint64) (uint64, error) {
	threshold := -n % n
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		v := rng.Uint64()
		if v >= threshold {
			return v % n, nil
		}
	}
}

func shuffled(ctx context.Context, rng random.Source, n int) ([]int, error) {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j, err := draw(ctx, rng, uint64(i+1))
		if err != nil {
			return nil, err
		}
		p[i], p[j] = p[j], p[i]
	}
	return p, nil
}

func oddPermutation(p []int) bool {
	odd := false
	for i := range p {
		for j := i + 1; j < len(p); j++ {
			if p[i] > p[j] {
				odd = !odd
			}
		}
	}
	return odd
}

// Complete the unique dependent corner orientation. gocube's representation
// uses the axis of the U/D sticker, rather than conventional clockwise trits.
// Adjacent positions alternate twist handedness along this corner cycle.
func completeOrientation(c *gocube.CubieCorners) {
	cycle := [8]int{0, 1, 5, 4, 6, 2, 3, 7}
	carried := c[cycle[0]].Orientation
	for _, position := range cycle[1:7] {
		carried = (c[position].Orientation + 1 - carried + 3) % 3
	}
	normalized := (2 - carried + 3) % 3
	c[7].Orientation = (2 - normalized + 3) % 3
}

func sample333(ctx context.Context, rng random.Source) (gocube.CubieCube, error) {
	state := gocube.SolvedCubieCube()
	corners, err := shuffled(ctx, rng, 8)
	if err != nil {
		return state, err
	}
	edges, err := shuffled(ctx, rng, 12)
	if err != nil {
		return state, err
	}
	if oddPermutation(corners) != oddPermutation(edges) {
		edges[10], edges[11] = edges[11], edges[10]
	}
	for i, piece := range corners {
		state.Corners[i].Piece = piece
	}
	for i := 0; i < 7; i++ {
		v, err := draw(ctx, rng, 3)
		if err != nil {
			return state, err
		}
		state.Corners[i].Orientation = int(v)
	}
	completeOrientation(&state.Corners)
	parity := false
	for i, piece := range edges {
		state.Edges[i].Piece = piece
		if i < 11 {
			v, err := draw(ctx, rng, 2)
			if err != nil {
				return state, err
			}
			state.Edges[i].Flip = v == 1
			parity = parity != state.Edges[i].Flip
		}
	}
	state.Edges[11].Flip = parity
	return state, nil
}

func sample222(ctx context.Context, rng random.Source) (gocube.CubieCorners, error) {
	corners := gocube.SolvedCubieCorners()
	permutation, err := shuffled(ctx, rng, 7)
	if err != nil {
		return corners, err
	}
	for i, piece := range permutation {
		corners[i+1].Piece = piece + 1
	}
	for i := 1; i < 7; i++ {
		v, err := draw(ctx, rng, 3)
		if err != nil {
			return corners, err
		}
		corners[i].Orientation = int(v)
	}
	completeOrientation(&corners)
	return corners, nil
}
