# Architecture

The first release uses the Go standard library only. Generation is synchronous,
offline, and local to each call. Puzzle and method are separate: `333` describes
a puzzle, and `random-move` describes an available method for it.

| Package | Responsibility |
| --- | --- |
| `cmd/turnsmith` | Build metadata, signal context, OS entropy, explicit registration, process exit |
| `internal/cli` | Strict flag parsing, explicit-length tracking, help, discovery, exit codes |
| `internal/app` | Resolve defaults and aliases, validate limits, seed, derive independent streams, assemble batches |
| `internal/puzzle` | Generator boundary, typed capabilities, immutable registry and sorted discovery |
| `internal/cube` | Face, turn, layer count, size-aware validation and notation |
| `internal/cube333` | 333 descriptor and filtered random-move selection |
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
members check cancellation. No runtime shell commands or file reads are used.

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
- Linker variables hold version, commit, and build date. They are the only
  package variables and are never changed by generation or CLI execution.
- A small Go developer tool builds all six archives. It is excluded from normal
  package builds and the product executable; no release tooling is a runtime
  dependency.
