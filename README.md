<img src="golang%20fonttools.png" alt="golang fonttools" width="400">

# fonttools

Go library for reading OpenType font data. Not a port of Python fontTools —
narrow scope: parse tables, extract structured data, return Go types.

[![CI](https://github.com/kamiwanai/golang-font-tools-lib/actions/workflows/ci.yml/badge.svg)](https://github.com/kamiwanai/golang-font-tools-lib/actions/workflows/ci.yml)

```
go get github.com/kamiwanai/golang-font-tools-lib
```

Requires Go 1.22+. Runtime depends only on the Go standard library; tests use
`golang.org/x/image/font/gofont` for fixtures.

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
| **Kerning (GPOS)** | PairPos pairs (fmt 1/2, extension type 9) | `font.GPOSKerning()` |
| **Kerning (legacy)** | `kern` table format 0 | `font.Kern()` |
| **Single pos (GPOS)** | SinglePos lookup type 1 (fmt 1/2) | `font.GPOSSinglePos()` |
| **Mark attachments** | MarkBase/MarkLig/MarkMark (types 4/5/6) | `font.GPOSMarkAttachments()` |
| **Cursive pos (GPOS)** | Entry/exit anchors (type 3) | `font.GPOSCursivePos()` |
| **Context pos (GPOS)** | ContextPos + ChainContextPos (types 7/8) | `font.GPOSContextPos()` / `font.GPOSChainContextPos()` |
| **GSUB single subst** | Glyph→glyph substitution (type 1) | `font.GSUBSingleSubst()` |
| **GSUB ligatures** | Ligature substitution (type 4) | `font.GSUBLigatureSubst()` |
| **GSUB multiple subst** | One→many substitution (type 2) | `font.GSUBMultipleSubst()` |
| **GSUB alternate subst** | One→pick-one substitution (type 3) | `font.GSUBAlternateSubst()` |
| **GSUB context subst** | Context + ChainContext (types 5/6) | `font.GSUBContextSubst()` / `font.GSUBChainContextSubst()` |
| **GSUB reverse subst** | Reverse chaining (type 8) | `font.GSUBReverseSubst()` |
| **Feature mapping** | GPOS/GSUB Script→Lang→Feature→Lookups | `font.GSUBActiveLookups(script, lang, feature)` |
| **GDEF** | GlyphClassDef, MarkAttachClass, AttachList, LigCaretList | `font.GDEFTable()` |
| **fvar** | Variation axes + named instances | `font.FVARTable()` / `font.FVARAxes()` / `font.FVARInstances()` |
| **gvar** | Glyph variations (tuple headers + packed deltas) | `font.GVARTable()` |
| **avar** | Axis variations normalization map | `font.AVARTable()` |
| **STAT** | Style attributes (design axes + axis values) | `font.STATTable()` |
| **MVAR** | Metrics variations (delta records) | `font.MVARTable()` |
| **HVAR** | Horizontal metrics variations (ItemVariationStore + DeltaSetIndexMap) | `font.HVARTable()` |
| **VVAR** | Vertical metrics variations | `font.VVARTable()` |
| **Font headers** | head, hhea, hmtx, OS/2, post | `font.Head()` / `Hhea()` / `Hmtx()` / `OS2()` / `Post()` |
| **Name records** | name table (all nameIDs, multi-language) | `font.NameRecords()` / `font.Name(ids...)` |
| **Font collections** | .ttc/.otc multi-font containers | `opentype.ParseCollection(data)` |

Use `fonttools.ParseFile(path)` once, then call methods on the returned `*fonttools.Font`.
For quick one-liners, `Extract*` convenience functions (e.g. `ExtractGPOSKerning(path)`)
still exist — they parse internally and delegate to the same methods.

---

## Package structure

| Package | Purpose |
|---------|---------|
| `fonttools` (root) | High-level `Font` type with typed extractor methods; `Extract*` one-liners |
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
| `internal/otfeatures` | Shared OpenType script/lang/feature-list decoder (used by `gpos` + `gsub`) |
| `internal/variation` | Shared ItemVariationStore / DeltaSetIndexMap / F2DOT14 decoder (used by `hvar` + `vvar` + `mvar`) |

---

## Example

The recommended approach: parse once with `fonttools.ParseFile()`, then call any number of
extractor methods on the returned `*fonttools.Font`. This avoids repeated file I/O.

```go
package main

import (
	"fmt"
	"log"

	"github.com/kamiwanai/golang-font-tools-lib"
	"github.com/kamiwanai/golang-font-tools-lib/gsub"
)

func main() {
	font, err := fonttools.ParseFile("Font.ttf")
	if err != nil {
		log.Fatal(err)
	}

	// Kerning pairs (GPOS PairPos)
	pairs, err := font.GPOSKerning()
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range pairs {
		fmt.Printf("%s %s %d
", p.LeftGlyph, p.RightGlyph, p.Value)
	}

	// Mark attachments (GPOS MarkBasePos / MarkLigPos / MarkMarkPos)
	marks, _ := font.GPOSMarkAttachments()
	for _, m := range marks {
		fmt.Printf("%s on %s: (%d,%d)
", m.MarkGlyph, m.BaseGlyph, m.BaseAnchor.X, m.BaseAnchor.Y)
	}

	// GSUB ligatures, filtered by feature
	lookups, _ := font.GSUBActiveLookups("latn", "dflt", "liga")
	fmt.Printf("liga lookups: %v
", lookups)

	ligs, _ := font.GSUBLigatureSubst()
	for _, l := range ligs {
		fmt.Printf("%s -> %v
", l.Ligature, l.Components)
	}

	// Variable font axes
	axes, _ := font.FVARAxes()
	for _, ax := range axes {
		fmt.Printf("%s: %v-%v (default %v)
", ax.Tag, ax.MinValue, ax.MaxValue, ax.DefaultValue)
	}
}
```

One-liner convenience functions (`ExtractGPOSKerning`, `ExtractGDEF`, etc.) still exist
for quick single-extraction scripts — they parse the file internally and delegate to the
same methods. New code should prefer `ParseFile` + methods for multi-extraction efficiency.

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

Contact: **kamiwanaiii@gmail.com**

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
