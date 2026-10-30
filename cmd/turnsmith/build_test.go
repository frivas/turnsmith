package main

import (
	"runtime/debug"
	"testing"

	"github.com/frivas/turnsmith/internal/cli"
)

func TestBuildMetadata(t *testing.T) {
	defaults := cli.Build{Version: "0.1.0-dev", Commit: "unknown", Date: "unknown"}
	settings := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "abc1234"},
		{Key: "vcs.time", Value: "2026-10-30T00:00:00Z"},
	}
	release := cli.Build{Version: "0.1.0", Commit: "def5678", Date: "2026-10-31T00:00:00Z"}
	for _, tc := range []struct {
		name  string
		build cli.Build
		info  *debug.BuildInfo
		want  cli.Build
	}{
		{"missing info", defaults, nil, defaults},
		{"empty info", defaults, &debug.BuildInfo{}, defaults},
		{"tagged module", defaults, &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}, cli.Build{Version: "0.1.0", Commit: "unknown", Date: "unknown"}},
		{"pseudo version", defaults, &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261030000000-abcdef123456"}}, cli.Build{Version: "0.0.0-20261030000000-abcdef123456", Commit: "unknown", Date: "unknown"}},
		{"local checkout", defaults, &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: settings}, cli.Build{Version: "0.1.0-dev", Commit: "abc1234", Date: "2026-10-30T00:00:00Z"}},
		{"dirty checkout", defaults, &debug.BuildInfo{Settings: append(append([]debug.BuildSetting{}, settings...), debug.BuildSetting{Key: "vcs.modified", Value: "true"})}, cli.Build{Version: "0.1.0-dev", Commit: "abc1234-dirty", Date: "2026-10-30T00:00:00Z"}},
		{"release overrides", release, &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}, Settings: settings}, release},
		{"independent defaults", cli.Build{Version: "0.1.0", Commit: "unknown", Date: "unknown"}, &debug.BuildInfo{Settings: settings}, cli.Build{Version: "0.1.0", Commit: "abc1234", Date: "2026-10-30T00:00:00Z"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveBuildMetadata(tc.build, tc.info); got != tc.want {
				t.Fatalf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}
