// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"testing"
)

func buildCOLRTable(version uint16, baseGlyphs []colorGlyphFixture, layers []colorLayerFixture) []byte {
	headerSize := 14
	baseRecSize := len(baseGlyphs) * 6
	layerRecSize := len(layers) * 4
	baseOff := headerSize
	layerOff := headerSize + baseRecSize

	data := make([]byte, headerSize+baseRecSize+layerRecSize)
	binary.BigEndian.PutUint16(data[0:2], version)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(baseGlyphs)))
	binary.BigEndian.PutUint32(data[4:8], uint32(baseOff))
	binary.BigEndian.PutUint32(data[8:12], uint32(layerOff))
	binary.BigEndian.PutUint16(data[12:14], uint16(len(layers)))

	for i, bg := range baseGlyphs {
		off := baseOff + i*6
		binary.BigEndian.PutUint16(data[off:off+2], bg.GlyphID)
		binary.BigEndian.PutUint16(data[off+2:off+4], bg.FirstLayer)
		binary.BigEndian.PutUint16(data[off+4:off+6], bg.NumLayers)
	}
	for i, l := range layers {
		off := layerOff + i*4
		binary.BigEndian.PutUint16(data[off:off+2], l.LayerGlyphID)
		binary.BigEndian.PutUint16(data[off+2:off+4], l.PaletteIndex)
	}
	return data
}

type colorGlyphFixture struct {
	GlyphID    uint16
	FirstLayer uint16
	NumLayers  uint16
}

type colorLayerFixture struct {
	LayerGlyphID uint16
	PaletteIndex uint16
}

func TestCOLRParsesSingleBaseGlyph(t *testing.T) {
	colr := buildCOLRTable(0,
		[]colorGlyphFixture{{GlyphID: 5, FirstLayer: 0, NumLayers: 2}},
		[]colorLayerFixture{
			{LayerGlyphID: 10, PaletteIndex: 0},
			{LayerGlyphID: 11, PaletteIndex: 1},
		},
	)
	data := buildFont(map[string][]byte{"COLR": colr})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	table, err := font.COLR()
	if err != nil {
		t.Fatalf("COLR: %v", err)
	}
	if table == nil {
		t.Fatal("COLR returned nil")
	}
	if table.Version != 0 {
		t.Fatalf("Version = %d, want 0", table.Version)
	}
	if len(table.BaseGlyphs) != 1 {
		t.Fatalf("BaseGlyphs count = %d, want 1", len(table.BaseGlyphs))
	}
	bg := table.BaseGlyphs[0]
	if bg.BaseGlyphID != 5 {
		t.Fatalf("BaseGlyphID = %d, want 5", bg.BaseGlyphID)
	}
	if bg.BaseGlyph != "glyph00005" {
		t.Fatalf("BaseGlyph = %q, want glyph00005", bg.BaseGlyph)
	}
	if len(bg.Layers) != 2 {
		t.Fatalf("Layers count = %d, want 2", len(bg.Layers))
	}
	if bg.Layers[0].LayerGlyphID != 10 || bg.Layers[0].PaletteIndex != 0 {
		t.Fatalf("Layer[0] = %+v, want {10 0}", bg.Layers[0])
	}
	if bg.Layers[1].LayerGlyphID != 11 || bg.Layers[1].PaletteIndex != 1 {
		t.Fatalf("Layer[1] = %+v, want {11 1}", bg.Layers[1])
	}
}

func TestCOLRWithNamesResolvesGlyphNames(t *testing.T) {
	colr := buildCOLRTable(0,
		[]colorGlyphFixture{{GlyphID: 2, FirstLayer: 0, NumLayers: 1}},
		[]colorLayerFixture{{LayerGlyphID: 3, PaletteIndex: 0}},
	)
	post := buildPostFormat2([]string{".notdef", "space", "A", "layerA"})
	maxp := buildMaxp(4)
	data := buildFont(map[string][]byte{
		"COLR": colr,
		"post": post,
		"maxp": maxp,
	})

	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	table, err := font.COLRWithNames()
	if err != nil {
		t.Fatalf("COLRWithNames: %v", err)
	}
	if table == nil {
		t.Fatal("COLRWithNames returned nil")
	}
	bg := table.BaseGlyphs[0]
	if bg.BaseGlyph != "A" {
		t.Fatalf("BaseGlyph = %q, want A", bg.BaseGlyph)
	}
}

func TestCOLRMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	table, err := font.COLR()
	if err != nil {
		t.Fatalf("COLR error: %v", err)
	}
	if table != nil {
		t.Fatal("COLR should be nil for missing table")
	}
}

func TestCOLRTruncated(t *testing.T) {
	data := buildFont(map[string][]byte{"COLR": {0, 0}})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.COLR()
	if err == nil {
		t.Fatal("COLR should fail for truncated table")
	}
}

func TestCOLRUnsupportedVersion(t *testing.T) {
	colr := buildCOLRTable(99,
		[]colorGlyphFixture{},
		[]colorLayerFixture{},
	)
	data := buildFont(map[string][]byte{"COLR": colr})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.COLR()
	if err == nil {
		t.Fatal("COLR should fail for unsupported version")
	}
}

func TestCOLRMultipleBaseGlyphs(t *testing.T) {
	colr := buildCOLRTable(0,
		[]colorGlyphFixture{
			{GlyphID: 1, FirstLayer: 0, NumLayers: 2},
			{GlyphID: 2, FirstLayer: 2, NumLayers: 1},
		},
		[]colorLayerFixture{
			{LayerGlyphID: 10, PaletteIndex: 0},
			{LayerGlyphID: 11, PaletteIndex: 1},
			{LayerGlyphID: 12, PaletteIndex: 2},
		},
	)
	data := buildFont(map[string][]byte{"COLR": colr})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	table, err := font.COLR()
	if err != nil {
		t.Fatalf("COLR: %v", err)
	}
	if len(table.BaseGlyphs) != 2 {
		t.Fatalf("BaseGlyphs count = %d, want 2", len(table.BaseGlyphs))
	}
	if len(table.BaseGlyphs[0].Layers) != 2 {
		t.Fatalf("First glyph layer count = %d, want 2", len(table.BaseGlyphs[0].Layers))
	}
	if len(table.BaseGlyphs[1].Layers) != 1 {
		t.Fatalf("Second glyph layer count = %d, want 1", len(table.BaseGlyphs[1].Layers))
	}
	if table.BaseGlyphs[1].Layers[0].LayerGlyphID != 12 {
		t.Fatalf("Second glyph layer = %+v, want glyph 12", table.BaseGlyphs[1].Layers[0])
	}
}

func TestCOLRV1ParsesV0BaseGlyphs(t *testing.T) {
	v1Header := make([]byte, 18)
	binary.BigEndian.PutUint16(v1Header[0:2], 1)
	binary.BigEndian.PutUint16(v1Header[2:4], 1)
	binary.BigEndian.PutUint32(v1Header[4:8], 18)
	binary.BigEndian.PutUint32(v1Header[8:12], 24)
	binary.BigEndian.PutUint16(v1Header[12:14], 1)
	binary.BigEndian.PutUint32(v1Header[14:18], 0)

	baseGlyphRec := make([]byte, 6)
	binary.BigEndian.PutUint16(baseGlyphRec[0:2], 5)
	binary.BigEndian.PutUint16(baseGlyphRec[2:4], 0)
	binary.BigEndian.PutUint16(baseGlyphRec[4:6], 1)

	layerRec := make([]byte, 4)
	binary.BigEndian.PutUint16(layerRec[0:2], 10)
	binary.BigEndian.PutUint16(layerRec[2:4], 0)

	colrData := append(v1Header, baseGlyphRec...)
	colrData = append(colrData, layerRec...)

	data := buildFont(map[string][]byte{"COLR": colrData})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	table, err := font.COLR()
	if err != nil {
		t.Fatalf("COLR v1: %v", err)
	}
	if table == nil {
		t.Fatal("COLR v1 returned nil")
	}
	if table.Version != 1 {
		t.Fatalf("Version = %d, want 1", table.Version)
	}
	if len(table.BaseGlyphs) != 1 || table.BaseGlyphs[0].BaseGlyphID != 5 {
		t.Fatal("COLR v1 failed to parse v0 base glyphs")
	}
	if len(table.BaseGlyphs) != 1 || table.BaseGlyphs[0].BaseGlyphID != 5 {
		t.Fatal("COLR v1 failed to parse v0 base glyphs")
	}
}
