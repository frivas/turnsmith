//go:build !windows

package terminal

import (
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sys/unix"
	"os"
)

func newScreen() (tcell.Screen, func(), error) {
	fd := os.Stdin.Fd()
	flags, err := unix.FcntlInt(fd, unix.F_GETFL, 0)
	if err != nil {
		return nil, nil, err
	}
	// tcell's Unix Drain enables O_NONBLOCK to interrupt its reader. Its tty
	// restoration covers termios, so restore descriptor flags separately to
	// keep subsequent ordinary stdin reads usable after this session.
	restore := func() { _, _ = unix.FcntlInt(fd, unix.F_SETFL, flags) }
	tty, err := tcell.NewStdIoTty()
	if err != nil {
		return nil, nil, err
	}
	screen, err := tcell.NewTerminfoScreenFromTty(tty)
	if err != nil {
		_ = tty.Close()
		return nil, nil, err
	}
	return screen, restore, nil
}
