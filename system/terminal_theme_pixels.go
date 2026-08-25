package system

import (
	"fmt"
	"image"
)

type terminalPixelRGB struct {
	R uint8
	G uint8
	B uint8
}

type terminalThemePixels struct {
	Background terminalPixelRGB
	Foreground terminalPixelRGB
	Cursor     terminalPixelRGB
}

func readTerminalThemePixels(pixels image.Image) (terminalThemePixels, error) {
	type counted struct {
		colour terminalPixelRGB
		count  int
	}
	counts := map[terminalPixelRGB]int{}
	bounds := pixels.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r16, g16, b16, a16 := pixels.At(x, y).RGBA()
			if a16 == 0 {
				continue
			}
			counts[terminalPixelRGB{R: uint8(r16 >> 8), G: uint8(g16 >> 8), B: uint8(b16 >> 8)}]++
		}
	}
	if len(counts) == 0 {
		return terminalThemePixels{}, fmt.Errorf("terminal screen has no opaque pixels")
	}
	most := func(accept func(terminalPixelRGB) bool) (counted, bool) {
		best := counted{}
		found := false
		for colour, count := range counts {
			if accept(colour) && (!found || count > best.count) {
				best, found = counted{colour: colour, count: count}, true
			}
		}
		return best, found
	}
	background, _ := most(func(terminalPixelRGB) bool { return true })
	brightness := func(c terminalPixelRGB) int { return int(c.R) + int(c.G) + int(c.B) }
	foreground, foregroundOK := most(func(c terminalPixelRGB) bool {
		return c != background.colour && brightness(c) >= 600 &&
			absInt(int(c.R)-int(c.G)) <= 24 && absInt(int(c.G)-int(c.B)) <= 24
	})
	cursor, cursorOK := most(func(c terminalPixelRGB) bool {
		return c.B >= 180 && int(c.B)-int(c.R) >= 40 && int(c.B)-int(c.G) >= 30
	})
	if !foregroundOK || !cursorOK {
		return terminalThemePixels{}, fmt.Errorf("terminal screen lacks rendered foreground or cursor pixels")
	}
	return terminalThemePixels{
		Background: background.colour, Foreground: foreground.colour, Cursor: cursor.colour,
	}, nil
}

func compareTerminalThemePixels(actual, expected terminalThemePixels) error {
	if actual != expected {
		return fmt.Errorf("rendered terminal theme differs: got=%+v want=%+v", actual, expected)
	}
	return nil
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
