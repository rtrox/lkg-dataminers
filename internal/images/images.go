package images

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.org/x/image/draw"
)

func ReadImage(filename string) (image.Image, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	img, _, err := image.Decode(fd)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func ScaleImage(img image.Image, scale draw.Scaler, scale_factor int) image.Image {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	rect := image.Rect(0, 0, width*scale_factor, height*scale_factor)
	dst := image.NewNRGBA(rect)
	scale.Scale(dst, rect, img, img.Bounds(), draw.Over, nil)
	return dst
}

func CropImage(img image.Image, crop image.Rectangle) (image.Image, error) {
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	// img is an Image interface. This checks if the underlying value has a
	// method called SubImage. If it does, then we can use SubImage to crop the
	// image.
	simg, ok := img.(subImager)
	if !ok {
		return nil, fmt.Errorf("image does not support cropping")
	}

	return simg.SubImage(crop), nil
}

func WriteImage(img image.Image, name string) error {
	fd, err := os.Create(name)
	if err != nil {
		return err
	}
	defer fd.Close()

	return png.Encode(fd, img)
}
