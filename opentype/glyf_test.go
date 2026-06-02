// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"testing"
)

// buildLocaShort builds a short-format (uint16*2) loca table.
func buildLocaShort(offsets []uint16) []byte {
	data := make([]byte, len(offsets)*2)
	for i, off := range offsets {
		binary.BigEndian.PutUint16(data[i*2:], off)
	}
	return data
}

// buildLocaLong builds a long-format (uint32) loca table.
func buildLocaLong(offsets []uint32) []byte {
	data := make([]byte, len(offsets)*4)
	for i, off := range offsets {
		binary.BigEndian.PutUint32(data[i*4:], off)
	}
	return data
}

// buildGlyph builds a minimal glyf entry with numberOfContours + bbox.
func buildGlyph(xMin, yMin, xMax, yMax int16) []byte {
	data := make([]byte, 10)
	binary.BigEndian.PutUint16(data[0:2], 1) // numberOfContours = 1 (simple glyph)
	binary.BigEndian.PutUint16(data[2:4], uint16(xMin))
	binary.BigEndian.PutUint16(data[4:6], uint16(yMin))
	binary.BigEndian.PutUint16(data[6:8], uint16(xMax))
	binary.BigEndian.PutUint16(data[8:10], uint16(yMax))
	return data
}

func TestGlyphBBoxesShortLoca(t *testing.T) {
	// Build glyf: 3 glyphs with different bboxes
	// loca short: offsets = [0, 10, 20, 30] -> actual = [0, 20, 40, 60]
	glyf := make([]byte, 0)
	glyf = append(glyf, buildGlyph(0, -200, 500, 700)...)  // glyph 0
	glyf = append(glyf, buildGlyph(10, 0, 600, 800)...)    // glyph 1
	glyf = append(glyf, buildGlyph(-50, -100, 400, 600)...) // glyph 2

	// Short format: each 10-byte glyph = 5 loca units
	loca := buildLocaShort([]uint16{0, 5, 10, 15})
	maxp := make([]byte, 6)
	binary.BigEndian.PutUint32(maxp[0:4], 0x00010000)
	binary.BigEndian.PutUint16(maxp[4:6], 3) // numGlyphs

	head := buildHeadTable(0, 0, 1000) // IndexToLocFormat = 0 (short)

	data := buildFont(map[string][]byte{
		"maxp": maxp,
		"head": head,
		"loca": loca,
		"glyf": glyf,
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	bboxes, err := font.GlyphBBoxes()
	if err != nil {
		t.Fatalf("GlyphBBoxes returned error: %v", err)
	}
	if len(bboxes) != 3 {
		t.Fatalf("len(bboxes) = %d, want 3", len(bboxes))
	}

	// Glyph 0
	if bboxes[0].XMin != 0 || bboxes[0].YMin != -200 {
		t.Fatalf("glyph 0: XMin=%d YMin=%d, want XMin=0 YMin=-200", bboxes[0].XMin, bboxes[0].YMin)
	}
	if bboxes[0].XMax != 500 || bboxes[0].YMax != 700 {
		t.Fatalf("glyph 0: XMax=%d YMax=%d, want XMax=500 YMax=700", bboxes[0].XMax, bboxes[0].YMax)
	}

	// Glyph 2
	if bboxes[2].XMin != -50 || bboxes[2].YMin != -100 {
		t.Fatalf("glyph 2: XMin=%d YMin=%d, want -50, -100", bboxes[2].XMin, bboxes[2].YMin)
	}
}

func TestGlyphBBoxesLongLoca(t *testing.T) {
	// Same glyphs but with long loca format
	glyf := make([]byte, 0)
	glyf = append(glyf, buildGlyph(0, 0, 600, 800)...)
	glyf = append(glyf, buildGlyph(-100, -200, 700, 900)...)

	loca := buildLocaLong([]uint32{0, 10, 20})
	maxp := make([]byte, 6)
	binary.BigEndian.PutUint32(maxp[0:4], 0x00010000)
	binary.BigEndian.PutUint16(maxp[4:6], 2)

	head := buildHeadTable(0, 1, 1000) // IndexToLocFormat = 1 (long)

	data := buildFont(map[string][]byte{
		"maxp": maxp,
		"head": head,
		"loca": loca,
		"glyf": glyf,
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	bboxes, err := font.GlyphBBoxes()
	if err != nil {
		t.Fatalf("GlyphBBoxes returned error: %v", err)
	}
	if len(bboxes) != 2 {
		t.Fatalf("len(bboxes) = %d, want 2", len(bboxes))
	}
	if bboxes[1].XMax != 700 || bboxes[1].YMax != 900 {
		t.Fatalf("glyph 1: XMax=%d YMax=%d, want 700, 900", bboxes[1].XMax, bboxes[1].YMax)
	}
}

func TestGlyphBBoxesEmptyGlyph(t *testing.T) {
	// One real glyph + one empty glyph (offset == nextOffset)
	g0 := buildGlyph(0, 0, 500, 500)
	glyf := make([]byte, len(g0))
	copy(glyf, g0)

	loca := buildLocaLong([]uint32{0, 10, 10}) // glyph 1 empty
	maxp := make([]byte, 6)
	binary.BigEndian.PutUint32(maxp[0:4], 0x00010000)
	binary.BigEndian.PutUint16(maxp[4:6], 2)

	head := buildHeadTable(0, 1, 1000)

	data := buildFont(map[string][]byte{
		"maxp": maxp,
		"head": head,
		"loca": loca,
		"glyf": glyf,
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	bboxes, err := font.GlyphBBoxes()
	if err != nil {
		t.Fatalf("GlyphBBoxes returned error: %v", err)
	}

	// Glyph 0: has bbox
	if bboxes[0].XMax != 500 {
		t.Fatalf("glyph 0: XMax=%d, want 500", bboxes[0].XMax)
	}
	// Glyph 1: empty — all zeros
	if bboxes[1].XMin != 0 || bboxes[1].XMax != 0 || bboxes[1].YMin != 0 || bboxes[1].YMax != 0 {
		t.Fatalf("glyph 1: expected zero bbox, got XMin=%d XMax=%d", bboxes[1].XMin, bboxes[1].XMax)
	}
}
