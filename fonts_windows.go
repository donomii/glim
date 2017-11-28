// GL Image library.  Routines for handling images and textures in GO OpenGL (especially with the GoMobile framework)
package glim

import (

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

	var f io.Reader
	folderPath, err := osext.ExecutableFolder()
	if err != nil {
		log.Printf("Could not get exec path, exiting because the go mobile team do not understand the concept of 'cross-paltform'\n")
		panic("Go mobile sucks")
	} else {
		//log.Println(fileName)
		file, err := os.Open(fmt.Sprintf("%v%v%v", folderPath, string(os.PathSeparator), fileName))
		if err != nil {
			//log.Fatal(err)
			log.Printf("Could not open %v, exiting because the go mobile team do not understand the concept of 'cross-platform'\n", fmt.Sprintf("%v%v%v", folderPath, string(os.PathSeparator), fileName))
			panic("Go mobile sucks")
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
