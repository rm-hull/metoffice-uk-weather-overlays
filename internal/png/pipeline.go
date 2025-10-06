package png

import (
	"image"
	"image/png"
	"io"
)

type PngImage struct {
	Img    image.Image
	Bounds image.Rectangle
}

type PipelineStage interface {
	Process(img *PngImage) error
}

func NewPngFromReader(r io.Reader) (*PngImage, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}
	return &PngImage{
		Img:    img,
		Bounds: img.Bounds(),
	}, nil
}

func (p *PngImage) Write(w io.Writer) error {
	return png.Encode(w, p.Img)
}

func (p *PngImage) Pipeline(stages ...PipelineStage) error {
	for _, stage := range stages {
		if err := stage.Process(p); err != nil {
			return err
		}
	}
	return nil
}
