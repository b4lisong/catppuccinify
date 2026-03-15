package converter

import (
	"image"
	"image/color"
	"math"

	"github.com/lucasb-eyer/go-colorful"
)

// sRGBToLinear applies the sRGB transfer function to convert a single
// sRGB component (0–1) to linear RGB.
func sRGBToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// Convert maps every pixel of img to the nearest Catppuccin Mocha palette
// color using CIEDE2000 distance and Floyd-Steinberg dithering.
// Error diffusion happens in sRGB space. Fully transparent pixels are
// preserved as-is.
func Convert(img image.Image, onProgress func(percent int)) *image.NRGBA {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Float64 pixel buffer in sRGB space (0–255 range) for error diffusion.
	type rgbPixel struct {
		R, G, B float64
	}
	buf := make([]rgbPixel, w*h)
	alpha := make([]uint8, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			idx := y*w + x

			a8 := uint8(a >> 8)
			alpha[idx] = a8

			if a8 == 0 {
				continue
			}

			// Convert pre-multiplied 16-bit to straight 8-bit sRGB.
			fa := float64(a)
			buf[idx] = rgbPixel{
				R: float64(r) / fa * 255.0,
				G: float64(g) / fa * 255.0,
				B: float64(b) / fa * 255.0,
			}
		}
	}

	out := image.NewNRGBA(bounds)

	for y := 0; y < h; y++ {
		if onProgress != nil {
			onProgress(y * 100 / h)
		}
		for x := 0; x < w; x++ {
			idx := y*w + x

			if alpha[idx] == 0 {
				out.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{0, 0, 0, 0})
				continue
			}

			// Step 4b: clamp to [0, 255].
			oldR := clamp(buf[idx].R, 0, 255)
			oldG := clamp(buf[idx].G, 0, 255)
			oldB := clamp(buf[idx].B, 0, 255)

			// Step 4c: convert sRGB to colorful.Color (with proper gamma).
			pixelColor := colorful.Color{
				R: sRGBToLinear(oldR / 255.0),
				G: sRGBToLinear(oldG / 255.0),
				B: sRGBToLinear(oldB / 255.0),
			}

			// Step 4d: find nearest palette color by CIEDE2000.
			best := 0
			bestDist := math.MaxFloat64
			for i, pc := range MochaPalette {
				dist := pixelColor.DistanceCIEDE2000(pc.Color)
				if dist < bestDist {
					bestDist = dist
					best = i
				}
			}

			pc := MochaPalette[best]

			// Step 4e: quantization error in sRGB space.
			errR := oldR - float64(pc.R)
			errG := oldG - float64(pc.G)
			errB := oldB - float64(pc.B)

			// Step 4f: Floyd-Steinberg error diffusion.
			diffuse := func(dx, dy int, factor float64) {
				nx, ny := x+dx, y+dy
				if nx < 0 || nx >= w || ny < 0 || ny >= h {
					return
				}
				ni := ny*w + nx
				if alpha[ni] == 0 {
					return
				}
				buf[ni].R += errR * factor
				buf[ni].G += errG * factor
				buf[ni].B += errB * factor
			}

			diffuse(1, 0, 7.0/16.0)
			diffuse(-1, 1, 3.0/16.0)
			diffuse(0, 1, 5.0/16.0)
			diffuse(1, 1, 1.0/16.0)

			// Step 4g: write palette color to output.
			out.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{
				R: pc.R,
				G: pc.G,
				B: pc.B,
				A: alpha[idx],
			})
		}
	}

	if onProgress != nil {
		onProgress(100)
	}

	return out
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
