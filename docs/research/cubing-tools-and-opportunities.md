# Terminal cubing tools and Turnsmith improvements

Research snapshot preceding implementation. The maintainer subsequently authorized
all ranked opportunities and daily sets through
[Deliver the complete terminal practice roadmap](https://github.com/frivas/turnsmith/issues/8).
The proposals and original staging recommendation below are historical research;
README and command help describe the delivered interfaces.


Researched 2026-10-30 against Turnsmith v0.2.0. Application source baseline:
`14aee572e02cf1d9968c4728d44835bdf286935b`.

## Recommendation

Make the next release better at guiding and checking a physical cube: inspect an
existing sequence, navigate its steps with the keyboard, and show a compact final
state. These extend capabilities we already have. A full timer is useful, but
terminal competitors already cover it well and it introduces a different set of
storage, timing, and input requirements.

This ranking is a product judgment based on the evidence below, not a measured
market-demand result. It assumes the audience suggested by the current product:
people who want an offline terminal tool for scrambles and understandable practice.
If the intended audience becomes primarily competitive speedcubers, trustworthy
random-state generation and timing move higher in priority.

## Scope and evidence

Research was narrowed first to six terminal-native CLI/TUI tools, then to five
adjacent projects whose features could transfer into a terminal. Only primary
sources were used: official repositories, documentation, product pages, and
terminal protocol specifications. Pinned source links are used where available.

Competitor applications were not installed or executed. A documented feature is
not an independently verified runtime result. Repository activity is contextual
metadata, not a quality score. This is a selected comparison, not an exhaustive
census or a ranking by popularity.

The current Turnsmith executable was inspected locally. It already has:

- Offline native binaries for macOS/Linux/Windows on AMD64 and ARM64.
- A versioned 333/random-move generator, unsigned 64-bit seeds, independent
  batches, replay metadata, text and JSON output, and golden vectors.
- A separate, tested facelet model for visualization, holding instructions,
  and colored per-move six-face diagrams with readable color letters.
- Explicit color controls, NO_COLOR handling, and plain output for pipes.

It does not yet accept an externally supplied move sequence, navigate a guide
interactively, display only the final state, time solves, or save sessions.
Only outer-face 3×3 moves are currently simulated. Random-move generation is
explicitly not uniform random-state sampling or competition-approved software.
See the [README](../../README.md) and [architecture](../architecture.md).

One concrete usability observation: the installed v0.2.0 command
`turnsmith --seed 42 --steps --color never | wc -l` produced **373 lines** for
its default 25 moves. This is an output-size observation, not a usability study
or a performance comparison.

## Terminal tools first

### Direct CLI comparisons

| Tool and primary evidence | Documented capabilities | Lesson for Turnsmith |
| --- | --- | --- |
| [TNoodle CLI](https://github.com/SpeedcuberOSS/tnoodle-cli/blob/de86192c9668254c862648a7a7770c76d830e318/README.md) | Lists puzzles, generates batches, writes JSON, and draws a supplied scramble as SVG. It wraps the TNoodle engine and offers platform distributions. Its README explicitly separates engine lineage from authorization for official events. | Separate generating from inspecting a supplied sequence. Broad puzzle support is established competition here; generation quality needs a deliberate engine strategy. SVG export is useful for sharing, but colored text remains the primary terminal view. |
| [cube-scramble-cli](https://github.com/purarue/cube-scramble-cli/blob/32680eab2fe20d372dc655a3ffbc86d9a1a1008c/README.md) | One-shot commands and a REPL with completion, history, Enter for another scramble, batches, multiple puzzle/training types, and a hideable stopwatch. The README lists Python and Java requirements. | Repeated practice can be easier than repeatedly invoking a command. Make interaction explicit so scripts retain predictable stdout. Turnsmith should preserve its native, self-contained installation. |
| [Cube Solver](https://github.com/itsdaveba/cube-solver/blob/0a92ffefa0b36c83ad0f6f544ea70462c73d7e68/README.rst) | Applies supplied maneuvers, prints unfolded states/facelets, generates scrambles, and solves supplied states or sequences. It documents potentially lengthy initial solver-table generation. | Supplied-sequence inspection is valuable without adding a solver. Replay and solving have different startup, memory, packaging, and correctness costs. |

### Adjacent terminal practice tools

| Tool and primary evidence | Documented capabilities | Lesson for Turnsmith |
| --- | --- | --- |
| [Term Timer](https://github.com/Fantomas42/term-timer/blob/d65f7c9c86374dd4f09c03f5ec8dadee982473cb/README.md) | Local JSON sessions, statistics, supplied scramble/file inputs, orientation settings, routines, daily challenges, training, and optional hardware integration. | Import/replay and orientation settings fit our current model. Daily shared practice could reuse deterministic seeds. A full training/hardware suite would be a product expansion, not a small parity feature. |
| [cube-tui](https://github.com/enekocm7/cube-tui/blob/02d43bbeea061b310647ac2ea348f0fd3b784a7d/README.md) | Terminal timing and local sessions. The current README describes an in-repository Rust TNoodle port for scrambling. Its [settings](https://github.com/enekocm7/cube-tui/blob/02d43bbeea061b310647ac2ea348f0fd3b784a7d/docs/settings.md) cover adaptive sizing, hideable panels, configurable keys, and a labeled colored net. | Design for terminal resizing and small panes. Keep the cube readable, hide secondary information before clipping it, and make interactive components optional. |
| [cuterm](https://github.com/daosyn/cuterm/blob/06b85ccccdd36c7012930df9a724ff40a6b58a3b/readme.md) | Go/Bubble Tea timer with a cube net, inspection, DNF/+2 editing, and ao5/ao12/best/worst statistics. Its README acknowledges simulating key release with a delay. | A terminal practice loop is feasible. A browser-style hold/release timer cannot be assumed to transfer faithfully to every terminal. The statistics and penalty rules were not audited in this research. |

### Freshness and code-reuse context

The following are observed latest default-branch commit dates, not release dates
or evidence of abandonment. All six repositories reported non-archived status
when checked through their official GitHub repository/commit APIs.

| Project | Latest observed commit date | Repository license source |
| --- | --- | --- |
| TNoodle CLI | 2025-01-04 | [GPL-3.0](https://github.com/SpeedcuberOSS/tnoodle-cli/blob/de86192c9668254c862648a7a7770c76d830e318/LICENSE.md) |
| cube-scramble-cli | 2026-01-23 | [GPL-3.0](https://github.com/purarue/cube-scramble-cli/blob/32680eab2fe20d372dc655a3ffbc86d9a1a1008c/LICENSE) |
| Cube Solver | 2025-07-13 | [MIT](https://github.com/itsdaveba/cube-solver/blob/0a92ffefa0b36c83ad0f6f544ea70462c73d7e68/LICENSE) |
| Term Timer | 2026-10-18 | [GPL-3.0](https://github.com/Fantomas42/term-timer/blob/d65f7c9c86374dd4f09c03f5ec8dadee982473cb/LICENSE) |
| cube-tui | 2026-10-16 | [GPL-3.0](https://github.com/enekocm7/cube-tui/blob/02d43bbeea061b310647ac2ea348f0fd3b784a7d/LICENSE) |
| cuterm | 2026-06-10 | [MIT](https://github.com/daosyn/cuterm/blob/06b85ccccdd36c7012930df9a724ff40a6b58a3b/license) |

Search snapshots can lag the source. For example, older cube-tui material refers
to Node/Bun-based scrambling, while the current pinned README describes its Rust
port without Java for core generation. Current source takes precedence here.
Engine claims were documented, not independently distribution-tested. A wrapper's
license label also does not audit its engine or transitive dependencies.

## Features from other tools that fit a terminal

| Tool and primary evidence | Feature worth adapting | Terminal version and boundary |
| --- | --- | --- |
| [csTimer](https://cstimer.net/) | Its official introduction documents training subsets, timing/statistics, sessions, scramble images, batch tools, and backups. | A final-state net is a small useful addition. Optional timing, session tables, and CSV/JSON export could follow. Batches and diagrams already exist in Turnsmith and are not new gaps by themselves. |
| [cubing.js / Twizzle](https://github.com/cubing/cubing.js), [algorithm API](https://js.cubing.net/cubing/alg/), [player API](https://js.cubing.net/cubing/twisty/) | Supplied algorithms, inversion/expansion, setup sequences, playback, and training-oriented sticker views. The project also provides a [scramble API and CLI](https://js.cubing.net/cubing/scramble/), so it bridges both categories. | Import and validate an algorithm, show its inverse, and provide keyboard playback. The useful abstraction is sequence navigation; a browser 3D renderer or embedded JavaScript runtime is unnecessary. Some player options are experimental. |
| [CubeDesk](https://www.cubedesk.io/), [official source](https://github.com/kash/cubedesk), [migration guide](https://www.cubedesk.io/guides/migrate-from-cstimer) | Timer statistics, algorithm training, and migration of times, penalties, scrambles, notes, and sessions from csTimer exports. | Local session reports and an explicit import/export format can work without accounts. Lossless migration must not be claimed without testing actual formats and unsupported fields. Social multiplayer is not a priority for this CLI. |
| [J Perm's PLL trainer](https://jperm.net/algs/pll) | Case selection, learning status, hint reveal, previous/next controls, and balanced or slow-case practice. | A named-algorithm drill with reveal/recall and a local learning queue. Begin with user-supplied cases; curated datasets, case setup states, attribution, and selection probabilities need separate work. |
| [TNoodle](https://github.com/thewca/tnoodle), [official scramble workflow](https://www.worldcubeassociation.org/regulations/scrambles/) | Structured event/round scramble sets and local export. | Named, reproducible practice sets with a manifest are useful. Competition secrecy, encrypted packets, and organizer PDF workflows do not serve the current casual-practice scope. Official program status is not inherited by a port or wrapper. |

The relevant sources have different reuse conditions: [csTimer](https://github.com/cs0x7f/cstimer/blob/master/LICENSE)
is GPL-3.0, [CubeDesk](https://github.com/kash/cubedesk/blob/main/README.md)
describes GPL-3.0-or-later, [TNoodle](https://github.com/thewca/tnoodle)
is AGPL-3.0, and [cubing.js](https://github.com/cubing/cubing.js#license)
explains component-specific licensing. These are inspiration sources, not a
recommendation to copy code into the current MIT project. Review exact versions,
files, and dependencies before choosing an integration or port.

## Ranked opportunities

All commands in this section are proposals, not implemented interfaces. Effort
is relative: small reuses an existing capability, medium introduces interaction
or a parser/workflow, and large introduces a new subsystem. These are not delivery
estimates. Value refers to the assumed guided-practice audience.

| Priority | Improvement | User value | Effort and dependencies |
| --- | --- | --- | --- |
| 1 | **Interactive guide:** one stable cube net, next/back, reset, current-move highlight, and progress. When stepping back, show the inverse turn the person should make. | High. Directly addresses the 373-line guide and helps recover from a missed move. | Medium. Reuses the simulator, but needs terminal input, resizing, clean exit, and accessibility tests. Keep streamed `--steps` for pipes and files. |
| 2 | **Import, inspect, and invert a sequence:** quoted moves, stdin/file input, and replay of Turnsmith JSON. Give errors with token positions. | High. Makes the visualizer useful for tutorials, another person's scramble, or a saved practice set. | Medium for input/replay, small for inversion within the current grammar. Start with supported outer-face notation. Rotations, slices, wide moves, groups and commutators require additional parser/model work. |
| 3 | **Final-state-only view:** a compact net after the complete sequence. | High for quick physical-cube checks; avoids requesting every intermediate diagram. | Small. Reuse the same state model and color/label conventions. Useful for generated and imported moves. |
| 4 | **Terminal portability and discoverability:** width-aware layouts, conservative color-depth fallback, preserved labels, examples, and shell completions. | Medium/high. Makes a useful feature work in more panes, remote shells, and terminal configurations. | Small/medium. Existing NO_COLOR, color letters, and pipe behavior must remain intact; they are not missing features. |
| 5 | **Focused drills:** named user algorithms, revealable hints, repetition and a learning/slow-case queue. | High for learners who practice specific cases. | Medium/large. Depends on imported sequences and deliberate setup states. A curated training dataset is a separate deliverable. |
| 6 | **Optional timer, history, and statistics:** sessions, penalties, notes, local export and restore. | High for daily timed practice, lower for users who only want scrambles or guidance. | Large as a dependable workflow. Timing, terminal input, persistent schemas, editing and recovery require their own design. |
| 7 | **Random-state 333 and then selected new events.** | High if serious speedcubing becomes the primary audience. | Large/very large. Requires a correct sampler plus solver/inversion path, resource measurements, independent tests, and engine/license decisions. The existing sticker model does not supply these. |

A low-cost secondary experiment is a daily/shared practice command built on
current deterministic seeds. It needs a defined date/timezone policy and recorded
algorithm/version metadata. Seeded sharing already exists, so this is a convenience
layer whose demand should be tested, not a replacement for the higher priorities.

## Terminal-specific constraints

1. **Keep the default command scriptable.** Interactive modes must be explicit
   and require a suitable terminal. Preserve the existing default text and JSON
   contracts. Never put a mandatory dashboard or hidden persistent writes in the
   one-shot scramble path.
2. **Key release is not universally available.** The [kitty keyboard protocol](https://sw.kovidgoyal.net/kitty/keyboard-protocol/)
   documents the limits of legacy terminal input and opt-in event reporting.
   A timer should have reliable press-based controls; enhanced hold/release
   behavior can be optional when support is detected. Do not present a delayed
   approximation as hardware-equivalent timing.
3. **Plan for small terminals and graceful color fallback.** Existing color
   letters are valuable. Capability detection should not assume that a character
   device guarantees a particular color depth. The [xterm control-sequence reference](https://invisible-island.net/xterm/ctlseqs/ctlseqs.pdf)
   distinguishes indexed and direct color. Preserve the existing behavior where
   explicit color flags can override automatic [NO_COLOR](https://no-color.org/)
   handling.
4. **A displayed state needs a starting-state contract.** Keep solved-start and
   orientation assumptions visible. Moving backwards should tell a human how
   to undo the last turn. Importing an arbitrary physical sticker layout is a
   separate validation problem, not part of simple algorithm replay.
5. **Persistence needs an exit path.** If sessions or case progress are added,
   provide a documented versioned format, atomic saving and export/restore with
   the feature. Test penalties and average calculations against known fixtures.
6. **Keep generation guarantees separate.** More puzzles, legal notation,
   random-state generation, and official competition authorization are different
   claims. Preserve current random-move replay as a distinct versioned method.

## Suggested next release

A coherent next release would be **guided practice and sequence inspection**.

- First add the input/validation path and compact final view using the existing
  outer-face grammar. A proposed `inspect` workflow should support generated
  moves, literal supplied moves, and saved JSON without silently changing them.
- Then let an explicit interactive guide navigate either kind of sequence in one
  screen. Keep orientation visible and explain inverse moves when going back.
- Add terminal-width/color fallbacks and completion alongside those workflows.
- Leave timer/history, broad training datasets, new puzzle engines, BLE devices,
  social accounts, and 3D graphics for separate product decisions.

Acceptance should include a physical practice session, not only simulator tests:
a learner can follow 25 moves without scrolling through the transcript, undo an
accidental step, and resume. Test a small terminal, a resized terminal, no-color
output, pipes, Ctrl-C cleanup, macOS, Linux, and Windows. Supplied sequences should
round-trip without changing move order, and default seeded output must stay stable.

Before committing to a timer or random-state engine, decide whether Turnsmith's
main job is guided learning, timed speedcubing, or a scriptable scramble utility.
That choice changes which of the later opportunities is worth its cost.
