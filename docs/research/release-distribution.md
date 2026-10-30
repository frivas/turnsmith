# First-release distribution and trust constraints

Researched 2026-10-30 for [Investigate first-release distribution and trust constraints](https://github.com/frivas/turnsmith/issues/3).
This is evidence for a maintainer decision, not approval to buy signing services,
change release policy, or publish v0.1.0.

## Existing implementation

At baseline `fcec07c`, [the release workflow](https://github.com/frivas/turnsmith/blob/fcec07c/.github/workflows/release.yml)
waits for the reusable CI workflow, cross-builds six archives, checks their
SHA-256 values, uploads them as workflow artifacts, and creates a draft GitHub
Release for a pushed `v0.*` tag. It supplies the tag version and commit to the
build. It has no signing, notarization, build-attestation, or SBOM steps.
Release immutability is a repository setting and cannot be inferred from this
workflow.

[The package script](https://github.com/frivas/turnsmith/blob/fcec07c/scripts/release.go)
builds darwin/linux/windows for amd64/arm64 with CGO disabled. Unix archives are
`.tar.gz`; Windows archives are `.zip`. Each contains the executable, README,
license, and notices. The executable, entrypoint, and archive prefixes still use
`scramble`; the selected project name is now `turnsmith`.

[CI](https://github.com/frivas/turnsmith/blob/fcec07c/.github/workflows/ci.yml)
requests a standard runner for each of three OSes. That matrix does not prove
native execution on all six OS/CPU combinations. A successful run on this
baseline would also precede the still-pending identity migration. Do not treat
this research or prior cross-builds as final candidate validation.

## GitHub archives and costs

GitHub releases attach downloadable assets to a Git tag. GitHub currently allows
up to 1,000 assets per release, each smaller than 2 GiB. The proposed six small
CLI archives and checksum file fit this model. GitHub does not impose an OS
code-signing requirement in its release-asset model. Choosing signed binaries
is a separate platform and maintainer policy question.
[GitHub release documentation](https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases)

Standard GitHub-hosted runners for public repositories are free. Larger runners
can incur charges even for public repositories; artifact storage has its own
allowances and billing. The current workflow uses standard runner labels and
uploads release archives as Actions artifacts, so retention and storage remain
worth checking rather than describing the whole pipeline as cost-free.
[GitHub Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions)

## Go installation

A remote command installation uses `go install package@version`. The package
path includes the canonical module path and command subdirectory. If the
maintainer chooses a complete rename, the intended command after a real tag
exists is `go install github.com/frivas/turnsmith/cmd/turnsmith@v0.1.0`.
This is conditional, not a working installation claim today. The module and
entrypoint must first be changed together. The current module requires Go
1.27.0; toolchain availability and installation must be validated on the
candidate. Go can select a newer toolchain when the user's configuration allows
it. [Go installation reference](https://go.dev/ref/mod#go-install),
[Go toolchains](https://go.dev/doc/toolchain)

There is a local metadata gap to decide. [The command entrypoint](https://github.com/frivas/turnsmith/blob/fcec07c/cmd/scramble/main.go)
initializes its own reported version to `0.1.0-dev` and commit/date to `unknown`.
The archive script overrides these through linker flags; an ordinary remote
`go install` invocation does not invoke that script. Inference from this code:
unless changed, the Go channel will report those defaults even when installed
at the release tag. Decide whether to derive version information or document
the difference before claiming equivalent metadata across channels.

## macOS signing and notarization

Apple's default protection for downloaded software checks developer identity
and notarization. Unsigned or unnotarized downloads can be blocked or require
user intervention. Actual behavior for this standalone CLI also depends on how
it was downloaded, quarantine attributes, and device policy. A successful local
build/run is not evidence that a browser-downloaded archive works on a clean
Mac. [Apple's Gatekeeper guidance](https://support.apple.com/en-us/102445)

The supported Developer ID route uses a Developer ID certificate and Apple's
notary service. The Apple Developer Program currently costs USD 99 per year,
with local prices and possible qualifying fee waivers. No membership or signing
identity was verified for this project.
[Developer ID certificates](https://developer.apple.com/help/account/certificates/create-developer-id-certificates),
[Apple program enrollment](https://developer.apple.com/programs/enroll/)

Apple's custom workflow accepts ZIP, disk-image, and signed flat installer
containers for submission. It can issue tickets for standalone binaries, but
cannot staple tickets to those binaries or directly to ZIPs. Gatekeeper can
retrieve a ticket online; stapling supported containers enables offline ticket
availability. Inference for Turnsmith: signing and notarizing a CLI does not
by itself prove a fully offline first-run experience for the current bare-binary
tarball. If that becomes a release requirement, decide whether a signed installer
or other supported container is needed and test the actual download path.
[Apple's custom notarization workflow](https://developer.apple.com/documentation/Security/customizing-the-notarization-workflow)

The current single Linux packaging job has no Apple signing/notarization tools
or credentials configured. Adding this route requires workflow changes and
credential handling. Any byte-changing signing or packaging steps must happen
before computing final archive checksums and attestations.

## Windows signing

Microsoft recommends signing non-Store software, but a valid signature does not
guarantee an immediate SmartScreen reputation bypass. New signed binaries can
still warn. Unsigned files build reputation anew per version, and enterprise
policy or Smart App Control can prevent execution. Therefore "unsigned Windows
archives are available" and "all Windows users can run them without prompts"
are different promises.

Microsoft currently recommends Artifact Signing for non-Store distribution and
lists a starting cost of USD 9.99/month, with identity validation required.
Eligibility, actual regional pricing, and access must be checked before choosing
it. No service was provisioned and no Windows signing identity was verified.
[Microsoft's current SmartScreen developer guidance](https://learn.microsoft.com/windows/apps/package-and-deploy/smartscreen-reputation)

## Checksums, build provenance, and immutable releases

The existing `SHA256SUMS` checks the bytes against a supplied checksum. Inference:
if both an archive and an unsigned checksum file were replaced together, that
check alone would not authenticate the publisher or build process. The workflow
currently provides checksums only; do not advertise provenance as implemented.

GitHub build artifact attestations are available for public repositories on
current plans. They require an attestation step and the relevant
`id-token: write` and `attestations: write` workflow permissions, in addition to
content access. Consumers can verify artifacts with `gh attestation verify`.
This establishes build provenance and does not substitute for macOS or Windows
platform signing. [GitHub build-attestation instructions](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)

GitHub immutable releases lock release assets and tags after publication and
create a release attestation binding the release tag, commit, and assets. This
is distinct from an attestation about how the build ran. GitHub recommends
preparing a draft with all assets before publishing it. Verification commands
include `gh release verify` and `gh release verify-asset`. Choose and enable this
policy explicitly; its presence was not verified for this repository.
[Immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases),
[release verification](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity)

## Homebrew later

A maintainer can create an external tap without acceptance into Homebrew core.
A GitHub repository named `homebrew-<tap>` supports the conventional short tap
name. A tap needs a maintained formula and real download/source URLs, checksums,
and tests. No tap or formula was created during this research.
[Creating and maintaining a tap](https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap)

Homebrew core has separate acceptance rules, including stable, versioned and
verifiable sources, source availability, and its supported CI matrix. A tap
that installs upstream platform binaries does not imply core eligibility.
The first release can therefore ship GitHub archives and Go installation
without claiming `brew install turnsmith` exists.
[Core formula requirements](https://docs.brew.sh/Acceptable-Formulae)

## Decisions left to the maintainer

- Whether the first release includes only GitHub archives and Go installation,
  with a tap deferred.
- Whether unsigned platform binaries are acceptable initially, with accurate
  installation guidance, or paid signing/notarization is a prerequisite.
- Whether GitHub build attestations and immutable releases are required, and
  who owns their configuration and verification.
- What version metadata a Go-installed command must report.
- Whether offline first launch on macOS is required, beyond offline operation
  after installation.

These choices belong in the existing identity and release-policy tickets.
No source implementation, policy setting, tag, release, signing enrollment,
or payment was changed by this research.
