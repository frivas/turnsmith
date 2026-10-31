package release

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// SourceArchive packages the committed source plus its vendored Go dependencies.
// It never reads the working tree's source or changes its module files. Vendoring
// honors the caller's module cache and GOPROXY settings; GOPROXY=off works when
// the pinned modules are already cached. The resulting tree builds with vendor
// mode and needs no module downloads. This function is used only by release tooling.
func SourceArchive(repoDir, outPath, commit string, timestamp time.Time) error {
	if !regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`).MatchString(commit) {
		return fmt.Errorf("source archive requires a hexadecimal Git commit")
	}
	repoDir, e := filepath.Abs(repoDir)
	if e != nil {
		return e
	}
	outPath, e = filepath.Abs(outPath)
	if e != nil {
		return e
	}
	resolve := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", "--end-of-options", commit+"^{commit}")
	resolved, e := resolve.Output()
	if e != nil {
		return fmt.Errorf("resolve source commit: %w", e)
	}
	revision := strings.TrimSpace(string(resolved))
	temp, e := os.MkdirTemp("", "turnsmith-source-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(temp)
	tree := filepath.Join(temp, "tree")
	if e = os.Mkdir(tree, 0700); e != nil {
		return e
	}
	// Stage git archive to disk rather than buffering potentially large source
	// trees or extracting a live process stream with an unobserved exit status.
	snapshot, e := os.Create(filepath.Join(temp, "commit.tar"))
	if e != nil {
		return e
	}
	git := exec.Command("git", "-C", repoDir, "archive", "--format=tar", revision)
	git.Stdout = snapshot
	var gitErrors strings.Builder
	git.Stderr = &gitErrors
	runErr := git.Run()
	closeErr := snapshot.Close()
	if runErr != nil {
		return fmt.Errorf("archive source commit: %w: %s", runErr, gitErrors.String())
	}
	if closeErr != nil {
		return closeErr
	}
	snapshot, e = os.Open(filepath.Join(temp, "commit.tar"))
	if e != nil {
		return e
	}
	executable := map[string]bool{}
	extractErr := extractSourceTreeModes(snapshot, tree, executable)
	closeErr = snapshot.Close()
	if extractErr != nil {
		return extractErr
	}
	if closeErr != nil {
		return closeErr
	}
	if _, e = os.Stat(filepath.Join(tree, "go.mod")); e != nil {
		return fmt.Errorf("source commit has no go.mod: %w", e)
	}
	vendor := exec.Command("go", "mod", "vendor")
	vendor.Dir = tree
	vendor.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	if output, err := vendor.CombinedOutput(); err != nil {
		return fmt.Errorf("vendor committed source: %w: %s", err, output)
	}
	if e = os.WriteFile(filepath.Join(tree, "SOURCE_COMMIT"), []byte(revision+"\n"), 0644); e != nil {
		return e
	}
	return archiveSourceTree(tree, outPath, timestamp, executable)
}

func extractSourceTree(reader io.Reader, root string) error {
	return extractSourceTreeModes(reader, root, nil)
}

func extractSourceTreeModes(reader io.Reader, root string, executable map[string]bool) error {
	tr := tar.NewReader(reader)
	for {
		header, e := tr.Next()
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return fmt.Errorf("read source snapshot: %w", e)
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		name := header.Name
		clean := path.Clean(name)
		if name == "" || strings.Contains(name, "\\") || path.IsAbs(name) || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, ":") {
			return fmt.Errorf("unsafe source snapshot path %q", name)
		}
		if clean == "." {
			if header.Typeflag == tar.TypeDir {
				continue
			}
			return fmt.Errorf("invalid source root entry")
		}
		target := filepath.Join(root, filepath.FromSlash(clean))
		switch header.Typeflag {
		case tar.TypeDir:
			if e = os.MkdirAll(target, 0755); e != nil {
				return e
			}
		case tar.TypeReg, tar.TypeRegA:
			if e = os.MkdirAll(filepath.Dir(target), 0755); e != nil {
				return e
			}
			if executable != nil {
				executable[clean] = header.Mode&0111 != 0
			}
			mode := os.FileMode(0644)
			if header.Mode&0111 != 0 {
				mode = 0755
			}
			file, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if e != nil {
				return e
			}
			_, copyErr := io.Copy(file, tr)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("unsupported source snapshot entry %q (links and special files are not extracted)", name)
		}
	}
}

func archiveSourceTree(tree, outPath string, timestamp time.Time, executable map[string]bool) (err error) {
	var paths []string
	if err = filepath.WalkDir(tree, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name != tree {
			paths = append(paths, name)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("source archive cannot contain symlink %q", name)
		}
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(paths)
	if err = os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(outPath), ".turnsmith-source-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer func() { _ = file.Close(); _ = os.Remove(temporary) }()
	timestamp = timestamp.UTC().Truncate(time.Second)
	gz := gzip.NewWriter(file)
	gz.Header.ModTime = timestamp
	gz.Header.OS = 255
	tw := tar.NewWriter(gz)
	rootHeader := &tar.Header{Name: "turnsmith-wca-source/", Mode: 0755, Typeflag: tar.TypeDir, ModTime: timestamp, Format: tar.FormatPAX}
	if err = tw.WriteHeader(rootHeader); err != nil {
		return err
	}
	for _, name := range paths {
		info, e := os.Lstat(name)
		if e != nil {
			return e
		}
		relative, e := filepath.Rel(tree, name)
		if e != nil {
			return e
		}
		archiveName := "turnsmith-wca-source/" + filepath.ToSlash(relative)
		mode := int64(0644)
		kind := byte(tar.TypeReg)
		size := info.Size()
		if info.IsDir() {
			mode = 0755
			kind = tar.TypeDir
			archiveName += "/"
			size = 0
		} else if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported source file %q", name)
		} else if executable[filepath.ToSlash(relative)] || info.Mode()&0111 != 0 {
			mode = 0755
		}
		header := &tar.Header{Name: archiveName, Mode: mode, Typeflag: kind, Size: size, ModTime: timestamp, Format: tar.FormatPAX}
		if err = tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			continue
		}
		input, e := os.Open(name)
		if e != nil {
			return e
		}
		_, copyErr := io.Copy(tw, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if err = tw.Close(); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	if err = file.Chmod(0644); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, outPath)
}
