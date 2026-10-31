package training

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/frivas/turnsmith/internal/dirlock"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var ErrLocked = errors.New("training store is locked")
var ErrConflict = errors.New("training progress conflicts")
var ErrCorrupt = errors.New("training progress is invalid")

type Store struct{ dir string }
type LockOwner struct {
	Token     string    `json:"token"`
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
}

func NewStore(dir string) (*Store, error) {
	if dir == "" {
		base, e := os.UserConfigDir()
		if e != nil {
			return nil, e
		}
		dir = filepath.Join(base, "turnsmith")
	}
	absolute, e := filepath.Abs(dir)
	if e != nil {
		return nil, e
	}
	return &Store{dir: absolute}, nil
}
func (s *Store) Path() string       { return filepath.Join(s.dir, "training.json") }
func (s *Store) BackupPath() string { return s.Path() + ".bak" }
func (s *Store) lockPath() string   { return filepath.Join(s.dir, ".training.lock") }
func id() (string, error) {
	var data [16]byte
	if _, e := rand.Read(data[:]); e != nil {
		return "", e
	}
	return hex.EncodeToString(data[:]), nil
}
func (s *Store) LockInfo() (LockOwner, error) {
	var owner LockOwner
	data, e := os.ReadFile(filepath.Join(s.lockPath(), "owner.json"))
	if e != nil {
		return owner, e
	}
	if e := json.Unmarshal(data, &owner); e != nil {
		return owner, e
	}
	if owner.Token == "" {
		return owner, fmt.Errorf("invalid lock metadata")
	}
	return owner, nil
}

// BreakLock requires an operator to stop the owner and supply its inspected token.
func (s *Store) BreakLock(token string) error {
	owner, e := s.LockInfo()
	if e != nil {
		return e
	}
	if token == "" || owner.Token != token {
		return ErrConflict
	}
	if e := os.Remove(filepath.Join(s.lockPath(), "owner.json")); e != nil {
		return e
	}
	return os.Remove(s.lockPath())
}
func (s *Store) locked(ctx context.Context, fn func() error) (err error) {
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := os.MkdirAll(s.dir, 0700); e != nil {
		return e
	}
	token, e := id()
	if e != nil {
		return e
	}

	if e := dirlock.Mkdir(ctx, s.lockPath(), 5*time.Second, 10*time.Millisecond); e != nil {
		if errors.Is(e, dirlock.ErrTimeout) && errors.Is(e, os.ErrExist) {
			return fmt.Errorf("%w: %w", ErrLocked, e)
		}
		return e
	}
	data, _ := json.Marshal(LockOwner{token, os.Getpid(), time.Now().UTC()})
	if e := os.WriteFile(filepath.Join(s.lockPath(), "owner.json"), data, 0600); e != nil {
		_ = os.Remove(filepath.Join(s.lockPath(), "owner.json"))
		_ = os.Remove(s.lockPath())
		return e
	}

	defer func() {
		if e := s.BreakLock(token); err == nil && e != nil {
			err = e
		}
	}()
	return fn()
}
func readPath(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return readBounded(f)
}
func decodeDocument(data []byte) (Document, error) {
	var d Document
	if e := decode(data, &d); e != nil {
		return d, fmt.Errorf("%w: %v", ErrCorrupt, e)
	}
	if e := validate(d); e != nil {
		return d, fmt.Errorf("%w: %v", ErrCorrupt, e)
	}
	return d, nil
}
func (s *Store) load() (Document, []byte, error) {
	data, e := readPath(s.Path())
	if errors.Is(e, os.ErrNotExist) {
		if _, e := os.Stat(s.BackupPath()); e == nil {
			return Document{}, nil, fmt.Errorf("%w: primary missing while backup exists; recover explicitly", ErrCorrupt)
		} else if !errors.Is(e, os.ErrNotExist) {
			return Document{}, nil, e
		}
		return Document{SchemaVersion: 1, Collections: []CollectionProgress{}}, nil, nil
	}
	if e != nil {
		return Document{}, nil, e
	}
	d, e := decodeDocument(data)
	return d, data, e
}
func atomicWrite(path string, data []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".training-write-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer func() { _ = f.Close(); _ = os.Remove(name) }()
	if e := f.Chmod(0600); e != nil {
		return e
	}
	if _, e := f.Write(data); e != nil {
		return e
	}
	if e := f.Sync(); e != nil {
		return e
	}
	if e := f.Close(); e != nil {
		return e
	}
	if e := os.Rename(name, path); e != nil {
		return e
	}
	if runtime.GOOS != "windows" {
		dir, e := os.Open(filepath.Dir(path))
		if e != nil {
			return e
		}
		defer dir.Close()
		return dir.Sync()
	}
	return nil
}
func (s *Store) update(ctx context.Context, mutate func(*Document) error) error {
	return s.locked(ctx, func() error {
		d, old, e := s.load()
		if e != nil {
			return e
		}
		if e := mutate(&d); e != nil {
			return e
		}
		if e := validate(d); e != nil {
			return e
		}
		data, e := json.MarshalIndent(d, "", "  ")
		if e != nil {
			return e
		}
		if len(data) > maxBytes {
			return fmt.Errorf("training data exceeds 16 MiB")
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		if old != nil {
			if e := atomicWrite(s.BackupPath(), old); e != nil {
				return e
			}
		}
		return atomicWrite(s.Path(), append(data, '\n'))
	})
}
func (s *Store) snapshot(ctx context.Context) (Document, error) {
	var d Document
	e := s.locked(ctx, func() error { var e error; d, _, e = s.load(); return e })
	return d, e
}
func (s *Store) ExportJSON(ctx context.Context, w io.Writer) error {
	d, e := s.snapshot(ctx)
	if e != nil {
		return e
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

// ImportJSON replaces explicitly or merges disjoint collections and identical data.
// Differing progress for an existing collection is a conflict, never overwritten.
func (s *Store) ImportJSON(ctx context.Context, r io.Reader, replace bool) error {
	data, e := readBounded(r)
	if e != nil {
		return e
	}
	incoming, e := decodeDocument(data)
	if e != nil {
		return e
	}
	return s.update(ctx, func(d *Document) error {
		if replace {
			*d = incoming
			return nil
		}
		for _, c := range incoming.Collections {
			found := false
			for _, existing := range d.Collections {
				if existing.CollectionID == c.CollectionID {
					a, _ := json.Marshal(c)
					b, _ := json.Marshal(existing)
					if string(a) != string(b) {
						return fmt.Errorf("%w: collection %s", ErrConflict, c.Name)
					}
					found = true
					break
				}
			}
			if !found {
				d.Collections = append(d.Collections, c)
			}
		}
		return nil
	})
}

// Recover restores only a validated backup. Existing primary bytes are preserved.
func (s *Store) Recover(ctx context.Context) error {
	return s.locked(ctx, func() error {
		data, e := readPath(s.BackupPath())
		if e != nil {
			return e
		}
		if _, e := decodeDocument(data); e != nil {
			return e
		}
		old, e := readPath(s.Path())
		if e == nil {
			suffix, e := id()
			if e != nil {
				return e
			}
			if e := atomicWrite(s.Path()+".before-recovery-"+suffix, old); e != nil {
				return e
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return e
		}
		return atomicWrite(s.Path(), data)
	})
}
