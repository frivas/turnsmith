// Package release supports development-time release tooling, not the executable.
package release

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateVersion accepts SemVer 2.0.0 without the Git tag's leading v.
func ValidateVersion(version string) error {
	pattern := `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`
	if !regexp.MustCompile(pattern).MatchString(version) {
		return fmt.Errorf("expected MAJOR.MINOR.PATCH with optional prerelease/build metadata, without leading v")
	}
	core, _, _ := strings.Cut(version, "+")
	_, prerelease, _ := strings.Cut(core, "-")
	for _, identifier := range strings.Split(prerelease, ".") {
		if len(identifier) > 1 && identifier[0] == '0' && strings.Trim(identifier, "0123456789") == "" {
			return fmt.Errorf("numeric prerelease identifiers cannot have leading zeros")
		}
	}
	return nil
}
