// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"testing"
)

// buildMinimalCFF2 creates a minimal CFF2 table with 2 glyphs and simple charstrings.
func buildMinimalCFF2() []byte {
	// CFF2 structure:
	// Offset 0-4: header (major=2, minor=0, hdrSize=5, topDictLen=2)
	// Offset 5-6: Top DICT: CharStrings offset = 9 (encoded as 9+139=148), operator 17
	// Offset 7-8: Global Subr INDEX: count=0
	// Offset 9-14: CharStrings INDEX: count=2, offSize=1, offsets: 1, 5, 12
	// Offset 15-26: object data: padding + obj0 + obj1
	data := []byte{
		2, 0, 5, 0, 2,       // header
		148, 17,               // Top DICT: CharStrings offset 9
		0, 0,                  // Global Subr: count=0
		0, 2, 1, 1, 5, 12,   // CharStrings INDEX: 2 entries
		0,                    // padding (offset 0)
		239, 239, 21, 14,    // obj0: rmoveto(100,100), endchar
		239, 239, 21,         // obj1: rmoveto(100,100)
		189, 189, 5,          //       rlineto(50,50)
		14,                    //       endchar
	}
	return data
}

func TestCFF2BBoxesBasic(t *testing.T) {
	data := buildMinimalCFF2()

	// Create a font with just a CFF2 table
	tag := "CFF2"
	tagBytes := make([]byte, 4)
	copy(tagBytes, tag)

	font := &Font{
		Tables: map[string]Table{
			tag: {Tag: tag, Offset: 0, Length: uint32(len(data))},
		},
		data: data,
	}

	bboxes, err := font.CFF2GlyphBBoxes()
	if err != nil {
		t.Fatalf("CFF2GlyphBBoxes error: %v", err)
	}
	if len(bboxes) != 2 {
		t.Fatalf("expected 2 bboxes, got %d", len(bboxes))
	}
	if bboxes[0].GlyphIndex != 0 {
		t.Fatalf("bbox[0].GlyphIndex = %d", bboxes[0].GlyphIndex)
	}
	if !bboxValid(int(bboxes[0].XMin), int(bboxes[0].YMin), int(bboxes[0].XMax), int(bboxes[0].YMax)) {
		t.Fatal("bbox[0] should be valid")
	}
	if !bboxValid(int(bboxes[1].XMin), int(bboxes[1].YMin), int(bboxes[1].XMax), int(bboxes[1].YMax)) {
		t.Fatal("bbox[1] should be valid")
	}
}

func TestCFF2BBoxesNoTable(t *testing.T) {
	font := &Font{
		Tables: map[string]Table{},
		data:   []byte{},
	}
	_, err := font.CFF2GlyphBBoxes()
	if err == nil {
		t.Fatal("expected error for missing CFF2 table")
	}
}

func TestCFF2BBoxesTruncated(t *testing.T) {
	font := &Font{
		Tables: map[string]Table{
			"CFF2": {Tag: "CFF2", Offset: 0, Length: 3},
		},
		data: []byte{2, 0, 5},
	}
	_, err := font.CFF2GlyphBBoxes()
	if err == nil {
		t.Fatal("expected error for truncated CFF2 table")
	}
}

func TestCFF2HasCFF2(t *testing.T) {
	data := buildMinimalCFF2()
	font := &Font{
		Tables: map[string]Table{
			"CFF2": {Tag: "CFF2", Offset: 0, Length: uint32(len(data))},
		},
		data: data,
	}
	if !font.HasCFF2() {
		t.Fatal("HasCFF2 should be true")
	}

	font2 := &Font{
		Tables: map[string]Table{},
		data:   []byte{},
	}
	if font2.HasCFF2() {
		t.Fatal("HasCFF2 should be false")
	}
}

func TestCFF2GlyphNamesDefault(t *testing.T) {
	data := buildMinimalCFF2()
	font := &Font{
		Tables: map[string]Table{
			"CFF2": {Tag: "CFF2", Offset: 0, Length: uint32(len(data))},
		},
		data: data,
	}
	names, err := font.CFF2GlyphNames()
	if err != nil {
		t.Fatalf("CFF2GlyphNames error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != ".notdef" {
		t.Fatalf("names[0] = %q, want .notdef", names[0])
	}
}
