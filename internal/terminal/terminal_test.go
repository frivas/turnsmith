package terminal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
)

func TestDecodeKey(t *testing.T) {
	for _, tc := range []struct {
		key  tcell.Key
		r    rune
		want Key
	}{
		{tcell.KeyEnter, 0, KeyEnter}, {tcell.KeyRune, ' ', KeySpace},
		{tcell.KeyRight, 0, KeyNext}, {tcell.KeyLeft, 0, KeyPrevious},
		{tcell.KeyHome, 0, KeyHome}, {tcell.KeyEnd, 0, KeyEnd},
		{tcell.KeyCtrlC, 0, KeyInterrupt}, {tcell.KeyCtrlD, 0, KeyEOF},
		{tcell.KeyEscape, 0, KeyQuit}, {tcell.KeyRune, 'q', KeyQuit},
		{tcell.KeyRune, 'n', KeyNext}, {tcell.KeyRune, 'p', KeyPrevious},
		{tcell.KeyRune, 'r', KeyHome}, {tcell.KeyRune, '+', KeyRune},
		{tcell.KeyF1, 0, ""},
	} {
		key, r := decodeKey(tcell.NewEventKey(tc.key, tc.r, tcell.ModNone))
		if key != tc.want || (tc.key == tcell.KeyRune && r != tc.r) {
			t.Errorf("%v/%q: got %q/%q", tc.key, tc.r, key, r)
		}
	}
}

func TestRunRejectsStreamsAndCanceledContext(t *testing.T) {
	if err := Run(context.Background(), strings.NewReader(""), &bytes.Buffer{}, Options{}, nil); !errors.Is(err, ErrNotTerminal) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Run(ctx, nil, nil, Options{}, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

type observedScreen struct {
	tcell.SimulationScreen
	finished, readerFinished chan struct{}
}

func (s *observedScreen) Init() error {
	if err := s.SimulationScreen.Init(); err != nil {
		return err
	}
	s.SetSize(80, 24)
	return nil
}
func (s *observedScreen) Fini() { s.SimulationScreen.Fini(); close(s.finished) }
func (s *observedScreen) ChannelEvents(events chan<- tcell.Event, quit <-chan struct{}) {
	defer close(s.readerFinished)
	s.SimulationScreen.ChannelEvents(events, quit)
}

func TestRunRestoresAndJoinsReader(t *testing.T) {
	sentinel := errors.New("callback failed")
	for _, tc := range []struct {
		name          string
		trigger       func(*observedScreen, context.CancelFunc)
		callbackError error
		done          bool
		want          error
	}{
		{name: "quit", trigger: func(s *observedScreen, _ context.CancelFunc) {
			_ = s.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'q', 0))
		}},
		{name: "control-c", trigger: func(s *observedScreen, _ context.CancelFunc) {
			_ = s.PostEvent(tcell.NewEventKey(tcell.KeyCtrlC, 0, 0))
		}, want: context.Canceled},
		{name: "control-d", trigger: func(s *observedScreen, _ context.CancelFunc) {
			_ = s.PostEvent(tcell.NewEventKey(tcell.KeyCtrlD, 0, 0))
		}},
		{name: "eof", trigger: func(s *observedScreen, _ context.CancelFunc) { _ = s.PostEvent(tcell.NewEventError(io.EOF)) }},
		{name: "context", trigger: func(_ *observedScreen, cancel context.CancelFunc) { cancel() }, want: context.Canceled},
		{name: "callback", callbackError: sentinel, want: sentinel},
		{name: "done", done: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &observedScreen{SimulationScreen: tcell.NewSimulationScreen("UTF-8"), finished: make(chan struct{}), readerFinished: make(chan struct{})}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			initial := make(chan struct{})
			result := make(chan error, 1)
			go func() {
				result <- runScreen(ctx, s, Options{}, func(e Event) (Frame, error) {
					if e.Key == KeyInitial {
						close(initial)
					}
					return Frame{Text: "hello", Done: tc.done}, tc.callbackError
				})
			}()
			select {
			case <-initial:
			case <-time.After(time.Second):
				t.Fatal("no initial frame")
			}
			if tc.trigger != nil {
				tc.trigger(s, cancel)
			}
			select {
			case err := <-result:
				if !errors.Is(err, tc.want) {
					t.Fatalf("got %v want %v", err, tc.want)
				}
			case <-time.After(time.Second):
				t.Fatal("session did not stop")
			}
			select {
			case <-s.finished:
			default:
				t.Fatal("screen not restored")
			}
			select {
			case <-s.readerFinished:
			default:
				t.Fatal("event reader still running")
			}
		})
	}
}

func TestTickAndResize(t *testing.T) {
	s := &observedScreen{SimulationScreen: tcell.NewSimulationScreen("UTF-8"), finished: make(chan struct{}), readerFinished: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	seenResize := false
	err := runScreen(ctx, s, Options{Tick: time.Millisecond}, func(e Event) (Frame, error) {
		if e.At.IsZero() {
			t.Error("missing timestamp")
		}
		if e.Key == KeyInitial {
			_ = s.PostEvent(tcell.NewEventResize(80, 24))
		}
		if e.Key == KeyResize {
			seenResize = true
		}
		return Frame{Done: e.Key == KeyTick && seenResize}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGuideNavigationAndSmallWindow(t *testing.T) {
	var states []string
	h, err := guideHandler(2, GuideOptions{}, func(n, w, h int, action string) (string, error) {
		states = append(states, fmt.Sprintf("%d:%s", n, action))
		return "frame", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []Key{KeyInitial, KeyNext, KeySpace, KeyEnter, KeyPrevious, KeyHome, KeyEnd} {
		if _, err := h(Event{Key: key, Width: 80, Height: 24}); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Join(states, ","); got != "0:start,1:next,2:next,1:previous,0:reset,2:end" {
		t.Fatal(got)
	}
	for _, key := range []Key{KeyNext, KeyPrevious, KeyHome, KeyEnd} {
		frame, err := h(Event{Key: key, Width: 20, Height: 10})
		if err != nil || !strings.Contains(frame.Text, "navigation paused") {
			t.Fatalf("%+v %v", frame, err)
		}
	}
	_, _ = h(Event{Key: KeyResize, Width: 80, Height: 24})
	if states[len(states)-1] != "2:resize" {
		t.Fatal(states)
	}
	if _, err := guideHandler(-1, GuideOptions{}, nil); err == nil {
		t.Fatal("invalid guide accepted")
	}
}

func TestPaintStylesControlsAndClipping(t *testing.T) {
	s := tcell.NewSimulationScreen("UTF-8")
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	defer s.Fini()
	s.SetSize(8, 3)
	paint(s, "\x1b[38;2;255;128;0mA\x1b[48;5;22mB\x1b[mC\x1b[2JD\x1b]0;title\aE\n界\u0301Z overflow\nlast\nclipped", 8, 3)
	fg, _, _ := contentStyle(s, 0, 0).Decompose()
	if fg != tcell.NewRGBColor(255, 128, 0) {
		t.Fatal(fg)
	}
	_, bg, _ := contentStyle(s, 1, 0).Decompose()
	if bg != tcell.PaletteColor(22) {
		t.Fatal(bg)
	}
	if contentStyle(s, 2, 0) != tcell.StyleDefault {
		t.Fatal("reset not applied")
	}
	for x, want := range "ABCDE" {
		got, _, _ := s.Get(x, 0)
		if got != string(want) {
			t.Errorf("%d: %q", x, got)
		}
	}
	got, _, width := s.Get(0, 1)
	if got != "界\u0301" || width != 2 {
		t.Fatalf("wide grapheme: %q %d", got, width)
	}
	got, _, _ = s.Get(2, 1)
	if got != "Z" {
		t.Fatal(got)
	}
}

func contentStyle(s tcell.Screen, x, y int) tcell.Style {
	_, style, _ := s.Get(x, y)
	return style
}

func TestPaintCombiningAcrossStyleChanges(t *testing.T) {
	for _, base := range []string{"A", "界"} {
		t.Run(base, func(t *testing.T) {
			s := tcell.NewSimulationScreen("UTF-8")
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			defer s.Fini()
			s.SetSize(8, 1)
			paint(s, base+"\x1b[31m\u0301Z", 8, 1)
			got, style, width := s.Get(0, 0)
			if got != base+"\u0301" || style != tcell.StyleDefault {
				t.Fatalf("combining mark changed the preceding cell: %q %v", got, style)
			}
			got, style, _ = s.Get(width, 0)
			fg, _, _ := style.Decompose()
			if got != "Z" || fg != tcell.PaletteColor(1) {
				t.Fatalf("following character has wrong position or color: %q %v", got, fg)
			}
		})
	}
}
