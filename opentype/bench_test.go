// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package opentype

import (
	"testing"
)

func BenchmarkParse(b *testing.B) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(256),
		"post": buildPostFormat2(makeNames(256)),
		"cmap": buildCmapFormat12(makeCmap(256)),
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(data)
	}
}

func BenchmarkGlyphOrder(b *testing.B) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(256),
		"post": buildPostFormat2(makeNames(256)),
	})
	font, _ := Parse(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = font.GlyphOrder()
	}
}

func BenchmarkBestCmap(b *testing.B) {
	data := buildFont(map[string][]byte{
		"cmap": buildCmapFormat12(makeCmap(2000)),
	})
	font, _ := Parse(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = font.BestCmap()
	}
}

func BenchmarkNameRecords(b *testing.B) {
	names := make(map[uint16]string)
	for i := uint16(1); i <= 50; i++ {
		names[i] = "NameRecord"
	}
	data := buildFont(map[string][]byte{
		"name": buildNameTable(names),
	})
	font, _ := Parse(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = font.NameRecords()
	}
}

func BenchmarkNumGlyphs(b *testing.B) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(256),
	})
	font, _ := Parse(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = font.NumGlyphs()
	}
}

func BenchmarkTableData(b *testing.B) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(256),
	})
	font, _ := Parse(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = font.TableData("maxp")
	}
}

func makeNames(count int) []string {
	names := make([]string, count)
	for i := range names {
		names[i] = "glyphName"
	}
	return names
}

func makeCmap(count int) map[uint32]uint16 {
	m := make(map[uint32]uint16, count)
	for i := 0; i < count; i++ {
		m[uint32(0x41+i)] = uint16(i + 1)
	}
	return m
}
