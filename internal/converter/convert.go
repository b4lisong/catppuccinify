package converter

import (
	"image"
	"image/color"
	"math"
	"sync"

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

// sRGBToLinearLUT precomputes sRGBToLinear for all 256 possible byte values.
var sRGBToLinearLUT [256]float64

func init() {
	for i := range sRGBToLinearLUT {
		sRGBToLinearLUT[i] = sRGBToLinear(float64(i) / 255.0)
	}
}

// rgbPixel is a floating-point pixel in sRGB space (0–255 range) for error diffusion.
type rgbPixel struct {
	R, G, B float64
}

var rgbPixelPool = sync.Pool{
	New: func() any {
		s := make([]rgbPixel, 0)
		return &s
	},
}

var alphaPool = sync.Pool{
	New: func() any {
		s := make([]uint8, 0)
		return &s
	},
}

// sq returns the square of a float64.
func sq(v float64) float64 { return v * v }

// distanceCIEDE2000Lab computes the CIEDE2000 color difference given two
// colors already in Lab space (L: 0-100, a/b: typically -128 to 127).
// This is the go-colorful formula without the redundant Lab() conversions.
func distanceCIEDE2000Lab(l1, a1, b1, l2, a2, b2 float64) float64 {
	cab1 := math.Sqrt(sq(a1) + sq(b1))
	cab2 := math.Sqrt(sq(a2) + sq(b2))
	cabmean := (cab1 + cab2) / 2

	g := 0.5 * (1 - math.Sqrt(math.Pow(cabmean, 7)/(math.Pow(cabmean, 7)+math.Pow(25, 7))))
	ap1 := (1 + g) * a1
	ap2 := (1 + g) * a2
	cp1 := math.Sqrt(sq(ap1) + sq(b1))
	cp2 := math.Sqrt(sq(ap2) + sq(b2))

	hp1 := 0.0
	if b1 != ap1 || ap1 != 0 {
		hp1 = math.Atan2(b1, ap1)
		if hp1 < 0 {
			hp1 += math.Pi * 2
		}
		hp1 *= 180 / math.Pi
	}
	hp2 := 0.0
	if b2 != ap2 || ap2 != 0 {
		hp2 = math.Atan2(b2, ap2)
		if hp2 < 0 {
			hp2 += math.Pi * 2
		}
		hp2 *= 180 / math.Pi
	}

	deltaLp := l2 - l1
	deltaCp := cp2 - cp1
	dhp := 0.0
	cpProduct := cp1 * cp2
	if cpProduct != 0 {
		dhp = hp2 - hp1
		if dhp > 180 {
			dhp -= 360
		} else if dhp < -180 {
			dhp += 360
		}
	}
	deltaHp := 2 * math.Sqrt(cpProduct) * math.Sin(dhp/2*math.Pi/180)

	lpmean := (l1 + l2) / 2
	cpmean := (cp1 + cp2) / 2
	hpmean := hp1 + hp2
	if cpProduct != 0 {
		hpmean /= 2
		if math.Abs(hp1-hp2) > 180 {
			if hp1+hp2 < 360 {
				hpmean += 180
			} else {
				hpmean -= 180
			}
		}
	}

	t := 1 - 0.17*math.Cos((hpmean-30)*math.Pi/180) + 0.24*math.Cos(2*hpmean*math.Pi/180) + 0.32*math.Cos((3*hpmean+6)*math.Pi/180) - 0.2*math.Cos((4*hpmean-63)*math.Pi/180)
	deltaTheta := 30 * math.Exp(-sq((hpmean - 275) / 25))
	rc := 2 * math.Sqrt(math.Pow(cpmean, 7)/(math.Pow(cpmean, 7)+math.Pow(25, 7)))
	sl := 1 + (0.015*sq(lpmean-50))/math.Sqrt(20+sq(lpmean-50))
	sc := 1 + 0.045*cpmean
	sh := 1 + 0.015*cpmean*t
	rt := -math.Sin(2*deltaTheta*math.Pi/180) * rc

	return math.Sqrt(sq(deltaLp/sl)+sq(deltaCp/sc)+sq(deltaHp/sh)+rt*(deltaCp/sc)*(deltaHp/sh)) * 0.01
}

// Convert maps every pixel of img to the nearest Catppuccin Mocha palette
// color using CIEDE2000 distance and Floyd-Steinberg dithering.
// Error diffusion happens in sRGB space. Fully transparent pixels are
// preserved as-is.
func Convert(img image.Image, onProgress func(percent int)) *image.NRGBA {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	n := w * h

	// Get buffers from pool and grow if needed.
	bufPtr := rgbPixelPool.Get().(*[]rgbPixel)
	if cap(*bufPtr) < n {
		*bufPtr = make([]rgbPixel, n)
	} else {
		*bufPtr = (*bufPtr)[:n]
		clear(*bufPtr)
	}
	buf := *bufPtr

	alphaPtr := alphaPool.Get().(*[]uint8)
	if cap(*alphaPtr) < n {
		*alphaPtr = make([]uint8, n)
	} else {
		*alphaPtr = (*alphaPtr)[:n]
		clear(*alphaPtr)
	}
	alpha := *alphaPtr

	defer func() {
		rgbPixelPool.Put(bufPtr)
		alphaPool.Put(alphaPtr)
	}()

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
	lastPct := -1

	for y := 0; y < h; y++ {
		if onProgress != nil {
			pct := y * 100 / h
			if pct != lastPct {
				lastPct = pct
				onProgress(pct)
			}
		}
		for x := 0; x < w; x++ {
			idx := y*w + x

			if alpha[idx] == 0 {
				out.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{0, 0, 0, 0})
				continue
			}

			// Clamp to [0, 255].
			oldR := clamp(buf[idx].R, 0, 255)
			oldG := clamp(buf[idx].G, 0, 255)
			oldB := clamp(buf[idx].B, 0, 255)

			// Convert sRGB to linear RGB via LUT, then to Lab for CIEDE2000.
			pixelColor := colorful.Color{
				R: sRGBToLinearLUT[int(oldR+0.5)],
				G: sRGBToLinearLUT[int(oldG+0.5)],
				B: sRGBToLinearLUT[int(oldB+0.5)],
			}
			pl, pa, pb := pixelColor.Lab()
			pl, pa, pb = pl*100.0, pa*100.0, pb*100.0

			// Find nearest palette color by CIEDE2000 using precomputed Lab values.
			best := 0
			bestDist := math.MaxFloat64
			for i, pc := range MochaPalette {
				dist := distanceCIEDE2000Lab(pl, pa, pb, pc.LabL, pc.LabA, pc.LabB)
				if dist < bestDist {
					bestDist = dist
					best = i
				}
			}

			pc := MochaPalette[best]

			// Quantization error in sRGB space.
			errR := oldR - float64(pc.R)
			errG := oldG - float64(pc.G)
			errB := oldB - float64(pc.B)

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
				buf[ni].R += errR * factor
				buf[ni].G += errG * factor
				buf[ni].B += errB * factor
			}

			diffuse(1, 0, 7.0/16.0)
			diffuse(-1, 1, 3.0/16.0)
			diffuse(0, 1, 5.0/16.0)
			diffuse(1, 1, 1.0/16.0)

			// Write palette color to output.
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
