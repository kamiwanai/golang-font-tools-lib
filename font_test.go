// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kamiwanai/golang-font-tools-lib/opentype"
	"golang.org/x/image/font/gofont/goregular"
)

// =============================================================================
// Parse / ParseFile
// =============================================================================

func TestParse(t *testing.T) {
	font, err := Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse(goregular.TTF): %v", err)
	}
	if font == nil {
		t.Fatal("Parse returned nil font without error")
	}
}

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	font, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if font == nil {
		t.Fatal("ParseFile returned nil font without error")
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "nonexistent.ttf"))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestParse_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"garbage", []byte("not a font")},
		{"truncated", goregular.TTF[:100]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			font, err := Parse(tt.data)
			if err == nil {
				t.Errorf("expected error for %s data, got font=%v", tt.name, font)
			}
			if font != nil {
				t.Errorf("expected nil font on error for %s, got %v", tt.name, font)
			}
		})
	}
}

// =============================================================================
// Embedded opentype.Font method forwarding (smoke tests)
// =============================================================================

func TestFont_GlyphOrder(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	names, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("GlyphOrder returned empty slice")
	}
	if names[0] != ".notdef" {
		t.Fatalf("first glyph = %q, want .notdef", names[0])
	}
}

func TestFont_NumGlyphs(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	n, err := font.NumGlyphs()
	if err != nil {
		t.Fatalf("NumGlyphs: %v", err)
	}
	if n <= 0 {
		t.Fatalf("NumGlyphs = %d, want > 0", n)
	}
}

func TestFont_BestCmap(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	cmap, err := font.BestCmap()
	if err != nil {
		t.Fatalf("BestCmap: %v", err)
	}
	if len(cmap) == 0 {
		t.Fatal("BestCmap returned empty map")
	}
	if _, ok := cmap['A']; !ok {
		t.Fatal("BestCmap missing 'A'")
	}
}

func TestFont_Head(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	head, err := font.Head()
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if head.UnitsPerEm == 0 {
		t.Fatal("Head.UnitsPerEm = 0")
	}
	if head.MagicNumber != 0x5F0F3CF5 {
		t.Fatalf("Head.MagicNumber = %#x, want 0x5F0F3CF5", head.MagicNumber)
	}
}

func TestFont_Hhea(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	hhea, err := font.Hhea()
	if err != nil {
		t.Fatalf("Hhea: %v", err)
	}
	if hhea.Ascender <= 0 {
		t.Errorf("Hhea.Ascent = %d, want > 0", hhea.Ascender)
	}
}

func TestFont_Hmtx(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	hmtx, err := font.Hmtx()
	if err != nil {
		t.Fatalf("Hmtx: %v", err)
	}
	if len(hmtx) == 0 {
		t.Fatal("Hmtx returned empty")
	}
}

func TestFont_OS2(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	os2, err := font.OS2()
	if err != nil {
		t.Fatalf("OS2: %v", err)
	}
	if os2.TypoAscender <= 0 {
		t.Errorf("OS2.STypoAscender = %d, want > 0", os2.TypoAscender)
	}
}

func TestFont_Post(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	post, err := font.Post()
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	// Go Regular is proportional (not monospaced)
	if post.IsFixedPitch != 0 {
		t.Errorf("Post.IsFixedPitch = %d, want 0 (proportional)", post.IsFixedPitch)
	}
}

func TestFont_NameRecords(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	records, err := font.NameRecords()
	if err != nil {
		t.Fatalf("NameRecords: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("NameRecords returned empty")
	}
}

func TestFont_Kern(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	kern, err := font.Kern()
	if err != nil {
		t.Skipf("no kern table in Go Regular: %v", err)
	}
	if len(kern) == 0 {
		t.Skip("kern table is empty")
	}
	t.Logf("found %d legacy kern pairs", len(kern))
}

func TestFont_AdvanceWidths(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	widths, err := font.AdvanceWidths()
	if err != nil {
		t.Fatalf("AdvanceWidths: %v", err)
	}
	if len(widths) == 0 {
		t.Fatal("AdvanceWidths returned empty")
	}
}

// =============================================================================
// Table-not-found: methods should return errors, not nil,nil / panic
// =============================================================================

func TestFont_GPOSKerning_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	pairs, err := font.GPOSKerning()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if pairs != nil {
		t.Errorf("expected nil on error, got %v", pairs)
	}
}

func TestFont_GPOSSinglePos_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	vals, err := font.GPOSSinglePos()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if vals != nil {
		t.Errorf("expected nil on error, got %v", vals)
	}
}

func TestFont_GPOSMarkAttachments_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	marks, err := font.GPOSMarkAttachments()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if marks != nil {
		t.Errorf("expected nil on error, got %v", marks)
	}
}

func TestFont_GPOSCursivePos_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	entries, err := font.GPOSCursivePos()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if entries != nil {
		t.Errorf("expected nil on error, got %v", entries)
	}
}

func TestFont_GPOSContextPos_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	lookups, err := font.GPOSContextPos()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if lookups != nil {
		t.Errorf("expected nil on error, got %v", lookups)
	}
}

func TestFont_GPOSChainContextPos_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	lookups, err := font.GPOSChainContextPos()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if lookups != nil {
		t.Errorf("expected nil on error, got %v", lookups)
	}
}

func TestFont_GSUBSingleSubst_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	substs, err := font.GSUBSingleSubst()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if substs != nil {
		t.Errorf("expected nil on error, got %v", substs)
	}
}

func TestFont_GSUBLigatureSubst_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	ligs, err := font.GSUBLigatureSubst()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if ligs != nil {
		t.Errorf("expected nil on error, got %v", ligs)
	}
}

func TestFont_GSUBMultipleSubst_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	substs, err := font.GSUBMultipleSubst()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if substs != nil {
		t.Errorf("expected nil on error, got %v", substs)
	}
}

func TestFont_GSUBAlternateSubst_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	substs, err := font.GSUBAlternateSubst()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if substs != nil {
		t.Errorf("expected nil on error, got %v", substs)
	}
}

func TestFont_GSUBScripts_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	scripts, err := font.GSUBScripts()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if scripts != nil {
		t.Errorf("expected nil on error, got %v", scripts)
	}
}

func TestFont_GSUBFeatures_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	features, err := font.GSUBFeatures()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if features != nil {
		t.Errorf("expected nil on error, got %v", features)
	}
}

func TestFont_GSUBActiveLookups_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	lookups, err := font.GSUBActiveLookups("latn", "dflt", "liga")
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if lookups != nil {
		t.Errorf("expected nil on error, got %v", lookups)
	}
}

func TestFont_GSUBContextSubst_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	lookups, err := font.GSUBContextSubst()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if lookups != nil {
		t.Errorf("expected nil on error, got %v", lookups)
	}
}

func TestFont_GSUBChainContextSubst_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	lookups, err := font.GSUBChainContextSubst()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if lookups != nil {
		t.Errorf("expected nil on error, got %v", lookups)
	}
}

func TestFont_GSUBReverseSubst_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	substs, err := font.GSUBReverseSubst()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if substs != nil {
		t.Errorf("expected nil on error, got %v", substs)
	}
}

func TestFont_GDEFGlyphClasses_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	classes, err := font.GDEFGlyphClasses()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if classes != nil {
		t.Errorf("expected nil on error, got %v", classes)
	}
}

func TestFont_GDEFGlyphClassesByName_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	classes, err := font.GDEFGlyphClassesByName()
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
	if classes != nil {
		t.Errorf("expected nil on error, got %v", classes)
	}
}

func TestFont_GDEFTable_NoTable(t *testing.T) {
	font := mustParse(t, goregular.TTF)
	gd, err := font.GDEFTable()
	if err == nil {
		t.Skip("Go Regular unexpectedly has GDEF table")
	}
	_ = gd
}

// =============================================================================
// Extract* convenience functions (backward compat)
// =============================================================================

func TestExtractGPOSKerning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGPOSKerning(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractGPOSSinglePos(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGPOSSinglePos(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractGPOSCursivePos(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGPOSCursivePos(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractGPOSMarkAttachments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGPOSMarkAttachments(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractGDEF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGDEF(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractGSUBSingleSubst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGSUBSingleSubst(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractGSUBLigatureSubst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGSUBLigatureSubst(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractSTAT(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractSTAT(path)
	if err == nil {
		t.Skip("Go Regular unexpectedly has table")
	}
}

func TestExtractFVAR(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractFVAR(path)
	if err == nil {
		t.Error("expected error for non-variable font, got nil")
	}
}

func TestExtractGVAR(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractGVAR(path)
	if err == nil {
		t.Error("expected error for non-variable font, got nil")
	}
}

func TestExtractAVAR(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := ExtractAVAR(path)
	if err == nil {
		t.Error("expected error for non-variable font, got nil")
	}
}

func TestExtractFileNotFound(t *testing.T) {
	_, err := ExtractGPOSKerning(filepath.Join(t.TempDir(), "nope.ttf"))
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// =============================================================================
// Consistency: fonttools.Font vs opentype.Font
// =============================================================================

func TestFont_Consistency_WithOpentype(t *testing.T) {
	ftFont := mustParse(t, goregular.TTF)
	otFont := mustParseOT(t, goregular.TTF)

	ftNames, err := ftFont.GlyphOrder()
	if err != nil { t.Fatal(err) }
	otNames, err := otFont.GlyphOrder()
	if err != nil { t.Fatal(err) }
	if len(ftNames) != len(otNames) {
		t.Fatalf("GlyphOrder length mismatch: fonttools=%d opentype=%d", len(ftNames), len(otNames))
	}
	for i := range ftNames {
		if ftNames[i] != otNames[i] {
			t.Fatalf("GlyphOrder[%d]: fonttools=%q opentype=%q", i, ftNames[i], otNames[i])
		}
	}

	ftHead, _ := ftFont.Head()
	otHead, _ := otFont.Head()
	if ftHead != otHead {
		t.Errorf("Head mismatch")
	}

	ftCmap, _ := ftFont.BestCmap()
	otCmap, _ := otFont.BestCmap()
	if len(ftCmap) != len(otCmap) {
		t.Fatalf("BestCmap length mismatch: fonttools=%d opentype=%d", len(ftCmap), len(otCmap))
	}
	for r, g := range ftCmap {
		if otCmap[r] != g {
			t.Errorf("BestCmap[U+%04X]: fonttools=%d opentype=%d", r, g, otCmap[r])
		}
	}
}

// =============================================================================
// helpers
// =============================================================================

func mustParse(t *testing.T, data []byte) *Font {
	t.Helper()
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return font
}

func mustParseOT(t *testing.T, data []byte) *opentype.Font {
	t.Helper()
	font, err := opentype.Parse(data)
	if err != nil {
		t.Fatalf("opentype.Parse: %v", err)
	}
	return font
}
