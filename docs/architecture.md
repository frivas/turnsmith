# Architecture

Generation is synchronous,
offline, and local to each call. Puzzle and method are separate: `333` describes
a puzzle, and `random-move` describes an available method for it.

| Package | Responsibility |
| --- | --- |
| `cmd/turnsmith` | Build metadata, signal context, OS entropy, explicit registration, process exit |
| `internal/cli` | Strict flag parsing, explicit-length tracking, help, discovery, exit codes |
| `internal/app` | Resolve defaults and aliases, validate limits, seed, derive independent streams, assemble batches |
| `internal/puzzle` | Generator boundary, typed capabilities, immutable registry and sorted discovery |
| `internal/cube` | Face, turn, layer count, size-aware validation and notation |
| `internal/cube333` | 333 generation, optional facelet state simulation, and terminal step diagrams |
| `internal/random` | Decimal seeds, entropy reads, SHA-256 stream encoding, PCG initialization, bounded integers |
| `internal/output` | Completed-result text, JSON, and discovery-table rendering |

The CLI sends a typed request. A nil length means the user did not supply one;
it is resolved against method capabilities in the application. An unsupported
explicit length fails even when it happens to equal a familiar default.
The generator receives a typed optional length, a context, and caller-owned
randomness. Its output is canonical notation tokens. The application checks
output token boundaries and exact length when declared, copies tokens into a
completed result, and serializes only after every scramble succeeds.

A generator failure discards the batch. Output writers can still fail midway
through serialization; this returns status 1 with useful context where stderr
is writable. Context cancellation propagates as status 130. The executable
handles Ctrl-C through `signal.NotifyContext`. All loops over moves and batch
members check cancellation. Generation uses no runtime shell commands or external table files. Explicit
sequence imports and session/training modes read user-selected or local data.

Registration is explicit in `main`, with no init functions or mutable generation
globals. Registry maps are private, descriptors are copied in and out including
optional lengths and slices, and listing is sorted. Canonical IDs and aliases
share one collision namespace. Methods are keyed inside each puzzle. Future
puzzles require a registration and implementation, with no CLI or renderer
switch statements. Generator implementations must be immutable and safe for
independent calls; registry construction cannot enforce that contract for
arbitrary implementations.

Cube generators use typed moves internally and render at their boundary.
Non-cube generators can emit their own notation, such as the test fixture's
`tip++` token. The application and serializers never interpret cube faces or
layer widths. There is no arbitrary options map, plugin loader, database,
configuration layer, or public library API.

Each scramble derives a separate stream from root seed, canonical puzzle,
method, algorithm version, and zero-based index. Draws in one scramble cannot
advance another. No full cube simulator, visited-state filter, or shared history
is needed to produce legal outer-face sequences.

## Consequential choices

- Go 1.27 is the minimum, matching the available supported stable toolchain.
  Standard-library CLI parsing is implemented explicitly to preserve strict
  decimal validation, duplicate detection, and command-specific flags without
  external dependencies.
- PCG raw words plus a project-owned bounded-integer operation define algorithm
  version 1. Convenience APIs do not decide the seeded sequence.
- The canonical module is `github.com/frivas/turnsmith`; the executable entrypoint
  is `cmd/turnsmith`. The historical RNG domain `scramble/rng/v1` stays fixed so
  the project rename does not change seeded output.
- Linker variables hold explicit version, commit, and build date overrides.
  Embedded Go module/VCS metadata fills available defaults once at startup.
  The resolved values are local and shared by version output and JSON metadata;
  generation and CLI execution never mutate the linker variables.
- A small Go developer tool builds all six archives and their Git changelog.
  Its SemVer/history helpers live in `internal/release`, tested in normal CI
  but never imported by the product executable. The tool's entrypoint is excluded
  from normal package builds. Release tooling is not a runtime dependency.

## Optional step diagrams

`--steps` selects a renderer registered by canonical puzzle ID in `cli.CLI.Guides`.
The renderer interface accepts a completed sequence, context, writer, and color
choice. The entrypoint wires the 333 guide independently of generator registration;
future puzzle guides can be registered without changing CLI dispatch or serializers.
Aliases resolve through the same puzzle registry. Unsupported guides and JSON/step
combinations fail explicitly. The diagram mode limits the whole batch to 1000 moves
before writing output.

The 333 facelet model owns a local solved state and applies outer-face turns to
positions and normals in integer 3D coordinates. It is never called during scramble
generation and does not track or reject repeated states. Each diagram is a snapshot
after a sequence prefix, with all six faces viewed directly from outside. Rendering
checks cancellation per step and propagates writer failures. ASCII color letters
remain present with and without ANSI backgrounds. Automatic color detection lives
at the executable boundary and respects NO_COLOR and TERM=dumb.

`internal/cube333/testdata/states.json` contains independently generated states
from [cubejs 1.3.2](https://github.com/ldez/cubejs), checked in URFDLB facelet order.
The fixtures cover all 18 single moves, all prefixes of the seed-42 scramble, and
noncommuting sequences. This reference was used only to produce test data; it is
not a runtime, Go-module, or CI dependency.

## Terminal practice extension

`alg` parses bounded sequences. `cubestate` independently simulates supported
cube sizes, and `view` renders their nets. `terminal` adapts tcell events and
restores terminal state. Their state never feeds back into legacy random-move
generation. The original cube333 model and reference fixtures remain an
independent check for the generalized model and random-state engine.

`randomstate` uses pinned native gocube transitions/tables with caller-local
sampling and bounded search. `cube444` is a separate random-move registration.
`daily` derives versioned UTC practice seeds through the ordinary app service.
`practice` owns solves/sessions; `training` owns case progress. Both stores use
separate versioned files, cross-process locks, atomic replacement, validated
backups and explicit recovery. No persistent write occurs in default generation.
The CLI owns interactive state machines and command-specific presentation.

Third-party Go dependencies are compiled into the executable. There is no
runtime JavaScript, Java, external solver process or network requirement.
