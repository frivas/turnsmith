package cube333

import (
	"bytes"
	"context"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
)

type guideFailure struct{}

func (guideFailure) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

type cancelGuide struct{ cancel context.CancelFunc }

func (w cancelGuide) Write(data []byte) (int, error) { w.cancel(); return len(data), nil }

func TestStepGuide(t *testing.T) {
	tokens := []string{"L2", "D'", "U"}
	var plain, colored bytes.Buffer
	for _, tc := range []struct {
		w     *bytes.Buffer
		color bool
	}{{&plain, false}, {&colored, true}} {
		if err := (StepGuide{}).Render(context.Background(), tc.w, tokens, tc.color); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range []string{"white center on top", "green center facing you", "Step 0/3: solved", "Step 1/3: L2", "Step 2/3: D'", "Step 3/3: U", "turn the bottom face 90 degrees counterclockwise", "L / left   F / front  R / right  B / back"} {
		if !strings.Contains(plain.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if len(regexp.MustCompile(`\[[WYORGB]\]`).FindAllString(plain.String(), -1)) != 54*4 {
		t.Fatal("expected 54 facelets for the initial state and each move")
	}
	if !strings.Contains(colored.String(), "\x1b[") || regexp.MustCompile("\x1b\\[[0-9;]*m").ReplaceAllString(colored.String(), "") != plain.String() {
		t.Fatal("color changed diagram content")
	}
	if err := (StepGuide{}).Render(context.Background(), guideFailure{}, tokens, true); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := (StepGuide{}).Render(ctx, cancelGuide{cancel}, tokens, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("missing mid-guide cancellation: %v", err)
	}
	var invalid bytes.Buffer
	if err := (StepGuide{}).Render(context.Background(), &invalid, []string{"R", "bad"}, true); err == nil || invalid.Len() != 0 {
		t.Fatal("invalid notation emitted a partial guide")
	}
}
