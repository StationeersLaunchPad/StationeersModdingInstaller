package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
	"image/png"
	_ "image/png"
	"os"
)

func main() {
	f, err := os.Open("assets/iconpng.png")
	if err != nil {
		panic(err)
	}
	src, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		panic(err)
	}

	sizes := []int{256, 128, 64, 48, 32, 16}

	type entry struct {
		data []byte
		w, h int
	}
	var entries []entry

	for _, sz := range sizes {
		scaled := resizeNN(src, sz)
		var buf bytes.Buffer
		if err := png.Encode(&buf, scaled); err != nil {
			panic(err)
		}
		entries = append(entries, entry{buf.Bytes(), sz, sz})
	}

	out, err := os.Create("build/windows/icon.ico")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	// ICONDIR header
	binary.Write(out, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(out, binary.LittleEndian, uint16(1)) // type: 1 = ICO
	binary.Write(out, binary.LittleEndian, uint16(len(entries)))

	offset := uint32(6 + 16*len(entries))
	for _, e := range entries {
		w := e.w
		if w >= 256 {
			w = 0
		}
		h := e.h
		if h >= 256 {
			h = 0
		}
		out.Write([]byte{byte(w), byte(h), 0, 0})
		binary.Write(out, binary.LittleEndian, uint16(1))  // color planes
		binary.Write(out, binary.LittleEndian, uint16(32)) // bits per pixel
		binary.Write(out, binary.LittleEndian, uint32(len(e.data)))
		binary.Write(out, binary.LittleEndian, offset)
		offset += uint32(len(e.data))
	}
	for _, e := range entries {
		out.Write(e.data)
	}
}

func resizeNN(src image.Image, size int) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sx := b.Min.X + x*(b.Max.X-b.Min.X)/size
			sy := b.Min.Y + y*(b.Max.Y-b.Min.Y)/size
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
