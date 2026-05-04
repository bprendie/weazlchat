package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type rgb struct {
	r float64
	g float64
	b float64
}

var logoStops = []rgb{
	hexRGB(0xF25D94),
	hexRGB(0x7D56F4),
	hexRGB(0x04B575),
	hexRGB(0xF7D774),
}

func gradientLogo(s string) string {
	lines := strings.Split(s, "\n")
	width := 1
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}

	var out strings.Builder
	for y, line := range lines {
		for x, r := range line {
			if r == ' ' {
				out.WriteRune(r)
				continue
			}
			t := float64(x) / float64(width-1)
			t = math.Mod(t+(float64(y)*0.08), 1)
			color := sampleGradient(t, logoStops)
			out.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(string(r)))
		}
		if y < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func sampleGradient(t float64, stops []rgb) string {
	if len(stops) == 0 {
		return "#FFFFFF"
	}
	if len(stops) == 1 {
		return stops[0].hex()
	}
	scaled := t * float64(len(stops)-1)
	i := int(math.Floor(scaled))
	if i >= len(stops)-1 {
		return stops[len(stops)-1].hex()
	}
	local := scaled - float64(i)
	a := stops[i]
	b := stops[i+1]
	return rgb{
		r: a.r + ((b.r - a.r) * local),
		g: a.g + ((b.g - a.g) * local),
		b: a.b + ((b.b - a.b) * local),
	}.hex()
}

func hexRGB(v int) rgb {
	return rgb{
		r: float64((v >> 16) & 0xff),
		g: float64((v >> 8) & 0xff),
		b: float64(v & 0xff),
	}
}

func (c rgb) hex() string {
	return fmt.Sprintf("#%02X%02X%02X", clamp(c.r), clamp(c.g), clamp(c.b))
}

func clamp(v float64) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return int(math.Round(v))
}
