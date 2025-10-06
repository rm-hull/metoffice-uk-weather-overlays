package png

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"

	"github.com/anthonynsimon/bild/blur"
	"golang.org/x/image/draw"
)

// adjust tolerance: higher means more aggressive removal
// adjust sigma: tweak for more/less blur
func Smooth(r io.Reader, w io.Writer, tolerance float64, sigma float64, replace color.Color, greyscale bool) error {

	img, err := png.Decode(r)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)

	replaceR, replaceG, replaceB, _ := replace.RGBA()
	rR, rG, rB := float64(replaceR>>8), float64(replaceG>>8), float64(replaceB>>8)

	// 1. Replace specified color with transparency
	//    with alpha scaled by distance from color within tolerance
	//    (i.e. pixels close to the target color become more transparent)
	//
	//    This helps avoid hard edges when blurring later
	//
	//    Note: this is done in RGBA space which is not perceptually uniform
	//          but is good enough for our purposes here
	//
	//    See: https://en.wikipedia.org/wiki/Alpha_compositing#Description
	//         https://en.wikipedia.org/wiki/Color_difference
	// Loop over pixels
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			R, G, B, A := float64(r>>8), float64(g>>8), float64(b>>8), float64(a>>8)

			// Distance from replace color
			dist := math.Sqrt((rR-R)*(rR-R) + (rG-G)*(rG-G) + (rB-B)*(rB-B))

			if dist < tolerance {
				// Replace with specified color and scale alpha
				alpha := uint8((dist / tolerance) * A)
				out.Set(x, y, color.NRGBA{uint8(R), uint8(G), uint8(B), alpha})
			} else {
				out.Set(x, y, color.NRGBA{uint8(R), uint8(G), uint8(B), uint8(A)})
			}
		}
	}

	// 2. If greyscale is true, convert to greyscale before blur
	//	This is useful for cloud amount to avoid color tints after blurring
	var imgForBlur image.Image = out
	if greyscale {
		gs := image.NewNRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, a := out.At(x, y).RGBA()
				if a == 0 {
					gs.Set(x, y, color.NRGBA{0, 0, 0, 0})
					continue
				}
				// Calculate luminance using standard formula
				lum := uint8(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8))
				// gs.Set(x, y, color.NRGBA{lum, lum, lum, uint8(a >> 8)})
				gs.Set(x, y, color.NRGBA{255, 255, 255, lum})
			}
		}
		imgForBlur = gs
	}

	// 3. Apply Gaussian blur to the (possibly greyscale) image
	//	This helps smooth edges created by color replacement above
	//
	//	Note: bild/blur.Gaussian uses a fast approximation which is good enough here
	//
	//	See: https://pkg.go.dev/github.com/anthonynsimon/bild/blur#Gaussian
	blurred := blur.Gaussian(imgForBlur, sigma)

	// 4. Now smooth edges with bicubic-like resampling
	//	This helps reduce any remaining artifacts from the blur step
	//
	//	See: https://pkg.go.dev/golang.org/x/image/draw#CatmullRom
	//
	//	Note: this is a bit of a hack but works well enough for our purposes here
	//	      as we are scaling to the same size (i.e. no actual scaling)
	//
	//	      We could use a proper bicubic filter but this is simpler and good enough
	//
	//	      See: https://en.wikipedia.org/wiki/Bicubic_interpolation
	//	           https://en.wikipedia.org/wiki/Image_scaling
	//
	smoothed := image.NewNRGBA(bounds)
	draw.CatmullRom.Scale(smoothed, bounds, blurred, bounds, draw.Over, nil)

	return png.Encode(w, smoothed)
}
