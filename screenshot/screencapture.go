package screenshot

import (
	"image"
	"unsafe"

	"github.com/kbinani/screenshot"
	"github.com/nfnt/resize"
)

import "C"

// GetScreenshot return rgba format
func GetScreenshot(cx, cy, cw, ch, rw, rh int) *image.RGBA {
	bounds := image.Rectangle{
		Min: image.Point{
			X: cx,
			Y: cy,
		},
		Max: image.Point{
			X: cx + cw,
			Y: cy + ch,
		},
	}
	img, err := screenshot.CaptureRect(bounds)

	if err != nil {
		panic(err)
	}
	img = resize.Resize(uint(rw), uint(rh), img, resize.Lanczos3).(*image.RGBA)
	return img
}

// GetScreenSize return screen size width and height
func GetScreenSize() (int, int) {
	bounds := screenshot.GetDisplayBounds(0)
	return bounds.Max.X, bounds.Max.Y
}

// RgbaToYuv convert to yuv from rgba
func RgbaToYuv(rgba *image.RGBA) []byte {
	w := rgba.Rect.Max.X
	h := rgba.Rect.Max.Y
	size := int(float32(w*h) * 1.5)
	stride := rgba.Stride - w*4
	yuv := make([]byte, size, size)
	C.rgba2yuv(unsafe.Pointer(&yuv[0]), unsafe.Pointer(&rgba.Pix[0]), C.int(w), C.int(h), C.int(stride))
	return yuv
}
