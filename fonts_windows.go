// GL Image library.  Routines for handling images and textures in GO OpenGL (especially with the GoMobile framework)
package glim

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"

	"golang.org/x/image/font/gofont/gomono"

	//_ "golang.org/x/image/font/gofont/goregular"

	"github.com/golang/freetype/truetype"
	"github.com/kardianos/osext"
)

func LoadFileFromExecFolder(fileName string) *truetype.Font {
	var f io.Reader
	var fontBytes []byte
	folderPath, err := osext.ExecutableFolder()
	fullPath := fmt.Sprintf("%v/%v", folderPath, fileName)
	fontBytes, err = ioutil.ReadFile(fullPath)

	if err != nil {
		log.Printf("Could not get exec path, falling back to system font\n")
		f = bytes.NewReader(gomono.TTF)
		fontBytes, err = ioutil.ReadAll(f)

		if err != nil {
			log.Println(err)
			panic(err)
		}
	}

	txtFont, err1 := truetype.Parse(fontBytes)
	if err1 != nil {
		log.Println(err1)
		panic(err1)
	}

	return txtFont
}

func LoadFont(fileName string) *truetype.Font {

	if fontCache == nil {
		fontCache = map[string]*truetype.Font{}
	}
	im, ok := fontCache[fileName]
	if ok {
		return im
	}

	//fontBytes := sysFont.Default()

	txtFont := LoadFileFromExecFolder(fileName)
	fontCache[fileName] = txtFont
	return txtFont
}
