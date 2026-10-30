//go:build ignore

// Run from the repository root: go run ./scripts/release.go --version 0.1.0
// This developer tool has no third-party dependencies; it is not part of turnsmith.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/frivas/turnsmith/internal/release"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}
func run() error {
	version := flag.String("version", "0.1.0-dev", "semantic version without v")
	commit := flag.String("commit", "unknown", "Git commit or unknown")
	date := flag.String("date", "", "RFC3339 build date; defaults to current UTC time")
	out := flag.String("out", "dist", "output directory")
	checkVersion := flag.Bool("check-version", false, "validate the version without building")
	changelog := flag.Bool("changelog", false, "generate and package the Git changelog")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if err := release.ValidateVersion(*version); err != nil {
		return err
	}
	if *checkVersion {
		return nil
	}
	if *commit != "unknown" && !regexp.MustCompile(`^[0-9a-f]{7,64}$`).MatchString(*commit) {
		return fmt.Errorf("invalid commit")
	}
	timestamp := time.Now().UTC().Truncate(time.Second)
	if *date != "" {
		var err error
		timestamp, err = time.Parse(time.RFC3339, *date)
		if err != nil {
			return fmt.Errorf("build date: %w", err)
		}
		timestamp = timestamp.UTC()
	}
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run from repository root: %w", err)
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		return err
	}
	changelogPath := ""
	if *changelog {
		notes, err := release.Changelog(".", *version, *commit)
		if err != nil {
			return fmt.Errorf("changelog: %w", err)
		}
		changelogPath = filepath.Join(*out, "CHANGELOG.md")
		if err := os.WriteFile(changelogPath, []byte(notes), 0644); err != nil {
			return err
		}
	}
	temp, err := os.MkdirTemp("", "turnsmith-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	var sums strings.Builder
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			executable := "turnsmith"
			extension := ".tar.gz"
			if goos == "windows" {
				executable += ".exe"
				extension = ".zip"
			}
			binary := filepath.Join(temp, executable)
			flags := fmt.Sprintf("-s -w -X main.version=%s -X main.commit=%s -X main.date=%s", *version, *commit, timestamp.Format(time.RFC3339))
			command := exec.Command("go", "build", "-trimpath", "-ldflags", flags, "-o", binary, "./cmd/turnsmith")
			command.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+arch, "CGO_ENABLED=0")
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			if err := command.Run(); err != nil {
				return fmt.Errorf("build %s/%s: %w", goos, arch, err)
			}
			name := fmt.Sprintf("turnsmith_%s_%s_%s%s", *version, goos, arch, extension)
			path := filepath.Join(*out, name)
			if err := archive(path, executable, binary, changelogPath, timestamp, goos == "windows"); err != nil {
				return fmt.Errorf("archive %s: %w", name, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fmt.Fprintf(&sums, "%x  %s\n", sha256.Sum256(data), name)
			fmt.Println(path)
		}
	}
	return os.WriteFile(filepath.Join(*out, "SHA256SUMS"), []byte(sums.String()), 0644)
}
func archive(path, executable, binary, changelog string, timestamp time.Time, windows bool) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if e := file.Close(); err == nil {
			err = e
		}
	}()
	type asset struct {
		name, path string
		mode       int64
	}
	assets := []asset{{executable, binary, 0755}, {"README.md", "README.md", 0644}, {"LICENSE", "LICENSE", 0644}, {"THIRD_PARTY_NOTICES.md", "THIRD_PARTY_NOTICES.md", 0644}}
	if changelog != "" {
		assets = append(assets, asset{"CHANGELOG.md", changelog, 0644})
	}
	if windows {
		writer := zip.NewWriter(file)
		defer func() {
			if e := writer.Close(); err == nil {
				err = e
			}
		}()
		for _, a := range assets {
			data, e := os.ReadFile(a.path)
			if e != nil {
				return e
			}
			header := &zip.FileHeader{Name: a.name, Method: zip.Deflate, Modified: timestamp}
			header.SetMode(os.FileMode(a.mode))
			entry, e := writer.CreateHeader(header)
			if e != nil {
				return e
			}
			if _, e = entry.Write(data); e != nil {
				return e
			}
		}
	} else {
		compressed := gzip.NewWriter(file)
		defer func() {
			if e := compressed.Close(); err == nil {
				err = e
			}
		}()
		writer := tar.NewWriter(compressed)
		defer func() {
			if e := writer.Close(); err == nil {
				err = e
			}
		}()
		for _, a := range assets {
			data, e := os.ReadFile(a.path)
			if e != nil {
				return e
			}
			if e = writer.WriteHeader(&tar.Header{Name: a.name, Mode: a.mode, Size: int64(len(data)), ModTime: timestamp, Typeflag: tar.TypeReg}); e != nil {
				return e
			}
			if _, e = io.Copy(writer, strings.NewReader(string(data))); e != nil {
				return e
			}
		}
	}
	return nil
}
