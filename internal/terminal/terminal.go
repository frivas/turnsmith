// Package terminal owns explicit, cancellable terminal sessions. It never opens
// an interactive screen when standard input or standard output is redirected.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"golang.org/x/term"
)

var ErrNotTerminal = errors.New("interactive mode requires terminal stdin and stdout; use stream mode when piping")

// Key names are portable actions, not raw escape sequences. Navigation aliases
// share a key, but Enter and Space remain distinct for press-based timers.
type Key string

const (
	KeyInitial   Key = "initial"
	KeyResize    Key = "resize"
	KeyTick      Key = "tick"
	KeyNext      Key = "next"
	KeyPrevious  Key = "previous"
	KeyHome      Key = "home"
	KeyEnd       Key = "end"
	KeyEnter     Key = "enter"
	KeySpace     Key = "space"
	KeyRune      Key = "rune"
	KeyQuit      Key = "quit"
	KeyInterrupt Key = "interrupt"
	KeyEOF       Key = "eof"
)

// Event.At contains Go's monotonic clock reading for reliable elapsed timing.
type Event struct {
	Key           Key
	Rune          rune
	Width, Height int
	At            time.Time
}

type Frame struct {
	Text string
	Done bool
}

type Handler func(Event) (Frame, error)

type Options struct {
	// Tick is optional. Zero disables periodic redraws. Positive durations are
	// clamped to 20ms to keep a mistaken interval from spinning the UI.
	Tick time.Duration
}

// IsTerminal reports real terminal capability, not merely a character device.
func IsTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// Run takes an exclusive lease on process-standard terminals and always restores
// the screen, cursor, input mode, and event reader before returning. q/Esc and
// Ctrl-D exit normally; Ctrl-C returns context.Canceled. Handler errors propagate.
// Handler receives initial, resize, optional tick, and decoded press events. Its
// text may contain ANSI SGR colors; other control sequences are never executed.
func Run(ctx context.Context, input io.Reader, output io.Writer, options Options, handler Handler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	in, inOK := input.(*os.File)
	out, outOK := output.(*os.File)
	if !inOK || !outOK || !IsTerminal(in) || !IsTerminal(out) || in.Fd() != os.Stdin.Fd() || out.Fd() != os.Stdout.Fd() || os.Getenv("TERM") == "dumb" {
		return ErrNotTerminal
	}
	if handler == nil {
		return fmt.Errorf("interactive frame handler is required")
	}
	screen, restore, err := newScreen()
	if err != nil {
		return fmt.Errorf("open interactive terminal: %w", err)
	}
	defer restore()
	// The backend interrupts pending input with a deadline before restoring the
	// tty. Clear that deadline after Fini so callers may read stdin again.
	defer func() { _ = in.SetReadDeadline(time.Time{}) }()
	return runScreen(ctx, screen, options, handler)
}

func runScreen(ctx context.Context, screen tcell.Screen, options Options, handler Handler) error {
	if err := screen.Init(); err != nil {
		return fmt.Errorf("initialize interactive terminal: %w", err)
	}
	quit := make(chan struct{})
	events := make(chan tcell.Event)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		screen.ChannelEvents(events, quit)
	}()
	defer func() {
		close(quit)
		screen.Fini()
		<-readerDone
	}()
	screen.HideCursor()
	var tick <-chan time.Time
	if options.Tick > 0 {
		ticker := time.NewTicker(max(options.Tick, 20*time.Millisecond))
		defer ticker.Stop()
		tick = ticker.C
	}
	draw := func(key Key, r rune, now time.Time) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		width, height := screen.Size()
		frame, err := handler(Event{Key: key, Rune: r, Width: width, Height: height, At: now})
		if err != nil {
			return false, fmt.Errorf("draw interactive frame: %w", err)
		}
		screen.Clear()
		paint(screen, frame.Text, width, height)
		screen.Show()
		return frame.Done, nil
	}
	if done, err := draw(KeyInitial, 0, time.Now()); err != nil || done {
		return err
	}
	for {
		var key Key
		var r rune
		var at time.Time
		select {
		case <-ctx.Done():
			return ctx.Err()
		case at = <-tick:
			key = KeyTick
		case event, ok := <-events:
			if !ok || event == nil {
				return nil
			}
			at = event.When()
			if at.IsZero() {
				at = time.Now()
			}
			switch event := event.(type) {
			case *tcell.EventResize:
				screen.Sync()
				key = KeyResize
			case *tcell.EventKey:
				key, r = decodeKey(event)
			case *tcell.EventError:
				if errors.Is(event, io.EOF) || event.Error() == io.EOF.Error() {
					return nil
				}
				return fmt.Errorf("read interactive terminal: %w", event)
			default:
				continue
			}
		}
		switch key {
		case KeyQuit, KeyEOF:
			return nil
		case KeyInterrupt:
			return context.Canceled
		case "":
			continue
		}
		if done, err := draw(key, r, at); err != nil || done {
			return err
		}
	}
}

func decodeKey(event *tcell.EventKey) (Key, rune) {
	switch event.Key() {
	case tcell.KeyCtrlC:
		return KeyInterrupt, 0
	case tcell.KeyCtrlD:
		return KeyEOF, 0
	case tcell.KeyEscape:
		return KeyQuit, 0
	case tcell.KeyEnter:
		return KeyEnter, 0
	case tcell.KeyRight, tcell.KeyDown:
		return KeyNext, 0
	case tcell.KeyLeft, tcell.KeyUp:
		return KeyPrevious, 0
	case tcell.KeyHome:
		return KeyHome, 0
	case tcell.KeyEnd:
		return KeyEnd, 0
	case tcell.KeyRune:
		r := event.Rune()
		switch r {
		case 'q', 'Q':
			return KeyQuit, r
		case 'n', 'N':
			return KeyNext, r
		case 'p', 'P':
			return KeyPrevious, r
		case 'r', 'R':
			return KeyHome, r
		case ' ':
			return KeySpace, r
		default:
			return KeyRune, r
		}
	}
	return "", 0
}
