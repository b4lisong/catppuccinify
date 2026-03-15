package converter

import (
	"github.com/lucasb-eyer/go-colorful"
)

// PaletteColor represents a single Catppuccin palette color.
type PaletteColor struct {
	Name  string
	Hex   string
	R, G, B uint8
	Color colorful.Color // precomputed at startup for CIEDE2000 distance
}

// MochaPalette contains all 26 Catppuccin Mocha colors.
var MochaPalette []PaletteColor

func init() {
	type entry struct {
		name string
		hex  string
	}

	colors := []entry{
		{"Rosewater", "#f5e0dc"},
		{"Flamingo", "#f2cdcd"},
		{"Pink", "#f5c2e7"},
		{"Mauve", "#cba6f7"},
		{"Red", "#f38ba8"},
		{"Maroon", "#eba0ac"},
		{"Peach", "#fab387"},
		{"Yellow", "#f9e2af"},
		{"Green", "#a6e3a1"},
		{"Teal", "#94e2d5"},
		{"Sky", "#89dceb"},
		{"Sapphire", "#74c7ec"},
		{"Blue", "#89b4fa"},
		{"Lavender", "#b4befe"},
		{"Text", "#cdd6f4"},
		{"Subtext1", "#bac2de"},
		{"Subtext0", "#a6adc8"},
		{"Overlay2", "#9399b2"},
		{"Overlay1", "#7f849c"},
		{"Overlay0", "#6c7086"},
		{"Surface2", "#585b70"},
		{"Surface1", "#45475a"},
		{"Surface0", "#313244"},
		{"Base", "#1e1e2e"},
		{"Mantle", "#181825"},
		{"Crust", "#11111b"},
	}

	MochaPalette = make([]PaletteColor, len(colors))
	for i, c := range colors {
		col, _ := colorful.Hex(c.hex)
		r, g, b := col.RGB255()
		MochaPalette[i] = PaletteColor{
			Name:  c.name,
			Hex:   c.hex,
			R:     r,
			G:     g,
			B:     b,
			Color: col,
		}
	}
}
