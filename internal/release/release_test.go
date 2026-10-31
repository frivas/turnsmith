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
	if strings.Contains(notes, "## Highlights") || !strings.Contains(notes, "## Changelog") {
		t.Fatal("missing changelog file changed the commit-only format")
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

func TestVersionHighlights(t *testing.T) {
	for _, heading := range []string{"## 0.3.0 - 2026-10-31", "## [0.3.0]", "## [0.3.0] - 2026-10-31"} {
		text := "# Changelog\n\n## 0.3.0-rc.1 - 2026-10-30\n\nWrong prerelease notes.\n\n" + heading + "\n\n### Added\n\n- Selected feature.\n\n## [0.2.0]\n\nOlder notes.\n"
		if got := versionHighlights(text, "0.3.0"); got != "### Added\n\n- Selected feature." {
			t.Fatalf("%q: %q", heading, got)
		}
	}
	for _, text := range []string{"", "## 0.3.0-rc.1\nWrong prerelease notes", "## 0.3.0+build.1\nOther build notes", "## [0.3.00]\nWrong version", "## 0.3.0\n\n## 0.2.0\nOther version"} {
		if got := versionHighlights(text, "0.3.0"); got != "" {
			t.Fatalf("unexpected highlights from %q: %q", text, got)
		}
	}
}

func TestChangelogHighlightsUseSelectedCommit(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		args = append([]string{"-c", "user.name=Release Test", "-c", "user.email=release@example.invalid", "-c", "commit.gpgsign=false"}, args...)
		command := exec.Command("git", args...)
		command.Dir = dir
		data, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, data)
		}
		return strings.TrimSpace(string(data))
	}
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "CHANGELOG.md"), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	git("init")
	write("# Changelog\n\n## 0.3.0 - 2026-10-31\n\n- Selected commit feature.\n\n## 0.2.0\n\n- Earlier feature.\n")
	git("add", "CHANGELOG.md")
	git("commit", "-m", "Prepare release")
	selected := git("rev-parse", "HEAD")
	write("## [0.3.0]\n\n- Newer HEAD feature.\n")
	git("add", "CHANGELOG.md")
	git("commit", "-m", "Update notes")
	write("## [0.3.0]\n\n- Dirty working-tree feature.\n")
	notes, err := Changelog(dir, "0.3.0", selected)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Highlights\n\n- Selected commit feature.\n\n## Commits", "Prepare release", "/commit/" + selected} {
		if !strings.Contains(notes, want) {
			t.Fatalf("missing %q:\n%s", want, notes)
		}
	}
	for _, unwanted := range []string{"Newer HEAD feature", "Dirty working-tree feature", "Earlier feature", "Update notes"} {
		if strings.Contains(notes, unwanted) {
			t.Fatalf("included %q:\n%s", unwanted, notes)
		}
	}
	notes, err = Changelog(dir, "0.4.0", selected)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(notes, "## Highlights") || !strings.Contains(notes, "## Changelog") || !strings.Contains(notes, "Prepare release") {
		t.Fatal("unmatched version lost commit-only fallback", notes)
	}
}
