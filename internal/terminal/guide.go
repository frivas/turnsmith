package terminal

import (
	"context"
	"fmt"
	"io"
)

type GuideOptions struct{ MinWidth, MinHeight int }

type GuideFrame func(index, width, height int, action string) (string, error)

// RunGuide navigates states 0..total. Frame actions are start, next, previous,
// reset, end, and resize. The caller explains the actual forward/inverse moves.
func RunGuide(ctx context.Context, input io.Reader, output io.Writer, total int, frame GuideFrame) error {
	return RunGuideWithOptions(ctx, input, output, total, GuideOptions{}, frame)
}

// RunGuideWithOptions pauses navigation below the configured minimum size, so
// unseen key presses cannot move the physical-cube guidance out of sync.
func RunGuideWithOptions(ctx context.Context, input io.Reader, output io.Writer, total int, options GuideOptions, frame GuideFrame) error {
	handler, err := guideHandler(total, options, frame)
	if err != nil {
		return err
	}
	return Run(ctx, input, output, Options{}, handler)
}

func guideHandler(total int, options GuideOptions, frame GuideFrame) (Handler, error) {
	if total < 0 || frame == nil {
		return nil, fmt.Errorf("guide requires a nonnegative step count and frame handler")
	}
	if options.MinWidth <= 0 {
		options.MinWidth = 44
	}
	if options.MinHeight <= 0 {
		options.MinHeight = 20
	}
	index := 0
	last := ""
	return func(event Event) (Frame, error) {
		if event.Width < options.MinWidth || event.Height < options.MinHeight {
			return Frame{Text: fmt.Sprintf("Resize to at least %d columns x %d rows.\nCurrent: %d x %d.\nStep %d/%d; navigation paused.\nq: quit", options.MinWidth, options.MinHeight, event.Width, event.Height, index, total)}, nil
		}
		action := ""
		switch event.Key {
		case KeyInitial:
			action = "start"
		case KeyResize:
			action = "resize"
		case KeyNext, KeyEnter, KeySpace:
			if index < total {
				index++
				action = "next"
			}
		case KeyPrevious:
			if index > 0 {
				index--
				action = "previous"
			}
		case KeyHome:
			index = 0
			action = "reset"
		case KeyEnd:
			index = total
			action = "end"
		}
		if action == "" && last != "" {
			return Frame{Text: last}, nil
		}
		text, err := frame(index, event.Width, event.Height, action)
		if err != nil {
			return Frame{}, err
		}
		last = text
		return Frame{Text: text}, nil
	}, nil
}
