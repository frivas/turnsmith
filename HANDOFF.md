# Turnsmith continuation

Updated 2026-10-31. Workspace `/Users/francisco/Code/personal/turnsmith`.
Current public stable release and installed Mac binary are still v0.2.0.
v0.3.0 has NOT been tagged or published yet. Do not infer publication from README's prepared links.

## Release verification in progress

GitHub access and automatic approval are restored. The maintainer explicitly
requested publication, all Go checks, upload and Mac installation. Full Go tests,
short race tests, vet, formatting and diff checks pass on the final WCA source.
The earlier credit failure did not execute any Git command. Continue with signed
commit/push, exact-commit packaging including corresponding source, hosted CI,
automatic v0.3.0 publication, published-asset verification, both Mac binaries and
remaining tracker closure. There is no current blocker.

## Active task and authorization

The maintainer requested autonomous implementation of all researched terminal
cubing improvements using Wayfinder, release and Mac installation, with questions
only for blockers. Then expanded scope to WCA-style valid scrambles, explicitly
rejected Java, and requested an own Go backend derived from TNoodle. Continue
until implementation, verification, publication, installation and tracker closure
are complete. Default 222/333/444 practice now uses method `wca`; explicit legacy
`random-move` and `random-state` methods and their vectors remain unchanged.

Map: [Deliver the complete terminal practice roadmap](https://github.com/frivas/turnsmith/issues/8).
Original six feature tickets are closed with evidence. Release ticket15 remains
open. Added native WCA sampling ticket16 and native Go 444 backend ticket17 are
open, assigned to frivas and blocking15. Map execution notes override skill's
chart-only and one-ticket limits and explicitly authorize parallel agents.

## Completed baseline commits

- `f41f378` complete terminal roadmap: input/inspect/inverse/final diagrams,
  interactive guide, portability/completions, sessions/timer/statistics,
  user-owned drills, native222/333 random-state, 444 random-move and daily sets.
- `27e2865` input bounds/resize fixes.
- `8cfeed0` Windows pending-directory-deletion lock retry, preserving ownership.
- `24f4489` detailed committed changelog highlights in generated releases.

All are pushed. [Native CI for24f4489](https://github.com/frivas/turnsmith/actions/runs/41181473996)
passed on macOS/Linux/Windows, including Linux full race and fuzz tests. Six
candidate archives and real Mac PTYs passed. These results predate the current
WCA expansion and do NOT validate its final source.

## Current WCA expansion, not committed yet

- `internal/randomstate/wca.go`: additive Cube222WCA/Cube333WCA, methodwca/v1,
  optimal-distance rejection below4/2, independently checked shallow states.
- `internal/random/entropy.go`, app/puzzle capability DirectEntropy: unseededwca
  uses fresh cryptographic words; Result.Seed empty and Randomnesscryptographic.
  Explicitseed remains deterministic, Randomnessseeded-practice. Original
  streams/JSON unchanged for legacy methods. Entropy failure cancels and emits no
  partial result. Timer must remain unseeded on later cryptographic attempts.
- `internal/randomstate/facelets.go`: MIT native333 reduction adapter, normalizes
  six center labels before conversion, validates counts/pieces/twist/flip/parity.
- `internal/tnoodle444`: native Go adaptation of Shuang Chen's GPL3+ threephase
  from thewca/tnoodle-lib commitd01a947d9f028f38085cda9b0507a9cf3d3f38a8.
  Keep GPL headers and COPYING. Do NOT relabel this code MIT. It is linked only
  by the separate `cmd/turnsmith-wca` GPL backend, never by the MIT main CLI.
  Three-phase reduction uses existing Go333 solver callback, no Java or runtime
  downloaded tables. All100 sampled targets passed independent geometry tests.
- `internal/wcaengine`: MIT process adapter to sibling turnsmith-wca, bounded
  JSON-line protocol1 and algorithmversion1, exact Uint64 entropy broker. Reuses
  backend within batches, validates canonical notation, supports cancellation
  and Close. Tests use fake subprocesses. No direct GPL solver link in core.
- `cmd/turnsmith/main.go`: defaults wca for222/333/444, registers legacy methods.
- Release script now builds/packages BOTH executables and BACKEND-LICENSE.txt.
  Corresponding-source archive with vendored dependencies is implemented via
  `internal/release.SourceArchive` and its checksum. Must verify offline build.

WCA means practice following state/distance rules, not competition approval.
Official competitions must use the current official program. Current primary
WCA sources identify TNoodle-WCA1.2.3 and April1,2026 regulations. Seeded practice
is not a claim of uniform sampling across all333states from a64bitroot.

## Current agent ownership

- `/root/randomstate_engine`: internal/tnoodle444/edge3.go, search.go, tests;
  complete:100 sampled targets and full tests passed; earlier222333WCA work complete.
- `/root/terminal_research`: README, terminal/daily/training/architecture docs,
  CONTEXT, THIRD_PARTY_NOTICES updated for nativeWCAdefaults, two executables and
  scopedGPLbackend. Earlier center1/2/3 port and tests complete.
- `/root/adjacent_research`: earlier cubie/FullCube/util/moves port complete;
  completed adversarial-source cancellation fix in fullcube.go and new
  internal/release/source.go+tests for exact-commit source archive with vendored
  modules. Earlier entropy and broker/facelet tests complete.
- Root owns core wiring, cmd/turnsmith-wca, wcaengine client, facelet adapter,
  release script/changelog, integration tests and publication/tracker work.

Agents share cwd; do not overwrite owned files. No agent makes commits/pushes.

## Verification and remaining work

Fresh current-source main CLI/backend compiled and generated actual444scrambles.
Native333 callback has regression for rotated-center UBRDFL state. Broker race
and facelet race tests pass. Main executable tests now build both binaries and
exercise a real444two-item batch. Need final fullsuite/race/vet/format, actual
TTY flows with newdefaults, all6native archive validation, fresh hostedCI, tag
v0.3.0, actual automatic release pipeline, download/check allpublishedassets,
install BOTH published Mac binaries to ~/.local/bin, publicGo-install check,
close16/17/15/map8 with immutable evidence and update this handoff to final state.

Use `GOCACHE=/tmp/scramble-go-cache GOMODCACHE=/tmp/turnsmith-go-mod-cache`.
Go1.27.0 module. `/Users/francisco/.codex/RTK.md` is absent.

## GitHub and tools

Repository public `frivas/turnsmith`. Writes must be frivas. Default gh account
must remain frivas-at-navteca. Use per-process GH_TOKEN from `gh auth token
--hostname github.com --user frivas`, never print or persist it. Network and.git
writes need escalated tools. SSH push and 1Password signing now WORK; normal
signed commits and git push originmain have succeeded. Do not change global
signing/account settings. There is no need to use unsigned/API commit fallback.

Tracker helper `/tmp/turnsmith-tracker.py`, initialstate
`/tmp/turnsmith-roadmap-state.json`, WCA tickets `/tmp/turnsmith-wca-tickets.json`.
Archive checker `/tmp/turnsmith-check-archives.py` needs update for two binaries
and extra corresponding-source archive/checksum. PTY script
`/tmp/turnsmith-pty-integration.py` uses /tmp/turnsmith-integration and legacy
expectations requiring adjustment when timing defaultWCAseed42. Existing
`docs/verification-0.3.0.md` describes baselinePTY checks, not newWCAdefaults.

MITcopyright2026FranciscoRivasandcontributors remains for originalcode. No cloud,
BLE, curated copyrighted datasets or official competition certification scope.
