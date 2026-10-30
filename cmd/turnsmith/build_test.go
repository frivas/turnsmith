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
		{"missing info", cli.Build{}, nil, defaults},
		{"empty info", cli.Build{}, &debug.BuildInfo{}, defaults},
		{"tagged module", cli.Build{}, &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}, cli.Build{Version: "0.1.0", Commit: "unknown", Date: "unknown"}},
		{"pseudo version", cli.Build{}, &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261030000000-abcdef123456"}}, cli.Build{Version: "0.0.0-20261030000000-abcdef123456", Commit: "unknown", Date: "unknown"}},
		{"local checkout", cli.Build{}, &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: settings}, cli.Build{Version: "0.1.0-dev", Commit: "abc1234", Date: "2026-10-30T00:00:00Z"}},
		{"dirty checkout", cli.Build{}, &debug.BuildInfo{Settings: append(append([]debug.BuildSetting{}, settings...), debug.BuildSetting{Key: "vcs.modified", Value: "true"})}, cli.Build{Version: "0.1.0-dev", Commit: "abc1234-dirty", Date: "2026-10-30T00:00:00Z"}},
		{"release overrides", release, &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}, Settings: settings}, release},
		{"independent defaults", cli.Build{Version: "0.1.0"}, &debug.BuildInfo{Settings: settings}, cli.Build{Version: "0.1.0", Commit: "abc1234", Date: "2026-10-30T00:00:00Z"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveBuildMetadata(tc.build, tc.info); got != tc.want {
				t.Fatalf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

func TestExplicitDevelopmentMetadata(t *testing.T) {
	explicit := cli.Build{Version: "0.1.0-dev", Commit: "unknown", Date: "unknown"}
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef1"}}}
	if got := resolveBuildMetadata(explicit, info); got != explicit {
		t.Fatalf("explicit development metadata changed: got %+v; want %+v", got, explicit)
	}
}
