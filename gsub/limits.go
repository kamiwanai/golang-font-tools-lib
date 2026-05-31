package gsub

// MaxLookupCount limits the number of lookups processed in GSUB.
var MaxLookupCount int = 65536

// MaxCoverageGlyphs limits the number of glyphs expanded from a Coverage
// format 2 range table.
var MaxCoverageGlyphs int = 65536

// MaxLigatureComponents limits the number of component glyphs in one ligature.
var MaxLigatureComponents int = 64

// MaxLigatureSetEntries limits the number of ligature entries per LigatureSet.
var MaxLigatureSetEntries int = 65536

// MaxMultipleSubstSequences limits the total number of substitution sequences
// across all MultipleSubst lookups.
var MaxMultipleSubstSequences int = 65536

// MaxContextRuleExpansion limits the total number of context rules expanded
// from class-based rules (format 2) to avoid combinatorial explosion.
var MaxContextRuleExpansion int = 65536

// MaxClassDefGlyphs limits the number of glyphs in a ClassDef table.
var MaxClassDefGlyphs int = 65536
