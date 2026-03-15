package converter

import (
	"github.com/lucasb-eyer/go-colorful"
)

// PaletteColor represents a single Catppuccin palette color.
type PaletteColor struct {
	Name        string
	Hex         string
	R, G, B     uint8
	Color       colorful.Color // precomputed at startup for CIEDE2000 distance
	LabL, LabA, LabB float64   // precomputed Lab values scaled to 0-100 range
}

// Flavor identifies a Catppuccin color scheme variant.
type Flavor string

const (
	Latte     Flavor = "latte"
	Frappe    Flavor = "frappe"
	Macchiato Flavor = "macchiato"
	Mocha     Flavor = "mocha"
)

// Palettes maps each Catppuccin flavor to its 26-color palette.
var Palettes map[Flavor][]PaletteColor

// MochaPalette is an alias for backwards compatibility.
var MochaPalette []PaletteColor

type entry struct {
	name string
	hex  string
}

func buildPalette(entries []entry) []PaletteColor {
	palette := make([]PaletteColor, len(entries))
	for i, c := range entries {
		col, _ := colorful.Hex(c.hex)
		r, g, b := col.RGB255()
		l, a, bVal := col.Lab()
		palette[i] = PaletteColor{
			Name:  c.name,
			Hex:   c.hex,
			R:     r,
			G:     g,
			B:     b,
			Color: col,
			LabL:  l * 100.0,
			LabA:  a * 100.0,
			LabB:  bVal * 100.0,
		}
	}
	return palette
}

// PaletteForFlavor returns the palette for the given flavor name.
// Returns (palette, true) on success. Defaults to Mocha on empty string.
// Returns (nil, false) for unknown flavors.
func PaletteForFlavor(flavor string) ([]PaletteColor, bool) {
	if flavor == "" {
		return Palettes[Mocha], true
	}
	p, ok := Palettes[Flavor(flavor)]
	return p, ok
}

func init() {
	Palettes = make(map[Flavor][]PaletteColor, 4)

	Palettes[Latte] = buildPalette([]entry{
		{"Rosewater", "#dc8a78"},
		{"Flamingo", "#dd7878"},
		{"Pink", "#ea76cb"},
		{"Mauve", "#8839ef"},
		{"Red", "#d20f39"},
		{"Maroon", "#e64553"},
		{"Peach", "#fe640b"},
		{"Yellow", "#df8e1d"},
		{"Green", "#40a02b"},
		{"Teal", "#179299"},
		{"Sky", "#04a5e5"},
		{"Sapphire", "#209fb5"},
		{"Blue", "#1e66f5"},
		{"Lavender", "#7287fd"},
		{"Text", "#4c4f69"},
		{"Subtext1", "#5c5f77"},
		{"Subtext0", "#6c6f85"},
		{"Overlay2", "#7c7f93"},
		{"Overlay1", "#8c8fa1"},
		{"Overlay0", "#9ca0b0"},
		{"Surface2", "#acb0be"},
		{"Surface1", "#bcc0cc"},
		{"Surface0", "#ccd0da"},
		{"Base", "#eff1f5"},
		{"Mantle", "#e6e9ef"},
		{"Crust", "#dce0e8"},
	})

	Palettes[Frappe] = buildPalette([]entry{
		{"Rosewater", "#f2d5cf"},
		{"Flamingo", "#eebebe"},
		{"Pink", "#f4b8e4"},
		{"Mauve", "#ca9ee6"},
		{"Red", "#e78284"},
		{"Maroon", "#ea999c"},
		{"Peach", "#ef9f76"},
		{"Yellow", "#e5c890"},
		{"Green", "#a6d189"},
		{"Teal", "#81c8be"},
		{"Sky", "#99d1db"},
		{"Sapphire", "#85c1dc"},
		{"Blue", "#8caaee"},
		{"Lavender", "#babbf1"},
		{"Text", "#c6d0f5"},
		{"Subtext1", "#b5bfe2"},
		{"Subtext0", "#a5adce"},
		{"Overlay2", "#949cbb"},
		{"Overlay1", "#838ba7"},
		{"Overlay0", "#737994"},
		{"Surface2", "#626880"},
		{"Surface1", "#51576d"},
		{"Surface0", "#414559"},
		{"Base", "#303446"},
		{"Mantle", "#292c3c"},
		{"Crust", "#232634"},
	})

	Palettes[Macchiato] = buildPalette([]entry{
		{"Rosewater", "#f4dbd6"},
		{"Flamingo", "#f0c6c6"},
		{"Pink", "#f5bde6"},
		{"Mauve", "#c6a0f6"},
		{"Red", "#ed8796"},
		{"Maroon", "#ee99a0"},
		{"Peach", "#f5a97f"},
		{"Yellow", "#eed49f"},
		{"Green", "#a6da95"},
		{"Teal", "#8bd5ca"},
		{"Sky", "#91d7e3"},
		{"Sapphire", "#7dc4e4"},
		{"Blue", "#8aadf4"},
		{"Lavender", "#b7bdf8"},
		{"Text", "#cad3f5"},
		{"Subtext1", "#b8c0e0"},
		{"Subtext0", "#a5adcb"},
		{"Overlay2", "#939ab7"},
		{"Overlay1", "#8087a2"},
		{"Overlay0", "#6e738d"},
		{"Surface2", "#5b6078"},
		{"Surface1", "#494d64"},
		{"Surface0", "#363a4f"},
		{"Base", "#24273a"},
		{"Mantle", "#1e2030"},
		{"Crust", "#181926"},
	})

	Palettes[Mocha] = buildPalette([]entry{
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
	})

	MochaPalette = Palettes[Mocha]
}
