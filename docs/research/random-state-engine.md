# Native random-state engines

Researched and implemented on 2026-10-31 for [ticket #9](https://github.com/frivas/turnsmith/issues/9).

## Decision

Use `github.com/unixpickle/gocube v0.1.1` for native cubie transitions and coordinate tables. Its [source](https://github.com/unixpickle/gocube/tree/v0.1.1) is Go and its [BSD-2-Clause license](https://github.com/unixpickle/gocube/blob/v0.1.1/LICENSE) permits redistribution with the required notices. The selected package has no transitive dependencies, external executables, CGO, network calls, or downloaded table files. Go's module checksum pins the dependency. Keep its copyright and license in distributed notices.

Turnsmith implements its own caller-local sampler and context-aware solver in `internal/randomstate`. The upstream [random-state sampler](https://github.com/unixpickle/gocube/blob/v0.1.1/random_state.go) uses package-global randomness. The upstream [solver wrapper](https://github.com/unixpickle/gocube/blob/v0.1.1/solver.go) uses goroutines and its search does not expose Turnsmith's context and resource contract. We therefore use the exported state/table types directly. This is an implementation decision based on reading the pinned source, rather than an upstream guarantee about our adapter.

## State distributions and construction

For 333, rejection-based bounded draws drive Fisher-Yates permutations. Sample all eight corner pieces and twelve edge pieces independently, then swap the final two edges if permutation parity differs. Every admissible permutation pair has exactly two preimages before that correction. Sample seven independent corner orientation coordinates and derive the final constrained coordinate. Sample eleven independent edge flips and derive their XOR. This gives uniform reachable cubie states when supplied uniform independent random words. The internal PCG stream provides deterministic pseudorandom words under Turnsmith's existing versioned derivation contract.

The orientation completion accounts for the axis-based convention in the upstream [corner representation](https://github.com/unixpickle/gocube/blob/v0.1.1/cubie_corners.go). Corner and edge parity and orientation constraints are enforced before solving. We solve the sampled target and invert the solution, then verify that the solver actually solved the target. Failed searches return an error. They never replace a difficult sample with an easier one. Uniform *sampling* does not promise a uniform distribution conditional on a bounded search succeeding.

The 333 solver uses two-phase coordinate heuristics, fixed move order, maximum phase-one depth 12 and total depth 30, with a hard limit of 50 million visited search nodes. It checks context cancellation during sampling, table waiting and every 1,024 search nodes. It returns errors on cancellation or exhausted bounds. It does not claim shortest scrambles or competition approval. The underlying phase coordinates and pruning values follow the pinned upstream [phase one](https://github.com/unixpickle/gocube/blob/v0.1.1/phase1_cube.go) and [phase two](https://github.com/unixpickle/gocube/blob/v0.1.1/phase2_cube.go) representations.

For 222, fix the BLD corner in its solved position and orientation to choose one representative per whole-cube rotation. Uniformly permute the other seven pieces and sample six free orientations. This gives `7! × 3^6 = 3,674,160` states. Jaap Scherphuis's original [pocket-cube analysis](https://www.jaapsch.net/puzzles/cube2.htm) explains the rotational quotient and publishes the eleven-move face-turn diameter. Turnsmith independently builds a complete distance table using U/F/R turns and tests that every coordinate is reachable and that its diameter is eleven. Following decreasing distances gives an optimal path in this fixed-corner move set. Invert that path to obtain the scramble. A sampled solved state emits the identity `R R'`, preserving the sampled state without rejecting it.

333 random-move generation remains separate and unchanged. Random-state does not accept a fixed `--length`. 222 is a genuine additional puzzle, rather than a name mapped to 333 random moves.

## Runtime and cancellation

Immutable tables initialize lazily once per puzzle. Callers waiting for initialization can cancel immediately. A cancelled waiter does not interrupt or poison the shared bounded initialization, which continues in a background goroutine. Subsequent generation reuses the same tables. Generation never shares mutable random streams or search state.

Measurements below used Go on darwin/arm64, Apple M5 Max. They describe this machine, not a latency guarantee. The benchmark tests are reproducible with `go test ./internal/randomstate -run '^$' -bench . -benchmem`.

| Operation | Measured time | Allocated bytes | Retained table bytes |
| --- | ---: | ---: | ---: |
| 333 table construction, one run | 166 ms | 48,738,800 | 14,562,368 |
| 222 table construction, one run | 88.5 ms | 18,494,808 | 3,778,032 |
| Warm 333, seeds 0–9 | 6.54 ms/op | 1,120/op | shared tables |
| Warm 222, seeds 0–9 | 2.66 µs/op | 395/op | shared tables |

Retained table byte counts sum Go array/struct sizes and the persistent distance slice. They exclude runtime/allocator overhead, temporary queues, stacks and executable size. Construction uses dense bounded integer queues instead of the upstream heuristic builder's much larger allocation churn. Both table sets together retain about 18.3 MB, excluding overhead. Search latency varies by state. The process remains native Go for the supported Darwin, Linux and Windows amd64/arm64 builds.

## Validation

Tests cover all 5,040 permutation and 729 orientation coordinate round trips, all 3,674,160 combined pocket states, and diameter eleven. A 1,000-seed 333 sweep checks that every returned solution solves its sampled state. For 100 pocket targets and twenty full-cube targets, tests separately apply emitted scramble tokens to Turnsmith's geometric facelet simulator and compare the result with the sampled target's stickers. The pocket comparison checks its 24 corner stickers. This independent representation catches inversion, notation, face-order and orientation mistakes that a solver's own inverse check could miss.

Versioned golden sequences, concurrent repeatability, pre-cancelled generation, search cancellation, node-budget enforcement, unsupported length rejection and the solved-state identity are covered. `go test -race ./internal/randomstate -short` passed. The extended sweep passed separately. No guarantee of competition certification follows from these checks.

## Integration

Append `randomstate.Cube333{}` to the existing 333 registration's generators. Register `randomstate.Registration222()` as another puzzle. Both descriptors use method `random-state`, algorithm version `1`, and no explicit length. The selected registration defaults 222 to random-state while 333 retains its existing default. Method/version identities keep the existing 333/random-move random streams unchanged.
