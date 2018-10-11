// Text editor library.  Routines to support a text editor
package glim

import "math"
import (
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"unicode"
	"unicode/utf8"


	"image/color"
	
)



//Holds all the configuration details for drawing a string into a texture.  This structure gets written to during the draw
type FormatParams struct {
	Colour            *color.RGBA //Text colour
	Line              int		  //The line number, i.e. the number of /n characters from the start
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

//Is v inside the box defined by min and max?
func InBounds(v, min, max Vec2) bool {
	if v.X < min.X {
		return false
	}
	if v.Y < min.Y {
		return false
	}
	if v.X > max.X {
		return false
	}
	if v.Y < min.Y {
		return false
	}
	return true
}

//Move v to the closest point inside the box defined by min.max
func MoveInBounds(v, min, max, charDim, charAdv, linAdv Vec2) (newPos Vec2) {
	//fmt.Printf("pos: (%v), min: (%v), max: (%v), charDim: (%v)\n",v, min, max, charDim)
	if v.X < min.X {
		return MoveInBounds(Vec2{v.X + 1, v.Y}, min, max, charDim, charAdv, linAdv)
	}
	if v.Y < min.Y { //FIXME?
		return MoveInBounds(Vec2{v.X, v.Y + 1}, min, max, charDim, charAdv, linAdv)
	}
	if v.X+charDim.X > max.X {
		return MoveInBounds(Vec2{v.X - 1, v.Y}, min, max, charDim, charAdv, linAdv)
	}
	if v.Y+charDim.Y > max.Y {
		return MoveInBounds(Vec2{v.X, v.Y - 1}, min, max, charDim, charAdv, linAdv)
	}
	return v
}

//Duplicate a formatter, that can be modified without changing the original
func CopyFormatter(inF *FormatParams) *FormatParams {
	out := NewFormatter()
	*out = *inF
	return out
}

//Draw some text into a 32bit RGBA byte array, wrapping where needed.  Supports all the options I need for a basic word processor, including vertical text, and different sized lines
//
//This was a bad idea.  Instead of all the if statements, we should just assume everything is left-to-right, top-to-bottom, and then rotate the entire block afterwards (we will also have to rotate the characters around their own center)
//
//Return the cursor position (number of characters from start of text) that is closest to the mouse cursor (cursorX, cursorY)
func RenderPara(f *FormatParams, xpos, ypos, orig_xpos, orig_ypos, maxX, maxY, clientWidth, clientHeight, cursorX, cursorY int, u8Pix []uint8, text string, transparent bool, doDraw bool, showCursor bool) (int, int, int) {
	cursorDist := 9999999
	seekCursorPos := 0
	vert := f.Vertical
	orig_colour := f.Colour
	foreGround := f.Colour
	selectColour := color.RGBA{255, 1, 1, 255}
	highlightColour := color.RGBA{1, 255, 1, 255}
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
	xpos = pos.X
	ypos = pos.Y
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
				foreGround = &highlightColour
			} else {
				foreGround = orig_colour
			}
		}
		if (i >= f.SelectStart) && (i <= f.SelectEnd) && (f.SelectStart != f.SelectEnd) {
			nf := CopyFormatter(f)
			nf.SelectStart = -1
			nf.SelectEnd = -1
			nf.Colour = &selectColour
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
				img, face := DrawStringRGBA(f.FontSize, *foreGround, v, "f1.ttf")
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
						xpos = pos.X
						ypos = pos.Y
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
				xpos = pos.X
				ypos = pos.Y

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
