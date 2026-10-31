# Turnsmith handoff

Updated 2026-10-31. Workspace `/Users/francisco/Code/personal/turnsmith`.

## Completed release

[Turnsmith v0.3.0](https://github.com/frivas/turnsmith/releases/tag/v0.3.0)
is published from `6d8dd8979ce429d21a7fac45278ea5bf6ff00f72`.
[Native candidate CI](https://github.com/frivas/turnsmith/actions/runs/41229226857)
and the [actual release pipeline](https://github.com/frivas/turnsmith/actions/runs/41229421613)
passed on macOS, Linux and Windows. All nine public assets were downloaded and
verified: six platform archives, backend corresponding source, changelog and
checksums. Each platform archive contains both native executables.

Both published macOS ARM64 binaries are installed in `~/.local/bin` and report
v0.3.0 with the tagged commit. The shell resolves `turnsmith` to that installation.
Installed 222, 333 and 444 generation passed. The installed binary hashes are:

- turnsmith: `be3c8fd7e5291ca9f3cd49c558a3230bd850092bc7aa86e82ba0b6b45f2a8991`
- turnsmith-wca: `d0ddbf396966fcba0390a3f3e586752080662a43b4a7985e4506e078de2f29fa`

The prior executable was backed up under the temporary directory
`turnsmith-before-0.3.0-etz55709` before replacement.

[Deliver the complete terminal practice roadmap](https://github.com/frivas/turnsmith/issues/8)
and every child ticket are closed with implementation and verification evidence.
No work remains pending for this request. The earlier approval-credit failure
was resolved and did not prevent final publication or installation.

## Behavior and boundaries

Default 222/333/444 practice uses `wca`. It applies minimum optimal distances of
four moves for 222 and two for 333/444. Unseeded generation uses direct
cryptographic randomness; JSON seed is empty and randomness is `cryptographic`.
Explicit seeds and daily sets are reproducible `seeded-practice`. These are
practice generators, not official WCA competition program approval.

Original explicit `333/random-move`, `444/random-move` and
`222/333/random-state` methods and their vectors remain unchanged. Choosing a
length requires `--method random-move`. The historical RNG domain stays
`scramble/rng/v1`. JSON schema remains 1 with optional additional metadata.

The original MIT CLI links permissive dependencies. Its separate native Go
`turnsmith-wca` backend contains the GPL-3.0-or-later TNoodle threephase port,
with Shuang Chen attribution and the pinned upstream revision recorded in the
source. Keep its COPYING and notices. Do not relabel that component MIT or link
it into the MIT executable. There is no Java, runtime table download or runtime
network dependency. Source archives include vendored dependencies for offline
builds, and binaries and corresponding source must come from the same clean
committed tree.

Other delivered features include imported algorithm inspection/inversion,
interactive colored guidance and final diagrams, portable colors/completions,
local sessions/timing/penalties/statistics and CSV/JSON recovery, user-owned
algorithm training, and daily/shared practice sets. Default generation writes no
practice files. See README and the user guides in docs.

## Verification

Go unit/integration tests, short race checks, vet, formatting and module
verification passed. Native full sweeps include 1000 333 targets and 100
independently checked 444 targets. Go build metadata, archive contents,
permissions, OS/CPU headers and checksums passed for all 12 packaged executables.
Both executables built offline from corresponding source; its complete test
suite passed offline during verification. Actual public Go installs at v0.3.0
succeeded for both commands and generated 444 scrambles together. Mac PTYs
verified navigation, timing, training, resize guards and terminal restoration.

The first Windows candidate exposed lost source-archive executable bits.
The final fix preserves Git tar permissions across staging filesystems, with a
regression test and successful native Windows CI. Do not weaken that check.

## Development and account rules

Module `github.com/frivas/turnsmith`, Go 1.27.0. Main entrypoints are
`cmd/turnsmith` and `cmd/turnsmith-wca`. Useful cache overrides in this sandbox:
`GOCACHE=/tmp/scramble-go-cache GOMODCACHE=/tmp/turnsmith-go-mod-cache`.
The RTK instruction file `/Users/francisco/.codex/RTK.md` is absent.

GitHub repository writes use `frivas`; the default active account remains
`frivas-at-navteca`. Use a process-scoped frivas credential for gh commands,
never print or save it. Signed Git commits and SSH pushes work. Do not alter
global account or signing settings. A valid SemVer tag automatically runs CI,
packages both binaries for all six targets and publishes the complete release.
