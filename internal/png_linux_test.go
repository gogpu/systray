//go:build linux

package internal

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// makePNG creates a PNG image in memory with the given pixels.
// Each pixel is specified as an NRGBA color.
func makePNG(t *testing.T, w, h int, pixels []color.NRGBA) []byte {
	t.Helper()

	if len(pixels) != w*h {
		t.Fatalf("makePNG: pixel count %d does not match %dx%d=%d", len(pixels), w, h, w*h)
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, pixels[y*w+x])
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}

	return buf.Bytes()
}

func TestPngToARGB_SingleRedPixel(t *testing.T) {
	t.Parallel()

	pngData := makePNG(t, 1, 1, []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 255},
	})

	w, h, argb, err := pngToARGB(pngData)
	if err != nil {
		t.Fatalf("pngToARGB returned error: %v", err)
	}

	if w != 1 || h != 1 {
		t.Fatalf("dimensions = %dx%d, want 1x1", w, h)
	}

	if len(argb) != 4 {
		t.Fatalf("argb length = %d, want 4", len(argb))
	}

	// ARGB big-endian: A=255, R=255, G=0, B=0
	pixel := binary.BigEndian.Uint32(argb)
	wantPixel := uint32(0xFF)<<24 | uint32(0xFF)<<16 | uint32(0x00)<<8 | uint32(0x00)
	if pixel != wantPixel {
		t.Errorf("pixel = 0x%08X, want 0x%08X", pixel, wantPixel)
	}
}

func TestPngToARGB_SingleGreenPixel(t *testing.T) {
	t.Parallel()

	pngData := makePNG(t, 1, 1, []color.NRGBA{
		{R: 0, G: 255, B: 0, A: 255},
	})

	_, _, argb, err := pngToARGB(pngData)
	if err != nil {
		t.Fatalf("pngToARGB returned error: %v", err)
	}

	// ARGB big-endian: A=255, R=0, G=255, B=0
	pixel := binary.BigEndian.Uint32(argb)
	wantPixel := uint32(0xFF)<<24 | uint32(0x00)<<16 | uint32(0xFF)<<8 | uint32(0x00)
	if pixel != wantPixel {
		t.Errorf("pixel = 0x%08X, want 0x%08X", pixel, wantPixel)
	}
}

func TestPngToARGB_SingleBluePixel(t *testing.T) {
	t.Parallel()

	pngData := makePNG(t, 1, 1, []color.NRGBA{
		{R: 0, G: 0, B: 255, A: 255},
	})

	_, _, argb, err := pngToARGB(pngData)
	if err != nil {
		t.Fatalf("pngToARGB returned error: %v", err)
	}

	// ARGB big-endian: A=255, R=0, G=0, B=255
	pixel := binary.BigEndian.Uint32(argb)
	wantPixel := uint32(0xFF)<<24 | uint32(0x00)<<16 | uint32(0x00)<<8 | uint32(0xFF)
	if pixel != wantPixel {
		t.Errorf("pixel = 0x%08X, want 0x%08X", pixel, wantPixel)
	}
}

func TestPngToARGB_PixelWithAlpha(t *testing.T) {
	t.Parallel()

	// NRGBA with 50% alpha (128).
	// image.NRGBA stores non-premultiplied. image.At().RGBA() returns premultiplied 16-bit.
	// R=200, A=128 → premul R = 200*128/255 ≈ 100 at 8-bit, but RGBA() returns 16-bit:
	// r16 = 200*128*0x101 = ... / 0xFF, then >>8 gives 8-bit.
	// Let's verify empirically by encoding and decoding.
	pngData := makePNG(t, 1, 1, []color.NRGBA{
		{R: 200, G: 100, B: 50, A: 128},
	})

	w, h, argb, err := pngToARGB(pngData)
	if err != nil {
		t.Fatalf("pngToARGB returned error: %v", err)
	}

	if w != 1 || h != 1 {
		t.Fatalf("dimensions = %dx%d, want 1x1", w, h)
	}

	// Decode with Go's image package to get expected premultiplied values.
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		t.Fatalf("image.Decode failed: %v", err)
	}
	r, g, b, a := img.At(0, 0).RGBA()
	wantA := uint8(a >> 8)
	wantR := uint8(r >> 8)
	wantG := uint8(g >> 8)
	wantB := uint8(b >> 8)

	gotA := argb[0]
	gotR := argb[1]
	gotG := argb[2]
	gotB := argb[3]

	if gotA != wantA {
		t.Errorf("alpha = %d, want %d", gotA, wantA)
	}
	if gotR != wantR {
		t.Errorf("red = %d, want %d", gotR, wantR)
	}
	if gotG != wantG {
		t.Errorf("green = %d, want %d", gotG, wantG)
	}
	if gotB != wantB {
		t.Errorf("blue = %d, want %d", gotB, wantB)
	}
}

func TestPngToARGB_TransparentPixel(t *testing.T) {
	t.Parallel()

	pngData := makePNG(t, 1, 1, []color.NRGBA{
		{R: 255, G: 255, B: 255, A: 0},
	})

	_, _, argb, err := pngToARGB(pngData)
	if err != nil {
		t.Fatalf("pngToARGB returned error: %v", err)
	}

	// Fully transparent: ARGB should be all zeros (premultiplied).
	pixel := binary.BigEndian.Uint32(argb)
	if pixel != 0x00000000 {
		t.Errorf("transparent pixel = 0x%08X, want 0x00000000", pixel)
	}
}

func TestPngToARGB_2x2Image(t *testing.T) {
	t.Parallel()

	pixels := []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 255},     // top-left: red
		{R: 0, G: 255, B: 0, A: 255},     // top-right: green
		{R: 0, G: 0, B: 255, A: 255},     // bottom-left: blue
		{R: 255, G: 255, B: 255, A: 255}, // bottom-right: white
	}
	pngData := makePNG(t, 2, 2, pixels)

	w, h, argb, err := pngToARGB(pngData)
	if err != nil {
		t.Fatalf("pngToARGB returned error: %v", err)
	}

	if w != 2 || h != 2 {
		t.Fatalf("dimensions = %dx%d, want 2x2", w, h)
	}

	expectedLen := 2 * 2 * 4
	if len(argb) != expectedLen {
		t.Fatalf("argb length = %d, want %d", len(argb), expectedLen)
	}

	// Verify each pixel (ARGB big-endian order, 4 bytes per pixel).
	type pixelCheck struct {
		offset     int
		name       string
		a, r, g, b uint8
	}
	checks := []pixelCheck{
		{0, "top-left (red)", 255, 255, 0, 0},
		{4, "top-right (green)", 255, 0, 255, 0},
		{8, "bottom-left (blue)", 255, 0, 0, 255},
		{12, "bottom-right (white)", 255, 255, 255, 255},
	}
	for _, c := range checks {
		gotA := argb[c.offset]
		gotR := argb[c.offset+1]
		gotG := argb[c.offset+2]
		gotB := argb[c.offset+3]
		if gotA != c.a || gotR != c.r || gotG != c.g || gotB != c.b {
			t.Errorf("%s: ARGB = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
				c.name, gotA, gotR, gotG, gotB, c.a, c.r, c.g, c.b)
		}
	}
}

func TestPngToARGB_InvalidData(t *testing.T) {
	t.Parallel()

	_, _, _, err := pngToARGB([]byte{0x00, 0x01, 0x02, 0x03})
	if err == nil {
		t.Error("expected error for invalid PNG data, got nil")
	}
}

func TestPngToARGB_EmptyData(t *testing.T) {
	t.Parallel()

	_, _, _, err := pngToARGB([]byte{})
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

func TestPngToARGB_NilData(t *testing.T) {
	t.Parallel()

	_, _, _, err := pngToARGB(nil)
	if err == nil {
		t.Error("expected error for nil data, got nil")
	}
}

func TestPngToARGB_Dimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		w, h int
	}{
		{"1x1", 1, 1},
		{"16x16", 16, 16},
		{"32x32", 32, 32},
		{"1x10", 1, 10},
		{"10x1", 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pixels := make([]color.NRGBA, tt.w*tt.h)
			for i := range pixels {
				pixels[i] = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
			}

			pngData := makePNG(t, tt.w, tt.h, pixels)
			w, h, argb, err := pngToARGB(pngData)
			if err != nil {
				t.Fatalf("pngToARGB returned error: %v", err)
			}

			if w != tt.w || h != tt.h {
				t.Errorf("dimensions = %dx%d, want %dx%d", w, h, tt.w, tt.h)
			}

			expectedLen := tt.w * tt.h * 4
			if len(argb) != expectedLen {
				t.Errorf("argb length = %d, want %d", len(argb), expectedLen)
			}
		})
	}
}
