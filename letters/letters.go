// letters.go
package main

import (
	"fmt"
	//	"log"

	"github.com/donomii/glim"
	//"github.com/donomii/goof"
)

func DumpBuff(buff []uint8, width, height uint) string {
	out := ""
	//log.Printf("Dumping buffer with width, height %v,%v\n", width, height)
	for y := uint(0); y < height; y = y + 2 {

		for x := uint(0); x < width; x++ {
			i := (x + y*width) * 4

			if buff[i+3] > 128 {
				out = out + "*"
			} else {
				out = out + " "
			}

		}
		out = out + "\n"
	}
	return out
}

func main() {
	for i := 0; i < 256; i++ {
		img, _ := glim.DrawStringRGBA(15, glim.RGBA{255, 255, 255, 255}, string(i), "f1.ttf")
		XmaX, YmaX := img.Bounds().Max.X, img.Bounds().Max.Y
		bts, X, Y := glim.GFormatToImage(img, nil, XmaX, YmaX)
		//log.Println(bts)
		letter := DumpBuff(bts, uint(X)/2, uint(Y))

		fmt.Println(output_c(i, letter))

	}
	fmt.Println(letterlookup_c())
}

func output_c(index int, letter string) string {
	out := fmt.Sprintf("char * letter_%v() {\n", index)
	out = out + fmt.Sprintf("return(\"%v\");\n", letter)
	out = out + "}"
	return out
}

func letterlookup_c() string {
	out := "char * letterlookup(int letter) {\n"
	for i := 0; i < 256; i++ {
		out = out + fmt.Sprintf("	if(letter==%v) { return(letter_%v();}\n", i, i)
	}
	out = out + "	return(\"whoops\");\n}"
	return out
}
