package freetype2

/*
#cgo CFLAGS: -I/usr/local/include/freetype2
#cgo LDFLAGS: -lfreetype
#include <ft2build.h>
#include FT_FREETYPE_H
#include FT_GLYPH_H
#include "errors.h"
*/
import "C"

import (
	"fmt"
	"image"
	"image/draw"
	"reflect"
	"runtime"
	"unsafe"
)

func errstr(code C.FT_Error) string {
	return C.GoString(C.ft_error_string(code))
}

// Library represents a collection of font face.s
type Library struct {
	handle C.FT_Library
}

// New allocates a new Library.
func New() (*Library, error) {
	lib := &Library{}
	errno := C.FT_Init_FreeType(&lib.handle)
	if errno != 0 {
		return lib, fmt.Errorf("freetype2: %s", errstr(errno))
	}
	runtime.SetFinalizer(lib, (*Library).release)
	return lib, nil
}

func (l *Library) release() {
	C.FT_Done_FreeType(l.handle)
}

// NewFace creates a new face from the bytes of a font file. If
func (l *Library) NewFace(data []byte, faceIndex int) (*Face, error) {
	// FIXME: library reference ensures faces get cleaned up before the
	// library, but probably a bad idea to rely on this.
	face := &Face{lib: l}
	buffer := (*C.FT_Byte)(unsafe.Pointer(&data[0]))
	errno := C.FT_New_Memory_Face(l.handle,
		buffer, C.FT_Long(len(data)),
		C.FT_Long(faceIndex),
		&face.handle)
	if errno != 0 {
		return nil, fmt.Errorf("freetype2: %s", errstr(errno))
	}
	if face.handle.face_flags&C.FT_FACE_FLAG_SCALABLE == 0 {
		face.release()
		return nil, fmt.Errorf("freetype2: only scalable fonts supported")
	}
	errno = C.FT_Select_Charmap(face.handle, C.FT_ENCODING_UNICODE)
	if errno != 0 {
		face.release()
		return nil, fmt.Errorf("freetype2: only unicode charmap supported")
	}
	runtime.SetFinalizer(face, (*Face).release)
	face.init()
	return face, nil
}

// Face represents a typeface of a single style.
type Face struct {
	lib     *Library
	handle  C.FT_Face
	kerning bool
}

func (f *Face) init() {
	f.kerning = f.handle.face_flags&C.FT_FACE_FLAG_KERNING != 0
}

func (f *Face) release() {
	C.FT_Done_Face(f.handle)
}

// NumFaces returns the number of faces in the font we loaded from.
func (f *Face) NumFaces() int {
	return int(f.handle.num_faces)
}

// Pt sets character size in points, and resolution in dots-per-inch (typically 72).
func (f *Face) Pt(pt, dpi int) error {
	height := C.FT_F26Dot6(pt << 6)
	res := C.FT_UInt(dpi)
	errno := C.FT_Set_Char_Size(f.handle, 0, height, res, res)
	if errno != 0 {
		return fmt.Errorf("freetype2: %s", errstr(errno))
	}
	return nil
}

// Pt sets character size in pixels.
func (f *Face) Px(px int) error {
	height := C.FT_UInt(px)
	errno := C.FT_Set_Pixel_Sizes(f.handle, 0, height)
	if errno != 0 {
		return fmt.Errorf("freetype2: %s", errstr(errno))
	}
	return nil
}

// FixedWidth returns true if font face is monospaced, where each glyph
// occupies the same horizontal space.
func (f *Face) FixedWidth() bool {
	return f.handle.face_flags&C.FT_FACE_FLAG_FIXED_WIDTH != 0
}

// Bounds returns the overall bounding box for the font, in pixels.
func (f *Face) Bounds() image.Rectangle {
	return image.Rect(f.xpixels(int(f.handle.bbox.xMin)),
		f.ypixels(int(f.handle.bbox.yMin)),
		f.xpixels(int(f.handle.bbox.xMax)),
		f.ypixels(int(f.handle.bbox.yMax)))
}

func (f *Face) xpixels(val int) int {
	invEM := 1.0 / float64(f.handle.units_per_EM)
	xscale := float64(f.handle.size.metrics.x_ppem) * invEM
	return int((float64(val) * xscale))
}

func (f *Face) ypixels(val int) int {
	invEM := 1.0 / float64(f.handle.units_per_EM)
	yscale := float64(f.handle.size.metrics.y_ppem) * invEM
	return int((float64(val) * yscale))
}

// Height returns the default line spacing in pixels.
func (f *Face) Height() int {
	return f.ypixels(int(f.handle.height))
}

// MaxAdvance returns the maximum advance width in pixels.
func (f *Face) MaxAdvance() int {
	return f.xpixels(int(f.handle.max_advance_width))
}

// MaxVerticalAdvance returns the maximum advance height (for vertical layout)
// in pixels.
func (f *Face) MaxVerticalAdvance() int {
	return f.ypixels(int(f.handle.max_advance_height))
}

// Metrics represents measurements of a glyph in pixels.
type Metrics struct {
	Width, Height      int
	HorizontalBearingX int
	HorizontalBearingY int
	AdvanceWidth       int
	VerticalBearingX   int
	VerticalBearingY   int
	AdvanceHeight      int
}

// Metrics computes the pixel unit metrics for the glyph.
func (f *Face) Metrics(dst *Metrics, ch rune) error {
	errno := C.FT_Load_Char(f.handle, C.FT_ULong(ch), C.FT_LOAD_DEFAULT)
	if errno != 0 {
		return fmt.Errorf("freetype2: %s", errstr(errno))
	}
	m := &f.handle.glyph.metrics
	/*
		s := &f.handle.size.metrics
		invEM := 64.0 / float64(f.handle.units_per_EM)
		xscale := float64(s.x_ppem) * invEM
		yscale := float64(s.y_ppem) * invEM
		dst.Width = int(float64(m.width) * xscale)
		dst.Height = int(float64(m.height) * yscale)
		dst.HorizontalBearingX = int(float64(m.horiBearingX) * xscale)
		dst.HorizontalBearingY = int(float64(m.horiBearingY) * yscale)
		dst.AdvanceWidth = int(float64(m.horiAdvance) * xscale)
		dst.VerticalBearingX = int(float64(m.vertBearingX) * xscale)
		dst.VerticalBearingY = int(float64(m.vertBearingY) * yscale)
		dst.AdvanceHeight = int(float64(m.vertAdvance) * yscale)
	*/
	dst.Width = int(m.width >> 6)
	dst.Height = int(m.height >> 6)
	dst.HorizontalBearingX = int(m.horiBearingX >> 6)
	dst.HorizontalBearingY = int(m.horiBearingY >> 6)
	dst.AdvanceWidth = int(m.horiAdvance >> 6)
	dst.VerticalBearingX = int(m.vertBearingX >> 6)
	dst.VerticalBearingY = int(m.vertBearingY >> 6)
	dst.AdvanceHeight = int(m.vertAdvance >> 6)
	return nil
}

// Image directly copies the rendered bitmap data for the glyph into dst with
// its top-left corner at pt. Currently, only *image.Alpha is supported.
func (f *Face) Image(dst draw.Image, pt image.Point, ch rune) error {
	errno := C.FT_Load_Char(f.handle, C.FT_ULong(ch), C.FT_LOAD_RENDER)
	if errno != 0 {
		return fmt.Errorf("freetype2: %s", errstr(errno))
	}
	bitmap := &f.handle.glyph.bitmap
	if bitmap.pixel_mode != C.FT_PIXEL_MODE_GRAY || bitmap.num_grays != 256 {
		return fmt.Errorf("freetype2: unsupported pixel mode")
	}
	src := imageFromBitmap(bitmap)
	switch dst.(type) {
	case *image.Alpha:
		drawAlpha(dst.(*image.Alpha), pt, src)
	default:
		return fmt.Errorf("freetype2: unsupported dst type %T", dst)
	}
	return nil
}

func imageFromBitmap(bitmap *C.FT_Bitmap) image.Alpha {
	size := int(bitmap.rows * bitmap.width)
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(bitmap.buffer)),
		Len:  size,
		Cap:  size,
	}
	return image.Alpha{
		Pix:    *(*[]byte)(unsafe.Pointer(&hdr)),
		Stride: int(bitmap.width),
		Rect:   image.Rect(0, 0, int(bitmap.width), int(bitmap.rows)),
	}
}

func drawAlpha(dst *image.Alpha, pt image.Point, src image.Alpha) {
	for y := 0; y < src.Rect.Max.Y; y++ {
		dsty := y + pt.Y
		if dsty >= dst.Rect.Max.Y {
			break
		}
		offs := y * src.Stride
		copy(dst.Pix[dsty*dst.Stride+pt.X:], src.Pix[offs:offs+src.Stride])
	}
}
