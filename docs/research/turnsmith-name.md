# Turnsmith naming and namespace research

Checked 2026-10-30 for
[Investigate Turnsmith naming and namespace conflicts](https://github.com/frivas/turnsmith/issues/2).
The user has selected the project name `turnsmith`. This research supplies
facts for the command/module identity decision; it does not change that choice.

## Findings

The selected GitHub repository exists as public
[`frivas/turnsmith`](https://github.com/frivas/turnsmith). GitHub's repository API
reported `full_name: frivas/turnsmith`, `private: false`, and default branch
`main`. This establishes the chosen repository namespace, without reserving
an executable name across other software ecosystems.
[Repository API](https://api.github.com/repos/frivas/turnsmith).

A current GitHub repository-name search for `turnsmith in:name` returned one
repository, this project. No other named CLI repository was found by that
query. This is a bounded search result, not proof that no CLI with that command
exists: private, unindexed, differently named, or non-GitHub projects are outside
its coverage.
[GitHub repository search API](https://api.github.com/search/repositories?q=turnsmith+in%3Aname&per_page=100).

There is already a software product called **Turnsmith**. Its own website
presents inventory-management software and supplier-order automation. This is
a confirmed naming collision in software, even though its stated purpose differs
from twisty-puzzle practice. The site also establishes that `turnsmith.com` is
already in use by that product.
[Turnsmith product website](https://turnsmith.com/).

Homebrew's official formula registry contained 8,686 entries and its cask
registry contained 7,819 at the time of inspection. There were no matches for
`turnsmith` across formula names, full names, aliases, or old names, or across
cask tokens, full tokens, old tokens, or display names. The local command
`brew search '/^turnsmith$/'` independently reported no formulae or casks found.
These results establish no observed collision in the checked registries, not a
reservation or a promise of future availability.
[Formula registry](https://formulae.brew.sh/api/formula.json),
[cask registry](https://formulae.brew.sh/api/cask.json).

An authenticated listing of the first 200 `frivas` repositories showed no
repository whose name starts with `homebrew-`. There is no selected tap in this
map, and the research did not create one. Any future tap must be inspected at
its actual owner/repository path before a formula is published there.
[Owner repository API](https://api.github.com/users/frivas/repos?per_page=100).

## What the namespaces allow

Go locates published modules through their module path and repository tags.
For this repository, `github.com/frivas/turnsmith` is the natural canonical
module path. Adopting it still requires changing `go.mod` and internal import
prefixes together. A `cmd/turnsmith` entrypoint would provide the corresponding
command package path. Neither the module rename nor the entrypoint rename has
been implemented by this research.
[Go module source conventions](https://go.dev/doc/modules/managing-source).

Homebrew supports formula names within an owner's tap. Fully qualified names
select a particular tap's formula, including where another tap or core has a
formula with the same short name. For a future tap named `frivas/tap`, the
conventional GitHub repository would be `frivas/homebrew-tap` and its formula
could be addressed as `frivas/tap/turnsmith`. Those are examples, not existing
installation instructions.
[Homebrew tap naming and duplicate-name rules](https://docs.brew.sh/Taps).

No search result establishes Homebrew core acceptance. Core has separate
requirements, including a stable immutable release and suitable source and
license conditions. Core acceptance remains outside this first-release map.
[Homebrew formula acceptance requirements](https://docs.brew.sh/Acceptable-Formulae).

## Implications for the identity decision

Use the owner-qualified repository and a description such as "offline
twisty-puzzle scramble CLI" in release material so users can distinguish this
project from the existing inventory product. The known collision is relevant
to discoverability and branding. This research does not assess trademark
registration, infringement, or legal clearance and makes no exclusivity claim.

The next ticket should settle command naming, canonical Go module, legacy
aliases, and attribution using these facts. It can retain the user-selected
project name while recording the software-name collision explicitly. No
compatibility requirement or mandatory rename follows from the checks alone.

## Reproduction

The checked queries were:

```sh
gh api 'search/repositories?q=turnsmith+in:name&per_page=100'
gh api repos/frivas/turnsmith
HOMEBREW_NO_AUTO_UPDATE=1 brew search '/^turnsmith$/'
gh repo list frivas --limit 200 --json name,isPrivate
```

The registry check fetched both official JSON arrays and searched only the
name/alias/token fields listed above, case-insensitively. It did not search
all third-party taps, package registries, DNS names, or operating-system PATHs.
Recheck the selected package namespace when a distribution channel is chosen.
