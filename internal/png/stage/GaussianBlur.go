package stage

import (
	"github.com/anthonynsimon/bild/blur"
	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
)

type GaussianBlurStage struct {
	Sigma float64
}

// Process applies a Gaussian blur to the image using the specified Sigma value
// Higher Sigma values result in a more pronounced blur effect
func (s *GaussianBlurStage) Process(p *png.PngImage) error {
	p.Img = blur.Gaussian(p.Img, s.Sigma)
	return nil
}
