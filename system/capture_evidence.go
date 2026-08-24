package system

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
)

func validateCaptureEvidence(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	image, err := png.Decode(file)
	if err != nil {
		return fmt.Errorf("capture is not a PNG: %w", err)
	}
	bounds := image.Bounds()
	if bounds.Dx() < 2 || bounds.Dy() < 2 {
		return fmt.Errorf("capture extent is %dx%d", bounds.Dx(), bounds.Dy())
	}
	first := image.At(bounds.Min.X, bounds.Min.Y)
	r0, g0, b0, a0 := first.RGBA()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := image.At(x, y).RGBA()
			if r != r0 || g != g0 || b != b0 || a != a0 {
				return nil
			}
		}
	}
	return fmt.Errorf("capture contains one uniform color")
}

type terminalPaletteCounts struct {
	Red   int
	Green int
	Blue  int
}

type terminalPaletteExpectation struct {
	Red   [3]uint8
	Green [3]uint8
	Blue  [3]uint8
}

// validateTerminalPaletteEvidence reads only the public terminal-screen rectangle. Window chrome
// is deliberately excluded: a drawn tab bar around a blank renderer is still a blank terminal.
// The fixture uses ANSI red, green and blue background cells. Shared focus lighting may dim those
// pixels, so the verdict uses hue dominance and a minimum painted area rather than one exact RGB
// byte value. All providers are judged by this one rule.
func validateTerminalPaletteEvidence(path string, expected terminalPaletteExpectation) (terminalPaletteCounts, error) {
	file, err := os.Open(path)
	if err != nil {
		return terminalPaletteCounts{}, err
	}
	defer file.Close()
	pixels, err := png.Decode(file)
	if err != nil {
		return terminalPaletteCounts{}, fmt.Errorf("terminal capture is not a PNG: %w", err)
	}
	counts := countTerminalPalettePixels(pixels, expected)
	const minimumSwatchPixels = 100
	if counts.Red < minimumSwatchPixels || counts.Green < minimumSwatchPixels || counts.Blue < minimumSwatchPixels {
		return counts, fmt.Errorf(
			"terminal palette is not painted: red=%d green=%d blue=%d, each must cover at least %d pixels",
			counts.Red, counts.Green, counts.Blue, minimumSwatchPixels,
		)
	}
	return counts, nil
}

func terminalPaletteExpectationFromBase(base []string) (terminalPaletteExpectation, error) {
	if len(base) != 16 {
		return terminalPaletteExpectation{}, fmt.Errorf("terminal palette has %d base colours, want 16", len(base))
	}
	red, err := parseTerminalColour(base[1])
	if err != nil {
		return terminalPaletteExpectation{}, err
	}
	green, err := parseTerminalColour(base[2])
	if err != nil {
		return terminalPaletteExpectation{}, err
	}
	blue, err := parseTerminalColour(base[4])
	if err != nil {
		return terminalPaletteExpectation{}, err
	}
	return terminalPaletteExpectation{Red: red, Green: green, Blue: blue}, nil
}

func parseTerminalColour(value string) ([3]uint8, error) {
	var result [3]uint8
	if len(value) != 7 || value[0] != '#' {
		return result, fmt.Errorf("invalid terminal colour: %s", value)
	}
	for index := range result {
		parsed, err := strconv.ParseUint(value[1+index*2:3+index*2], 16, 8)
		if err != nil {
			return result, fmt.Errorf("invalid terminal colour: %s", value)
		}
		result[index] = uint8(parsed)
	}
	return result, nil
}

func countTerminalPalettePixels(pixels image.Image, expected terminalPaletteExpectation) terminalPaletteCounts {
	return terminalPaletteCounts{
		Red:   largestPaletteRegion(pixels, expected.Red),
		Green: largestPaletteRegion(pixels, expected.Green),
		Blue:  largestPaletteRegion(pixels, expected.Blue),
	}
}

func largestPaletteRegion(pixels image.Image, expected [3]uint8) int {
	bounds := pixels.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	matched := make([]bool, width*height)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r16, g16, b16, _ := pixels.At(x, y).RGBA()
			colour := [3]uint8{uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8)}
			matched[(y-bounds.Min.Y)*width+x-bounds.Min.X] = sameDominantHue(colour, expected)
		}
	}
	largest := 0
	queue := make([]int, 0, 256)
	for start := range matched {
		if !matched[start] {
			continue
		}
		matched[start] = false
		queue = append(queue[:0], start)
		count := 0
		for len(queue) > 0 {
			index := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			count++
			x, y := index%width, index/width
			for _, next := range []int{index - 1, index + 1, index - width, index + width} {
				if next < 0 || next >= len(matched) || !matched[next] {
					continue
				}
				nextX, nextY := next%width, next/width
				if abs(nextX-x)+abs(nextY-y) != 1 {
					continue
				}
				matched[next] = false
				queue = append(queue, next)
			}
		}
		if count > largest {
			largest = count
		}
	}
	return largest
}

func sameDominantHue(colour, expected [3]uint8) bool {
	dominant := 0
	for index := 1; index < len(expected); index++ {
		if expected[index] > expected[dominant] {
			dominant = index
		}
	}
	second := uint8(0)
	for index, value := range colour {
		if index != dominant && value > second {
			second = value
		}
	}
	return colour[dominant] >= 64 && int(colour[dominant])-int(second) >= 24
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
