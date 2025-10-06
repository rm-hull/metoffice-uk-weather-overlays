package stage

import (
	"image"
	"image/color"

	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
)

type GreyscaleStage struct{}

// Process converts the image to greyscale using luminance calculation
// The alpha channel is set based on the luminance value, with higher luminance resulting in higher opacity
// Fully transparent pixels remain transparent
func (s *GreyscaleStage) Process(p *png.PngImage) error {
	gs := image.NewNRGBA(p.Bounds)
	for y := p.Bounds.Min.Y; y < p.Bounds.Max.Y; y++ {
		for x := p.Bounds.Min.X; x < p.Bounds.Max.X; x++ {
			r, g, b, a := p.Img.At(x, y).RGBA()
			if a == 0 {
				gs.Set(x, y, color.NRGBA{0, 0, 0, 0})
				continue
			}
			// Calculate luminance using standard coefficients
			// Reference: https://en.wikipedia.org/wiki/Grayscale#Luma_coding_in_video_systems
			lum := uint8(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8))
			gs.Set(x, y, color.NRGBA{255, 255, 255, lum})
		}
	}
	p.Img = gs
	return nil
}
