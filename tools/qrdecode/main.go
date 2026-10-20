package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func main() {
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		panic(err)
	}
	res, err := qrcode.NewQRCodeReader().DecodeWithoutHints(bmp)
	if err != nil {
		fmt.Println("DECODE_GAGAL:", err)
		return
	}
	fmt.Println("DECODE_OK")
	fmt.Println("ISI:", res.GetText())
}
