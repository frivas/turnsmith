# Codex handoff: turnsmith

Updated 2026-10-30. Workspace: `/Users/francisco/Code/personal/turnsmith`.
The folder is named turnsmith; the implemented project and executable are named
scramble. Keep that distinction when locating files.

## Current continuation status

The user confirmed **turnsmith** as the project name and authorized a public
repository at https://github.com/frivas/turnsmith. It was created by the `frivas`
GitHub account, and the active CLI account was immediately restored to
`frivas-at-navteca`. Local Git initialization now works, with `main` and an
`origin` remote. The implementation still uses `scramble`; its rename has not
been performed or silently decided.

The canonical release-planning artifact is
[Map the route to Turnsmith's first public release](https://github.com/frivas/turnsmith/issues/1).
It has six native child issues, native blocking relationships, and two AFK
research tickets. See [docs/wayfinding.md](docs/wayfinding.md) for tracker
operations and the frontier query. The destination is the first public release,
not a broader roadmap. Continue through that map. Human decisions remain open;
no release tag or release publication is authorized by charting the map.

The implementation-session notes below are historical. In particular, their
statements that Git was unavailable and no repository or map existed have been
superseded by this continuation. The local v0.1.0 checks described below have
not been rerun merely for creating the tracker; no Go source changed.

## Implementation-session record

## Current status

The requested v0.1.0 local implementation is complete and passed the checks
listed below. Source, tests, documentation, workflows, a native development
binary, and six release archives are present. Nothing has been published.
The directory has no `.git`: `git init` failed with `Operation not permitted`
under the previous session's workspace permissions. There are no commits,
branches, remotes, tags, or hosted CI results from this work.

The user initially invoked Wayfinder, then explicitly chose **implement v0.1.0
locally** over planning. No Wayfinder map or issue tracker was created. Continue
from the implementation; there is no pending planning interview. The latest
request was to save this handoff for another Codex session.

## Implemented behavior

- Default command emits exactly 25 filtered random outer-face moves for `333`.
- Puzzle aliases `3x3` and `3x3x3` normalize to `333`.
- Root generation flags, batches, full unsigned decimal 64-bit seeds, text and
  schema-version-1 JSON, discovery, help, and version commands work.
- Length/count/total-move limits, strict input validation, exit statuses,
  entropy failure, generation failure, output failure, and cancellation are
  handled. Batches finish generation before serialization.
- Both redundancy rules are implemented with direct valid-face enumeration.
  There is no simulator or visited-state tracking.
- Puzzle/method registry and typed capabilities support new implementations
  without changing CLI dispatch or serializers. Test-only non-cube and
  non-length methods exercise that boundary.
- All generation state and randomness are local. Each zero-based batch index
  gets a separately derived SHA-256/PCG stream, preserving batch prefixes.
- MIT license, Go/PCG notices, contributor documentation, CI, and draft-release
  packaging are included. There are no external Go module dependencies.

Only `333/random-move` is implemented. It does not uniformly sample reachable
states and has no competition approval. Solvers, other working puzzles,
timers, history, diagrams, plugins, configuration files, and telemetry remain
outside the agreed release scope.

## Where to read next

| Need | Source |
| --- | --- |
| Usage, installation, scope | [README.md](README.md) |
| Responsibilities and consequential choices | [docs/architecture.md](docs/architecture.md) |
| Exact face/turn ordering and restrictions | [docs/algorithms.md](docs/algorithms.md) |
| Byte encoding, PCG arithmetic, version policy | [docs/reproducibility.md](docs/reproducibility.md) |
| Golden sequences | [internal/cli/testdata/vectors.json](internal/cli/testdata/vectors.json) |
| Extension contract | [docs/adding-a-puzzle.md](docs/adding-a-puzzle.md) |
| Ownership, release and Homebrew steps | [docs/releasing.md](docs/releasing.md) |
| Domain glossary | [CONTEXT.md](CONTEXT.md) |

The executable entry is `cmd/scramble/main.go`. Implementation packages are
`internal/{cli,app,puzzle,cube,cube333,random,output}`. Development tools are
`scripts/check-format.go` and `scripts/release.go`, both run explicitly with
`go run` and excluded from normal package builds by build tags.

## Verification already completed

These passed on macOS ARM64 with Go 1.27.3 during the implementation session:

```sh
go test ./...
go vet ./...
go test -race ./...
go run ./scripts/check-format.go
```

Additional successful checks:

- Three-second fuzz runs, two workers each, for `FuzzParseSeed`,
  `FuzzResolution`, and `FuzzMoveValidation`; over 500,000 executions each.
- Native executable integration tests, local `go build`, and local
  `go install ./cmd/scramble` with an isolated writable GOBIN.
- Golden sequences and raw PCG outputs compared against a separate Python
  calculation of the documented integer arithmetic. That temporary reference
  script is not a maintained repository tool; committed fixtures and the
  arithmetic specification are the durable artifacts.
- Concurrent independent calls, mid-generation cancellation, entropy/writer
  failures, no partial batch on generation failure, alias/format equivalence,
  batch-prefix stability, and stream independence when one scramble consumes
  more random draws.
- All six CGO-free release targets cross-compiled. Archive SHA-256 values,
  filenames, contents, Unix executable permissions, and binary OS/CPU headers
  were inspected. The packaged macOS ARM64 executable was run and its release
  version metadata and JSON verified.
- Both benchmarks completed. Observed application-level baselines were about
  993 ns / 1280 B / 13 allocations for one default scramble and 1.079 ms /
  1,100,351 B / 7006 allocations for 1000 default scrambles on an Apple M5 Max.
  These are observations, not CI thresholds.

Linux and Windows binaries were cross-built but **not executed locally**.
`.github/workflows/ci.yml` configures native integration checks on all three
OSes, plus Linux race and fuzz checks. Those hosted jobs have not run yet.
The release workflow has also not run on GitHub.

## Local commands and artifacts

The module is currently `scramble`; go.mod declares Go 1.27.0. The short module
name is provisional because the user has not selected a public repository URL.

```sh
go build -trimpath -o bin/scramble ./cmd/scramble
./bin/scramble --seed 42
./bin/scramble --seed 42 --count 3 --json
go install ./cmd/scramble
```

Plain builds report `0.1.0-dev`, commit `unknown`, build date `unknown`.
`bin/scramble` already exists as a native development binary.

`dist/` contains `SHA256SUMS` and `scramble_0.1.0_<os>_<arch>` archives for
darwin/linux/windows and amd64/arm64. Unix archives use `.tar.gz`; Windows
archives use `.zip`. Each includes the executable, README, LICENSE, and
THIRD_PARTY_NOTICES.md. They were generated with:

```sh
go run ./scripts/release.go --version 0.1.0 --date 2026-10-30T00:00:00Z
```

Their embedded commit is `unknown`. Rebuild with the actual commit and desired
build date for publication. `bin/` and `dist/` are gitignored generated outputs.
macOS binaries are unsigned/unnotarized; Windows binaries are unsigned.

## Remaining work before publication

1. Confirm the next requested task with the user if none is supplied. Local
   implementation is done; publication is not an implicit next action.
2. In a Git-writable environment, initialize the repository and choose the
   actual owner, remote, and canonical Go module path. Update go.mod and every
   `scramble/internal/` import prefix together, then rerun checks.
3. Check and record working-name availability in the selected namespace,
   existing CLI projects, and Homebrew before publishing packages or creating
   a formula. No name availability conclusion has been established yet.
4. Review the MIT ownership line and attribution for the chosen repository,
   push the reviewed source, enable Actions, and obtain successful native CI
   results on macOS, Linux, and Windows.
5. Rebuild with real version/commit/date metadata and verify final archives.
   Push a reviewed `v0.1.0` tag when publication is authorized. The workflow
   creates a draft GitHub Release; a maintainer reviews and publishes it.
6. After real releases exist, document the actual remote `go install` command.
   A Homebrew tap/formula can follow the name check and release infrastructure.
   There is no Homebrew formula or claimed Homebrew core availability today.

## Environment notes for the next agent

- The previous sandbox blocked `.git` creation and Rex socket access. Do not
  mistake that for missing source files or a failing Go implementation.
- Tests used `GOCACHE=/tmp/scramble-go-cache` because the normal user cache
  directory was outside writable roots. Apply that environment variable to
  Go commands if the same restriction persists. The install smoke check used
  `GOBIN=/tmp/scramble-install`.
- `/Users/francisco/.codex/RTK.md`, referenced by the user's initial instructions,
  was absent when read. Follow current instructions if it becomes available.
- No source changes or new test runs were made for this handoff. The file list
  and local artifacts were rechecked before writing it. Revalidate after
  implementation, module-path, toolchain, or packaging changes; preserve or
  version intentional changes to golden output.
