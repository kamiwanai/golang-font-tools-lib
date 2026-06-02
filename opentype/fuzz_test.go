// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte{})
	f.Add(makeValidFont())

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		if font == nil {
			t.Fatal("Parse returned nil font without error")
		}
		_ = font.ScalerType
		for tag := range font.Tables {
			tableData, _ := font.TableData(tag)
			_ = tableData
		}
	})
}

func FuzzBestCmap(f *testing.F) {
	f.Add(buildFont(map[string][]byte{
		"cmap": buildCmapFormat12(map[uint32]uint16{0x41: 1, 0x56: 2}),
	}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		cmap, err := font.BestCmap()
		if err != nil {
			return
		}
		_ = cmap
	})
}

func FuzzGlyphOrder(f *testing.F) {
	f.Add(buildFont(map[string][]byte{
		"maxp": buildMaxp(1),
	}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		names, err := font.GlyphOrder()
		if err != nil {
			return
		}
		_ = names
	})
}

func FuzzNameRecords(f *testing.F) {
	f.Add(buildFont(map[string][]byte{
		"name": buildNameTable(map[uint16]string{1: "Fixture"}),
	}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		records, err := font.NameRecords()
		if err != nil {
			return
		}
		_ = records
	})
}

func makeValidFont() []byte {
	return buildFont(map[string][]byte{
		"maxp": buildMaxp(1),
	})
}


func FuzzCOLR(f *testing.F) {
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{"COLR": {0, 0, 0, 1, 0, 0, 0, 14, 0, 0, 0, 18, 0, 1}}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		table, _ := font.COLR()
		_ = table
	})
}

func FuzzCPAL(f *testing.F) {
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{"CPAL": {0, 0, 0, 2, 0, 1, 0, 2, 0, 0, 0, 14}}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		table, _ := font.CPAL()
		_ = table
	})
}

func FuzzSVG(f *testing.F) {
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{"SVG ": {0, 0, 0, 0, 0, 10, 0, 0, 0, 0}}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		docs, _ := font.SVGDocuments()
		_ = docs
	})
}

func FuzzHead(f *testing.F) {
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{"head": buildHeadTable(0, 0, 1000)}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		head, _ := font.Head()
		_ = head
	})
}

func FuzzKern(f *testing.F) {
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{
		"kern": {0, 1, 0, 0},
	}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		kern, _ := font.Kern()
		_ = kern
	})
}

func FuzzPost(f *testing.F) {
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{
		"post": {0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		post, _ := font.Post()
		_ = post
	})
}

func FuzzCFF(f *testing.F) {
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{"CFF ": {1, 0, 0, 0}}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		names, _ := font.CFFGlyphNames()
		_ = names
	})
}

func FuzzGlyphBBox(f *testing.F) {
	glyphData := buildGlyph(0, 0, 100, 100)
	loca := buildLocaLong([]uint32{0, uint32(len(glyphData))})
	head := buildHeadTable(0, 1, 1000)
	f.Add([]byte{})
	f.Add(buildFont(map[string][]byte{
		"head": head,
		"maxp": buildMaxp(2),
		"loca": loca,
		"glyf": glyphData,
	}))

	f.Fuzz(func(t *testing.T, data []byte) {
		font, err := Parse(data)
		if err != nil {
			return
		}
		bboxes, _ := font.GlyphBBoxes()
		_ = bboxes
	})
}
