// GL Image library.  Routines for handling images and textures in GO OpenGL (especially with the GoMobile framework)
package glim

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/kardianos/osext"

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

	var f io.Reader
	folderPath, err := osext.ExecutableFolder()
	if err != nil {
		log.Printf("Could not get exec path, falling back to system font\n")
		fontBytes := sysFont.Monospace()
		f = bytes.NewReader(fontBytes)
	} else {
		//log.Println(fileName)
		file, err := os.Open(fmt.Sprintf("%v%v%v", folderPath, string(os.PathSeparator), fileName))
		if err != nil {
			//log.Fatal(err)
			log.Printf("Could not open %v, falling back to system font\n", fmt.Sprintf("%v%v%v", folderPath, string(os.PathSeparator), fileName))
			fontBytes := sysFont.Monospace()
			f = bytes.NewReader(fontBytes)

		} else {
			defer file.Close()
			f = file
		}
	}
	fontBytes, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	txtFont, err1 := truetype.Parse(fontBytes)
	if err1 != nil {
		log.Println(err1)
		panic(err1)
	}

	fontCache[fileName] = txtFont
	return txtFont
}
