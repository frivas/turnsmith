package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func sourceCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	args = append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)
	c := exec.Command("git", args...)
	c.Dir = dir
	out, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %v\n%s", args, e, out)
	}
	return strings.TrimSpace(string(out))
}
func sourceWrite(t *testing.T, dir, name, text string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(text), 0644); e != nil {
		t.Fatal(e)
	}
}
func sourceFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	sourceCommand(t, dir, "init", "-q")
	sourceWrite(t, dir, "go.mod", "module example.test/source\n\ngo 1.23\n\nrequire example.test/localdep v0.0.0\nreplace example.test/localdep => ./dep\n")
	sourceWrite(t, dir, "main.go", "package main\nimport (\"fmt\";\"example.test/localdep\")\nfunc main(){fmt.Println(localdep.Value)}\n")
	sourceWrite(t, dir, "dep/go.mod", "module example.test/localdep\n\ngo 1.23\n")
	sourceWrite(t, dir, "dep/value.go", "package localdep\nconst Value = \"committed value\"\n")
	sourceWrite(t, dir, "dep/LICENSE", "Fixture dependency license\n")
	sourceWrite(t, dir, "COPYING", "Fixture root license\n")
	sourceWrite(t, dir, "run.sh", "#!/bin/sh\nprintf source\n")
	if e := os.Chmod(filepath.Join(dir, "run.sh"), 0755); e != nil {
		t.Fatal(e)
	}
	sourceCommand(t, dir, "add", ".")
	sourceCommand(t, dir, "update-index", "--chmod=+x", "run.sh")
	sourceCommand(t, dir, "-c", "user.name=Source Fixture", "-c", "user.email=fixture@example.test", "commit", "-qm", "fixture")
	return dir, sourceCommand(t, dir, "rev-parse", "HEAD")
}

type sourceEntry struct {
	header tar.Header
	data   []byte
}

func readSourceArchive(t *testing.T, path string) (map[string]sourceEntry, []string) {
	t.Helper()
	file, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer file.Close()
	gz, e := gzip.NewReader(file)
	if e != nil {
		t.Fatal(e)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	entries := map[string]sourceEntry{}
	var names []string
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(tr)
		if e != nil {
			t.Fatal(e)
		}
		entries[h.Name] = sourceEntry{*h, data}
		names = append(names, h.Name)
	}
	return entries, names
}
func TestSourceArchiveCommittedTreeVendorAndDeterminism(t *testing.T) {
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOWORK", "off")
	repo, commit := sourceFixture(t)
	// These edits must never enter the archive, even though SourceArchive executes
	// vendoring from the repository's current environment.
	sourceWrite(t, repo, "main.go", "dirty working tree\n")
	sourceWrite(t, repo, "dep/value.go", "dirty dependency\n")
	sourceWrite(t, repo, "untracked-secret", "untracked\n")
	stamp := time.Date(2026, 10, 31, 12, 34, 56, 987, time.FixedZone("fixture", 3600))
	out := filepath.Join(t.TempDir(), "source.tar.gz")
	if e := SourceArchive(repo, out, commit, stamp); e != nil {
		t.Fatal(e)
	}
	entries, names := readSourceArchive(t, out)
	prefix := "turnsmith-wca-source/"
	for _, name := range []string{"main.go", "go.mod", "COPYING", "dep/value.go", "vendor/example.test/localdep/value.go", "vendor/example.test/localdep/LICENSE", "vendor/modules.txt", "SOURCE_COMMIT"} {
		if _, ok := entries[prefix+name]; !ok {
			t.Fatal("missing committed or vendored source", name)
		}
	}
	if _, ok := entries[prefix+"untracked-secret"]; ok {
		t.Fatal("untracked data leaked")
	}
	if got := string(entries[prefix+"vendor/example.test/localdep/value.go"].data); !strings.Contains(got, "committed value") || strings.Contains(got, "dirty") {
		t.Fatal("vendored live tree instead of pinned commit", got)
	}
	if got := string(entries[prefix+"SOURCE_COMMIT"].data); got != commit+"\n" {
		t.Fatal(got)
	}
	if !sort.StringsAreSorted(names) {
		t.Fatal("archive entries not sorted", names)
	}
	for name, entry := range entries {
		if !entry.header.ModTime.Equal(stamp.UTC().Truncate(time.Second)) || entry.header.Uid != 0 || entry.header.Gid != 0 || entry.header.Uname != "" || entry.header.Gname != "" {
			t.Fatal("nondeterministic archive metadata", name, entry.header)
		}
	}
	if entries[prefix+"run.sh"].header.Mode != 0755 {
		t.Fatal("executable mode lost")
	}
	other := filepath.Join(t.TempDir(), "again.tar.gz")
	if e := SourceArchive(repo, other, commit, stamp); e != nil {
		t.Fatal(e)
	}
	a, _ := os.ReadFile(out)
	b, _ := os.ReadFile(other)
	if !bytes.Equal(a, b) {
		t.Fatal("same commit and timestamp produced different bytes")
	}
	live, _ := os.ReadFile(filepath.Join(repo, "main.go"))
	if string(live) != "dirty working tree\n" {
		t.Fatal("source archive mutated working tree")
	}
	if _, e := os.Stat(filepath.Join(repo, "vendor")); !os.IsNotExist(e) {
		t.Fatal("vendor written to live tree", e)
	}
}
func TestSourceArchiveNoDependenciesAndFailuresPreserveOutput(t *testing.T) {
	repo := t.TempDir()
	sourceCommand(t, repo, "init", "-q")
	sourceWrite(t, repo, "go.mod", "module example.test/nodeps\ngo 1.23\n")
	sourceWrite(t, repo, "main.go", "package main\nfunc main(){}\n")
	sourceCommand(t, repo, "add", ".")
	sourceCommand(t, repo, "-c", "user.name=Source Fixture", "-c", "user.email=fixture@example.test", "commit", "-qm", "no dependencies")
	commit := sourceCommand(t, repo, "rev-parse", "HEAD")
	out := filepath.Join(t.TempDir(), "source.tar.gz")
	if e := SourceArchive(repo, out, commit, time.Unix(123456, 0)); e != nil {
		t.Fatal(e)
	}
	original, _ := os.ReadFile(out)
	for _, revision := range []string{"HEAD", "--output=elsewhere", strings.Repeat("f", 40)} {
		if e := SourceArchive(repo, out, revision, time.Now()); e == nil {
			t.Fatal("invalid revision accepted", revision)
		}
		after, _ := os.ReadFile(out)
		if !bytes.Equal(original, after) {
			t.Fatal("failed archive replaced existing output")
		}
	}
	sourceWrite(t, repo, "main.go", "package main\nimport _ \"missing.example.test/dependency\"\nfunc main(){}\n")
	sourceCommand(t, repo, "add", ".")
	sourceCommand(t, repo, "-c", "user.name=Source Fixture", "-c", "user.email=fixture@example.test", "commit", "-qm", "invalid module")
	broken := sourceCommand(t, repo, "rev-parse", "HEAD")
	t.Setenv("GOPROXY", "off")
	if e := SourceArchive(repo, out, broken, time.Now()); e == nil || !strings.Contains(e.Error(), "vendor committed source") {
		t.Fatal("vendoring failure hidden", e)
	}
	after, _ := os.ReadFile(out)
	if !bytes.Equal(original, after) {
		t.Fatal("vendor failure replaced existing archive")
	}
}
func TestExtractSourceTreeRejectsUnsafeEntries(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind byte
		link string
	}{{"../escape", tar.TypeReg, ""}, {"/absolute", tar.TypeReg, ""}, {`..\escape`, tar.TypeReg, ""}, {"C:/absolute", tar.TypeReg, ""}, {"link", tar.TypeSymlink, "../escape"}, {"hard", tar.TypeLink, "../escape"}, {"device", tar.TypeChar, ""}} {
		t.Run(strings.ReplaceAll(tc.name, "/", "_"), func(t *testing.T) {
			var data bytes.Buffer
			tw := tar.NewWriter(&data)
			if e := tw.WriteHeader(&tar.Header{Name: tc.name, Typeflag: tc.kind, Linkname: tc.link, Mode: 0644}); e != nil {
				t.Fatal(e)
			}
			if e := tw.Close(); e != nil {
				t.Fatal(e)
			}
			if e := extractSourceTree(&data, t.TempDir()); e == nil {
				t.Fatal("unsafe archive entry accepted", tc)
			}
		})
	}
}

// Windows does not expose Unix executable bits through os.Stat. Preserve the
// authoritative mode from Git's tar headers even when the staging FS loses it.
func TestSourceArchivePreservesGitModeWithoutFilesystemExecBits(t *testing.T) {
	var snapshot bytes.Buffer
	writer := tar.NewWriter(&snapshot)
	if e := writer.WriteHeader(&tar.Header{Name: "run.sh", Mode: 0755, Size: 2, Typeflag: tar.TypeReg}); e != nil {
		t.Fatal(e)
	}
	if _, e := writer.Write([]byte("sh")); e != nil {
		t.Fatal(e)
	}
	if e := writer.Close(); e != nil {
		t.Fatal(e)
	}
	tree := t.TempDir()
	modes := map[string]bool{}
	if e := extractSourceTreeModes(&snapshot, tree, modes); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(filepath.Join(tree, "run.sh"), 0644); e != nil {
		t.Fatal(e)
	}
	out := filepath.Join(t.TempDir(), "source.tar.gz")
	if e := archiveSourceTree(tree, out, time.Unix(0, 0), modes); e != nil {
		t.Fatal(e)
	}
	entries, _ := readSourceArchive(t, out)
	if entries["turnsmith-wca-source/run.sh"].header.Mode != 0755 {
		t.Fatal("lost Git executable mode")
	}
}
