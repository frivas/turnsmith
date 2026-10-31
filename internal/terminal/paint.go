package terminal

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
)

// paint interprets SGR styles only. Cursor commands, OSC hyperlinks/title changes,
// and other embedded controls never reach the terminal. Content is clipped to
// the supplied frame; the frame callback is responsible for responsive layout.
func paint(screen tcell.Screen, text string, width, height int) {
	x, y := 0, 0
	lastX, lastY := -1, -1
	style := tcell.StyleDefault
	for len(text) > 0 && y < height {
		switch text[0] {
		case '\x1b':
			var sequence string
			text, sequence = consumeEscape(text)
			style = sgr(style, sequence)
			continue
		case '\n':
			x, y, lastX = 0, y+1, -1
			text = text[1:]
			continue
		case '\r':
			text = text[1:]
			continue
		case '\t':
			x = (x/4 + 1) * 4
			text = text[1:]
			continue
		}
		end := strings.IndexAny(text, "\x1b\n\r\t")
		if end < 0 {
			end = len(text)
		}
		segment := text[:end]
		text = text[end:]
		graphemes := uniseg.NewGraphemes(segment)
		for graphemes.Next() {
			runes := graphemes.Runes()
			if len(runes) == 0 || unicode.IsControl(runes[0]) {
				continue
			}
			cellWidth := graphemes.Width()
			if cellWidth == 0 {
				if lastX >= 0 && lastY == y {
					primary, combining, previousStyle, _ := screen.GetContent(lastX, y)
					screen.SetContent(lastX, y, primary, append(combining, runes...), previousStyle)
				}
				continue
			}
			if x+cellWidth <= width {
				screen.SetContent(x, y, runes[0], runes[1:], style)
				lastX, lastY = x, y
			}
			x += cellWidth
		}
	}
}

func consumeEscape(text string) (rest, sgrSequence string) {
	if len(text) < 2 {
		return "", ""
	}
	switch text[1] {
	case '[':
		for i := 2; i < len(text); i++ {
			if text[i] >= 0x40 && text[i] <= 0x7e {
				if text[i] == 'm' {
					sequence := text[2:i]
					if sequence == "" {
						sequence = "0"
					}
					return text[i+1:], sequence
				}
				return text[i+1:], ""
			}
		}
		return "", ""
	case ']':
		for i := 2; i < len(text); i++ {
			if text[i] == '\a' {
				return text[i+1:], ""
			}
			if text[i] == '\x1b' && i+1 < len(text) && text[i+1] == '\\' {
				return text[i+2:], ""
			}
		}
		return "", ""
	default:
		return text[2:], ""
	}
}

func sgr(style tcell.Style, sequence string) tcell.Style {
	if sequence == "" {
		return style
	}
	parts := strings.Split(sequence, ";")
	values := make([]int, len(parts))
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return style
		}
		values[i] = n
	}
	for i := 0; i < len(values); i++ {
		code := values[i]
		switch {
		case code == 0:
			style = tcell.StyleDefault
		case code == 1:
			style = style.Bold(true)
		case code == 22:
			style = style.Bold(false)
		case code == 39:
			style = style.Foreground(tcell.ColorDefault)
		case code == 49:
			style = style.Background(tcell.ColorDefault)
		case code >= 30 && code <= 37:
			style = style.Foreground(tcell.PaletteColor(code - 30))
		case code >= 40 && code <= 47:
			style = style.Background(tcell.PaletteColor(code - 40))
		case code >= 90 && code <= 97:
			style = style.Foreground(tcell.PaletteColor(code - 90 + 8))
		case code >= 100 && code <= 107:
			style = style.Background(tcell.PaletteColor(code - 100 + 8))
		case (code == 38 || code == 48) && i+2 < len(values) && values[i+1] == 5:
			if values[i+2] >= 0 && values[i+2] < 256 {
				color := tcell.PaletteColor(values[i+2])
				if code == 38 {
					style = style.Foreground(color)
				} else {
					style = style.Background(color)
				}
			}
			i += 2
		case (code == 38 || code == 48) && i+4 < len(values) && values[i+1] == 2:
			r, g, b := values[i+2], values[i+3], values[i+4]
			if r >= 0 && r < 256 && g >= 0 && g < 256 && b >= 0 && b < 256 {
				color := tcell.NewRGBColor(int32(r), int32(g), int32(b))
				if code == 38 {
					style = style.Foreground(color)
				} else {
					style = style.Background(color)
				}
			}
			i += 4
		}
	}
	return style
}
