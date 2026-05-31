// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package opentype

import (
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestGoRegularFixtureParity(t *testing.T) {
	font, err := Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		t.Fatalf("NumGlyphs returned error: %v", err)
	}
	if numGlyphs <= 0 {
		t.Fatalf("NumGlyphs = %d, want positive", numGlyphs)
	}

	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder returned error: %v", err)
	}
	if len(glyphOrder) != numGlyphs {
		t.Fatalf("GlyphOrder length = %d, want %d", len(glyphOrder), numGlyphs)
	}
	if glyphOrder[0] != ".notdef" {
		t.Fatalf("GlyphOrder[0] = %q, want .notdef", glyphOrder[0])
	}

	cmap, err := font.BestCmap()
	if err != nil {
		t.Fatalf("BestCmap returned error: %v", err)
	}
	glyphID := cmap['A']
	if glyphID == 0 {
		t.Fatal("BestCmap missing nonzero glyph for A")
	}
	if int(glyphID) >= len(glyphOrder) {
		t.Fatalf("A glyph ID = %d outside glyph order length %d", glyphID, len(glyphOrder))
	}

	if family := font.Name(1); family == "" {
		t.Fatal("Name(1) returned empty family name")
	}
	records, err := font.NameRecords()
	if err != nil {
		t.Fatalf("NameRecords returned error: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("NameRecords returned no records")
	}
}
