package converter

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/lucasb-eyer/go-colorful"
)

// TestSinglePixelRedMapping verifies that a pure red pixel maps to the
// nearest Catppuccin Mocha color via CIEDE2000.
func TestSinglePixelRedMapping(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

	out := Convert(img, MochaPalette, nil)
	c := out.NRGBAAt(0, 0)

	found := false
	for _, pc := range MochaPalette {
		if pc.R == c.R && pc.G == c.G && pc.B == c.B {
			found = true
			t.Logf("pure red mapped to %s (%s)", pc.Name, pc.Hex)
			break
		}
	}
	if !found {
		t.Errorf("output pixel (%d,%d,%d) is not a valid palette color", c.R, c.G, c.B)
	}
}

// TestAllOutputPixelsAreValid converts a small gradient image and checks
// that every opaque output pixel matches a palette color.
func TestAllOutputPixelsAreValid(t *testing.T) {
	const size = 16
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 16),
				G: uint8(y * 16),
				B: 128,
				A: 255,
			})
		}
	}

	paletteSet := make(map[[3]uint8]bool, len(MochaPalette))
	for _, pc := range MochaPalette {
		paletteSet[[3]uint8{pc.R, pc.G, pc.B}] = true
	}

	out := Convert(img, MochaPalette, nil)
	bounds := out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := out.NRGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			key := [3]uint8{c.R, c.G, c.B}
			if !paletteSet[key] {
				t.Fatalf("pixel (%d,%d) = (%d,%d,%d) is not a palette color", x, y, c.R, c.G, c.B)
			}
		}
	}
}

// TestAlphaPreservation verifies that fully transparent pixels remain
// transparent and that semi-opaque alpha values are preserved.
func TestAlphaPreservation(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 0})   // transparent
	img.SetNRGBA(1, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 128}) // semi-opaque
	img.SetNRGBA(2, 0, color.NRGBA{R: 200, G: 100, B: 50, A: 255}) // opaque

	out := Convert(img, MochaPalette, nil)

	if a := out.NRGBAAt(0, 0).A; a != 0 {
		t.Errorf("transparent pixel alpha: got %d, want 0", a)
	}
	if a := out.NRGBAAt(1, 0).A; a != 128 {
		t.Errorf("semi-opaque pixel alpha: got %d, want 128", a)
	}
	if a := out.NRGBAAt(2, 0).A; a != 255 {
		t.Errorf("opaque pixel alpha: got %d, want 255", a)
	}
}

// TestBoundaryConditions verifies that dithering doesn't panic on edge
// pixels where some Floyd-Steinberg neighbors are out of bounds.
func TestBoundaryConditions(t *testing.T) {
	// 1x1 image: no neighbors at all.
	img1 := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img1.SetNRGBA(0, 0, color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	out1 := Convert(img1, MochaPalette, nil)
	c := out1.NRGBAAt(0, 0)
	if c.A != 255 {
		t.Errorf("1x1 image: expected alpha 255, got %d", c.A)
	}

	// 2x2 image: corners have limited neighbors.
	img2 := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img2.SetNRGBA(x, y, color.NRGBA{R: 100, G: 50, B: 200, A: 255})
		}
	}
	out2 := Convert(img2, MochaPalette, nil)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			if out2.NRGBAAt(x, y).A != 255 {
				t.Errorf("2x2 pixel (%d,%d): expected alpha 255", x, y)
			}
		}
	}
}

// TestDistanceCIEDE2000Lab verifies that our inlined CIEDE2000 formula
// matches go-colorful's DistanceCIEDE2000 for several color pairs.
func TestDistanceCIEDE2000Lab(t *testing.T) {
	pairs := [][2]colorful.Color{
		{colorful.Color{R: 1, G: 0, B: 0}, colorful.Color{R: 0, G: 1, B: 0}},
		{colorful.Color{R: 0, G: 0, B: 1}, colorful.Color{R: 1, G: 1, B: 0}},
		{colorful.Color{R: 0.5, G: 0.5, B: 0.5}, colorful.Color{R: 0.2, G: 0.3, B: 0.8}},
		{colorful.Color{R: 0, G: 0, B: 0}, colorful.Color{R: 1, G: 1, B: 1}},
		{colorful.Color{R: 0.9, G: 0.1, B: 0.5}, colorful.Color{R: 0.1, G: 0.9, B: 0.5}},
	}

	for i, pair := range pairs {
		expected := pair[0].DistanceCIEDE2000(pair[1])

		l1, a1, b1 := pair[0].Lab()
		l2, a2, b2 := pair[1].Lab()
		got := distanceCIEDE2000Lab(l1*100, a1*100, b1*100, l2*100, a2*100, b2*100)

		if math.Abs(got-expected) > 1e-10 {
			t.Errorf("pair %d: distanceCIEDE2000Lab = %v, want %v (diff %v)", i, got, expected, math.Abs(got-expected))
		}
	}
}

// TestAllFlavorsConvert verifies that converting with each flavor produces
// valid output without panics and all pixels are valid palette colors.
func TestAllFlavorsConvert(t *testing.T) {
	const size = 4
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 64),
				G: uint8(y * 64),
				B: 128,
				A: 255,
			})
		}
	}

	for _, flavor := range []Flavor{Latte, Frappe, Macchiato, Mocha} {
		palette := Palettes[flavor]
		t.Run(string(flavor), func(t *testing.T) {
			paletteSet := make(map[[3]uint8]bool, len(palette))
			for _, pc := range palette {
				paletteSet[[3]uint8{pc.R, pc.G, pc.B}] = true
			}

			out := Convert(img, palette, nil)
			bounds := out.Bounds()
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					c := out.NRGBAAt(x, y)
					if c.A == 0 {
						continue
					}
					key := [3]uint8{c.R, c.G, c.B}
					if !paletteSet[key] {
						t.Fatalf("pixel (%d,%d) = (%d,%d,%d) is not a %s palette color", x, y, c.R, c.G, c.B, flavor)
					}
				}
			}
		})
	}
}

func makeGradientImage(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 255 / size),
				G: uint8(y * 255 / size),
				B: uint8((x + y) * 128 / size),
				A: 255,
			})
		}
	}
	return img
}

func BenchmarkConvert(b *testing.B) {
	img := makeGradientImage(256)
	b.ResetTimer()
	for b.Loop() {
		Convert(img, MochaPalette, nil)
	}
}

func BenchmarkConvertLarge(b *testing.B) {
	img := makeGradientImage(1024)
	b.ResetTimer()
	for b.Loop() {
		Convert(img, MochaPalette, nil)
	}
}
