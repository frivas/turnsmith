//go:build ignore

// Cross-platform gofmt check used by CI and contributors.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	failed := false
	for _, root := range []string{"cmd", "internal", "scripts"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			original, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			formatted, err := format.Source(original)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if !bytes.Equal(original, formatted) {
				fmt.Fprintln(os.Stderr, "run gofmt -w", path)
				failed = true
			}
			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}
