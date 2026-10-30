# Releasing

The public source repository is [frivas/turnsmith](https://github.com/frivas/turnsmith).
The executable is `turnsmith`, and its canonical Go module is
`github.com/frivas/turnsmith`. No release tag, published release, Homebrew tap,
or formula exists yet. Follow the [release map](https://github.com/frivas/turnsmith/issues/1)
for remaining decisions and validation evidence.

## Before publishing

1. Review the [naming research](https://github.com/frivas/turnsmith/issues/2),
   including the existing inventory software product called Turnsmith. Recheck
   the chosen package namespace before adding a distribution channel.
2. Validate the canonical module and `cmd/turnsmith` entrypoint. After a real
   release tag exists, the remote installation command will be
   `go install github.com/frivas/turnsmith/cmd/turnsmith@v0.1.0`. Versioned Go
   installs derive their version from embedded module metadata. Missing commit
   and date information stays `unknown`; it is not inferred from the tag.
3. Review the MIT ownership line, notices, and selected binary trust policy.
   Obtain successful native macOS, Linux, and Windows CI results for the actual
   candidate. The initial Windows formatting failure was reproduced as checkout
   CRLF conversion; `.gitattributes` now keeps Go source LF on every OS. Track
   the resulting native CI evidence in the validation ticket.
4. Run the checks below and inspect final archive contents, executable names,
   permissions, metadata, and checksums. Cross-compilation alone does not prove
   native execution on other OSes.
5. When the maintainer authorizes publication, tag the reviewed commit `v0.1.0`
   and push that tag. The release workflow runs CI, packages six CGO-free
   targets, verifies checksums, uploads artifacts, and creates a draft GitHub
   Release. Review and publish the draft as a separate maintainer step.

## Local checks and packaging

```sh
go run ./scripts/check-format.go
go test ./...
go vet ./...
go test -race ./...
go run ./scripts/release.go --version 0.1.0
```

`make check`, `make build`, `make bench`, and `make release` are convenient
alternatives where Make is installed. Go commands also work directly on Windows.

The release script runs from the repository root and accepts `--commit`, `--date`
RFC3339, and `--out`. Defaults are version `0.1.0-dev`, commit `unknown`, and
current UTC time. Example with explicit metadata:

```sh
go run ./scripts/release.go --version 0.1.0 --commit abcdef1234567 --date 2026-10-30T00:00:00Z
```

Archives contain the executable at the root, README.md, LICENSE, and
THIRD_PARTY_NOTICES.md. Targets are darwin, linux, and windows crossed with
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
