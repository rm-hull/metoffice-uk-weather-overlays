package imageprocessing

import (
	"image"
	"image/png"
	"io"

	"github.com/chai2010/webp"
)

type ProcessedImage struct {
	Img image.Image
}

type PipelineStage interface {
	Process(img *ProcessedImage) error
}

func NewImageFromReader(r io.Reader) (*ProcessedImage, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	return &ProcessedImage{
		Img: img,
	}, nil
}

func (p *ProcessedImage) Write(w io.Writer) error {
	return webp.Encode(w, p.Img, &webp.Options{Quality: 80})
}

func (p *ProcessedImage) Pipeline(stages ...PipelineStage) error {
	for _, stage := range stages {
		if err := stage.Process(p); err != nil {
			return err
		}
	}
	return nil
}
