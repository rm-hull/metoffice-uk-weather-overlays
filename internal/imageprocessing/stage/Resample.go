package stage

import (
	"image"

	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/imageprocessing"
	"golang.org/x/image/draw"
)

type ResampleStage struct{}

// Process applies a Catmull-Rom resampling to smooth the image
// This can help reduce artifacts introduced by other processing stages
// such as color replacement and blurring
func (s *ResampleStage) Process(p *imageprocessing.ProcessedImage) error {
	bounds := p.Img.Bounds()
	smoothed := image.NewNRGBA(bounds)
	draw.CatmullRom.Scale(smoothed, bounds, p.Img, bounds, draw.Over, nil)
	p.Img = smoothed
	return nil
}
