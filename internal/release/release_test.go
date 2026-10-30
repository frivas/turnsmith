package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateVersion(t *testing.T) {
	for _, version := range []string{"0.1.0", "1.0.0", "2.12.30", "1.0.0-rc.1", "1.0.0-alpha.0", "1.0.0-0", "1.0.0-01a", "1.0.0+build.01", "1.0.0-rc.1+build.2"} {
		if err := ValidateVersion(version); err != nil {
			t.Errorf("valid %q rejected: %v", version, err)
		}
	}
	for _, version := range []string{"", "v1.0.0", "1.0", "01.0.0", "1.01.0", "1.0.01", "1.0.0-01", "1.0.0-rc.01", "1.0.0-", "1.0.0+", "1.0.0-a..b", "1.0.0+a..b", "1.0.0 ", "1.0.0\n", "-1.0.0", "1.0.0/a"} {
		if err := ValidateVersion(version); err == nil {
			t.Errorf("invalid %q accepted", version)
		}
	}
}

func TestChangelog(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		args = append([]string{"-c", "user.name=Release Test", "-c", "user.email=release@example.invalid", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)
		command := exec.Command("git", args...)
		command.Dir = dir
		data, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	commit := func(message string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte(message), 0644); err != nil {
			t.Fatal(err)
		}
		git("add", "source.txt")
		git("commit", "-m", message)
		return git("rev-parse", "HEAD")
	}
	git("init")
	first := commit("Initial generator")
	second := commit("Add batch generation")
	notes, err := Changelog(dir, "0.1.0", second)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"First release", "Initial generator", "Add batch generation", "/commit/" + first, "/commit/" + second} {
		if !strings.Contains(notes, want) {
			t.Errorf("initial changelog missing %q:\n%s", want, notes)
		}
	}
	for _, os := range []string{"darwin", "linux", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			ext := ".tar.gz"
			if os == "windows" {
				ext = ".zip"
			}
			if !strings.Contains(notes, "turnsmith_0.1.0_"+os+"_"+arch+ext) {
				t.Errorf("missing download for %s/%s", os, arch)
			}
		}
	}
	git("tag", "-a", "v0.1.0", "-m", "First release")
	latest := commit("Fix direct-push bug")
	git("tag", "vnot-semver")
	git("tag", "v0.1.1")
	notes, err = Changelog(dir, "0.1.1", latest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "Fix direct-push bug") || !strings.Contains(notes, "/compare/v0.1.0...v0.1.1") || strings.Contains(notes, "Initial generator") || strings.Contains(notes, "Add batch generation") {
		t.Fatalf("incorrect incremental changelog:\n%s", notes)
	}
	// A second version may intentionally release the same source commit.
	notes, err = Changelog(dir, "0.1.2", latest)
	if err != nil || !strings.Contains(notes, "No source changes since") {
		t.Fatalf("same-source release: %v\n%s", err, notes)
	}
	if _, err := Changelog(dir, "01.0.0", latest); err == nil {
		t.Fatal("invalid version accepted for changelog")
	}
}
