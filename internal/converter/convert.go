package converter

import (
	"image"
	"image/color"
	"math"

	"github.com/lucasb-eyer/go-colorful"
)

// Convert maps every pixel of img to the nearest Catppuccin Mocha palette
// color using CIELAB distance and Floyd-Steinberg dithering.
// Fully transparent pixels are preserved as-is.
func Convert(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Build float64 pixel buffer in CIELAB space and a parallel alpha buffer.
	type labPixel struct {
		L, A, B float64
	}
	buf := make([]labPixel, w*h)
	alpha := make([]uint8, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			idx := y*w + x

			// Store original alpha (pre-multiplied -> straight).
			a8 := uint8(a >> 8)
			alpha[idx] = a8

			if a8 == 0 {
				continue
			}

			// Convert pre-multiplied to straight, then to [0,1].
			fa := float64(a)
			fr := float64(r) / fa
			fg := float64(g) / fa
			fb := float64(b) / fa

			col := colorful.Color{R: fr, G: fg, B: fb}
			l, aa, bb := col.Lab()
			buf[idx] = labPixel{L: l, A: aa, B: bb}
		}
	}

	out := image.NewNRGBA(bounds)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x

			if alpha[idx] == 0 {
				out.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{0, 0, 0, 0})
				continue
			}

			oldL := buf[idx].L
			oldA := buf[idx].A
			oldB := buf[idx].B

			// Find nearest palette color in CIELAB.
			best := 0
			bestDist := math.MaxFloat64
			for i, pc := range MochaPalette {
				dL := oldL - pc.L
				dA := oldA - pc.A
				dB := oldB - pc.Bfield
				dist := dL*dL + dA*dA + dB*dB
				if dist < bestDist {
					bestDist = dist
					best = i
				}
			}

			pc := MochaPalette[best]

			// Compute quantisation error in CIELAB.
			errL := oldL - pc.L
			errA := oldA - pc.A
			errB := oldB - pc.Bfield

			// Floyd-Steinberg error diffusion.
			diffuse := func(dx, dy int, factor float64) {
				nx, ny := x+dx, y+dy
				if nx < 0 || nx >= w || ny < 0 || ny >= h {
					return
				}
				ni := ny*w + nx
				if alpha[ni] == 0 {
					return
				}
				buf[ni].L += errL * factor
				buf[ni].A += errA * factor
				buf[ni].B += errB * factor
			}

			diffuse(1, 0, 7.0/16.0)
			diffuse(-1, 1, 3.0/16.0)
			diffuse(0, 1, 5.0/16.0)
			diffuse(1, 1, 1.0/16.0)

			out.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{
				R: pc.R,
				G: pc.G,
				B: pc.B,
				A: alpha[idx],
			})
		}
	}

	return out
}
