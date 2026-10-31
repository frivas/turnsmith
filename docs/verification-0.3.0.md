# v0.3.0 terminal integration checks

Verified on macOS on 2026-10-31 using a freshly built executable and Python's
standard-library PTY support. These are automated executable checks, not a human
physical-cube usability test. Windows and Linux interactive behavior still need
platform-specific confirmation.

Commands used:

```sh
GOCACHE=/tmp/scramble-go-cache GOMODCACHE=/tmp/turnsmith-go-mod-cache go build -o /tmp/turnsmith-integration ./cmd/turnsmith
python3 /tmp/turnsmith-pty-integration.py
```

The temporary script created isolated data directories and a user-owned two-move
case collection. It opened real PTYs at 80 columns × 40 rows and 48 columns × 24
rows, sent keys to the executable, inspected stored JSON, and compared terminal
attributes before and after each run. It also checked the terminal's alternate
screen exit sequence and successful process exit.

Passed flows at both sizes:

- Timer: start and stop two attempts, apply +2 to the first and DNF to the second,
  select the next scramble, and exit. Exactly two positive-duration solves were
  persisted with the requested penalties, generation metadata and distinct
  per-attempt seeds. The normal exit printed the saved session count.
- Guide: advance, go back with inverse guidance, reset, and exit. The rendered
  instructions included `Apply: R`, `Undo:`, and `Reset: solve cube`.
- Training: use the supplied `R U` case, show inverse setup `U' R'`, reveal the
  hint, time an attempt, explicitly confirm success, and exit. Stored progress
  recorded hint use and success. At 48 × 24, training correctly refused to start
  until the PTY was resized to 48 × 40; the 3×3 training frame needs 26 rows.

All six sessions exited with status 0, restored the original terminal attributes
(excluding macOS's transient PENDIN retype flag), and exited the alternate screen.
Training at 48 × 24 was verified as a guarded small-screen state, not a complete
training layout at that height.

The script and raw terminal traces are temporary verification artifacts, not
runtime dependencies or bundled training content. No default user data directory
was used during these checks.


## Native WCA extension checks

After the native Go WCA backend was added, the complete unit/integration suite
passed again. This includes exhaustive shallow-state rejection for 222/333,
100 independent full 444 sampled targets, native executable generation through
the process broker, cryptographic-vs-seeded metadata, and the unchanged explicit
legacy random-move seed42 sequence. Center normalization at the 444-to-333
reduction boundary has a specific rotated-center regression fixture.

The WCA backend is entirely Go. Tests compare emitted444 scrambles to their
sampled targets with the independent geometric model modulo24 whole-cube
rotations. Coordinate, symmetry, pruning, cancellation, bounded failure,
concurrent generation, exact entropy-word forwarding and process cleanup checks
are separate from the statistical uniformity argument. Source attribution,
licensing and generation rules are recorded in the WCA research documents.
