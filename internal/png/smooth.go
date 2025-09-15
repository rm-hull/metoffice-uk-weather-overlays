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
func Smooth(r io.Reader, w io.Writer, tolerance float64, sigma float64) error {

	img, err := png.Decode(r)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	out := image.NewNRGBA(bounds)

	// Thresholds
	whiteR, whiteG, whiteB := 255.0, 255.0, 255.0

	// Loop over pixels
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			R, G, B, A := float64(r>>8), float64(g>>8), float64(b>>8), float64(a>>8)

			// Distance from white
			dist := math.Sqrt((whiteR-R)*(whiteR-R) + (whiteG-G)*(whiteG-G) + (whiteB-B)*(whiteB-B))

			if dist < tolerance {
				// Scale alpha based on closeness to white
				alpha := uint8((dist / tolerance) * A)
				out.Set(x, y, color.NRGBA{uint8(R), uint8(G), uint8(B), alpha})
			} else {
				out.Set(x, y, color.NRGBA{uint8(R), uint8(G), uint8(B), uint8(A)})
			}
		}
	}

	// Apply Gaussian blur to the whole RGBA image
	blurred := blur.Gaussian(out, sigma)

	// Now smooth edges with bicubic-like resampling
	smoothed := image.NewNRGBA(bounds)
	draw.CatmullRom.Scale(smoothed, bounds, blurred, bounds, draw.Over, nil)

	return png.Encode(w, smoothed)
}
