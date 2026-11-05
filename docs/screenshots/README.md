# README screenshots

These PNGs show frames captured from the running CLI in an 88-column terminal.
The capture script renders terminal cells with a dark palette and window title
bar. It keeps the app's text, sticker colors, and layout.

| Image | Captured state |
| --- | --- |
| `guide.png` | `R U R' U'`, after two forward moves |
| `timer.png` | Ready to time, with six illustrative solves in a temporary session |
| `training.png` | The README's right-trigger collection, with its hint revealed |

## Refresh the images

Build the current CLI and install the capture dependencies in a virtual
environment. Run these commands from the repository root:

```sh
go build -trimpath -o bin/turnsmith ./cmd/turnsmith
python3 -m venv /tmp/turnsmith-screenshot-tools
/tmp/turnsmith-screenshot-tools/bin/pip install Pillow pyte
/tmp/turnsmith-screenshot-tools/bin/python scripts/readme-screenshots.py
```

The script uses macOS's Andale Mono font by default. On Linux, pass a monospace
font such as `--font /usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf`.
The capture script requires macOS or Linux for pseudo-terminal support. Images
can be viewed on any platform.

Use `--binary /path/to/turnsmith` to capture another build. Sample sessions and
training data live in a temporary directory and are removed after capture.
The timer uses seed 42 for a repeatable scramble. The training screenshot uses
the same collection JSON shown in the README.

Review all three images after regeneration. Check that the cube net, move
instructions, hints, and keyboard controls are visible before committing them.
