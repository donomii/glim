// GL Image library.  Routines for handling images and textures in GO OpenGL (especially with the GoMobile framework)
package glim

import "math"
import (
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"unicode"
	"unicode/utf8"

	"fmt"
	"log"
	"os"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"golang.org/x/mobile/gl"

	"image"
	"image/color"
	"image/png"
)

type Thunk func()

//Return screen (or main window) size
func ScreenSize(glctx gl.Context) (int, int) {
	outbuff := []int32{0, 0, 0, 0}
	glctx.GetIntegerv(outbuff, gl.VIEWPORT)
	screenWidth := int(outbuff[2])
	screenHeight := int(outbuff[3])
	return screenWidth, screenHeight
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

//Save the currently display to a file on disk
func ScreenShot(glctx gl.Context, filename string) {
	screenWidth, screenHeight := ScreenSize(glctx)
	//log.Printf("Saving width: %v, height: %v\n", screenWidth, screenHeight)
	SaveBuff(screenWidth, screenHeight, CopyScreen(glctx, int(screenWidth), int(screenHeight)), filename)
}

//Copies an image to a correctly-packed texture data array, where "correctly packed" means a byte array suitable for loading into OpenGL as a 32-bit RGBA byte blob
//
//Returns the array, modified in place.  If u8Pix is nil or texWidth is 0, it creates a new texture array and returns that.  Texture is assumed to be square.
func PaintTexture(img image.Image, u8Pix []uint8, clientWidth int) []uint8 {
	bounds := img.Bounds()
	newW := bounds.Max.X
	newH := bounds.Max.Y

	//if uint(newW) != clientWidth || uint(newH) != clientWidth {
	if (int(newW) > clientWidth) || (int(newH) > clientWidth) {
		panic(fmt.Sprintf("ClientWidth (%v) does not match image width(%v) and height(%v)", clientWidth, newW, newH))
	}
	if u8Pix == nil {
		dim := clientWidth*clientWidth*4 + 4
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
	return u8Pix
}

//The core of ScreenShot, it copies the display into a RGBA byte array
func CopyScreen(glctx gl.Context, clientWidth, clientHeight int) []byte {
	buff := make([]byte, clientWidth*clientHeight*4, clientWidth*clientHeight*4)
	if clientWidth == 0 || clientHeight == 0 {
		return buff
	}
	//fmt.Printf("reading pixels: %v, %v\n", clientWidth, clientHeight)
	//glctx.BindFramebuffer(gl.FRAMEBUFFER, rtt_frameBuff)
	glctx.ReadPixels(buff, 0, 0, clientWidth, clientHeight, gl.RGBA, gl.UNSIGNED_BYTE)
	//glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	return buff
}

//Copy the render buffer to golangs's horrible image format.  Use CopyScreen instead.
func CopyScreenToGFormat(glctx gl.Context, clientWidth, clientHeight int) image.Image {
	buff := CopyScreen(glctx, clientWidth, clientHeight)
	rect := image.Rectangle{image.Point{0, clientWidth}, image.Point{0, clientHeight}}
	rgba := image.NewNRGBA(rect)
	if clientWidth == 0 || clientHeight == 0 {
		return rgba
	}
	rgba.Pix = buff
	return rgba
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
func GDiff(m, m1 image.Image) int64 {
	bounds := m.Bounds()

	diff := int64(0)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := m.At(x, y).RGBA()
			r1, g1, b1, _ := m1.At(x, y).RGBA()
			diff = diff + int64(Abs32(udiff(r>>8, r1>>8))+Abs32(udiff(g>>8, g1>>8))+Abs32(udiff(b>>8, b1>>8)))
		}
	}
	return diff
}

// Abs32 returns the absolute value of uint32 x.
func Abs32(x uint32) uint32 {
	if x < 0 {
		return -x
	}
	return x
}

//Copy frame buffer to 32bit RGBA byte array
//
//Only useful if you are using additional framebuffers, this retrieves the contents of a framebuffer of your choice
func CopyFrameBuff(glctx gl.Context, rtt_frameBuff gl.Framebuffer, clientWidth, clientHeight int) []byte {
	buff := make([]byte, clientWidth*clientHeight*4, clientWidth*clientHeight*4)
	//fmt.Printf("reading pixels: ")
	//glctx.BindFramebuffer(gl.FRAMEBUFFER, rtt_frameBuff)
	glctx.ReadPixels(buff, 0, 0, clientWidth, clientHeight, gl.RGBA, gl.UNSIGNED_BYTE)
	//glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	return buff
}

//Dumps a go image format thing to disk
func SaveImage(m *image.RGBA, filename string) {
	f, _ := os.OpenFile(filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0666)
	defer f.Close()
	png.Encode(f, m)
}

//Saves a 32 bit RGBA byte array to a PNG file
func SaveBuff(texWidth, texHeight int, buff []byte, filename string) {
	m := image.NewNRGBA(image.Rectangle{image.Point{0, 0}, image.Point{int(texWidth), int(texHeight)}})
	if buff != nil {
		//log.Printf("Saving buffer: %v,%v", texWidth, texHeight)
		for y := 0; y < texHeight; y++ {
			for x := 0; x < texWidth; x++ {
				i := (x + y*texWidth) * 4
				m.Set(int(x), int(texHeight-y), color.NRGBA{uint8(buff[i]), uint8(buff[i+1]), uint8(buff[i+2]), 255})
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

//Render-to-texture
//
//Instead of drawing to the screen, draw into a texture.  You must create the framebuffer and texture first, and do whatever setup is required to make them valid.
//
//Thunk is a function that takes no args, returns no values, but draws the gl screen
//
//Rtt does the correct setup to prepare the texture for drawing, calls thunk() to draw it, then it restores the default frambuffer
//
//i.e. frambuffer 0 is active at the end of the call, so make sure you switch to the correct one before doing anymore drawing!  I should probably take that out, and figure out how to restore the currect framebuff
func Rtt(glctx gl.Context, rtt_frameBuff gl.Framebuffer, rtt_tex gl.Texture, texWidth, texHeight int, program gl.Program, thunk Thunk) {
	glctx.BindFramebuffer(gl.FRAMEBUFFER, rtt_frameBuff)
	glctx.Viewport(0, 0, texWidth, texHeight)
	glctx.ActiveTexture(gl.TEXTURE0)
	glctx.BindTexture(gl.TEXTURE_2D, rtt_tex)
	//draw here the content you want in the texture
	log.Printf("+Framebuffer status: %v\n", glctx.CheckFramebufferStatus(gl.FRAMEBUFFER))

	//rtt_tex is now a texture with the drawn content
	depthbuffer := glctx.CreateRenderbuffer()
	glctx.BindRenderbuffer(gl.RENDERBUFFER, depthbuffer)
	glctx.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT16, texWidth, texHeight)
	glctx.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthbuffer)
	//     glctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, rtt_tex, 0)

	glctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, rtt_tex, 0)

	glctx.Enable(gl.BLEND)
	glctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	glctx.Enable(gl.DEPTH_TEST)
	glctx.DepthFunc(gl.LEQUAL)
	glctx.DepthMask(true)
	glctx.ClearColor(0, 0, 0, 1)
	glctx.UseProgram(program) //FIXME - may cause graphics glitches
	glctx.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	thunk()

	glctx.Flush()
	glctx.GenerateMipmap(gl.TEXTURE_2D)

	buff := CopyFrameBuff(glctx, rtt_frameBuff, texWidth, texHeight)
	SaveBuff(int(texWidth), int(texHeight), buff, fmt.Sprintf("x_%v.png", fname_int))
	fname_int += 1
	glctx.BindTexture(gl.TEXTURE_2D, gl.Texture{0})
	glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	//log.Println("Finished Rtt")
	//fmt.Printf("done \n")
}

//Prints the contents of a 32bit RGBA byte array as ASCII text
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

//Renders a string into a openGL texture.  No guarantees are made that the text will fit
func String2Tex(glctx gl.Context, str string, tSize float64, glTex gl.Texture) {
	img, _ := DrawStringRGBA(tSize, color.RGBA{255, 255, 255, 255}, str)
	SaveImage(img, "texttest.png")
	w := 256 //img.Bounds().Max.X  //FIXME
	buff := PaintTexture(img, nil, int(w))
	//DumpBuff(buff, uint(w), uint(w))
	UploadTex(glctx, glTex, w, w, buff)
}

//Will attempt to load the contents of a 32bit RGBA byte array into an existing openGL texture.  The texture will be uploaded with the right options for displaying text i.e. clamp_to_edge and filter nearest.
func UploadTex(glctx gl.Context, glTex gl.Texture, w, h int, buff []uint8) {
	glctx.BindTexture(gl.TEXTURE_2D, glTex)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	glctx.TexImage2D(gl.TEXTURE_2D, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, buff)
	glctx.GenerateMipmap(gl.TEXTURE_2D)
}

func checkGlError(glctx gl.Context) {
	err := glctx.GetError()
	if err > 0 {
		errStr := fmt.Sprintf("GLerror: %v\n", err)
		fmt.Printf(errStr)
		panic(errStr)
	}
}


//Creates a new framebuffer and texture, with the texture attached to the frame buffer
//
func GenTextureAndFramebuffer(glctx gl.Context, w, h int, format gl.Enum) (gl.Framebuffer, gl.Texture) {
	f := glctx.CreateFramebuffer()
	checkGlError(glctx)
	/*glctx.BindFramebuffer(gl.FRAMEBUFFER, f)
	glctx.ActiveTexture(gl.TEXTURE0)
	t := glctx.CreateTexture()
	log.Printf("Texture created: %v", t)

	glctx.BindTexture(gl.TEXTURE_2D, t)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	checkGlError(glctx)

	glctx.TexImage2D(gl.TEXTURE_2D, 0, w, h, format, gl.UNSIGNED_INT, nil)
	checkGlError(glctx)
	//glctx.TexImage2D(gl.TEXTURE_2D, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	glctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t, 0)
	checkGlError(glctx)
	*/

	/*
	   depthbuffer := glctx.CreateRenderbuffer()
	   glctx.BindRenderbuffer(gl.RENDERBUFFER, depthbuffer)
	   glctx.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT16, w, h)
	   glctx.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthbuffer)
	*/

	//status := glctx.CheckFramebufferStatus(gl.FRAMEBUFFER)
	//if status != gl.FRAMEBUFFER_COMPLETE {
	//log.Fatal(fmt.Sprintf("Gentexture failed: Framebuffer status: %v\n", status))
	//}
	//glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	return f, GenTextureOnFramebuffer(glctx, f, w, h, format)
}

//Creates a new framebuffer and texture, with the texture attached to the frame buffer
//
func GenTextureOnFramebuffer(glctx gl.Context, f gl.Framebuffer, w, h int, format gl.Enum) gl.Texture {
	glctx.BindFramebuffer(gl.FRAMEBUFFER, f)
	checkGlError(glctx)
	glctx.ActiveTexture(gl.TEXTURE0)
	checkGlError(glctx)
	t := glctx.CreateTexture()
	checkGlError(glctx)
	log.Printf("Texture created: %v", t)

	glctx.BindTexture(gl.TEXTURE_2D, t)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	checkGlError(glctx)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	checkGlError(glctx)

	log.Printf("Creating texture of width %v and height %v", w, h)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, w, h, format, gl.UNSIGNED_BYTE, nil)
	checkGlError(glctx)
	//glctx.TexImage2D(gl.TEXTURE_2D, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	glctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, t, 0)
	checkGlError(glctx)

	/*
	   depthbuffer := glctx.CreateRenderbuffer()
	   glctx.BindRenderbuffer(gl.RENDERBUFFER, depthbuffer)
	   glctx.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT16, w, h)
	   glctx.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthbuffer)
	*/

	//status := glctx.CheckFramebufferStatus(gl.FRAMEBUFFER)
	//if status != gl.FRAMEBUFFER_COMPLETE {
	//log.Fatal(fmt.Sprintf("Gentexture failed: Framebuffer status: %v\n", status))
	//}
	glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	checkGlError(glctx)
	return t
}

var renderCache map[string]*image.RGBA
var faceCache map[string]*font.Face
var fontCache map[string]*truetype.Font

//Dump the rendercache, facecache and fontcache
func ClearAllCaches() {
	renderCache = map[string]*image.RGBA{}
	faceCache = map[string]*font.Face{}
	fontCache = map[string]*truetype.Font{}
}

//Creates a texture and draws a string to it
//
//FIXME some fonts might not compeletely fit in the texture (usually the decorative ones which extend into another letter)
func DrawStringRGBA(txtSize float64, fontColor color.RGBA, txt string) (*image.RGBA, *font.Face) {
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
	txtFont := LoadFont("f1.ttf")
	d := &font.Drawer{
		Src: image.NewUniform(fontColor), // 字体颜色
		Face: truetype.NewFace(txtFont, &truetype.Options{
			Size:    txtSize,
			DPI:     72,
			Hinting: font.HintingNone,
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
	//ascend := fuckedRect.Min.Y
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

//Holds all the configuration details for drawing a string into a texture.  This structure gets written to during the draw
type FormatParams struct {
	Colour            *color.RGBA //Text colour
	Line              int
	Cursor            int         //The cursor position, in characters from the start of the text
	SelectStart       int         //Start of the selection box, counted from the start of document
	SelectEnd         int         //End of the selection box, counted from the start of document
	StartLinePos      int         //Updated during render, holds the closest start of line, including soft line breaks
	FontSize          float64     //Fontsize, in points or something idfk
	FirstDrawnCharPos int         //The first character to draw on the screen.  Anything before this is ignored
	LastDrawnCharPos  int         //The last character that we were able to fit on the screen
	TailBuffer        bool        //Nothing for now
	Outline           bool        //Nothing for now
	Vertical          bool        //Draw texture vertically for Chinese/Japanese rendering
	SelectColour      *color.RGBA //Selection text colour
}

//Create a new text formatter, with useful default parameters
func NewFormatter() *FormatParams {
	return &FormatParams{&color.RGBA{5, 5, 5, 255}, 0, 0, 0, 0, 0, 22.0, 0, 0, false, true, false, &color.RGBA{255, 128, 128, 255}}
}

//Draw a cursor shape
func DrawCursor(xpos, ypos, height, clientWidth int, u8Pix []byte) {
	colour := byte(0)
	for xx := int(0); xx < 3; xx++ {
		for yy := int(0); yy < height; yy++ {
			offset := (yy+ypos)*clientWidth*4 + (xx+xpos)*4
			//log.Printf("Drawpos: %v", offset)
			if offset >= 0 && offset < (len(u8Pix)) {
				u8Pix[offset] = colour
				u8Pix[offset+1] = colour
				u8Pix[offset+2] = colour
				u8Pix[offset+3] = 255
			}
		}
	}
}

//Check and correct formatparams to make sure e.g. cursor is always on the screen
func SanityCheck(f *FormatParams, txt string) {
	if f.Cursor < 0 {
		f.Cursor = 0
	}
	if f.Cursor >= len(txt)-1 {
		f.Cursor = len(txt) - 1
	}
	if f.FirstDrawnCharPos < 0 {
		f.FirstDrawnCharPos = 0
	}
	if f.FirstDrawnCharPos >= len(txt)-1 {
		f.FirstDrawnCharPos = len(txt) - 1
	}

}

func Fixed2int(n fixed.Int26_6) int {
	return n.Round()
}

type Vec2 struct {
	x, y int
}

func InBounds(v, min, max, charDim Vec2) bool {
	if v.x < min.x {
		return false
	}
	if v.y < min.y {
		return false
	}
	if v.x > max.x {
		return false
	}
	if v.y < min.y {
		return false
	}
	return true
}

func MoveInBounds(v, min, max, charDim, charAdv, linAdv Vec2) (newPos Vec2) {
	//fmt.Printf("pos: (%v), min: (%v), max: (%v), charDim: (%v)\n",v, min, max, charDim)
	if v.x < min.x {
		return MoveInBounds(Vec2{v.x + 1, v.y}, min, max, charDim, charAdv, linAdv)
	}
	if v.y < min.y {
		return MoveInBounds(Vec2{v.x, v.y + 1}, min, max, charDim, charAdv, linAdv)
	}
	if v.x+charDim.x > max.x {
		return MoveInBounds(Vec2{v.x - 1, v.y}, min, max, charDim, charAdv, linAdv)
	}
	if v.y+charDim.y > max.y {
		return MoveInBounds(Vec2{v.x, v.y - 1}, min, max, charDim, charAdv, linAdv)
	}
	return v
}

//Get the maximum pixel size needed to hold a string
func GetGlyphSize(size float64, str string) (int, int) {
	_, str_size := utf8.DecodeRuneInString(str)
	img, _ := DrawStringRGBA(size, color.RGBA{1.0, 1.0, 1.0, 1.0}, str[0:str_size])
	XmaX, YmaX := img.Bounds().Max.X, img.Bounds().Max.Y
	if XmaX > 4000 {
		panic("X can't be that big")
	}
	return XmaX, YmaX
}

func copyFormatter(inF *FormatParams) *FormatParams {
	out := NewFormatter()
	*out = *inF
	return out
}

//Draw some text into a 32bit RGBA byte array, wrapping where needed.  Supports all the options I need for a basic word processor, including vertical text, and different sized lines
//This was a bad idea.  Instead of all the if statements, we should just assume everything is left-to-right, top-to-bottom, and then rotate the entire block afterwards (we will also have to rotate the characters around their own center)
//Arabic will still need special code - better to separate into two completely different routines?
//Return the cursor position (number of characters from start of text) that is closest to the mouse cursor (cursorX, cursorY)
func RenderPara(f *FormatParams, xpos, ypos, orig_xpos, orig_ypos, maxX, maxY, clientWidth, clientHeight, cursorX, cursorY int, u8Pix []uint8, text string, transparent bool, doDraw bool, showCursor bool) (int, int, int) {
	cursorDist := 9999999
	seekCursorPos := 0
	vert := f.Vertical
	orig_colour := f.Colour
	foreGround := f.Colour
	colSwitch := false
	if f.TailBuffer {
		//f.Cursor = len(text)
		//scrollToCursor(f, text)  //Use pageup function, once it is fast enough
	}
	//log.Printf("Cursor: %v\n", f.Cursor)
	letters := strings.Split(text, "")
	letters = append(letters, " ")
	orig_fontSize := f.FontSize
	defer func() {
		f.FontSize = orig_fontSize
		SanityCheck(f, text)
	}()
	//xpos := orig_xpos
	//ypos := orig_ypos
	if vert {
		xpos = maxX
	}
	gx, gy := GetGlyphSize(f.FontSize, text)
	//fmt.Printf("Chose position %v, maxX: %v\n", pos, maxX)
	pos := MoveInBounds(Vec2{xpos, ypos}, Vec2{orig_xpos, orig_ypos}, Vec2{maxX, maxY}, Vec2{gx, gy}, Vec2{0, 1}, Vec2{-1, 0})
	xpos = pos.x
	ypos = pos.y
	maxHeight := 0
	letterWidth := 100
	wobblyMode := false
	if f.Cursor > len(letters) {
		f.Cursor = len(letters)
	}
	//sanityCheck(f,txt)
	for i, v := range letters {
		if i < f.FirstDrawnCharPos {
			continue
		}
		if (f.Cursor == i) && doDraw {
			DrawCursor(xpos, ypos, maxHeight, clientWidth, u8Pix)
		}
		if i >= len(letters)-1 {
			continue
		}
		//foreGround = orig_colour

		if unicode.IsSpace([]rune(v)[0]) {
			//if i>0 && letters[i-1] == " " {
			//f.Colour = &color.RGBA{255,0,0,255}
			//f.FontSize = f.FontSize*1.2
			////log.Printf("Oversize start for %v at %v\n", v, i)
			//} else {
			//f.Colour = &color.RGBA{1,1,1,255}
			//}
			colSwitch = !colSwitch
			if colSwitch {
				foreGround = &color.RGBA{255, 1, 1, 255}
			} else {
				foreGround = orig_colour
			}
		}
		if (i >= f.SelectStart) && (i <= f.SelectEnd) && (f.SelectStart != f.SelectEnd) {
			nf := copyFormatter(f)
			nf.SelectStart = -1
			nf.SelectEnd = -1
			nf.Colour = &color.RGBA{255, 1, 1, 255}
			/*if i-1<f.SelectStart {
			      _, xpos, ypos = RenderPara(nf, xpos, ypos, 0, 0, maxX, maxY, clientWidth, clientHeight, cursorX, cursorY, u8Pix, "{", transparent, doDraw, showCursor)
			  }
			  if i+1>f.SelectEnd {
			      _, xpos, ypos = RenderPara(nf, xpos, ypos, 0, 0, maxX, maxY, clientWidth, clientHeight, cursorX, cursorY, u8Pix, "}", transparent, doDraw, showCursor)
			  }*/

			//fmt.Printf("%v is between %v and %v\n", i , f.SelectStart, f.SelectEnd)
			foreGround = nf.Colour
		}
		//fmt.Printf("%v: %V\n", i , f)
		if (string(text[i]) == " ") || (string(text[i]) == "\n") {
			f.FontSize = orig_fontSize
			//log.Printf("Oversize end for %v at %v\n", v, i)
		}
		if string(text[i]) == "\n" {
			if vert {
				xpos = xpos - maxHeight
				ypos = orig_ypos
			} else {
				ypos = ypos + maxHeight
				xpos = orig_xpos
				if i > 0 && string(text[i-1]) != "\n" {
					maxHeight = 12 //FIXME
				}
			}
			f.Line++
			f.StartLinePos = i
			if f.Cursor == i && showCursor {
				DrawCursor(xpos, ypos, maxHeight, clientWidth, u8Pix)
			}
		} else {
			if i >= f.FirstDrawnCharPos {
				ytweak := 0
				if wobblyMode {
					ytweak = int(math.Sin(float64(xpos)) * 5.0)
				}
				img, face := DrawStringRGBA(f.FontSize, *foreGround, v)
				XmaX, YmaX := img.Bounds().Max.X, img.Bounds().Max.Y
				imgBytes := img.Pix
				//imgBytes := Rotate270(XmaX, YmaX, img.Pix)
				//XmaX, YmaX = YmaX, XmaX
				fa := *face
				glyph, _ := utf8.DecodeRuneInString(v)
				letterWidth_F, _ := fa.GlyphAdvance(glyph)
				letterWidth = Fixed2int(letterWidth_F)
				//fuckedRect, _, _ := fa.GlyphBounds(glyph)
				//letterHeight := fixed2int(fuckedRect.Max.Y)
				letterHeight := Fixed2int(fa.Metrics().Height)
				//letterWidth := XmaX
				//letterHeight = letterHeight

				if vert && (xpos < 0) {
					if vert {
						f.LastDrawnCharPos = i - 1
						return seekCursorPos, xpos, ypos
					} else {
						pos := MoveInBounds(Vec2{xpos, ypos}, Vec2{orig_xpos, orig_ypos}, Vec2{maxX, maxY}, Vec2{gx, gy}, Vec2{0, 1}, Vec2{-1, 0})
						xpos = pos.x
						ypos = pos.y
					}
				}
				if xpos+XmaX > maxX {
					if !vert {
						ypos = ypos + maxHeight
						maxHeight = 0
						xpos = orig_xpos
						f.Line++
						f.StartLinePos = i
					}
				}

				if (ypos+YmaX+ytweak+1 > maxY) || (ypos+ytweak < 0) {
					if vert {
						xpos = xpos - maxHeight
						maxHeight = 0
						ypos = orig_ypos
						f.Line++
						f.StartLinePos = i
					} else {
						f.LastDrawnCharPos = i - 1
						return seekCursorPos, xpos, ypos
					}
				}
				pos := MoveInBounds(Vec2{xpos, ypos}, Vec2{orig_xpos, orig_ypos}, Vec2{maxX, maxY}, Vec2{XmaX, YmaX}, Vec2{0, 1}, Vec2{-1, 0})
				xpos = pos.x
				ypos = pos.y

				if doDraw {
					//PasteImg(img, xpos, ypos + ytweak, u8Pix, transparent)
					//PasteBytes(XmaX, YmaX, imgBytes, xpos, ypos+ytweak, int(clientWidth), int(clientHeight), u8Pix, transparent)
					PasteBytes(XmaX, YmaX, imgBytes, xpos, ypos+ytweak, int(clientWidth), int(clientHeight), u8Pix, true, false, true)
				}

				if f.Cursor == i && showCursor {
					DrawCursor(xpos, ypos, maxHeight, clientWidth, u8Pix)
				}

				f.LastDrawnCharPos = i
				maxHeight = MaxI(maxHeight, letterHeight)

				if vert {
					ypos += maxHeight
				} else {
					xpos += letterWidth
				}
			}
		}
		d := (cursorX-xpos+letterWidth)*(cursorX-xpos+letterWidth) + (cursorY-ypos-maxHeight/2)*(cursorY-ypos-maxHeight/2)
		if d < cursorDist {
			cursorDist = d
			seekCursorPos = i
		}

	}
	SanityCheck(f, text)
	return seekCursorPos, xpos, ypos
}

//Return the larger of two integers
func MaxI(a, b int) int {
	if a > b {
		return a
	}
	return b
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

func PasteText(tSize float64, xpos, ypos, clientWidth, clientHeight int, text string, u8Pix []uint8, transparent bool) {
	img, _ := DrawStringRGBA(tSize, color.RGBA{255, 255, 255, 255}, text)
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

//Rotates a 32bit byte array into a new byte array.  The target array will be created with the correct dimensions
func Rotate270(srcW, srcH int, src []byte) []byte {
	//log.Printf("Rotating image (%v,%v)\n",srcW, srcH)
	dstW := srcH
	dstH := srcW
	dst := make([]byte, dstW*dstH*4)

	for dstY := 0; dstY < dstH; dstY++ {
		for dstX := 0; dstX < dstW; dstX++ {
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
