package dirlock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func denied(path string) error { return &os.PathError{Op: "mkdir", Path: path, Err: syscall.Errno(5)} }
func TestWindowsPendingDeletionRetries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	calls := 0
	err := mkdir(context.Background(), path, time.Second, time.Millisecond, func(path string, mode os.FileMode) error {
		calls++
		if calls == 1 {
			return denied(path)
		}
		return os.Mkdir(path, mode)
	}, "windows")
	if err != nil || calls != 2 {
		t.Fatal(calls, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
func TestWindowsPermanentDeniedIsBoundedAndPreserved(t *testing.T) {
	calls := 0
	start := time.Now()
	err := mkdir(context.Background(), "lock", 5*time.Millisecond, time.Millisecond, func(path string, _ os.FileMode) error { calls++; return denied(path) }, "windows")
	if !errors.Is(err, ErrTimeout) || !errors.Is(err, syscall.Errno(5)) || calls < 1 || time.Since(start) > time.Second {
		t.Fatal(calls, err, time.Since(start))
	}
	var pathError *os.PathError
	if !errors.As(err, &pathError) || pathError.Op != "mkdir" {
		t.Fatal("lost cause", err)
	}
}
func TestExistingOwnerIsNotStolen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	owner := filepath.Join(path, "owner.json")
	if err := os.WriteFile(owner, []byte("owner"), 0600); err != nil {
		t.Fatal(err)
	}
	err := mkdir(context.Background(), path, 5*time.Millisecond, time.Millisecond, os.Mkdir, "windows")
	if !errors.Is(err, ErrTimeout) {
		t.Fatal(err)
	}
	data, err := os.ReadFile(owner)
	if err != nil || string(data) != "owner" {
		t.Fatal("owner removed", err)
	}
}
func TestOtherErrorsAndUnixDeniedDoNotRetry(t *testing.T) {
	for _, goos := range []string{"darwin", "linux"} {
		calls := 0
		err := mkdir(context.Background(), "lock", time.Second, time.Millisecond, func(path string, _ os.FileMode) error { calls++; return denied(path) }, goos)
		if calls != 1 || !errors.Is(err, syscall.Errno(5)) {
			t.Fatal(goos, calls, err)
		}
	}
	calls := 0
	err := mkdir(context.Background(), "lock", time.Second, time.Millisecond, func(string, os.FileMode) error { calls++; return os.ErrPermission }, "windows")
	if calls != 1 || !errors.Is(err, os.ErrPermission) {
		t.Fatal(calls, err)
	}
}
func TestCancelledWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	err := mkdir(ctx, "lock", time.Second, time.Millisecond, func(string, os.FileMode) error { cancel(); return os.ErrExist }, "windows")
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
