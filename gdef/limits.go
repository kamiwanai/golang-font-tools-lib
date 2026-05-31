package gdef

// MaxClassDefGlyphs limits the number of glyph entries expanded from a
// ClassDef table (both format 1 and format 2).
var MaxClassDefGlyphs int = 65536

// MaxClassDefRanges limits the number of range records in a ClassDef format 2.
var MaxClassDefRanges int = 65536

// MaxCoverageGlyphs limits the number of glyphs expanded from a Coverage
// format 2 range table.
var MaxCoverageGlyphs int = 65536

// MaxCoverageRanges limits the number of ranges in a Coverage format 2 table.
var MaxCoverageRanges int = 65536

// MaxAttachListGlyphs limits the number of glyphs in an AttachList.
var MaxAttachListGlyphs int = 65536

// MaxAttachPointCount limits the number of point indices in an AttachPoint.
var MaxAttachPointCount int = 65536

// MaxLigCaretCount limits the number of ligature glyphs in a LigCaretList.
var MaxLigCaretCount int = 65536

// MaxLigCaretValues limits the number of caret values per ligature glyph.
var MaxLigCaretValues int = 256