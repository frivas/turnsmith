//go:build windows

package terminal

import "github.com/gdamore/tcell/v2"

// The maintained backend owns Win32 console handles, cancellation events, and
// mode restoration. No platform ioctls or unsafe console code live in Turnsmith.
func newScreen() (tcell.Screen, func(), error) {
	screen, err := tcell.NewScreen()
	return screen, func() {}, err
}
