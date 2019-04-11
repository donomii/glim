// GL Image library.  Routines for handling images and textures in GO OpenGL (especially with the GoMobile framework)
package glim

import (
	"golang.org/x/image/font/gofont/gomono"
	_ "golang.org/x/image/font/gofont/goregular"

	"github.com/golang/freetype/truetype"
)

//Attempts to load a font using goMobile's truetype font library
func LoadFont(fileName string) *truetype.Font {

	if fontCache == nil {
		fontCache = map[string]*truetype.Font{}
	}
	im, ok := fontCache[fileName]
	if ok {
		return im
	}

	//fontBytes := sysFont.Default()

	txtFont, _ := truetype.Parse(gomono.TTF)
	fontCache[fileName] = txtFont
	return txtFont
}
