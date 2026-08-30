package main

import (
	"image"
	"image/color"
	"testing"
)

func TestSignatureNormalizesForegroundAndBackgroundColors(t *testing.T) {
	want := uint64(1)<<0 | uint64(1)<<9 | uint64(1)<<63
	makeImage := func(background, foreground color.RGBA) image.Image {
		value := image.NewRGBA(image.Rect(0, 0, 16, 16))
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				value.Set(x, y, background)
			}
		}
		for _, point := range []image.Point{{0, 0}, {2, 2}, {14, 14}} {
			for y := 0; y < 2; y++ {
				for x := 0; x < 2; x++ {
					value.Set(point.X+x, point.Y+y, foreground)
				}
			}
		}
		return value
	}
	for _, value := range []image.Image{
		makeImage(color.RGBA{A: 255}, color.RGBA{R: 255, A: 255}),
		makeImage(color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}),
	} {
		if got := signature(value, 0, 0, 2); got != want {
			t.Fatalf("signature=%016x want %016x", got, want)
		}
	}
}

func TestSignatureBreaksEqualColorCountsByFirstPixel(t *testing.T) {
	value := image.NewRGBA(image.Rect(0, 0, 8, 8))
	background := color.RGBA{G: 255, A: 255}
	foreground := color.RGBA{R: 255, A: 255}
	var want uint64
	for index := 0; index < 64; index++ {
		current := background
		if index%2 == 1 {
			current = foreground
			want |= uint64(1) << index
		}
		value.Set(index%8, index/8, current)
	}
	if got := signature(value, 0, 0, 1); got != want {
		t.Fatalf("signature=%016x want %016x", got, want)
	}
}
