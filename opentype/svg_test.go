// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"testing"
)

func buildSVGTable(docs []svgDocFixture) []byte {
	// Header: version(2) + docListOffset(4) + reserved(4) = 10
	// Doc list: numEntries(2) + entries*12 + svg strings
	headerSize := 10
	listHeader := 2
	n := len(docs)
	entriesSize := n * 12

	// Calculate SVG string offsets
	docListOffset := headerSize
	svgDataStart := docListOffset + listHeader + entriesSize
	svgData := make([]byte, 0)
	offsets := make([]int, n)
	lengths := make([]int, n)
	for i, d := range docs {
		offsets[i] = svgDataStart + len(svgData)
		lengths[i] = len(d.SVG)
		svgData = append(svgData, []byte(d.SVG)...)
	}

	totalSize := svgDataStart + len(svgData)
	data := make([]byte, totalSize)
	binary.BigEndian.PutUint16(data[0:2], 0)              // version
	binary.BigEndian.PutUint32(data[2:6], uint32(docListOffset))
	// reserved [6:10] is zero

	binary.BigEndian.PutUint16(data[docListOffset:docListOffset+2], uint16(n))
	for i := 0; i < n; i++ {
		off := docListOffset + 2 + i*12
		binary.BigEndian.PutUint16(data[off:off+2], docs[i].StartGlyphID)
		binary.BigEndian.PutUint16(data[off+2:off+4], docs[i].EndGlyphID)
		binary.BigEndian.PutUint32(data[off+4:off+8], uint32(offsets[i]))
		binary.BigEndian.PutUint32(data[off+8:off+12], uint32(lengths[i]))
	}
	copy(data[svgDataStart:], svgData)
	return data
}

type svgDocFixture struct {
	StartGlyphID uint16
	EndGlyphID   uint16
	SVG          string
}

func TestSVGParsesSingleDocument(t *testing.T) {
	svgData := buildSVGTable([]svgDocFixture{
		{StartGlyphID: 3, EndGlyphID: 3, SVG: "<svg xmlns='http://www.w3.org/2000/svg'><rect width='100' height='100'/></svg>"},
	})
	data := buildFont(map[string][]byte{"SVG ": svgData})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	docs, err := font.SVGDocuments()
	if err != nil {
		t.Fatalf("SVGDocuments: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("Docs count = %d, want 1", len(docs))
	}
	d := docs[0]
	if d.StartGlyphID != 3 || d.EndGlyphID != 3 {
		t.Fatalf("Glyph range = [%d,%d], want [3,3]", d.StartGlyphID, d.EndGlyphID)
	}
	expected := "<svg xmlns='http://www.w3.org/2000/svg'><rect width='100' height='100'/></svg>"
	if d.Data != expected {
		t.Fatalf("SVG data mismatch: got %q, want %q", d.Data, expected)
	}
}

func TestSVGMultipleDocuments(t *testing.T) {
	svgData := buildSVGTable([]svgDocFixture{
		{StartGlyphID: 1, EndGlyphID: 1, SVG: "<svg id='g1'/>"},
		{StartGlyphID: 2, EndGlyphID: 5, SVG: "<svg id='range'/>"},
		{StartGlyphID: 10, EndGlyphID: 10, SVG: "<svg id='g10'/>"},
	})
	data := buildFont(map[string][]byte{"SVG ": svgData})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	docs, err := font.SVGDocuments()
	if err != nil {
		t.Fatalf("SVGDocuments: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("Docs count = %d, want 3", len(docs))
	}
	if docs[0].StartGlyphID != 1 {
		t.Fatalf("Doc 0 start = %d, want 1", docs[0].StartGlyphID)
	}
	if docs[1].EndGlyphID != 5 {
		t.Fatalf("Doc 1 end = %d, want 5", docs[1].EndGlyphID)
	}
	if docs[2].StartGlyphID != 10 {
		t.Fatalf("Doc 2 start = %d, want 10", docs[2].StartGlyphID)
	}
}

func TestSVGMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	docs, err := font.SVGDocuments()
	if err != nil {
		t.Fatalf("SVGDocuments error: %v", err)
	}
	if docs != nil {
		t.Fatal("Expected nil docs for missing table")
	}
}

func TestSVGTruncated(t *testing.T) {
	data := buildFont(map[string][]byte{"SVG ": {0, 0}})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.SVGDocuments()
	if err == nil {
		t.Fatal("SVGDocuments should fail for truncated table")
	}
}

func TestSVGUnsupportedVersion(t *testing.T) {
	svgData := make([]byte, 10)
	binary.BigEndian.PutUint16(svgData[0:2], 99)
	data := buildFont(map[string][]byte{"SVG ": svgData})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.SVGDocuments()
	if err == nil {
		t.Fatal("SVGDocuments should fail for unsupported version")
	}
}

func TestSVGDocListOffsetOutOfBounds(t *testing.T) {
	svgData := make([]byte, 10)
	binary.BigEndian.PutUint32(svgData[2:6], 99999) // doc list way beyond table
	data := buildFont(map[string][]byte{"SVG ": svgData})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.SVGDocuments()
	if err == nil {
		t.Fatal("SVGDocuments should fail for out-of-bounds doc list offset")
	}
}

func TestSVGDocSVGDataOutOfBounds(t *testing.T) {
	// Build a doc list that points SVG data beyond table bounds
	docListOff := 10
	header := make([]byte, docListOff)
	binary.BigEndian.PutUint32(header[2:6], uint32(docListOff))

	docList := make([]byte, 14) // 2 (count) + 12 (one entry)
	binary.BigEndian.PutUint16(docList[0:2], 1)
	binary.BigEndian.PutUint16(docList[2:4], 1)    // startGlyphID
	binary.BigEndian.PutUint16(docList[4:6], 1)    // endGlyphID
	binary.BigEndian.PutUint32(docList[6:10], 9999) // SVG offset — way out
	binary.BigEndian.PutUint32(docList[10:14], 100) // SVG length

	data := append(header, docList...)

	fontData := buildFont(map[string][]byte{"SVG ": data})
	font, err := Parse(fontData)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.SVGDocuments()
	if err == nil {
		t.Fatal("SVGDocuments should fail when SVG offset is out of bounds")
	}
}

func TestSVGDocumentContentIntegrity(t *testing.T) {
	expectedSVG := "<svg><path d='M0,0 L10,10' fill='red'/></svg>"
	svgData := buildSVGTable([]svgDocFixture{
		{StartGlyphID: 5, EndGlyphID: 5, SVG: expectedSVG},
	})
	data := buildFont(map[string][]byte{"SVG ": svgData})
	font, _ := Parse(data)
	docs, err := font.SVGDocuments()
	if err != nil {
		t.Fatalf("SVGDocuments: %v", err)
	}
	if docs[0].Data != expectedSVG {
		t.Fatalf("SVG data = %q, want %q", docs[0].Data, expectedSVG)
	}
}
