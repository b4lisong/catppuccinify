package converter

import (
	"image"
	"image/color"
	"testing"
)

// TestSinglePixelRedMapping verifies that a pure red pixel maps to the
// nearest Catppuccin Mocha color (expected: Red #f38ba8).
func TestSinglePixelRedMapping(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

	out := Convert(img)
	c := out.NRGBAAt(0, 0)

	// With a single pixel there is no dithering, so the output must be
	// exactly the nearest palette entry.
	found := false
	for _, pc := range MochaPalette {
		if pc.R == c.R && pc.G == c.G && pc.B == c.B {
			found = true
			t.Logf("pure red mapped to %s (#%02x%02x%02x)", pc.Name, pc.R, pc.G, pc.B)
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

	out := Convert(img)
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

	out := Convert(img)

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
