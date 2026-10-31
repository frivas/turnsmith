package randomstate

import (
	"context"
	"fmt"
	"github.com/unixpickle/gocube"
	"strings"
)

// Solve333Facelets solves a reduced cube in canonical URFDLB facelet order.
// It is the native reduction boundary used by the separate 4x4 backend.
func Solve333Facelets(ctx context.Context, facelets string) ([]string, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if len(facelets) != 54 {
		return nil, fmt.Errorf("reduced cube needs 54 facelets")
	}
	var stickers gocube.StickerCube
	// gocube uses U,D,F,B,R,L face order and the same face-local reading order.
	sourceFaces := []int{0, 3, 2, 5, 1, 4}
	colors := map[byte]int{}
	faceColors := []int{1, 5, 3, 2, 6, 4}
	for f, code := range faceColors {
		center := facelets[f*9+4]
		if _, exists := colors[center]; exists {
			return nil, fmt.Errorf("reduced cube centers must be distinct")
		}
		if !strings.ContainsRune("URFDLB", rune(center)) {
			return nil, fmt.Errorf("invalid reduced center")
		}
		colors[center] = code
	}
	counts := [7]int{}
	for f, src := range sourceFaces {
		for i := 0; i < 9; i++ {
			v, ok := colors[facelets[src*9+i]]
			if !ok {
				return nil, fmt.Errorf("invalid reduced facelet")
			}
			stickers[f*9+i] = v
			counts[v]++
		}
	}
	for _, n := range counts[1:] {
		if n != 9 {
			return nil, fmt.Errorf("reduced cube must have nine stickers per color")
		}
	}
	target, e := stickers.CubieCube()
	if e != nil {
		return nil, fmt.Errorf("invalid reduced cube: %w", e)
	}
	corners := make([]int, 8)
	edges := make([]int, 12)
	seenC := [8]bool{}
	seenE := [12]bool{}
	flip := false
	for i, piece := range target.Corners {
		if piece.Piece < 0 || piece.Piece >= 8 || seenC[piece.Piece] {
			return nil, fmt.Errorf("duplicate or invalid reduced corner")
		}
		seenC[piece.Piece] = true
		corners[i] = piece.Piece
	}
	for i, piece := range target.Edges {
		if piece.Piece < 0 || piece.Piece >= 12 || seenE[piece.Piece] {
			return nil, fmt.Errorf("duplicate or invalid reduced edge")
		}
		seenE[piece.Piece] = true
		edges[i] = piece.Piece
		flip = flip != piece.Flip
	}
	checkCorners := target.Corners
	completeOrientation(&checkCorners)
	if flip || checkCorners[7].Orientation != target.Corners[7].Orientation || oddPermutation(corners) != oddPermutation(edges) {
		return nil, fmt.Errorf("unreachable reduced cube orientation or parity")
	}
	tables, e := load333(ctx)
	if e != nil {
		return nil, e
	}
	solution, e := solve333(ctx, *target, tables)
	if e != nil {
		return nil, e
	}
	check := *target
	tokens := make([]string, len(solution))
	for i, move := range solution {
		check.Move(move)
		tokens[i] = move.String()
	}
	if !check.Solved() {
		return nil, fmt.Errorf("reduced cube solver verification failed")
	}
	return tokens, nil
}
