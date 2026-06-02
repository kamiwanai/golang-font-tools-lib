// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package fonttools

import (
	"os"
	"path/filepath"
	"testing"
)

// goldenFonts lists font files under testdata/ to validate.
// Add font files to testdata/ to enable these tests.
// Supported: .ttf, .otf, .woff
var goldenFonts []string

func init() {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		switch ext {
		case ".ttf", ".otf", ".woff":
			goldenFonts = append(goldenFonts, filepath.Join("testdata", e.Name()))
		}
	}
}

func TestGoldenFonts_Parse(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files in testdata/ — download Inter, Roboto, etc. to enable")
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			if font == nil {
				t.Fatal("ParseFile returned nil font")
			}
		})
	}
}

func TestGoldenFonts_GlyphCount(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			numGlyphs, err := font.NumGlyphs()
			if err != nil {
				t.Fatalf("NumGlyphs: %v", err)
			}
			if numGlyphs <= 0 {
				t.Fatalf("NumGlyphs = %d, want positive", numGlyphs)
			}
			// Reasonable upper bound for any font
			if numGlyphs > 100000 {
				t.Fatalf("NumGlyphs = %d, unreasonable", numGlyphs)
			}
			t.Logf("glyphs: %d", numGlyphs)
		})
	}
}

func TestGoldenFonts_GlyphOrder(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			glyphOrder, err := font.GlyphOrder()
			if err != nil {
				t.Fatalf("GlyphOrder: %v", err)
			}
			numGlyphs, _ := font.NumGlyphs()
			if len(glyphOrder) != numGlyphs {
				t.Fatalf("GlyphOrder length %d != NumGlyphs %d", len(glyphOrder), numGlyphs)
			}
			if len(glyphOrder) > 0 && glyphOrder[0] != ".notdef" {
				t.Fatalf("First glyph is %q, want .notdef", glyphOrder[0])
			}
		})
	}
}

func TestGoldenFonts_BestCmap(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			cmap, err := font.BestCmap()
			if err != nil {
				t.Fatalf("BestCmap: %v", err)
			}
			if len(cmap) == 0 {
				t.Fatal("BestCmap is empty" )
			}
			// Verify 'A' is mapped for most fonts
			if gid, ok := cmap['A']; ok {
				t.Logf("A -> glyph %d", gid)
			}
			t.Logf("cmap entries: %d", len(cmap))
		})
	}
}

func TestGoldenFonts_HeadTable(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			head, err := font.Head()
			if err != nil {
				t.Fatalf("Head: %v", err)
			}
			if head.UnitsPerEm == 0 {
				t.Fatal("UnitsPerEm is 0" )
			}
			t.Logf("unitsPerEm=%d indexToLocFormat=%d", head.UnitsPerEm, head.IndexToLocFormat)
		})
	}
}

func TestGoldenFonts_NameRecords(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			records, err := font.NameRecords()
			if err != nil {
				t.Fatalf("NameRecords: %v", err)
			}
			if len(records) == 0 {
				t.Fatal("NameRecords is empty" )
			}
			family := font.Name(1)
			if family == "" {
				t.Fatal("Family name (ID 1) is empty" )
			}
			t.Logf("family=%q records=%d", family, len(records))
		})
	}
}

func TestGoldenFonts_GPOSKerning(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			pairs, err := font.GPOSKerning()
			if err != nil {
				t.Fatalf("GPOSKerning: %v", err)
			}
			t.Logf("kerning pairs: %d", len(pairs))
		})
	}
}

func TestGoldenFonts_GSUBLookups(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			subs, err := font.GSUBSingleSubst()
			if err != nil {
				t.Fatalf("GSUBSingleSubst: %v", err)
			}
			ligs, err := font.GSUBLigatureSubst()
			if err != nil {
				t.Fatalf("GSUBLigatureSubst: %v", err)
			}
			t.Logf("single=%d ligature=%d", len(subs), len(ligs))
		})
	}
}

func TestGoldenFonts_FVAR(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			axes, err := font.FVARAxes()
			if err != nil {
				// FVAR is optional — not all fonts have it
				t.Logf("FVAR not present (expected for static fonts)")
				return
			}
			t.Logf("variable axes: %d", len(axes))
			for _, a := range axes {
				t.Logf("  axis: tag=%s min=%.1f max=%.1f default=%.1f", a.Tag, a.MinValue, a.MaxValue, a.DefaultValue)
			}
		})
	}
}

func TestGoldenFonts_GDEF(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files" )
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			classes, err := font.GDEFGlyphClasses()
			if err != nil {
				t.Logf("GDEF not present")
				return
			}
			t.Logf("GDEF classes: %d glyphs", len(classes))
		})
	}
}

func TestGoldenFonts_FVARInstances(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files")
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			fv, err := font.FVARTable()
			if err != nil {
				t.Logf("FVAR not present (expected for static fonts)")
				return
			}
			if len(fv.Axes) > 0 && len(fv.Instances) == 0 {
				t.Logf("variable font with %d axes but no named instances", len(fv.Axes))
			}
			t.Logf("axes=%d instances=%d", len(fv.Axes), len(fv.Instances))
			for _, inst := range fv.Instances {
				t.Logf("  instance: nameID=%d", inst.SubfamilyNameID)
			}
		})
	}
}

func TestGoldenFonts_GVAR(t *testing.T) {
	if len(goldenFonts) == 0 {
		t.Skip("No golden font files")
	}
	for _, path := range goldenFonts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			font, err := ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			gv, err := font.GVARTable()
			if err != nil {
				t.Logf("GVAR not present (expected for static fonts)")
				return
			}
			numGlyphs, _ := font.NumGlyphs()
			if len(gv.Glyphs) != numGlyphs {
				t.Errorf("GVAR GlyphCount=%d, font NumGlyphs=%d", len(gv.Glyphs), numGlyphs)
			}
			t.Logf("GVAR: glyphs=%d offsets=%d sharedTuples=%d",
				len(gv.Glyphs), len(gv.Glyphs), len(gv.SharedTuples))
		})
	}
}
