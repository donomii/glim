// Text editor library.  Routines to support a text editor
package glim

import (
	"math"
	_ "net/http/pprof"

	//"fmt"
	_ "fmt"
	_ "image/jpeg"
	"log"
	"regexp"

	_ "image/png"
)

// Holds all the configuration details for drawing a string into a texture.  This structure gets written to during the draw
type FormatParams struct {
	Colour            *RGBA   // Text colour
	Line              int     // The line number, i.e. the number of /n characters from the start
	Cursor            int     // The cursor position, in characters from the start of the text
	SelectStart       int     // Start of the selection box, counted from the start of document
	SelectEnd         int     // End of the selection box, counted from the start of document
	StartLinePos      int     // Updated during render, holds the closest start of line, including soft line breaks
	FontSize          float64 // Fontsize, in points or something idfk
	FirstDrawnCharPos int     // The first character to draw on the screen.  Anything before this is ignored
	LastDrawnCharPos  int     // The last character that we were able to fit on the screen
	TailBuffer        bool    // Nothing for now
	Outline           bool    // Nothing for now
	Vertical          bool    // Draw texture vertically for Chinese/Japanese rendering
	SelectColour      *RGBA   // Selection text colour
	CursorColour      *RGBA
	HighlightColour   *RGBA
}

// Create a new text formatter, with useful default parameters
func NewFormatter() *FormatParams {
	return &FormatParams{&RGBA{5, 5, 5, 255}, 0, 0, 0, 0, 0, 22.0, 0, 0, false, true, false, &RGBA{255, 128, 128, 255}, &RGBA{255, 0, 0, 255}, &RGBA{255, 255, 0, 255}}
}

// Draw a cursor shape
func DrawCursor(xpos, ypos, height, pixWidth int, u8Pix []byte, cursorColour *RGBA) {
	colour := *cursorColour
	for xx := int(0); xx < 6; xx++ {
		for yy := int(0); yy < height; yy++ {
			offset := (yy+ypos)*pixWidth*4 + (xx+xpos)*4
			// log.Printf("Drawpos: %v", offset)
			if offset >= 0 && offset < (len(u8Pix)) {
				u8Pix[offset] = colour[0]
				u8Pix[offset+1] = colour[1]
				u8Pix[offset+2] = colour[2]
				u8Pix[offset+3] = 255
			}
		}
	}
}

// Check and correct formatparams to make sure e.g. cursor is always on the screen
func SanityCheck(f *FormatParams, txt string) {
	log.Println("Sanity check")
	if f.Cursor < 0 {
		f.Cursor = 0
	}
	if f.Cursor > len(txt) {
		f.Cursor = len(txt)
	}
	if f.FirstDrawnCharPos < 0 {
		f.FirstDrawnCharPos = 0
	}
	if f.FirstDrawnCharPos >= len(txt)-1 {
		f.FirstDrawnCharPos = len(txt) - 1
	}
	if f.Cursor < 0 {
		f.Cursor = 0
	}
}

// Is v inside the box defined by min and max?
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

// Move v to the closest point inside the box defined by min.max
func MoveInBounds(v, min, max, charDim, charAdv, linAdv Vec2, attempts int) (newPos Vec2) {
	attempts = attempts - 1
	if attempts < 0 {
		return v
	}
	// fmt.Printf("pos: (%v), min: (%v), max: (%v), charDim: (%v)\n",v, min, max, charDim)
	if v.X < min.X {
		return MoveInBounds(Vec2{v.X + 1, v.Y}, min, max, charDim, charAdv, linAdv, attempts)
	}
	if v.Y < min.Y { // FIXME?
		return MoveInBounds(Vec2{v.X, v.Y + 1}, min, max, charDim, charAdv, linAdv, attempts)
	}
	if v.X+charDim.X > max.X {
		return MoveInBounds(Vec2{v.X - 1, v.Y}, min, max, charDim, charAdv, linAdv, attempts)
	}
	if v.Y+charDim.Y > max.Y {
		return MoveInBounds(Vec2{v.X, v.Y - 1}, min, max, charDim, charAdv, linAdv, attempts)
	}
	return v
}

// Duplicate a formatter, that can be modified without changing the original
func CopyFormatter(inF *FormatParams) *FormatParams {
	out := NewFormatter()
	*out = *inF
	return out
}

func CopyPix(pix []uint8) []uint8 {
	var new []uint8 = make([]uint8, len(pix))
	copy(new, pix)
	return new
}

// Draw some text into a 32bit RGBA byte array, wrapping where needed.  Supports all the options I need for a basic word processor, including vertical text, and different sized lines
//
// This was a bad idea.  Instead of all the if statements, we should just assume everything is left-to-right, top-to-bottom, and then rotate the entire block afterwards (we will also have to rotate the characters around their own center)
//
// Return the cursor position (number of characters from start of text) that is closest to the mouse cursor (cursorX, cursorY)
//
// xpos, ypos - The starting draw position, in 0<=xpos<=pixWidth, 0<=y<=pixHeight
// minX, minY - The leftmost part of the draw subregion.  To fill the whole pix, set to 0,0
// maxX, maxY - The rightmost edge of draw subregion.  To fill the whole pix, set to pixWidth, pixHeight
// pixWidth, pixHeight - the size of the bitmap (e.g. in screen coordinates)
// cursorX, cursorY - Mouse cursor coordinates, relative to whole image
func RenderPara(f *FormatParams, xpos, ypos, minX, minY, maxX, maxY, pixWidth, pixHeight, cursorX, cursorY int, u8Pix []uint8, text string, transparent bool, doDraw bool, showCursor bool) (int, int, int) {
	// re := regexp.MustCompile(`\t`)
	// text = re.ReplaceAllLiteralString(text, "    ")
	// strs := strings.SplitAfter(text, " ")
	letterz := []rune(text)
	out := []Token{}
	for _, v := range letterz {
		out = append(out, Token{string(v), Style{ForegroundColour: f.Colour}})
	}
	return RenderTokenPara(f, xpos, ypos, minX, minY, maxX, maxY, pixWidth, pixHeight, cursorX, cursorY, u8Pix, out, transparent, doDraw, showCursor)
}

func isNewLine(v string) bool {
	return (v == "\n") || (v == `\n`)
}

type Style struct {
	ForegroundColour *RGBA // Text colour
}

type Token struct {
	Text  string
	Style Style
}

func RenderTokenPara(f *FormatParams, xpos, ypos, minX, minY, maxX, maxY, pixWidth, pixHeight, cursorX, cursorY int, u8Pix []uint8, tokens []Token, transparent bool, doDraw bool, showCursor bool) (int, int, int) {
	cursorDist := 9999999
	seekCursorPos := 0
	vert := f.Vertical
	// selectColour := color.RGBA{255, 1, 1, 255}
	// highlightColour := color.RGBA{1, 255, 1, 255}
	// colSwitch := false
	if f.TailBuffer {
		// f.Cursor = len(text)
		// scrollToCursor(f, text)  //Use pageup function, once it is fast enough
	}
	// log.Printf("Cursor: %v\n", f.Cursor)
	var letters []string
	var markup []Style
	for _, v := range tokens {
		re := regexp.MustCompile(`\\t`)
		t := re.ReplaceAllLiteralString(v.Text, "    ")
		letters = append(letters, t)
		markup = append(markup, v.Style)
	}

	letters = append(letters, " ")
	markup = append(markup, Style{})
	orig_fontSize := f.FontSize
	defer func() {
		f.FontSize = orig_fontSize
		// SanityCheck(f, text)
	}()
	// xpos := minX
	// ypos := minY
	if vert {
		xpos = maxX
	}
	gx, gy := GetGlyphSize(f.FontSize, letters[0])
	// fmt.Printf("Chose position %v, maxX: %v\n", pos, maxX)
	pos := MoveInBounds(Vec2{xpos, ypos}, Vec2{minX, minY}, Vec2{maxX, maxY}, Vec2{gx, gy}, Vec2{0, 1}, Vec2{-1, 0}, 10)
	xpos = pos.X
	ypos = pos.Y
	maxHeight := 0
	letterWidth := 100
	wobblyMode := false
	if f.Cursor > len(letters) {
		f.Cursor = len(letters)
	}
	// sanityCheck(f,txt)
	for i, v := range letters {

		style := markup[i]

		foreGround := style.ForegroundColour
		if foreGround == nil {
			foreGround = &RGBA{255, 255, 255, 255}
		}

		// fmt.Printf("%v: '%v'(%V)\n", i, v, v)
		if isNewLine(v) {
			v = "\n"
		}
		if v == `\t` {
			v = "    "
		}
		if i < f.FirstDrawnCharPos {
			continue
		}
		if (showCursor && f.Cursor == i) && doDraw {
			DrawCursor(xpos, ypos, maxHeight, pixWidth, u8Pix, f.CursorColour)
		}
		if i >= len(letters)-1 {
			continue
		}
		// foreGround = orig_colour

		if v == " " || isNewLine(v) {
			f.FontSize = orig_fontSize
			// log.Printf("Oversize end for %v at %v\n", v, i)
		}
		if isNewLine(v) {
			if vert {
				xpos = xpos - maxHeight
				ypos = minY
			} else {
				ypos = ypos + maxHeight
				xpos = minX
				if i > 0 && !isNewLine(letters[i-1]) {
					maxHeight = 12 // FIXME
				}
			}
			// fmt.Printf("Newline char forces line++\n")
			f.Line = f.Line + 1
			f.StartLinePos = i
			if f.Cursor == i && showCursor {
				DrawCursor(xpos, ypos, maxHeight, pixWidth, u8Pix, f.CursorColour)
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
				// imgBytes := Rotate270(XmaX, YmaX, img.Pix)
				// XmaX, YmaX = YmaX, XmaX
				fa := *face
				// glyph, _ := utf8.DecodeRuneInString(v)
				// letterWidth_F, _ := fa.GlyphAdvance(glyph)
				// letterWidth = Fixed2int(letterWidth_F)
				// fuckedRect, _, _ := fa.GlyphBounds(glyph)
				// letterHeight := fixed2int(fuckedRect.Max.Y)
				letterHeight := Fixed2int(fa.Metrics().Height)
				letterWidth := XmaX / 2
				// letterHeight = letterHeight

				if vert && (xpos < 0) {
					if vert {
						f.LastDrawnCharPos = i - 1
						return seekCursorPos, xpos, ypos
					} else {
						pos := MoveInBounds(Vec2{xpos, ypos}, Vec2{minX, minY}, Vec2{maxX, maxY}, Vec2{gx, gy}, Vec2{0, 1}, Vec2{-1, 0}, 10)
						xpos = pos.X
						ypos = pos.Y
					}
				}
				/*if xpos+XmaX > maxX {
					if !vert {
						ypos = ypos + maxHeight
						maxHeight = 0
						// fmt.Printf("OOB X forces line++\n")
						xpos = minX
						f.Line++
						f.StartLinePos = i
					}
				}*/

				if (ypos+YmaX+ytweak+1 > maxY) || (ypos+ytweak < 0) {
					if vert {
						xpos = xpos - maxHeight
						maxHeight = 0
						ypos = minY
						// fmt.Printf("OOB Y forces line++\n")
						f.Line++
						f.StartLinePos = i
					} else {
						f.LastDrawnCharPos = i - 1
						return seekCursorPos, xpos, ypos
					}
				}
				pos := MoveInBounds(Vec2{xpos, ypos}, Vec2{minX, minY}, Vec2{maxX, maxY}, Vec2{XmaX, YmaX}, Vec2{0, 1}, Vec2{-1, 0}, 10)
				xpos = pos.X
				ypos = pos.Y

				if doDraw {
					// PasteImg(img, xpos, ypos + ytweak, u8Pix, transparent)
					// PasteBytes(XmaX, YmaX, imgBytes, xpos, ypos+ytweak, int(pixWidth), int(pixHeight), u8Pix, transparent)
					PasteBytes(XmaX, YmaX, imgBytes, xpos, ypos+ytweak, int(pixWidth), int(pixHeight), u8Pix, true, false, false)
				}

				if f.Cursor == i && showCursor {
					DrawCursor(xpos, ypos, maxHeight, pixWidth, u8Pix, f.CursorColour)
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
	// SanityCheck(f, text)
	return seekCursorPos, xpos, ypos
}

// Return the larger of two integers
func MaxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}
