package main

import (
	"runtime/debug"
	"strings"

	"github.com/frivas/turnsmith/internal/cli"
)

// Explicit release metadata wins; embedded Go metadata fills available defaults.
func resolveBuildMetadata(build cli.Build, info *debug.BuildInfo) cli.Build {
	if info == nil {
		return build
	}
	if build.Version == developmentVersion && info.Main.Version != "" && info.Main.Version != "(devel)" {
		build.Version = strings.TrimPrefix(info.Main.Version, "v")
	}
	var revision, timestamp string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			timestamp = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if build.Commit == "unknown" && revision != "" {
		build.Commit = revision
		if modified {
			build.Commit += "-dirty"
		}
	}
	if build.Date == "unknown" && timestamp != "" {
		build.Date = timestamp
	}
	return build
}
