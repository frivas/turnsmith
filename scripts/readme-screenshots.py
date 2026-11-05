#!/usr/bin/env python3
"""Capture real terminal frames. Requires Pillow, pyte, and a built turnsmith.

Usage: python scripts/readme-screenshots.py --binary ./bin/turnsmith
Pass --font /path/to/monospace.ttf on systems without macOS fonts.
All sample practice data is created in a temporary directory.
"""

import argparse
import fcntl
import json
import os
from pathlib import Path
import pty
import select
import struct
import subprocess
import tempfile
import termios
import time

import pyte
from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parent.parent
BACKGROUND = "#101820"
FOREGROUND = "#dce5ed"
PALETTE = {
    "black": "#101820", "red": "#ed6a68", "green": "#8ecb9a",
    "brown": "#e6bd70", "blue": "#7aaee8", "magenta": "#c39ce0",
    "cyan": "#7fcbd0", "white": "#dce5ed",
    "brightblack": "#8494a5", "brightred": "#ff8c89",
    "brightgreen": "#b2e5b5", "brightbrown": "#f4d597",
    "brightblue": "#a0c9ff", "brightmagenta": "#dcc0f4",
    "brightcyan": "#a7e2e5", "brightwhite": "#ffffff",
}


def capture(binary, args, keys=b"", rows=32, columns=88):
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", rows, columns, 0, 0))
    env = {**os.environ, "TERM": "xterm-256color", "COLORTERM": "truecolor",
           "TCELL_ALTSCREEN": "enable"}
    env.pop("NO_COLOR", None)
    screen = pyte.Screen(columns, rows)
    stream = pyte.ByteStream(screen)
    process = subprocess.Popen([binary, *args], stdin=slave, stdout=slave,
                               stderr=slave, env=env)

    def collect(seconds):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            if select.select([master], [], [], max(0, deadline - time.monotonic()))[0]:
                stream.feed(os.read(master, 65536))

    try:
        collect(2)
        if process.poll() is not None:
            raise RuntimeError("Terminal command failed: " + "\n".join(screen.display))
        for key in keys:
            os.write(master, bytes([key]))
            collect(0.15)
        return screen
    finally:
        if process.poll() is None:
            os.write(master, b"q")
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
        os.close(master)
        os.close(slave)


def color(value, default):
    if value == "default":
        return default
    return PALETTE.get(value, "#" + value)


def render(screen, title, target, font_path):
    scale = 2
    cell_w, cell_h, padding, bar = 10 * scale, 22 * scale, 28 * scale, 48 * scale
    font = ImageFont.truetype(font_path, 16 * scale)
    label = ImageFont.truetype(font_path, 12 * scale)
    used_rows = max(i for i, line in enumerate(screen.display) if line.strip()) + 1
    width = screen.columns * cell_w + padding * 2
    height = bar + padding * 2 + used_rows * cell_h
    image = Image.new("RGB", (width, height), BACKGROUND)
    draw = ImageDraw.Draw(image)
    draw.rectangle((0, 0, width, bar), fill="#1b2732")
    for index, fill in enumerate(("#ff6058", "#febc2e", "#28c840")):
        x = padding + index * 19 * scale
        draw.ellipse((x, 19 * scale, x + 10 * scale, 29 * scale), fill=fill)
    draw.text((padding + 80 * scale, 15 * scale), title, font=label, fill="#a9bacb")
    for y in range(used_rows):
        for x in range(screen.columns):
            char = screen.buffer[y][x]
            fg, bg = color(char.fg, FOREGROUND), color(char.bg, BACKGROUND)
            if char.reverse:
                fg, bg = bg, fg
            left, top = padding + x * cell_w, bar + padding + y * cell_h
            draw.rectangle((left, top, left + cell_w - 1, top + cell_h - 1), fill=bg)
            draw.text((left, top), char.data, font=font, fill=fg)
    image.save(target, optimize=True)
    print(f"{target.relative_to(ROOT)}: {width} x {height}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", default=str(ROOT / "bin/turnsmith"))
    parser.add_argument("--font", default="/System/Library/Fonts/Supplemental/Andale Mono.ttf")
    options = parser.parse_args()
    binary = str(Path(options.binary).resolve())
    output = ROOT / "docs/screenshots"
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="turnsmith-screenshots-") as work:
        data = str(Path(work) / "practice")
        collection = Path(work) / "my-cases.json"
        collection.write_text(json.dumps({
            "schema_version": 1, "name": "My practice cases",
            "cases": [{"id": "right-trigger", "name": "Right trigger", "puzzle": "333",
                       "algorithm": "R U R' U'"}],
        }) + "\n")
        # Seed a real disposable session with illustrative solve times.
        subprocess.run([binary, "sessions", "new", "practice", "--data-dir", data],
                       check=True, capture_output=True)
        for seconds in ("18.420", "16.830", "19.110", "17.560", "15.940", "17.210"):
            subprocess.run([binary, "sessions", "add", "practice", "--time", seconds,
                            "--scramble", "R U R' U'", "--data-dir", data],
                           check=True, capture_output=True)
        frames = [
            ("guide", "turnsmith guide --alg \"R U R' U'\"",
             ["guide", "--alg", "R U R' U'", "--color-depth", "truecolor"], b"nn"),
            ("timer", "turnsmith timer --session practice --inspection 15",
             ["timer", "--session", "practice", "--inspection", "15", "--seed", "42",
              "--data-dir", data], b""),
            ("training", "turnsmith train --file my-cases.json --mode sequential",
             ["train", "--file", str(collection), "--mode", "sequential", "--seed", "42",
              "--data-dir", data, "--color-depth", "truecolor"], b"h"),
        ]
        for name, title, args, keys in frames:
            render(capture(binary, args, keys), title, output / f"{name}.png", options.font)


if __name__ == "__main__":
    main()
