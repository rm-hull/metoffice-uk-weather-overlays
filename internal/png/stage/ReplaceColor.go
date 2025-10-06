package stage

import (
	"image"
	"image/color"
	"math"

	"github.com/rm-hull/metoffice-uk-weather-overlays/internal/png"
)

type ReplaceColorStage struct {
	Tolerance float64
	Replace   color.Color
}

// Process replaces pixels close to the specified color with transparency based on the distance to that color
// Tolerance defines how close a pixel must be to the target color to be affected
// A pixel exactly matching the target color becomes fully transparent, one at the edge of the tolerance remains opaque
func (s *ReplaceColorStage) Process(p *png.PngImage) error {
	out := image.NewNRGBA(p.Bounds)
	replaceR, replaceG, replaceB, _ := s.Replace.RGBA()
	rR, rG, rB := float64(replaceR>>8), float64(replaceG>>8), float64(replaceB>>8)
	for y := p.Bounds.Min.Y; y < p.Bounds.Max.Y; y++ {
		for x := p.Bounds.Min.X; x < p.Bounds.Max.X; x++ {
			r, g, b, a := p.Img.At(x, y).RGBA()
			R, G, B, A := float64(r>>8), float64(g>>8), float64(b>>8), float64(a>>8)
			dist := math.Sqrt((rR-R)*(rR-R) + (rG-G)*(rG-G) + (rB-B)*(rB-B))
			if dist < s.Tolerance {
				alpha := uint8((dist / s.Tolerance) * A)
				out.Set(x, y, color.NRGBA{uint8(R), uint8(G), uint8(B), alpha})
			} else {
				out.Set(x, y, color.NRGBA{uint8(R), uint8(G), uint8(B), uint8(A)})
			}
		}
	}
	p.Img = out
	return nil
}
