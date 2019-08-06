// GL Image library.  Routines for handling images and textures in GO OpenGL (especially with the GoMobile framework)
package glim

import "math"
import (
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"unicode/utf8"

	"fmt"
	"log"
	"os"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"image"
	"image/color"
	"image/png"
	"math/rand"
)

type Thunk func()

type RGBA []uint8

var renderCache map[string]*image.RGBA
var faceCache map[string]*font.Face
var fontCache map[string]*truetype.Font

//Make a random picture to display.  Handy for debugging
func RandPic(width, height int) []uint8 {
	pic := make([]uint8, width*height*4)
	for i, _ := range pic {
		pic[i] = uint8(rand.Intn(256))
	}
	return pic
}

//Dump the rendercache, facecache and fontcache
func ClearAllCaches() {
	renderCache = map[string]*image.RGBA{}
	faceCache = map[string]*font.Face{}
	fontCache = map[string]*truetype.Font{}
}

func ToChar(i int) rune {
	return rune(i)
}

//Load a image from disk, return a byte array, width, height
func LoadImage(path string) ([]byte, int, int) {
	infile, _ := os.Open(path)
	defer infile.Close()

	src, _, _ := image.Decode(infile)
	rect := src.Bounds()
	rgba := image.NewNRGBA(rect)
	draw.Draw(rgba, rect, src, rect.Min, draw.Src)
	return rgba.Pix, rect.Max.X, rect.Max.Y
}

// Abs64 returns the absolute value of x.
func Abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// Calculate the pixel difference between two images.
//
// This does a simple pixel compare, by sutracting the RGB pixel values.  Itdoes not take into account perceptual differences or gamma or anything clever like that
//
// The higher the returned number, the more different the pictures are
func CalcDiff(renderPix, refImage []byte, width, height int) (int64, []byte) {
	diffbuff := make([]byte, len(refImage))
	diff := int64(0)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for z := 0; z < 3; z++ {
				i := (x+y*width)*4 + z
				d := Abs64(int64(renderPix[i]) - int64(refImage[i]))
				diff = diff + d
				diffbuff[i] = byte(d)
			}
		}
	}
	return diff, diffbuff
}

//Copies an image to a correctly-packed texture data array, where "correctly packed" means a byte array suitable for loading into OpenGL as a 32-bit RGBA byte blob
//
//Other formats are not currently supported, patches welcome, etc
//
//Returns the array, modified in place.  If u8Pix is nil or texWidth is 0, it creates a new texture array and returns that.  Texture is assumed to be square (this used to be required for OpenGL, not sure now?).
func PaintTexture(img image.Image, u8Pix []uint8, clientWidth int) []uint8 {
	out, _, _ := GFormatToImage(img, u8Pix, 0, 0)
	return out
}
func GFormatToImage(img image.Image, u8Pix []uint8, clientWidth, clientHeight int) ([]uint8, int, int) {
	bounds := img.Bounds()
	newW := bounds.Max.X
	newH := bounds.Max.Y

	if int(clientWidth) == 0 {
		clientWidth = newW
	}
	if int(clientHeight) == 0 {
		clientHeight = newH
	}
	if (int(newW) > clientWidth) || (int(newH) > clientHeight) {
		panic(fmt.Sprintf("ClientWidth (%v) is not large enough for image of width(%v) and height(%v)", clientWidth, newW, newH))
	}
	if u8Pix == nil {
		dim := clientWidth*clientHeight*4 + 4
		u8Pix = make([]uint8, dim, dim)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// A color's RGBA method returns values in the range [0, 65535].
			start := int(y)*clientWidth*4 + int(x)*4
			u8Pix[start] = uint8(r * 255 / 65535)
			u8Pix[start+1] = uint8(g * 255 / 65535)
			u8Pix[start+2] = uint8(b * 255 / 65535)
			u8Pix[start+3] = uint8(a * 255 / 65535)
		}
	}
	return u8Pix, clientWidth, clientHeight
}

//Abs difference of two uint32 numbers
func udiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	} else {
		return b - a
	}
}

//Returns a number representing the graphical difference between two images.
//
//This difference is calculated by comparing each pixel and summing the difference in colour
//
//This difference function is used in some other programs to power some approximation functions, it doesn't actually mean anything
func GDiff(m, m1 image.Image) int64 {
	bounds := m.Bounds()

	diff := int64(0)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := m.At(x, y).RGBA()
			r1, g1, b1, _ := m1.At(x, y).RGBA()
			diff = diff + int64(udiff(r>>8, r1>>8)+udiff(g>>8, g1>>8)+udiff(b>>8, b1>>8))
		}
	}
	return diff
}

// Abs32 returns the absolute value of int32 x.
func Abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// Abs8 returns the absolute value of int8 x.
func Abs8(x int8) int8 {
	if x < 0 {
		return -x
	}
	return x
}

// AbsInt returns the absolute value of int x.
func AbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

//Dumps a go image format thing to disk
func SaveImage(picture *image.RGBA, filename string) {
	f, _ := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	defer f.Close()
	png.Encode(f, picture)
}

//Converts an image to google's image format, RGBA type
func ImageToGFormat(texWidth, texHeight int, buff []byte) image.Image {
	m := image.NewNRGBA(image.Rectangle{image.Point{0, 0}, image.Point{int(texWidth), int(texHeight)}})
	if buff != nil {
		for y := 0; y < texHeight; y++ {
			for x := 0; x < texWidth; x++ {
				i := (x + y*texWidth) * 4
				m.Set(int(x), int(texHeight-y), color.NRGBA{uint8(buff[i]), uint8(buff[i+1]), uint8(buff[i+2]), uint8(buff[i+3])})
			}
		}
	}
	return m
}

//Converts an image to google's image format, NRGBA format
func ImageToGFormatRGBA(texWidth, texHeight int, buff []byte) *image.RGBA {
	m := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{int(texWidth), int(texHeight)}})
	if buff != nil {
		for y := 0; y < texHeight; y++ {
			for x := 0; x < texWidth; x++ {
				i := (x + y*texWidth) * 4
				m.Set(int(x), int(texHeight-y), color.NRGBA{uint8(buff[i]), uint8(buff[i+1]), uint8(buff[i+2]), uint8(buff[i+3])})
			}
		}
	}
	return m
}

//Saves a 32 bit RGBA format byte array to a PNG file
//
// i.e. 1 byte for each R,B,G,A
func SaveBuff(texWidth, texHeight int, buff []byte, filename string) {
	m := image.NewNRGBA(image.Rectangle{image.Point{0, 0}, image.Point{int(texWidth), int(texHeight)}})
	if buff != nil {
		//log.Printf("Saving buffer: %v,%v", texWidth, texHeight)
		for y := 0; y < texHeight; y++ {
			for x := 0; x < texWidth; x++ {
				i := (x + y*texWidth) * 4
				m.Set(int(x), int(texHeight-y), color.NRGBA{uint8(buff[i]), uint8(buff[i+1]), uint8(buff[i+2]), uint8(buff[i+3])})
				//if buff[i]>0 { fmt.Printf("Found colour\n") }
				//if buff[i+1]>0 { fmt.Printf("Found colour\n") }
				//if buff[i+2]>0 { fmt.Printf("Found colour\n") }
			}
		}
	}
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		panic(fmt.Sprintf("Could not save buffer to %v : %v", filename, err))
	}
	defer f.Close()
	png.Encode(f, m)
}

var fname_int int

//Prints the contents of a 32bit RGBA byte array as ASCII text
//
//For debugging.  Any pixel with a red value over 128 will be drawn as "I", otherwise "_"
func DumpBuff(buff []uint8, width, height uint) {
	log.Printf("Dumping buffer with width, height %v,%v\n", width, height)
	for y := uint(0); y < height; y++ {
		for x := uint(0); x < width; x++ {
			i := (x + y*width) * 4
			//log.Printf("Index: %v\n", i)
			if buff[i] > 128 {
				fmt.Printf("I")
			} else {
				fmt.Printf("_")
			}
		}
		fmt.Println("")
	}
}

//Convert an array of uint8 to byte, because somehow golang manages to pack them differently in memeory
func Uint8ToBytes(in []uint8) []byte {
	out := make([]byte, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

//Convert to go's image library color
func RGBAtoColor(in RGBA) color.RGBA {
	out := color.RGBA{in[0], in[1], in[2], in[3]}
	return out
}

//Creates a texture and draws a string to it
//
//FIXME some fonts might not compeletely fit in the texture (usually the decorative ones which extend into another letter)
func DrawStringRGBA(txtSize float64, fontColor RGBA, txt, fontfile string) (*image.RGBA, *font.Face) {
	//log.Printf("Drawing text (%v), colour (%v), size(%v)\n", txt, fontColor, txtSize)
	cacheKey := fmt.Sprintf("%v,%v,%v", txtSize, fontColor, txt)
	if renderCache == nil {
		renderCache = map[string]*image.RGBA{}
	}
	if faceCache == nil {
		faceCache = map[string]*font.Face{}
	}
	im, ok := renderCache[cacheKey]
	face, ok1 := faceCache[cacheKey]
	if ok && ok1 {
		return im, face
	}
	txtFont := LoadFont(fontfile)
	d := &font.Drawer{
		Src: image.NewUniform(RGBAtoColor(fontColor)), // 字体颜色
		Face: truetype.NewFace(txtFont, &truetype.Options{
			Size:    txtSize,
			DPI:     96,
			Hinting: font.HintingFull,
		}),
	}
	fface := d.Face
	glyph, _ := utf8.DecodeRuneInString(txt)
	fuckedRect, _, _ := fface.GlyphBounds(glyph)
	// letterWidth := fixed2int(fuckedRect.Max.X)
	Xadj := Fixed2int(fuckedRect.Min.X)
	if Xadj < 0 {
		Xadj = Xadj * -1
	}
	// fuckedRect, _, _ = fface.GlyphBounds(glyph)
	// letterHeight := fixed2int(fuckedRect.Max.Y)
	//ascend := fuckedRect.min.Y
	//]
	targetWidth := d.MeasureString(txt).Ceil() * 2
	targetHeight := int(txtSize) * 3
	rect := image.Rect(0, 0, targetWidth, targetHeight)
	//rect := image.Rect(0, 0, 30, 30)
	rgba := image.NewRGBA(rect)
	d.Dst = rgba

	d.Dot = fixed.Point26_6{
		X: fixed.I(Xadj),
		Y: fixed.I(targetHeight * 1 / 3), //fixed.I(rect.Max.Y/3), //rect.Max.Y*2/3), //FIXME
	}
	d.DrawString(txt)
	renderCache[cacheKey] = rgba
	faceCache[cacheKey] = &d.Face
	//imgBytes := rgba.Pix
	//for i,v := range imgBytes {
	//imgBytes[i] = 255 - v
	//}

	return rgba, &d.Face
}

func Fixed2int(n fixed.Int26_6) int {
	return n.Round()
}

type Vec2 struct {
	X, Y int
}

//Get the maximum pixel size needed to hold a string
func GetGlyphSize(size float64, str string) (int, int) {
	_, str_size := utf8.DecodeRuneInString(str)
	img, _ := DrawStringRGBA(size, RGBA{1.0, 1.0, 1.0, 1.0}, str[0:str_size], "f1.ttf")
	XmaX, YmaX := img.Bounds().Max.X, img.Bounds().Max.Y
	if XmaX > 4000 {
		panic("X can't be that big")
	}
	return XmaX, YmaX
}

//PasteBytes
//
// Takes a bag of bytes, and some dimensions, and pastes it into another bag of bytes
// It's the basic image combining routine
func PasteBytes(srcWidth, srcHeight int, srcBytes []byte, xpos, ypos, dstWidth, dstHeight int, u8Pix []uint8, transparent, showBorder bool, copyAlpha bool) {
	//log.Printf("Copying source image (%v,%v) into destination image (%v,%v) at point (%v, %v)\n", srcWidth, srcHeight, dstWidth, dstHeight, xpos, ypos)
	bpp := 4 //bytes per pixel

	for i := 0; i < srcHeight; i++ {
		if transparent {
			for j := 0; j < srcWidth; j++ {
				srcOff := i*srcWidth*4 + j*4
				dstOff := (ypos+i)*dstWidth*bpp + xpos*bpp + j*bpp

				r := srcBytes[srcOff+0]
				g := srcBytes[srcOff+1]
				b := srcBytes[srcOff+2]

				dstR := u8Pix[srcOff+0]
				dstG := u8Pix[srcOff+1]
				dstB := u8Pix[srcOff+2]

				srcA := float64(srcBytes[srcOff+3]) / 255.0
				dstA := 0.0 //float64(u8Pix[dstOff+3])/255.0

				outA := srcA + dstA*(1-srcA)

				outR := byte(0)
				outG := byte(0)
				outB := byte(0)
				outR = byte((float64(r)*srcA + float64(dstR)*dstA*(1-srcA)) / outA)
				outG = byte((float64(g)*srcA + float64(dstG)*dstA*(1-srcA)) / outA)
				outB = byte((float64(b)*srcA + float64(dstB)*dstA*(1-srcA)) / outA)
				//if srcBytes[i*srcWidth*4+j*4] > u8Pix[(ypos+i)*dstWidth*bpp+xpos*bpp+j*bpp] {
				////log2Buff(fmt.Sprintf("Source: (%v,%v), destination: (%v,%v)\n", j,i,xpos+j, ypos+i))
				//copy(u8Pix[dstOff:dstOff+4], srcBytes[srcOff:srcOff+4])
				//}
				u8Pix[dstOff+0] = outR
				u8Pix[dstOff+1] = outG
				u8Pix[dstOff+2] = outB
				if copyAlpha { //Needed because the default alpha is 0, which causes multiple pastes to fully overwrite the previous pastes
					if srcBytes[srcOff+3] > u8Pix[dstOff+3] {
						u8Pix[dstOff+3] = srcBytes[srcOff+3]
					}
				}
				//copy(u8Pix[dstOff:dstOff+4], srcBytes[srcOff:srcOff+4])
				if showBorder {
					if i == 0 || j == 0 || i == srcHeight-1 || j == srcWidth-1 {
						u8Pix[dstOff+0] = 255
						u8Pix[dstOff+3] = 255

					}
				}
			}
		} else {
			srcOff := i * srcWidth * 4
			dstOff := (ypos+i)*dstWidth*bpp + xpos*bpp
			copy(u8Pix[dstOff:dstOff+4*srcWidth], srcBytes[srcOff:srcOff+4*srcWidth]) //FIXME move this outside the line loop so we can copy entire lines in one call
		}
	}
}

//Pastes a go format image into a bag of bytes image
func PasteImg(img *image.RGBA, xpos, ypos, clientWidth, clientHeight int, u8Pix []uint8, transparent bool) {
	po2 := int(MaxI(NextPo2(img.Bounds().Max.X), NextPo2(img.Bounds().Max.Y)))
	//log.Printf("Chose texture size: %v\n", po2)
	wordBuff := PaintTexture(img, nil, po2)
	bpp := int(4) //bytes per pixel

	h := img.Bounds().Max.Y
	w := img.Bounds().Max.X
	for i := int(0); i < int(h); i++ {
		for j := int(0); j < w; j++ {
			if (wordBuff[i*po2*4+j*4] > 128) || !transparent {
				u8Pix[(ypos+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp] = wordBuff[i*po2*4+j*4]
				u8Pix[(int(ypos)+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp+1] = wordBuff[i*po2*4+j*4+1]
				u8Pix[(int(ypos)+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp+2] = wordBuff[i*po2*4+j*4+2]
				u8Pix[(int(ypos)+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp+3] = wordBuff[i*po2*4+j*4+3]
			}
		}
	}
}

//Write some text into a bag of bytes image.
func PasteText(tSize float64, xpos, ypos, clientWidth, clientHeight int, text string, u8Pix []uint8, transparent bool) {
	img, _ := DrawStringRGBA(tSize, RGBA{255, 255, 255, 255}, text, "f1.ttf")
	po2 := int(MaxI(NextPo2(img.Bounds().Max.X), NextPo2(img.Bounds().Max.Y)))
	//log.Printf("Chose texture size: %v\n", po2)
	wordBuff := PaintTexture(img, nil, po2)
	bpp := int(4) //bytes per pixel

	h := img.Bounds().Max.Y
	w := int(img.Bounds().Max.X)
	for i := int(0); i < int(h); i++ {
		for j := int(0); j < w; j++ {
			if (wordBuff[i*po2*4+j*4] > 128) || !transparent {
				u8Pix[(int(ypos)+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp] = wordBuff[i*po2*4+j*4]
				u8Pix[(int(ypos)+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp+1] = wordBuff[i*po2*4+j*4+1]
				u8Pix[(int(ypos)+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp+2] = wordBuff[i*po2*4+j*4+2]
				u8Pix[(int(ypos)+i)*clientWidth*bpp+int(xpos)*bpp+j*bpp+3] = wordBuff[i*po2*4+j*4+3]
			}
		}
	}
}

func NextPo2(n int) int {
	return int(math.Pow(2, math.Ceil(math.Log2(float64(n)))))
}

//Rotate a 32bit byte array into a new byte array.  The target array will be created with the correct dimensions
func Rotate90(srcW, srcH int, src []byte) []byte {
	//log.Printf("Rotating image (%v,%v)\n",srcW, srcH)
	dstW := srcH
	dstH := srcW
	dst := make([]byte, dstW*dstH*4)

	for dstY := 0; dstY < dstH; dstY++ {
		for dstX := 0; dstX < dstW; dstX++ {
			srcX := dstY
			//srcY := dstW - dstX - 1
			srcY := dstX

			srcOff := srcY*srcW*4 + srcX*4
			dstOff := dstY*dstW*4 + dstX*4

			copy(dst[dstOff:dstOff+4], src[srcOff:srcOff+4])
		}
	}

	return dst
}

//Rotate a 32bit byte array into a new byte array.  The target array will be created with the correct dimensions
func Rotate270(srcW, srcH int, src []byte) []byte {
	//log.Printf("Rotating image (%v,%v)\n",srcW, srcH)
	dstW := srcH
	dstH := srcW
	dst := make([]byte, dstW*dstH*4)

	for dstY := 0; dstY < dstH; dstY++ {
		for dstX := 0; dstX < dstW; dstX++ {
			//log.Printf("dstX: %v, dstY: %v\n", dstX, dstY)
			//srcX := dstH - dstY  -1
			srcX := dstY
			srcY := dstW - dstX - 1
			//srcY := dstX

			srcOff := srcY*srcW*4 + srcX*4
			dstOff := dstY*dstW*4 + dstX*4

			copy(dst[dstOff:dstOff+4], src[srcOff:srcOff+4])
		}
	}

	return dst
}

//Flips a 32bit byte array picture upside down.  Creates a target array with with the correct dimensions and returns it
func FlipUp(srcW, srcH int, src []byte) []byte {
	//log.Printf("Rotating image (%v,%v)\n",srcW, srcH)
	dstW := srcW
	dstH := srcH
	dst := make([]byte, dstW*dstH*4)

	for dstY := 0; dstY < dstH; dstY++ {
		for dstX := 0; dstX < dstW; dstX++ {
			srcX := dstX
			srcY := dstH - dstY - 1
			//srcY := dstX

			srcOff := srcY*srcW*4 + srcX*4
			dstOff := dstY*dstW*4 + dstX*4

			copy(dst[dstOff:dstOff+4], src[srcOff:srcOff+4])
		}
	}

	return dst
}

//Turn all pixels of a colour into transparent pixels
//
//i.e. set the alpha to zero if the RGB matches the colour
//
//The alpha value of the input colour is ignored
func MakeTransparent(m []byte, col color.RGBA) []byte {
	for i := 0; i < len(m); i = i + 4 {
		if m[i] == col.R ||
			m[i+1] == col.B ||
			m[i+2] == col.G {
			m[i+3] = 0
		}
	}
	return m
}
