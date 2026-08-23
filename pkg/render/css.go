package render

import (
	"image/color"
	"strings"
)

// ParseCSSColor extracts and parses a hex color from a style string.
func ParseCSSColor(style string) (color.Color, bool) {
	if style == "" {
		return nil, false
	}

	_, after, found := strings.Cut(style, "color:")
	if !found {
		return nil, false
	}

	value, _, _ := strings.Cut(after, ";")
	_, hexPart, foundHex := strings.Cut(value, "#")
	if !foundHex {
		return nil, false
	}

	hexStr := strings.TrimSpace(hexPart)
	switch len(hexStr) {
	case 6:
		r, ok1 := parseHexPair(hexStr[0], hexStr[1])
		g, ok2 := parseHexPair(hexStr[2], hexStr[3])
		b, ok3 := parseHexPair(hexStr[4], hexStr[5])
		if !ok1 || !ok2 || !ok3 {
			return nil, false
		}
		return color.RGBA{R: r, G: g, B: b, A: 0xff}, true

	case 3: // Shorthand like #fff -> #ffffff
		r, ok1 := parseHexPair(hexStr[0], hexStr[0])
		g, ok2 := parseHexPair(hexStr[1], hexStr[1])
		b, ok3 := parseHexPair(hexStr[2], hexStr[2])
		if !ok1 || !ok2 || !ok3 {
			return nil, false
		}
		return color.RGBA{R: r, G: g, B: b, A: 0xff}, true
	}

	return nil, false
}

// parseHexPair converts two hex characters into a single byte via fast bit shifting.
func parseHexPair(c1, c2 byte) (byte, bool) {
	h1, ok1 := hexCharToByte(c1)
	h2, ok2 := hexCharToByte(c2)
	if !ok1 || !ok2 {
		return 0, false
	}
	return (h1 << 4) | h2, true
}

// hexCharToByte maps an individual ASCII byte to its numeric hex value.
func hexCharToByte(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
