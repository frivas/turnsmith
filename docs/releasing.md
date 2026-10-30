# Releasing

The public source repository is [frivas/turnsmith](https://github.com/frivas/turnsmith).
The executable is `turnsmith`, and its canonical Go module is
`github.com/frivas/turnsmith`. Published versions and downloads appear on
[Releases](https://github.com/frivas/turnsmith/releases). A Homebrew tap/formula
is not configured. Follow the [release map](https://github.com/frivas/turnsmith/issues/1)
for remaining decisions and validation evidence.

## Tag and publish

Push an annotated SemVer tag to publish a release automatically after validation:

```sh
git switch main
git pull --ff-only
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

These commands are an example for the first release, not a claim that it exists.
Choose the version and source commit before running them. Pushing the tag is the
publication action; a separate draft-review click is not required.

The [Release workflow](../.github/workflows/release.yml) validates the version,
runs native macOS/Linux/Windows CI, builds all six archives with the tag version
and exact commit embedded, verifies SHA-256 checksums, and uploads the files.
It prepares a draft with all assets and its changelog, then publishes it in the
same workflow after those steps succeed. Failed checks do not publish a release.
If upload or publication fails after draft creation, inspect the draft before
retrying; the workflow does not overwrite an existing release.

| OS | Architectures | Package | Executable |
| --- | --- | --- | --- |
| macOS (`darwin`) | AMD64, ARM64 | `.tar.gz` | `turnsmith` |
| Linux | AMD64, ARM64 | `.tar.gz` | `turnsmith` |
| Windows | AMD64, ARM64 | `.zip` | `turnsmith.exe` |

Each release includes all six packages, `SHA256SUMS`, and `CHANGELOG.md`.
The generated changelog is the release body and is also included inside each
package. The repository's [CHANGELOG.md](../CHANGELOG.md) provides a curated
release history visible without downloading an artifact; update it before tagging.
It lists commits since the closest reachable SemVer tag, including direct pushes,
with commit links, a comparison link, and download links. The first release
lists the complete history. Use meaningful commit subjects. Full Git history
and tags are fetched for changelog generation.

Tags follow [SemVer 2.0.0](https://semver.org/), prefixed with `v`, for example
`v0.1.0`, `v1.0.0`, or `v1.1.0-rc.1`. The tooling also accepts SemVer build
metadata. Leading zeros in core version numbers or numeric prerelease identifiers
are rejected. Prerelease tags publish GitHub prereleases. Choose patch versions
for fixes, minor versions for compatible features, and major versions for
incompatible CLI or JSON contract changes. Algorithm versions remain a separate
reproducibility contract. Go module paths for v2 and later must follow Go's
major-version suffix rules before offering that version through `go install`.

## Preview without publication

Run the Release workflow manually from GitHub Actions, or use:

```sh
gh workflow run release.yml --repo frivas/turnsmith -f version=0.1.0
```

Manual runs execute the same validation, CI, packaging, changelog, and checksum
steps and upload a `turnsmith-<version>` Actions artifact. They never create a tag
or GitHub release. Inspect that artifact before selecting a release tag.

Before tagging, review the MIT notices, name/discoverability research, chosen
binary trust policy, and final candidate evidence in the release map. Checksums
are implemented; platform signing and build provenance are separate policy
questions. Native CI covers the three runner OSes, while the six OS/CPU targets
are cross-built. It does not claim native execution of every CPU target.

## Local checks and packaging

```sh
go run ./scripts/check-format.go
go test ./...
go vet ./...
go test -race ./...
go run ./scripts/release.go --version 0.1.0 --changelog
```

`make check`, `make build`, `make bench`, and `make release VERSION=0.1.0` are
convenient alternatives where Make is installed. Local `make release` builds
packages and a changelog; it does not tag or publish. Go commands also work
directly on Windows.

The release script runs from the repository root and accepts `--commit`, `--date`
RFC3339, and `--out`. Defaults are version `0.1.0-dev`, commit `unknown`, and
current UTC time. Example with explicit metadata:

```sh
go run ./scripts/release.go --version 0.1.0 --commit abcdef1234567 --date 2026-10-30T00:00:00Z --changelog
```

Archives contain the executable at the root, README.md, LICENSE,
THIRD_PARTY_NOTICES.md, and CHANGELOG.md when `--changelog` is enabled. The
workflow always enables it. `--check-version` validates a version without
building or writing artifacts. Targets are darwin, linux, and windows crossed with
amd64 and arm64. Unix executables have mode 0755. Windows archives contain
`turnsmith.exe`. SHA256SUMS uses archive basenames; verify from `dist`:

```sh
cd dist
sha256sum --check SHA256SUMS
```

On macOS, use `shasum -a 256 -c SHA256SUMS`. On Windows compare each
`Get-FileHash` result with its entry. Build metadata is injected with Go linker
flags, taking precedence over embedded Go metadata. Without explicit values,
[Go build information](https://pkg.go.dev/runtime/debug#BuildInfo) supplies the
module version and available VCS revision/time. The fallback is `0.1.0-dev`
with `unknown` for missing commit/date. A modified checkout marks a derived
commit with `-dirty`. The `date:` line is a source commit time when derived and
a build timestamp when explicitly injected; it is never the runtime clock.
Archive timestamps are fixed to the supplied build date, and binaries use
`-trimpath`. This is not a promise of identical archives across toolchains.

macOS binaries are unsigned and unnotarized; no signing identity was provided.
Windows binaries are unsigned. Any later signing must occur before final archive
checksums are generated. No competition certification is implied by a release.

## Homebrew after the first release

After repository and release infrastructure exist and the name check is
recorded, the owner can create a separate tap with a formula named `turnsmith`.
Select the matching macOS/Linux archive and SHA-256 for Intel or ARM, install
`bin.install "turnsmith"`, declare the MIT license, and add a test invoking
`--seed 42 --length 1`. Run `brew audit` and `brew test` against the actual
formula. Then document `brew install frivas/TAP/turnsmith` with real identifiers.
No formula or claimed install command is included before those prerequisites.
