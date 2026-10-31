package release

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// Changelog lists commits since the closest reachable SemVer tag. With no
// earlier release tag it lists the full history. The release's own tag is excluded.
// Matching CHANGELOG.md highlights come from the resolved source commit, never
// from the working tree or a newer HEAD.
func Changelog(dir, version, commit string) (string, error) {
	if err := ValidateVersion(version); err != nil {
		return "", err
	}
	git := func(args ...string) (string, error) {
		command := exec.Command("git", args...)
		command.Dir = dir
		data, err := command.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(string(data)))
		}
		return strings.TrimSpace(string(data)), nil
	}
	if commit == "unknown" {
		commit = "HEAD"
	}
	head, err := git("rev-parse", "--verify", commit+"^{commit}")
	if err != nil {
		return "", err
	}
	tags, err := git("tag", "--merged", head)
	if err != nil {
		return "", err
	}
	args := []string{"describe", "--tags", "--abbrev=0"}
	for _, tag := range strings.Fields(tags) {
		if strings.HasPrefix(tag, "v") && tag != "v"+version && ValidateVersion(strings.TrimPrefix(tag, "v")) == nil {
			args = append(args, "--match", tag)
		}
	}
	previous := ""
	if len(args) > 3 {
		previous, err = git(append(args, head)...)
		if err != nil {
			return "", err
		}
	}
	span := head
	if previous != "" {
		span = previous + ".." + head
	}
	log, err := git("log", "--reverse", "--format=%H%x09%s", span)
	if err != nil {
		return "", err
	}
	const repository = "https://github.com/frivas/turnsmith"
	var notes strings.Builder
	fmt.Fprintf(&notes, "# Turnsmith v%s\n\n", version)
	// Highlights are optional for older commits and repositories without a
	// changelog. Commit-derived notes remain available in either case.
	changelog, _ := git("show", head+":CHANGELOG.md")
	if highlights := versionHighlights(changelog, version); highlights != "" {
		fmt.Fprintf(&notes, "## Highlights\n\n%s\n\n## Commits\n\n", highlights)
	} else {
		notes.WriteString("## Changelog\n\n")
	}
	if previous == "" {
		notes.WriteString("First release. Changes from the initial commit:\n\n")
	}
	if log == "" {
		notes.WriteString("No source changes since the previous release.\n")
	} else {
		escape := strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;")
		for _, line := range strings.Split(log, "\n") {
			sha, subject, _ := strings.Cut(line, "\t")
			fmt.Fprintf(&notes, "- %s ([%s](%s/commit/%s))\n", escape.Replace(subject), sha[:7], repository, sha)
		}
	}
	if previous != "" {
		fmt.Fprintf(&notes, "\n[Full comparison](%s/compare/%s...%s)\n", repository, url.PathEscape(previous), url.PathEscape("v"+version))
	}
	fmt.Fprintf(&notes, "\nSource commit: [%s](%s/commit/%s).\n\n## Downloads\n\n", head[:7], repository, head)
	notes.WriteString("| Platform | Architecture | Archive |\n| --- | --- | --- |\n")
	for _, os := range []string{"darwin", "linux", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			extension := ".tar.gz"
			if os == "windows" {
				extension = ".zip"
			}
			name := fmt.Sprintf("turnsmith_%s_%s_%s%s", version, os, arch, extension)
			fmt.Fprintf(&notes, "| %s | %s | [%s](%s/releases/download/%s/%s) |\n", os, arch, name, repository, url.PathEscape("v"+version), url.PathEscape(name))
		}
	}
	fmt.Fprintf(&notes, "\nArchives include the CLI and its separate GPL-licensed native Go 4x4 backend, with no Java requirement. [Backend corresponding source with vendored dependencies](%s/releases/download/%s/turnsmith-wca_%s_source.tar.gz). See BACKEND-LICENSE.txt and THIRD_PARTY_NOTICES.md in each package.\n", repository, url.PathEscape("v"+version), url.PathEscape(version))
	notes.WriteString("\nVerify downloads against the attached `SHA256SUMS`. macOS and Windows binaries are unsigned.\n")
	return notes.String(), nil
}

func versionHighlights(changelog, version string) string {
	lines := strings.Split(strings.ReplaceAll(changelog, "\r\n", "\n"), "\n")
	start := -1
	for index, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		if start >= 0 {
			return strings.TrimSpace(strings.Join(lines[start:index], "\n"))
		}
		fields := strings.Fields(strings.TrimPrefix(line, "## "))
		if len(fields) > 0 && (fields[0] == version || fields[0] == "["+version+"]") {
			start = index + 1
		}
	}
	if start >= 0 {
		return strings.TrimSpace(strings.Join(lines[start:], "\n"))
	}
	return ""
}
