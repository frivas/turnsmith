# Releasing

No public repository, package namespace, or Homebrew formula has been created
by this implementation. The executable working name is `scramble`. Publication
requires an owner, credentials, and a name check; local archives do not reserve
a name or establish availability.

## Before publishing

1. Check the chosen repository namespace and existing CLI projects for naming
   conflicts. Check `brew search scramble` and Homebrew's formula registry
   before creating a formula, as well as names in the chosen tap. Record the
   results and date. Do not claim Homebrew core availability without acceptance.
2. Choose the public repository URL and module path, e.g.
   `github.com/OWNER/scramble`. Replace `module scramble` in go.mod and every
   `scramble/internal/` import prefix with the real path. Run all checks again.
   The remote installation command will then be
   `go install github.com/OWNER/scramble/cmd/scramble@v0.1.0`, using the real
   owner. Do not publish the example placeholder.
3. Initialize Git if needed, review the MIT ownership line and third-party
   notices, and push reviewed source and workflows to the chosen repository.
   Select branch protection and enable GitHub Actions. No credentials are
   stored in source. The implementation environment blocked `.git` creation;
   repository initialization must be performed in a writable Git environment.
4. Run the checks below, inspect native integration results for all three OSes,
   and verify archive contents, executable names, permissions, and checksums.
   Native tests on one machine do not establish other-OS execution results.
5. Tag the reviewed commit `v0.1.0` and push that tag. The release workflow runs
   the full CI matrix, packages all six targets with CGO disabled, verifies
   checksums, uploads workflow artifacts, and creates a **draft** GitHub Release
   using the built-in repository-scoped token. Review and publish that draft.

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
`scramble.exe`. SHA256SUMS uses archive basenames; verify from `dist`:

```sh
cd dist
sha256sum --check SHA256SUMS
```

On macOS, use `shasum -a 256 -c SHA256SUMS`. On Windows compare each
`Get-FileHash` result with its entry. Build metadata is injected with Go linker
flags. Plain `go build` reports `0.1.0-dev`, commit `unknown`, date `unknown`.
Archive timestamps are fixed to the supplied build date, and binaries use
`-trimpath`. This is not a promise of identical archives across toolchains.

macOS binaries are unsigned and unnotarized; no signing identity was provided.
Windows binaries are unsigned. Any later signing must occur before final archive
checksums are generated. No competition certification is implied by a release.

## Homebrew after the first release

After repository and release infrastructure exist and the name check is
recorded, the owner can create a separate tap with a formula named `scramble`.
Select the matching macOS/Linux archive and SHA-256 for Intel or ARM, install
`bin.install "scramble"`, declare the MIT license, and add a test invoking
`--seed 42 --length 1`. Run `brew audit` and `brew test` against the actual
formula. Then document `brew install OWNER/TAP/scramble` with real identifiers.
No formula or claimed install command is included before those prerequisites.
