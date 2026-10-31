# Terminal use

Ordinary generation, JSON, inspections, and streamed diagrams work in pipes and
redirected files. Interactive modes are explicit: `guide`, `timer`, and `train`.
They require both standard input and standard output to be real terminals and
refuse `TERM=dumb`. These modes use one screen and restore the previous screen,
cursor, input mode, and Unix descriptor flags when they finish, including after
Ctrl-C, cancellation, or an application error. Terminal handling uses
[tcell's native Windows and Unix backends](https://github.com/gdamore/tcell/tree/v2.13.10).

Generation defaults to `wca` for 222, 333, and 444. Without `--seed`, the
sampler draws directly from OS cryptographic entropy. `--seed` selects
reproducible seeded practice with the same minimum-distance filter. Neither mode
makes Turnsmith an official WCA competition program. Daily sets are always
seeded practice. Supplied algorithms replay the moves you give them.

For 444, install the native Go `turnsmith-wca` companion beside `turnsmith`, or
put it on PATH. It ships in the release archive and needs no Java or runtime
downloads. First use builds its tables in memory. Missing companions and bounded
initialization/search failures return an error without substituting a method.
See [installation](../README.md#installation).

## Follow a scramble

```sh
turnsmith guide --seed 42
turnsmith guide --alg "R U R' U'"
turnsmith guide --file daily.json --index 1
```

Hold the cube with white on top and green facing you before starting from the
solved state. The labeled faces and tile letters remain visible without colors:
W white, Y yellow, O orange, R red, G green, and B blue. A clockwise move is viewed
while looking directly at the named face. An apostrophe reverses it; `2` is a
half turn. Keep this orientation unless the displayed algorithm explicitly
rotates the cube.

| Key | Action |
| --- | --- |
| Enter, Space, right/down arrow, `n` | Apply the next move |
| Left/up arrow, `p` | Go back; follow the displayed inverse move |
| Home, `r` | Reset the displayed guide to its initial state |
| End | Display the final state |
| `q`, Escape, Ctrl-D | Exit |
| Ctrl-C | Cancel |

Home and End change the displayed state immediately. Match your physical cube
to that state before continuing. The guide starts at state zero and stops at
the last move; extra Next/Back presses do not wrap around.

The diagrams adapt between a cube net and grouped face panels as width changes.
Minimum guide sizes are 22 columns × 15 rows for 2×2, 22 × 18 for 3×3, and
26 × 21 for 4×4. Below the minimum, a resize message replaces the frame and
navigation pauses. Resizing back keeps the same step.

For a transcript, a pipe, or a screen reader, use explicit stream output:

```sh
turnsmith inspect --alg "R U R' U'" --steps --color never
turnsmith inspect --file daily.json --index 1 --diagram
turnsmith --seed 42 --diagram --color never
```

## Colors

`--color auto|always|never` controls diagrams. Automatic color uses terminal
capability and respects `NO_COLOR`. `--color never` always produces plain tile
letters. `--color always` is an explicit color override; it can put ANSI codes
in redirected stream output.

`--color-depth auto|16|256|truecolor` selects the palette. Automatic mode chooses
from the terminal environment; explicit depth is useful for remote shells or
terminal multiplexers whose advertised capabilities differ from their display.
The interactive backend maps colors to its supported terminal palette. Every
tile also carries a letter, so hue is never the only way to identify a face.
The 16-color palette shares the nearest available color for orange and yellow;
use their O/Y letters or choose 256 colors for a clearer distinction.

```sh
NO_COLOR=1 turnsmith guide --seed 42
turnsmith guide --seed 42 --color-depth 256
turnsmith inspect --alg "R U" --diagram --color always --color-depth 16
```

## Local practice and daily sets

The timer uses presses: Space or Enter starts and stops. It does not claim
keyboard release detection or Stackmat hardware support. `n` selects the next
scramble; `2`, `d`, and `0` set +2, DNF, or no penalty for the last solve.
An optional `--inspection 15` supplies inspection guidance. Run `timer --help`
and `train --help` for local storage and training controls.

Daily practice is independent of interactive mode:

```sh
turnsmith daily --date 2026-10-31 --name "Weekend practice" --output daily.json
turnsmith daily --date 2026-10-31 --count 10 --json
turnsmith guide --file daily.json --index 3
```

The default date is today's UTC calendar date, the default count is five, and
the default method is `wca`. Daily sets use a predictable derived seed and report
`randomness: seeded-practice`; they are not secret competition scrambles.
Seed policy version 1 derives the root seed from the date, canonical puzzle ID,
and method. The name, count, and length do not change that seed. The manifest
records the application and algorithm versions needed to identify its generator.
The stored moves remain replayable even if a future generator changes. `--seed`
is unavailable in daily mode; use ordinary generation to choose a seed.

`--output PATH` writes a JSON manifest regardless of the stdout format. It
replaces an existing destination only after generation, validation, and a
completed temporary-file write. `--json` selects JSON for stdout too. Keep shared
sets unchanged when asking others to practise the same moves.
Daily manifests are limited to 1 MiB so that `inspect` and `guide` can load them;
reduce the count if a set exceeds that limit. Move length is configurable only
with explicit `--method random-move`; `wca` chooses its own output length.

## Shell completions

Generate definitions from the installed binary. They include its commands,
command-specific flags, supported puzzle IDs and aliases, generation methods,
and fixed values such as color depth. Turnsmith prints these definitions; loading
them is an explicit shell action.

For the current Bash session:

```bash
source <(turnsmith completion bash)
```

For Zsh after completion initialization:

```zsh
autoload -Uz compinit
compinit
source <(turnsmith completion zsh)
```

For Fish, install its generated completion file:

```fish
mkdir -p ~/.config/fish/completions
turnsmith completion fish > ~/.config/fish/completions/turnsmith.fish
source ~/.config/fish/completions/turnsmith.fish
```

For PowerShell 7, save and dot-source a script:

```powershell
turnsmith completion powershell | Out-File -Encoding utf8 ./turnsmith-completion.ps1
. ./turnsmith-completion.ps1
```

To persist Bash, Zsh, or PowerShell definitions, save the generated script in a
location you manage and source it from your shell profile. Regenerate it after
upgrading Turnsmith. Fish uses its
[completion file conventions](https://fishshell.com/docs/current/completions.html),
and the PowerShell script registers a
[native argument completer](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/register-argumentcompleter).
