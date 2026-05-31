// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

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
