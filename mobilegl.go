// mobilegl.go

//+build !coregl

package glim

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"golang.org/x/mobile/gl"
)

//Return screen (or main window) size
//
//Will only work with x/mobile/gl I guess
func ScreenSize(glctx gl.Context) (int, int) {
	outbuff := []int32{0, 0, 0, 0}
	glctx.GetIntegerv(outbuff, gl.VIEWPORT)
	screenWidth := int(outbuff[2])
	screenHeight := int(outbuff[3])
	return screenWidth, screenHeight
}

//Save the currently display to a file on disk
func ScreenShot(glctx gl.Context, filename string) {
	screenWidth, screenHeight := ScreenSize(glctx)
	//log.Printf("Saving width: %v, height: %v\n", screenWidth, screenHeight)
	SaveBuff(screenWidth, screenHeight, CopyScreen(glctx, int(screenWidth), int(screenHeight)), filename)
}

//Renders a string into a openGL texture.  No guarantees are made that the text will fit
func String2Tex(glctx gl.Context, str string, tSize float64, glTex gl.Texture, texSize int) {
	img, _ := DrawStringRGBA(tSize, RGBA{255, 255, 255, 255}, str, "f1.ttf")
	//SaveImage(img, "texttest.png")

	buff := PaintTexture(img, nil, int(texSize))
	//DumpBuff(buff, uint(w), uint(w))
	UploadTex(glctx, glTex, texSize, texSize, buff)
}

//Will attempt to load the contents of a 32bit RGBA byte array into an existing openGL texture.  The texture will be uploaded with the right options for displaying text i.e. clamp_to_edge and filter nearest.
func UploadTex(glctx gl.Context, glTex gl.Texture, w, h int, buff []uint8) {
	glctx.BindTexture(gl.TEXTURE_2D, glTex)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	glctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, w, h, gl.RGBA, gl.UNSIGNED_BYTE, buff)
	glctx.GenerateMipmap(gl.TEXTURE_2D)
}

func checkGlError(glctx gl.Context) {
	err := glctx.GetError()
	if err > 0 {
		errStr := fmt.Sprintf("GLerror: %v\n", err)
		log.Printf(errStr)
		//panic(errStr)
	}
}

//Creates a new framebuffer and texture, with the texture attached to the frame buffer
//
func GenTextureAndFramebuffer(glctx gl.Context, w, h int, format gl.Enum) (gl.Framebuffer, gl.Texture) {
	f := glctx.CreateFramebuffer()
	checkGlError(glctx)
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
	glctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, w, h, format, gl.UNSIGNED_BYTE, nil)
	checkGlError(glctx)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, w, h, gl.RGBA, gl.UNSIGNED_BYTE, nil)

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

//Render-to-texture
//
//Instead of drawing to the screen, draw into a texture.  You must create the framebuffer and texture first, and do whatever setup is required to make them valid.
//
//Thunk is a function that takes no args, returns no values, but draws the gl screen
//
//Rtt does the correct setup to prepare the texture for drawing, calls thunk() to draw it, then it restores the default frambuffer
//
// If filename is not "", then Rtt will save the contents of the framebuffer to the filename, followed by a number for each frame.
//
//i.e. frambuffer 0 will be active at the end of the call, so make sure you switch to the correct one before doing anymore drawing!  I should probably take that out, and figure out how to restore the currect framebuff
func Rtt(glctx gl.Context, rtt_frameBuff gl.Framebuffer, rtt_tex gl.Texture, texWidth, texHeight int, filename string, thunk Thunk) {
	if texWidth != texHeight {
		panic(fmt.Sprintf("You must provide equal width and height, you gave width: %v, height %v", texWidth, texHeight))
	}
	glctx.BindFramebuffer(gl.FRAMEBUFFER, rtt_frameBuff)
	checkGlError(glctx)
	glctx.Viewport(0, 0, texWidth, texHeight)
	checkGlError(glctx)
	glctx.ActiveTexture(gl.TEXTURE0)
	checkGlError(glctx)
	glctx.BindTexture(gl.TEXTURE_2D, rtt_tex)
	checkGlError(glctx)
	//draw here the content you want in the texture
	log.Printf("+Framebuffer status: %v\n", glctx.CheckFramebufferStatus(gl.FRAMEBUFFER))

	//rtt_tex is now a texture with the drawn content
	depthbuffer := glctx.CreateRenderbuffer()
	checkGlError(glctx)
	glctx.BindRenderbuffer(gl.RENDERBUFFER, depthbuffer)
	checkGlError(glctx)
	glctx.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT16, texWidth, texHeight)
	checkGlError(glctx)
	glctx.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthbuffer)
	checkGlError(glctx)
	//     glctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, rtt_tex, 0)

	glctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, rtt_tex, 0)
	checkGlError(glctx)
	status := glctx.CheckFramebufferStatus(gl.FRAMEBUFFER)
	log.Println("Framebuffer status: ", status)

	glctx.Enable(gl.BLEND)
	checkGlError(glctx)
	glctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	checkGlError(glctx)
	glctx.Enable(gl.DEPTH_TEST)
	checkGlError(glctx)
	glctx.DepthFunc(gl.LEQUAL)
	checkGlError(glctx)
	glctx.DepthMask(true)
	checkGlError(glctx)
	glctx.ClearColor(0, 0, 0, 1)
	checkGlError(glctx)
	//glctx.UseProgram(program) //FIXME - may cause graphics glitches
	//checkGlError(glctx)
	glctx.Clear(gl.DEPTH_BUFFER_BIT)
	checkGlError(glctx)
	glctx.Clear(gl.COLOR_BUFFER_BIT)
	checkGlError(glctx)
	thunk()

	glctx.Flush()
	checkGlError(glctx)
	glctx.GenerateMipmap(gl.TEXTURE_2D)
	checkGlError(glctx)

	if filename != "" {
		buff := CopyFrameBuff(glctx, rtt_frameBuff, texWidth, texHeight)
		checkGlError(glctx)
		SaveBuff(int(texWidth), int(texHeight), buff, fmt.Sprintf(filename+"_%04d.png", fname_int)) //FIXME - make the numbers an option
		fname_int += 1
	}
	glctx.BindTexture(gl.TEXTURE_2D, gl.Texture{0})
	checkGlError(glctx)
	//glctx.BindRenderbuffer(gl.FRAMEBUFFER, gl.Renderbuffer{0})
	//checkGlError(glctx)
	glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{0})
	checkGlError(glctx)
	glctx.DeleteRenderbuffer(depthbuffer) //FIXME - slow!
	//log.Println("Finished Rtt")
	//fmt.Printf("done \n")
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
