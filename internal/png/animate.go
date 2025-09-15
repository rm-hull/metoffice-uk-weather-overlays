package png

import (
	"bytes"
	"image/png"
	"os"

	"github.com/kettek/apng"
)

func Animate(files []string, frameDelay float64) ([]byte, error) {

	a := apng.APNG{
		Frames:    make([]apng.Frame, len(files)),
		LoopCount: 0,
	}

	for i, fname := range files {
		f, err := os.Open(fname)
		if err != nil {
			return nil, err
		}

		img, err := png.Decode(f)
		if err != nil {
			return nil, err
		}

		err = f.Close()
		if err != nil {
			return nil, err
		}

		a.Frames[i] = apng.Frame{
			Image:            img,
			DelayNumerator:   uint16(frameDelay * 1000),
			DelayDenominator: 1000,
		}
	}

	var buf bytes.Buffer
	if err := apng.Encode(&buf, a); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
