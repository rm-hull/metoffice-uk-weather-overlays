package imageprocessing

import (
	"image"
	"image/png"
	"io"

	"github.com/chai2010/webp"
)

type PngImage struct {
	Img image.Image
}

type PipelineStage interface {
	Process(img *PngImage) error
}

func NewImageFromReader(r io.Reader) (*PngImage, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	return &PngImage{
		Img: img,
	}, nil
}

func (p *PngImage) Write(w io.Writer) error {
	return webp.Encode(w, p.Img, &webp.Options{Quality: 80})
}

func (p *PngImage) Pipeline(stages ...PipelineStage) error {
	for _, stage := range stages {
		if err := stage.Process(p); err != nil {
			return err
		}
	}
	return nil
}
