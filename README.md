<img src="golang%20fonttools.png" alt="golang fonttools" width="400">

# fonttools

Go library for reading OpenType font data. Not a port of Python fontTools —
narrow scope: parse tables, extract structured data, return Go types.

[![CI](https://github.com/kamiwanai/golang-font-tools-lib/actions/workflows/ci.yml/badge.svg)](https://github.com/kamiwanai/golang-font-tools-lib/actions/workflows/ci.yml)

```
go get github.com/kamiwanai/golang-font-tools-lib
```

Requires Go 1.22+. Depends only on `golang.org/x/image`.

---

## What it does

| Area | What you get | How |
|------|-------------|-----|
| **Font parsing** | Open/sfnt parse, table directory, raw table bytes | `opentype.ParseFile(path)` / `Parse(data)` |
| **Glyph order** | PostScript glyph names in order | `font.GlyphOrder()` |
| **Unicode mapping** | Best cmap, reverse cmap (glyph→rune) | `font.BestCmap()` / `font.ReverseCmap()` |
| **Glyph names** | CFF charset names (.otf) | `font.CFFGlyphNames()` |
| **Glyph bboxes** | TrueType glyf bbox | `font.GlyphBBoxes()` |
| **CFF bboxes** | CFF/CFF2 CharString bbox (Type 2 interpreter) | `font.CFFGlyphBBoxes()` / `font.CFF2GlyphBBoxes()` |
| **Advance widths** | Per-glyph advance widths | `font.AdvanceWidths()` |
| **Kerning (GPOS)** | PairPos pairs (fmt 1/2, extension type 9) | `fonttools.ExtractGPOSKerning(path)` |
| **Kerning (legacy)** | `kern` table format 0 | `font.Kern()` |
| **Single pos (GPOS)** | SinglePos lookup type 1 (fmt 1/2) | `fonttools.ExtractGPOSSinglePos(path)` |
| **Mark attachments** | MarkBase/MarkLig/MarkMark (types 4/5/6) | `fonttools.ExtractGPOSMarkAttachments(path)` |
| **Cursive pos (GPOS)** | Entry/exit anchors (type 3) | `fonttools.ExtractGPOSCursivePos(path)` |
| **Context pos (GPOS)** | ContextPos + ChainContextPos (types 7/8) | `fonttools.ExtractGPOSContextPos(path)` |
| **GSUB single subst** | Glyph→glyph substitution (type 1) | `fonttools.ExtractGSUBSingleSubst(path)` |
| **GSUB ligatures** | Ligature substitution (type 4) | `fonttools.ExtractGSUBLigatureSubst(path)` |
| **GSUB multiple subst** | One→many substitution (type 2) | `fonttools.ExtractGSUBMultipleSubst(path)` |
| **GSUB alternate subst** | One→pick-one substitution (type 3) | `fonttools.ExtractGSUBAlternateSubst(path)` |
| **GSUB context subst** | Context + ChainContext (types 5/6) | `fonttools.ExtractGSUBContextSubst(path)` / `ChainContext` |
| **GSUB reverse subst** | Reverse chaining (type 8) | `fonttools.ExtractGSUBReverseChain(path)` |
| **Feature mapping** | GPOS/GSUB Script→Lang→Feature→Lookups | `gsub.ActiveLookups(data, script, lang, feature)` |
| **GDEF** | GlyphClassDef, MarkAttachClass, AttachList, LigCaretList | `fonttools.ExtractGDEF(path)` |
| **fvar** | Variation axes + named instances | `fonttools.ExtractFVAR(path)` |
| **gvar** | Glyph variations (tuple headers + packed deltas) | `fonttools.ExtractGVAR(path)` |
| **avar** | Axis variations normalization map | `fonttools.ExtractAVAR(path)` |
| **STAT** | Style attributes (design axes + axis values) | `fonttools.ExtractSTAT(path)` |
| **MVAR** | Metrics variations (delta records) | `fonttools.ExtractMVAR(path)` |
| **HVAR** | Horizontal metrics variations (ItemVariationStore + DeltaSetIndexMap) | `fonttools.ExtractHVAR(path)` |
| **VVAR** | Vertical metrics variations | `fonttools.ExtractVVAR(path)` |
| **Font headers** | head, hhea, hmtx, OS/2, post | `font.Head()` / `Hhea()` / `Hmtx()` / `OS2()` / `Post()` |
| **Name records** | name table (all nameIDs, multi-language) | `font.NameRecords()` / `font.Name(ids...)` |
| **Font collections** | .ttc/.otc multi-font containers | `opentype.ParseCollection(data)` |

Every `Extract*` function has a `*FromFont` variant that takes an already-parsed `*opentype.Font`
to avoid re-parsing when you need multiple extractions.

---

## Package structure

| Package | Purpose |
|---------|---------|
| `fonttools` (root) | High-level `Extract*` convenience API |
| `opentype` | Low-level sfnt parser, table readers, Font type |
| `gpos` | GPOS table decoding (PairPos, SinglePos, MarkPos, CursivePos, ContextPos, features) |
| `gsub` | GSUB table decoding (Single, Ligature, Multiple, Alternate, Context, Reverse, features) |
| `gdef` | GDEF table decoding (GlyphClassDef, AttachList, LigCaretList, MarkAttachClassDef) |
| `fvar` | fvar table: variation axes and named instances |
| `gvar` | gvar table: glyph variation data |
| `avar` | avar table: axis variation normalization maps |
| `stat` | STAT table: style attributes |
| `mvar` | MVAR table: metrics variations |
| `hvar` | HVAR table: horizontal metrics variations |
| `vvar` | VVAR table: vertical metrics variations |

---

## Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/kamiwanai/golang-font-tools-lib"
	"github.com/kamiwanai/golang-font-tools-lib/gsub"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

func main() {
	// Kerning pairs
	pairs, err := fonttools.ExtractGPOSKerning("Font.ttf")
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range pairs {
		fmt.Printf("%s %s %d\n", p.LeftGlyph, p.RightGlyph, p.Value)
	}

	// Mark attachments (diacritics)
	marks, _ := fonttools.ExtractGPOSMarkAttachments("Font.ttf")
	for _, m := range marks {
		fmt.Printf("%s on %s: (%d,%d)\n", m.MarkGlyph, m.BaseGlyph, m.BaseAnchor.X, m.BaseAnchor.Y)
	}

	// GSUB ligatures, filtered by feature
	font, _ := opentype.ParseFile("Font.ttf")
	gsubData, _ := font.GSUB()
	glyphOrder, _ := font.GlyphOrder()

	lookups, _ := gsub.ActiveLookups(gsubData, "latn", "dflt", "liga")
	fmt.Printf("liga lookups: %v\n", lookups)

	ligs, _ := gsub.DecodeLigatureSubstLookups(gsubData, glyphOrder)
	for _, l := range ligs {
		fmt.Printf("%s -> %v\n", l.Ligature, l.Components)
	}

	// Variable font axes
	fv, _ := fonttools.ExtractFVAR("Variable.ttf")
	for _, ax := range fv.Axes {
		fmt.Printf("%s: %v-%v (default %v)\n", ax.Tag, ax.MinValue, ax.MaxValue, ax.DefaultValue)
	}
}
```

---

## Supported formats

| Format | Status |
|--------|--------|
| TrueType (.ttf) | ✓ |
| CFF OpenType (.otf) | ✓ |
| Font collections (.ttc/.otc) | ✓ |
| cmap format 4 | ✓ |
| cmap format 12 | ✓ |
| post format 2 | ✓ |
| CFF charset names | ✓ |
| CFF CharStrings bbox (Type 2 interpreter with subroutines) | ✓ |
| CFF2 bbox (FDArray, FDSelect) | ✓ |
| COLR v1 (color gradients) | ✗ |
| WOFF / WOFF2 | ✗ |
| Apple AAT (Feat, Morx) | ✗ |
| Bitmap glyphs (EBDT, CBDT) | ✗ |
| Shaping | ✗ |

---

## Limits

All parsers have configurable upper bounds to prevent DoS from malformed fonts.

| Package | Key limits |
|---------|-----------|
| `opentype` | `MaxCmapGroupCodepoints`, `MaxCmapFormat4Segments`, `MaxPostCustomNames`, `MaxNameRecords`, `MaxCollectionFonts`, `MaxCFF2FDRanges` |
| `gpos` | `MaxCoverageGlyphs`, `MaxClassDefGlyphs`, `MaxClassPairExpansion`, `MaxLookupCount`, `MaxMarkCount` |
| `gsub` | `MaxLookupCount`, `MaxCoverageGlyphs`, `MaxLigatureComponents`, `MaxLigatureSetEntries`, `MaxMultipleSubstSequences`, `MaxContextRuleExpansion`, `MaxClassDefGlyphs` |
| `gdef` | `MaxClassDefGlyphs`, `MaxClassDefRanges`, `MaxCoverageGlyphs`, `MaxCoverageRanges`, `MaxAttachListGlyphs`, `MaxLigCaretCount` |
| `fvar` | `MaxAxes`, `MaxInstances` |
| `avar` | `MaxAxes`, `MaxSegmentMaps` |
| `stat` | `MaxDesignAxes`, `MaxAxisValues` |
| `mvar` | `MaxValueRecords` |
| `hvar` / `vvar` | `MaxItemVariationDataCount`, `MaxVariationAxes`, `MaxVariationRegions`, `MaxDeltaItems`, `MaxDeltaSetIndexEntries` |

Each limit is exported as a package-level `var` — override at init if a font legitimately exceeds defaults.

---

## CLI

```
go run ./cmd/export-gpos-kerning <font.ttf>
```

Prints all GPOS kerning pairs to stdout.

---

## License

**AGPL-3.0** — see [LICENSE](LICENSE).

This means: you can use, modify, and distribute this library freely, but if you
incorporate it into a network-accessible service or a larger work, the entire
derivative work must be released under AGPL-3.0 with full source code.

### Commercial license

If AGPL-3.0 obligations do not fit your use case (closed-source product, SaaS
backend, embedded firmware, etc.), a commercial license is available.

| Scope | Price | What you get |
|-------|-------|-------------|
| Single product / company | $700/year | Proprietary license, no source disclosure obligation, 1 year of updates |
| OEM / redistribution | $2,000/year | Sublicensing rights, custom terms |

Contact: **inari1337@gmail.com**

### Enforcement

- Every source file carries a copyright header with license reference.
- This repository is monitored for unlicensed use via automated code scanning.
- Unauthorized use in a closed-source or network-accessible product without a
  commercial license will be pursued. First step is a cease-and-desist; if
  unresolved, formal legal action under AGPL-3.0 terms.
- Estimated damages basis: commercial license price × years of unlicensed use.

### Contributing

By submitting a pull request you agree to the terms in [CONTRIBUTING.md](CONTRIBUTING.md),
including the Contributor License Agreement (CLA) that grants the copyright holder
the right to relicense your contribution.
