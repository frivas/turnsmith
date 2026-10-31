package practice

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound = errors.New("session or solve not found")
	ErrConflict = errors.New("practice data conflict")
	ErrLocked   = errors.New("practice store is locked")
	ErrCorrupt  = errors.New("practice data is invalid")
)

type Store struct {
	dir      string
	lockWait time.Duration
}

type LockOwner struct {
	Token     string    `json:"token"`
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
}

func DefaultDataDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "turnsmith"), nil
}

// New resolves a path but does not create directories or read/write data.
func New(dir string) (*Store, error) {
	var err error
	if dir == "" {
		dir, err = DefaultDataDir()
		if err != nil {
			return nil, err
		}
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &Store{dir: dir, lockWait: 5 * time.Second}, nil
}

func (s *Store) Dir() string        { return s.dir }
func (s *Store) Path() string       { return filepath.Join(s.dir, "sessions.json") }
func (s *Store) BackupPath() string { return s.Path() + ".bak" }
func (s *Store) lockPath() string   { return filepath.Join(s.dir, ".practice.lock") }

func newID() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

// LockInfo allows an explicit recovery command to identify an abandoned owner.
// Expiry or PID reuse alone is never treated as proof that an owner is dead.
func (s *Store) LockInfo() (LockOwner, error) {
	var owner LockOwner
	data, err := os.ReadFile(filepath.Join(s.lockPath(), "owner.json"))
	if err != nil {
		return owner, err
	}
	if err := json.Unmarshal(data, &owner); err != nil || owner.Token == "" {
		return owner, fmt.Errorf("invalid lock ownership metadata")
	}
	return owner, nil
}

// BreakLock is an explicit administrative operation. The caller must first
// verify that no process is using this store. Matching a token prevents deleting
// a different lock that replaced the one inspected by the caller.
func (s *Store) BreakLock(expectedToken string) error {
	owner, err := s.LockInfo()
	if err != nil {
		return err
	}
	if expectedToken == "" || owner.Token != expectedToken {
		return fmt.Errorf("%w: lock owner changed", ErrConflict)
	}
	return s.release(expectedToken)
}

func (s *Store) acquire(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return "", err
	}
	token, err := newID()
	if err != nil {
		return "", err
	}
	deadline := time.NewTimer(s.lockWait)
	defer deadline.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		err := os.Mkdir(s.lockPath(), 0700)
		if err == nil {
			data, err := json.Marshal(LockOwner{Token: token, PID: os.Getpid(), CreatedAt: time.Now().UTC()})
			if err == nil {
				err = os.WriteFile(filepath.Join(s.lockPath(), "owner.json"), data, 0600)
			}
			if err != nil {
				_ = os.Remove(filepath.Join(s.lockPath(), "owner.json"))
				_ = os.Remove(s.lockPath())
				return "", err
			}
			return token, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "", fmt.Errorf("%w at %s; inspect owner.json and only recover an abandoned lock after stopping its owner", ErrLocked, s.lockPath())
		case <-tick.C:
		}
	}
}

func (s *Store) release(token string) error {
	owner, err := s.LockInfo()
	if err != nil {
		return err
	}
	if owner.Token != token {
		return fmt.Errorf("%w: lock owner changed", ErrConflict)
	}
	if err := os.Remove(filepath.Join(s.lockPath(), "owner.json")); err != nil {
		return err
	}
	return os.Remove(s.lockPath())
}

func (s *Store) locked(ctx context.Context, fn func() error) (err error) {
	token, err := s.acquire(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if releaseErr := s.release(token); err == nil && releaseErr != nil {
			err = fmt.Errorf("operation finished but lock release failed: %w", releaseErr)
		}
	}()
	return fn()
}

func limitedRead(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxDataBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDataBytes {
		return nil, fmt.Errorf("practice data exceeds %d bytes", maxDataBytes)
	}
	return data, nil
}

func decode(data []byte) (Document, error) {
	var doc Document
	if err := checkJSON(data); err != nil {
		return doc, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return doc, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return doc, fmt.Errorf("%w: expected one JSON document", ErrCorrupt)
	}
	if err := validate(doc); err != nil {
		return doc, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	return doc, nil
}

func readFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return limitedRead(file)
}

func (s *Store) load() (Document, []byte, error) {
	data, err := readFile(s.Path())
	if errors.Is(err, os.ErrNotExist) {
		if _, backupErr := os.Stat(s.BackupPath()); backupErr == nil {
			return Document{}, nil, fmt.Errorf("%w: sessions.json is missing but a backup exists; use explicit recovery", ErrCorrupt)
		} else if !errors.Is(backupErr, os.ErrNotExist) {
			return Document{}, nil, backupErr
		}
		return Document{SchemaVersion: SchemaVersion, Sessions: []Session{}}, nil, nil
	}
	if err != nil {
		return Document{}, nil, err
	}
	doc, err := decode(data)
	if err != nil {
		return Document{}, nil, fmt.Errorf("%s: %w; inspect the file or use explicit backup recovery", s.Path(), err)
	}
	return doc, data, nil
}

func atomicWrite(path string, data []byte) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".practice-write-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(name) }()
	if err := file.Chmod(0600); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	// Directory fsync is supported on Unix. Windows does not expose it through
	// os.File.Sync; the synced file and prior good backup remain the boundary.
	if runtime.GOOS != "windows" {
		dir, err := os.Open(filepath.Dir(path))
		if err != nil {
			return err
		}
		defer dir.Close()
		return dir.Sync()
	}
	return nil
}

func (s *Store) update(ctx context.Context, mutate func(*Document) (bool, error)) error {
	return s.locked(ctx, func() error {
		doc, previous, err := s.load()
		if err != nil {
			return err
		}
		changed, err := mutate(&doc)
		if err != nil || !changed {
			return err
		}
		if err := validate(doc); err != nil {
			return err
		}
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if bytes.Equal(bytes.TrimSpace(data), bytes.TrimSpace(previous)) {
			return nil
		}
		if len(data) > maxDataBytes {
			return fmt.Errorf("practice data exceeds %d bytes", maxDataBytes)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if previous != nil {
			if err := atomicWrite(s.BackupPath(), previous); err != nil {
				return fmt.Errorf("backup: %w", err)
			}
		}
		return atomicWrite(s.Path(), data)
	})
}

func (s *Store) snapshot(ctx context.Context) (Document, error) {
	var doc Document
	err := s.locked(ctx, func() error {
		var err error
		doc, _, err = s.load()
		if err == nil {
			err = ctx.Err()
		}
		return err
	})
	return doc, err
}

func (s *Store) Sessions(ctx context.Context) ([]Session, error) {
	doc, err := s.snapshot(ctx)
	return doc.Sessions, err
}

func sessionIndex(doc Document, name string) (int, error) {
	for i, session := range doc.Sessions {
		if session.ID == name {
			return i, nil
		}
	}
	for i, session := range doc.Sessions {
		if session.Name == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("%w: session %q", ErrNotFound, name)
}

func (s *Store) Get(ctx context.Context, name string) (Session, error) {
	doc, err := s.snapshot(ctx)
	if err != nil {
		return Session{}, err
	}
	i, err := sessionIndex(doc, name)
	if err != nil {
		return Session{}, err
	}
	return doc.Sessions[i], nil
}

func (s *Store) Create(ctx context.Context, name string) (Session, error) {
	id, err := newID()
	if err != nil {
		return Session{}, err
	}
	session := Session{ID: id, Name: name, CreatedAt: time.Now().UTC(), Solves: []Solve{}}
	err = s.update(ctx, func(doc *Document) (bool, error) {
		for _, existing := range doc.Sessions {
			if existing.Name == name {
				return false, fmt.Errorf("%w: session %q already exists", ErrConflict, name)
			}
		}
		doc.Sessions = append(doc.Sessions, session)
		return true, nil
	})
	return session, err
}

func (s *Store) Record(ctx context.Context, name string, solve Solve) (Solve, error) {
	if solve.ID == "" {
		var err error
		solve.ID, err = newID()
		if err != nil {
			return Solve{}, err
		}
	}
	if solve.CreatedAt.IsZero() {
		solve.CreatedAt = time.Now().UTC()
	} else {
		solve.CreatedAt = solve.CreatedAt.UTC()
	}
	if solve.Penalty == "" {
		solve.Penalty = None
	}
	if err := validateSolve(solve); err != nil {
		return Solve{}, err
	}
	err := s.update(ctx, func(doc *Document) (bool, error) {
		i, err := sessionIndex(*doc, name)
		if err != nil {
			return false, err
		}
		doc.Sessions[i].Solves = append(doc.Sessions[i].Solves, solve)
		return true, nil
	})
	return solve, err
}

func (s *Store) Update(ctx context.Context, name, id string, edit Edit) (Solve, error) {
	if edit.DurationNS == nil && edit.Penalty == nil && edit.Notes == nil {
		return Solve{}, fmt.Errorf("update requires duration, penalty, or notes")
	}
	var result Solve
	err := s.update(ctx, func(doc *Document) (bool, error) {
		i, err := sessionIndex(*doc, name)
		if err != nil {
			return false, err
		}
		for j, solve := range doc.Sessions[i].Solves {
			if solve.ID != id {
				continue
			}
			if edit.DurationNS != nil {
				solve.DurationNS = *edit.DurationNS
			}
			if edit.Penalty != nil {
				solve.Penalty = *edit.Penalty
			}
			if edit.Notes != nil {
				solve.Notes = *edit.Notes
			}
			doc.Sessions[i].Solves[j], result = solve, solve
			return true, nil
		}
		return false, fmt.Errorf("%w: solve %q", ErrNotFound, id)
	})
	return result, err
}

func (s *Store) Delete(ctx context.Context, name, id string) error {
	return s.update(ctx, func(doc *Document) (bool, error) {
		i, err := sessionIndex(*doc, name)
		if err != nil {
			return false, err
		}
		for j, solve := range doc.Sessions[i].Solves {
			if solve.ID == id {
				doc.Sessions[i].Solves = append(doc.Sessions[i].Solves[:j], doc.Sessions[i].Solves[j+1:]...)
				return true, nil
			}
		}
		return false, fmt.Errorf("%w: solve %q", ErrNotFound, id)
	})
}

func (s *Store) DeleteSession(ctx context.Context, name string) error {
	return s.update(ctx, func(doc *Document) (bool, error) {
		i, err := sessionIndex(*doc, name)
		if err != nil {
			return false, err
		}
		doc.Sessions = append(doc.Sessions[:i], doc.Sessions[i+1:]...)
		return true, nil
	})
}

// Recover explicitly restores the previous known-good snapshot. It never
// replaces the good backup with corrupt or missing primary data.
func (s *Store) Recover(ctx context.Context) error {
	return s.locked(ctx, func() error {
		data, err := readFile(s.BackupPath())
		if err != nil {
			return err
		}
		if _, err := decode(data); err != nil {
			return fmt.Errorf("backup: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return atomicWrite(s.Path(), data)
	})
}

// checkJSON rejects duplicate keys before typed decoding can overwrite them.
func checkJSON(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON must be valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 64 {
			return fmt.Errorf("JSON nesting exceeds 64 levels")
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			keys := map[string]bool{}
			for decoder.More() {
				token, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := token.(string)
				if !ok {
					return fmt.Errorf("invalid JSON object key")
				}
				if key != strings.ToLower(key) {
					return fmt.Errorf("JSON field names must use lowercase schema names")
				}
				if keys[key] {
					return fmt.Errorf("duplicate JSON key %q", key)
				}
				keys[key] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			close, err := decoder.Token()
			if err != nil || close != json.Delim('}') {
				return fmt.Errorf("invalid JSON object")
			}
		case '[':
			for decoder.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			close, err := decoder.Token()
			if err != nil || close != json.Delim(']') {
				return fmt.Errorf("invalid JSON array")
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter")
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("expected one JSON document")
	}
	return nil
}
