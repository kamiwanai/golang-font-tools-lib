package gpos

// MaxCoverageGlyphs limits the number of glyphs expanded from a Coverage
// format 2 range table. The default is large enough for any valid font.
var MaxCoverageGlyphs int = 65536

// MaxClassDefGlyphs limits the number of glyphs expanded from a ClassDef
// format 2 range table.
var MaxClassDefGlyphs int = 65536

// MaxClassPairExpansion limits the total number of kerning pairs produced from
// PairPos format 2 class expansion.
var MaxClassPairExpansion int = 10_000_000

// MaxLookupCount limits the number of lookups processed in GPOS.
var MaxLookupCount int = 65536
