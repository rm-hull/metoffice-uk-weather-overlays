package stage

import (
	"image"

	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
	"golang.org/x/image/draw"
)

type ResampleStage struct{}

// Process applies a Catmull-Rom resampling to smooth the image
// This can help reduce artifacts introduced by other processing stages
// such as color replacement and blurring
func (s *ResampleStage) Process(p *png.PngImage) error {
	smoothed := image.NewNRGBA(p.Bounds)
	draw.CatmullRom.Scale(smoothed, p.Bounds, p.Img, p.Bounds, draw.Over, nil)
	p.Img = smoothed
	return nil
}
