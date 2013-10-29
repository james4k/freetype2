package freetype2

import (
	"io/ioutil"
	"testing"
)

func TestSanityMetrics(t *testing.T) {
	lib, err := New()
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile("testdata/luximr.ttf")
	if err != nil {
		t.Fatal(err)
	}
	face, err := lib.NewFace(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = face.Pt(44, 300)
	if err != nil {
		t.Fatal(err)
	}
	var m Metrics
	err = face.Metrics(&m, ',')
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("%+v", m)
	//t.Log(face.MaxAdvance(), face.Height())
	// TODO: check for values we would expect from this font and glyph
}
