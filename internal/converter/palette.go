package converter

import (
	"github.com/lucasb-eyer/go-colorful"
)

// PaletteColor represents a single Catppuccin palette color with both
// sRGB and CIELAB components.
type PaletteColor struct {
	Name           string
	R, G, B        uint8
	L, A, Bfield   float64 // CIELAB components
}

// MochaPalette contains all 26 Catppuccin Mocha colors with pre-computed
// CIELAB values.
var MochaPalette []PaletteColor

func init() {
	type entry struct {
		name string
		r, g, b uint8
	}

	colors := []entry{
		{"Rosewater", 0xf5, 0xe0, 0xdc},
		{"Flamingo", 0xf2, 0xcd, 0xcd},
		{"Pink", 0xf5, 0xc2, 0xe7},
		{"Mauve", 0xcb, 0xa6, 0xf7},
		{"Red", 0xf3, 0x8b, 0xa8},
		{"Maroon", 0xeb, 0xa0, 0xac},
		{"Peach", 0xfa, 0xb3, 0x87},
		{"Yellow", 0xf9, 0xe2, 0xaf},
		{"Green", 0xa6, 0xe3, 0xa1},
		{"Teal", 0x94, 0xe2, 0xd5},
		{"Sky", 0x89, 0xdc, 0xeb},
		{"Sapphire", 0x74, 0xc7, 0xec},
		{"Blue", 0x89, 0xb4, 0xfa},
		{"Lavender", 0xb4, 0xbe, 0xfe},
		{"Text", 0xcd, 0xd6, 0xf4},
		{"Subtext1", 0xba, 0xc2, 0xde},
		{"Subtext0", 0xa6, 0xad, 0xc8},
		{"Overlay2", 0x93, 0x99, 0xb2},
		{"Overlay1", 0x7f, 0x84, 0x9c},
		{"Overlay0", 0x6c, 0x70, 0x86},
		{"Surface2", 0x58, 0x5b, 0x70},
		{"Surface1", 0x45, 0x47, 0x5a},
		{"Surface0", 0x31, 0x32, 0x44},
		{"Base", 0x1e, 0x1e, 0x2e},
		{"Mantle", 0x18, 0x18, 0x25},
		{"Crust", 0x11, 0x11, 0x1b},
	}

	MochaPalette = make([]PaletteColor, len(colors))
	for i, c := range colors {
		col := colorful.Color{
			R: float64(c.r) / 255.0,
			G: float64(c.g) / 255.0,
			B: float64(c.b) / 255.0,
		}
		l, a, b := col.Lab()
		MochaPalette[i] = PaletteColor{
			Name:   c.name,
			R:      c.r,
			G:      c.g,
			B:      c.b,
			L:      l,
			A:      a,
			Bfield: b,
		}
	}
}
