package randomstate

import (
	"context"
	"github.com/frivas/turnsmith/internal/puzzle"
	"github.com/frivas/turnsmith/internal/random"
	"testing"
)

func TestManyStates(t *testing.T) {
	if testing.Short() {
		t.Skip("extended solver sweep")
	}
	for i := uint64(0); i < 1000; i++ {
		_, err := (Cube333{}).Generate(context.Background(), random.Derive(i, "333", "random-state", "1", 0), puzzle.Options{})
		if err != nil {
			t.Fatalf("seed%d: %v", i, err)
		}
	}
}
