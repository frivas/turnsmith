package randomstate

import (
	"context"
	"github.com/unixpickle/gocube"
	"sync"
)

type tables333 struct {
	p1 *gocube.Phase1Moves
	h1 *gocube.Phase1Heuristic
	p2 *gocube.Phase2Moves
	h2 *gocube.Phase2Heuristic
}

var initialize333 sync.Once
var ready333 chan struct{}
var cached333 *tables333

func load333(ctx context.Context) (*tables333, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	initialize333.Do(func() { ready333 = make(chan struct{}); go func() { cached333 = build333(); close(ready333) }() })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ready333:
		return cached333, nil
	}
}

func build333() *tables333 {
	t := &tables333{p1: gocube.NewPhase1Moves(), p2: gocube.NewPhase2Moves(), h1: new(gocube.Phase1Heuristic), h2: new(gocube.Phase2Heuristic)}
	// Dense integer queues avoid the upstream heuristic constructor's large
	// allocation churn. Each coordinate enters the queue at most once.
	fillPhase1(t.h1.COEO[:], 1093*2048, 2187*2048, func(hash, m int) int { return t.p1.COMoves[hash/2048][m]*2048 + t.p1.EOMoves[hash%2048][m] })
	fillPhase1(t.h1.EOSlice[:], 220*2048, 495*2048, func(hash, m int) int { return t.p1.ESliceMoves[hash/2048][m]*2048 + t.p1.EOMoves[hash%2048][m] })
	fillPhase2(t.h2.CornersSlice[:], 11, func(hash, m int) int { return t.p2.CornerMoves[hash/24][m]*24 + t.p2.SliceMoves[hash%24][m] })
	fillPhase2(t.h2.EdgesSlice[:], 8, func(hash, m int) int { return t.p2.EdgeMoves[hash/24][m]*24 + t.p2.SliceMoves[hash%24][m] })
	return t
}
func fillPhase1(distance []uint8, start, capacity int, next func(int, int) int) {
	for i := range distance {
		distance[i] = 8
	}
	distance[start] = 0
	queue := make([]uint32, 1, capacity)
	queue[0] = uint32(start)
	for front := 0; front < len(queue); front++ {
		hash := int(queue[front])
		depth := distance[hash]
		if depth == 7 {
			continue
		}
		for m := 0; m < 18; m++ {
			n := next(hash, m)
			if distance[n] != 8 {
				continue
			}
			distance[n] = depth + 1
			queue = append(queue, uint32(n))
		}
	}
}
func fillPhase2(distance []int8, maxDepth int, next func(int, int) int) {
	for i := range distance {
		distance[i] = -1
	}
	distance[0] = 0
	queue := make([]uint32, 1, len(distance))
	for front := 0; front < len(queue); front++ {
		hash := int(queue[front])
		depth := distance[hash]
		if int(depth) == maxDepth {
			continue
		}
		for m := 0; m < 10; m++ {
			n := next(hash, m)
			if distance[n] >= 0 {
				continue
			}
			distance[n] = depth + 1
			queue = append(queue, uint32(n))
		}
	}
}
