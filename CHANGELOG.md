# Changelog

Release downloads and the full commit-by-commit notes are available on
[GitHub Releases](https://github.com/frivas/turnsmith/releases).

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
