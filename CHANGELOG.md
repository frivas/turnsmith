# Changelog

## Unreleased

- Updated `golang.org/x/text` to v0.41.0 to resolve GO-2026-6629 and GO-2026-5970.
- Resolved all 18 Staticcheck diagnostics and added pinned static analysis and
  Go vulnerability scans to CI, including checks of unused required modules.
- Preserve the native 4×4 companion's final error output before reporting a failed
  protocol read. Waiting for stderr collection also fixes an intermittent macOS
  CI failure; a regression test holds back stderr delivery to exercise the race.

## 0.3.0 - 2026-10-31

- Default practice now uses WCA-style state sampling: 2x2 excludes states closer than four moves, 3x3/4x4 exclude states closer than two. Unseeded generation uses fresh cryptographic words; explicit seeds are labeled reproducible seeded practice.
- Added a native Go 4x4 three-phase backend adapted from TNoodle, independently checked against sampled targets. It ships as a separate GPL-licensed executable with corresponding source and needs no Java. Official WCA competitions still require the current official scramble program.

- Interactive Next/Back cube guidance with inverse undo moves and compact final diagrams.
- Algorithm import, inspection and inversion, including saved JSON, wide/slice moves, rotations, groups, commutators and conjugates.
- Portable terminal input, resize handling, color depth options and Bash/Zsh/Fish/PowerShell completions.
- Local solve sessions, press-based timing, inspection, penalties, notes, averages and JSON/CSV import/export with backups and recovery.
- User-owned algorithm drills with inverse setups, hints, learning status and balanced/learning/slow-case selection.
- Native 333 random-state, 222 random-state and 444 wide-turn random-move generators. Existing 333 random-move seeds remain unchanged.
- Named daily/shared practice sets with UTC dates, versioned seeds and replayable manifests.


Release downloads and the full commit-by-commit notes are available on
[GitHub Releases](https://github.com/frivas/turnsmith/releases).

## [0.2.0] - 2026-10-30

### Added

- Optional `--steps` mode with cube-holding instructions, a solved reference,
  and colored six-face diagrams after every move.
- `--color auto|always|never`, readable color-letter labels, terminal/piping
  detection, and NO_COLOR support for step diagrams.
- Independently checked 3×3×3 facelet simulation, used only for visualization.
- README orientation guide and a labeled reference cube diagram.

## [0.1.0] - 2026-10-30

### Added

- Offline `turnsmith` command for casual 3×3×3 practice, generating 25 moves by
  default with filters for redundant adjacent and opposite-face moves.
- Configurable length and batch size, unsigned 64-bit seeds, reproducible
  independent batch streams, and text or schema-version-1 JSON output.
- Puzzle/method discovery, aliases, strict input validation, cancellation,
  and explicit runtime/output error handling.
- Extensible puzzle and method registry with a documented reproducibility
  contract and golden test vectors.
- Native CI on macOS, Linux, and Windows, with Linux race and fuzz checks.
- SemVer tag releases that automatically publish a Git changelog, SHA-256
  checksums, and compiled macOS/Linux/Windows packages for AMD64 and ARM64.
- Non-publishing release previews with downloadable artifacts.
- Version metadata derived from Go module/VCS information, with explicit
  release metadata taking precedence in packaged binaries.

### Fixed

- Windows checkout line endings no longer cause the Go formatting check to fail.
- Explicit development versions and unknown metadata survive release overrides.

### Release scope

Only `333/random-move` is implemented. It does not uniformly sample reachable
states and has no official competition approval. Platform binaries are unsigned;
checksums are included. Homebrew, platform signing, notarization, and build
provenance are follow-up work.

[0.1.0]: https://github.com/frivas/turnsmith/releases/tag/v0.1.0

[0.2.0]: https://github.com/frivas/turnsmith/releases/tag/v0.2.0
