// Package dirlock acquires lock directories without removing an existing owner.
package dirlock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
)

var ErrTimeout = errors.New("timed out acquiring lock directory")

// Mkdir waits for an exclusive lock directory. Only the successful creator owns
// it. On Windows, a recently removed directory may remain pending deletion and
// CreateDirectory can temporarily return ERROR_ACCESS_DENIED. Retrying that
// error does not imply that an existing owner is stale. Permanent denial returns
// after the same bounded wait, preserving its underlying error for diagnosis.
func Mkdir(ctx context.Context, path string, wait, interval time.Duration) error {
	return mkdir(ctx, path, wait, interval, os.Mkdir, runtime.GOOS)
}

func retryable(err error, goos string) bool {
	// Win32 ERROR_ACCESS_DENIED is 5. Runtime gating is essential because errno 5
	// has a different meaning on Unix. Keep the code portable for injected tests.
	return errors.Is(err, os.ErrExist) || (goos == "windows" && errors.Is(err, syscall.Errno(5)))
}

func mkdir(ctx context.Context, path string, wait, interval time.Duration, attempt func(string, os.FileMode) error, goos string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deadline := time.NewTimer(max(wait, 0))
	defer deadline.Stop()
	ticker := time.NewTicker(max(interval, time.Millisecond))
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := attempt(path, 0700)
		if err == nil {
			return nil
		}
		if !retryable(err, goos) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("%w at %s; last mkdir error: %w", ErrTimeout, path, err)
		case <-ticker.C:
		}
	}
}
