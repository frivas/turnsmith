# scramble

Offline Rubik's Cube scrambles in a single executable. Run `scramble` to print
25 moves for casual 3×3×3 practice, with no headings or extra output.

Only **333 / random-move** is implemented. This method filters simple redundant
moves. It does **not** uniformly sample reachable cube states and is **not**
approved for official competition. The 25-move default is a product choice,
not a competition standard.

## Installation

Requires Go 1.27 or later to build. The installed executable needs no Go toolchain,
network access, configuration files, or other programs.

From this source directory:

```sh
go build -trimpath -o bin/scramble ./cmd/scramble
./bin/scramble
```

On Windows, use `go build -trimpath -o bin/scramble.exe ./cmd/scramble` and run
`.\bin\scramble.exe`.

To install with Go from a local checkout:

```sh
go install ./cmd/scramble
```

Go installs to `GOBIN`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. Add that
directory to your PATH. The provisional module name is `scramble`; a remote
`go install ...@v0.1.0` command will be supplied after the owner chooses a repository
and replaces the module path and internal import prefixes. No remote install
path or published release is claimed yet.

Release packaging produces `.tar.gz` archives for macOS and Linux and `.zip`
archives for Windows, on AMD64 and ARM64. Once a release is published, download
the archive for your OS and architecture from that repository's Releases page,
verify its SHA-256 against `SHA256SUMS`, extract it, and put the executable on PATH.
Use `shasum -a 256 ARCHIVE` on macOS, `sha256sum ARCHIVE` on Linux, or
`Get-FileHash ARCHIVE -Algorithm SHA256` in PowerShell.

A Homebrew tap is not available yet. Name availability must be checked before
publishing packages or creating a formula. See [release instructions](docs/releasing.md).

## Quick start

```sh
scramble
scramble --length 30
scramble --count 10
scramble --seed 42
scramble --puzzle 333 --method random-move
scramble --count 5 --seed 42 --json
scramble puzzles
scramble puzzles --json
scramble version
scramble --version
scramble help
```

Aliases `3x3` and `3x3x3` select `333`. Unsupported identifiers, including `444`,
fail without substituting another puzzle. Successful text output is one sequence
per line, single ASCII spaces, no numbering, wrapping, ANSI codes, or metadata,
and a final newline. Batches are generated completely before output starts.

## Flags

| Flag | Short | Default | Allowed values |
| --- | --- | --- | --- |
| `--puzzle` | `-p` | `333` | `333`, `3x3`, `3x3x3` |
| `--method` | | Puzzle default | `random-move` for 333 |
| `--length` | `-l` | `25` | Decimal integer 1–1000 |
| `--count` | `-n` | `1` | Decimal integer 1–10000 |
| `--seed` | | OS entropy | Unsigned decimal 64-bit integer |
| `--json` | | `false` | Boolean |
| `--help` | `-h` | | Show help |
| `--version` | | | Name and version only |

Use `--flag value`, `--flag=value`, or short equivalents such as `-l 30`.
Boolean flags accept `=true` or `=false`. Repeated flags, short-flag bundles,
positionals, and unknown flags are rejected. `puzzles` accepts only `--json`
and help; `version` accepts help; `help` takes no positional arguments.
`--version` cannot be combined with other flags. A request may contain at most
1,000,000 moves. Errors go to stderr with `error:` and no scramble output for
invalid requests. Exit codes: 0 success, 1 runtime or output failure, 2 invalid
input, 130 interruption. An output failure may leave already-written bytes.

## Notation and algorithm

Outer faces are `U D L R F B`. A bare letter means a clockwise quarter turn
when looking directly at that face. `2` means a half turn; a straight ASCII
apostrophe means a counterclockwise quarter turn. `R`, `R2`, and `R'` each count
as one token. Rotations, slices, lowercase, and wide moves are not generated.

Adjacent moves cannot use the same face. A face cannot repeat across its
opposite face, so `R L R'`, `U D2 U2`, and `F B F` are excluded. Opposite-face
pairs such as `R L` remain legal. The generator selects directly from valid
faces in a fixed order; it has no simulator or visited-state history.
Uniform next-move selection does not imply uniform state sampling.

See [algorithms](docs/algorithms.md) for the exact ordering and selection rules.

## Seeds and JSON

Seeds include zero and `18446744073709551615`. Negative, fractional, hexadecimal,
empty, overflowing, signed, and whitespace-padded values are rejected. Automatic
seeds come from the OS cryptographic random source; entropy failure is fatal.
Plain text deliberately omits seeds, so supply a seed or save JSON for replay.

The same root seed, canonical puzzle, method, algorithm version, length, and
zero-based batch index reproduce the same sequence. Increasing batch size
preserves earlier results. Aliases and output format do not change the stream.
Seed alone is insufficient after an algorithm version change. See the
[reproducibility contract and vectors](docs/reproducibility.md).

`--json` emits one document with schema version 1. Seeds are decimal strings
so JavaScript does not lose precision. Output includes application and algorithm
versions, canonical puzzle, method, parameters, and one-based scramble indices.
Each scramble includes `moves`, `sequence`, and `move_count`. There are no
timestamps or host paths. Inapplicable lengths can be omitted by future methods.

```json
{
  "schema_version": 1,
  "app_version": "0.1.0",
  "puzzle": "333",
  "method": "random-move",
  "algorithm_version": "1",
  "seed": "42",
  "parameters": {"length": 3, "count": 1},
  "scrambles": [
    {"index": 1, "moves": ["L2", "D'", "U"], "sequence": "L2 D' U", "move_count": 3}
  ]
}
```

## Development and contributing

```sh
go test ./...
go vet ./...
go test -race ./...
go run ./scripts/check-format.go
go test ./internal/app -run '^$' -bench . -benchmem
```

Run `gofmt -w cmd internal scripts` before contributing. Changes should include
tests of observable behavior. Golden vector changes require an explanation and
an algorithm-version decision. CI runs integration tests on macOS, Linux, and
Windows, with race and short fuzz checks on Linux. Benchmarks have no hardware
thresholds. No external module downloads are required to build or test.

See [architecture](docs/architecture.md), [adding a puzzle](docs/adding-a-puzzle.md),
and [releasing](docs/releasing.md). Solvers, timers, history, diagrams, telemetry,
plugins, configuration files, and other working puzzles are outside v0.1.0.
Implementation packages are internal; there is no stable public Go API.

## License and attribution

[MIT](LICENSE), copyright scramble contributors.
[Third-party notices](THIRD_PARTY_NOTICES.md) credit Go and PCG. This is an
independent implementation of the provided specification, with no copied
scrambler code.
