<p align="center">
  <img src="./golang%20fonttools.png" alt="fonttools logo" width="420">
</p>

# fonttools

Small Go library for reading OpenType font data and exporting GPOS PairPos kerning.

This is not a full Go version of Python `fontTools`. It has a focused scope:
read font tables, read simple font metadata, read glyph names, read Unicode cmap data,
and extract GPOS kerning pairs.

## Install

```bash
go get github.com/kamiwanai/fonttools
```

Requires Go 1.22 or newer.

## Quick Start

Extract all supported GPOS kerning pairs from a font file:

```go
package main

import (
	"fmt"
	"log"

	"github.com/kamiwanai/fonttools"
)

func main() {
	pairs, err := fonttools.ExtractGPOSKerning("MyFont.ttf")
	if err != nil {
		log.Fatal(err)
	}

	for _, pair := range pairs {
		fmt.Printf("%s %s %d\n", pair.LeftGlyph, pair.RightGlyph, pair.Value)
	}
}
```

Example output:

```text
A V -80
A W -70
T o -40
```

Values are OpenType font units. Negative values usually move glyphs closer together.

## Ways To Use

### 1. Extract Kerning From A File

Use this when you only need kerning pairs.

```go
pairs, err := fonttools.ExtractGPOSKerning("MyFont.ttf")
if err != nil {
	return err
}

for _, pair := range pairs {
	fmt.Println(pair.LeftGlyph, pair.RightGlyph, pair.Value)
}
```

Each pair is a `gpos.PairValue`:

```go
type PairValue struct {
	LeftGlyph  string
	RightGlyph string
	Value      int
	Sources    []PairSource
}
```

`Sources` shows where the value came from when the same pair appears in more than one lookup.

### 2. Parse A Font Once, Then Reuse It

Use this when you need metadata and kerning from the same font.

```go
font, err := opentype.ParseFile("MyFont.ttf")
if err != nil {
	return err
}

family := font.Name(16, 1) // Typographic Family, then Family fallback.
glyphs, err := font.GlyphOrder()
if err != nil {
	return err
}
pairs, err := fonttools.ExtractGPOSKerningFromFont(font)
if err != nil {
	return err
}

fmt.Println(family, len(glyphs), len(pairs))
```

Imports:

```go
import (
	"github.com/kamiwanai/fonttools"
	"github.com/kamiwanai/fonttools/opentype"
)
```

### 3. Parse Font Bytes

Use this when the font comes from a database, HTTP upload, embedded file, or another source.

```go
data, err := os.ReadFile("MyFont.ttf")
if err != nil {
	return err
}

font, err := opentype.Parse(data)
if err != nil {
	return err
}

fmt.Println(font.Tables)
```

### 4. Read Font Metadata

Use the `opentype` package for table-level font information.

```go
font, err := opentype.ParseFile("MyFont.ttf")
if err != nil {
	return err
}

numGlyphs, err := font.NumGlyphs()
if err != nil {
	return err
}

glyphOrder, err := font.GlyphOrder()
if err != nil {
	return err
}

cmap, err := font.BestCmap()
if err != nil {
	return err
}

familyName := font.Name(16, 1)
subfamilyName := font.Name(17, 2)

fmt.Println(numGlyphs)
fmt.Println(glyphOrder[0])
fmt.Println(cmap['A'])
fmt.Println(familyName, subfamilyName)
```

Useful methods:

- `ParseFile(path)` reads a font from disk.
- `Parse(data)` reads a font from bytes.
- `TableData(tag)` returns raw table bytes, for example `"GPOS"` or `"name"`.
- `NumGlyphs()` returns the glyph count from `maxp`.
- `GlyphOrder()` returns glyph names by glyph ID.
- `BestCmap()` returns Unicode codepoint to glyph ID mappings.
- `NameRecords()` returns decoded `name` table records.
- `Name(ids...)` returns the first non-empty matching name.

### 5. Decode Raw GPOS Data

Use this when you already have a GPOS table and glyph order.

```go
font, err := opentype.ParseFile("MyFont.ttf")
if err != nil {
	return err
}

glyphOrder, err := font.GlyphOrder()
if err != nil {
	return err
}

gposData, err := font.TableData("GPOS")
if err != nil {
	return err
}

pairs, err := gpos.DecodePairPosLookups(gposData, glyphOrder)
if err != nil {
	return err
}

fmt.Println(len(pairs))
```

Import:

```go
import "github.com/kamiwanai/fonttools/gpos"
```

Advanced users can also decode a single PairPos subtable with:

- `gpos.DecodePairPosFormat1(data, glyphOrder)`
- `gpos.DecodePairPosFormat2(data, glyphOrder)`

### 6. Use The Command Line Example

The repository includes a small command that prints kerning pairs:

```bash
go run ./cmd/export-gpos-kerning MyFont.ttf
```

Build it:

```bash
go build ./cmd/export-gpos-kerning
```

Run it:

```bash
./export-gpos-kerning MyFont.ttf
```

Output format:

```text
leftGlyph rightGlyph value
```

## Common Use Cases

- Export kerning pairs from `.ttf` or `.otf` files.
- Check whether a font contains GPOS PairPos kerning.
- Compare kerning between font versions.
- Build a font QA or regression test tool.
- Convert GPOS kerning into another app-specific format.
- Inspect glyph order, Unicode cmap mappings, and name records.
- Read raw OpenType table bytes without writing a full parser.

## Supported

- `.ttf` and `.otf` OpenType table directories.
- Glyph order from `post` format 2.
- Stable fallback glyph names like `glyph00001` when real names are unavailable.
- Unicode `cmap` format 4 and format 12.
- Decoded `name` table records.
- GPOS lookup type 2 PairPos.
- GPOS extension lookup type 9 when it targets PairPos.
- PairPos format 1 glyph pair kerning.
- PairPos format 2 class kerning expansion.
- Repeated pair aggregation.
- Source tracking for repeated pair contributions.

## Not Supported Yet

- Full shaping.
- Contextual positioning extraction.
- CFF glyph-name extraction.
- Font collections (`.ttc` or `.otc`).
- All OpenType tables.
- All GPOS lookup types.
- Legacy `kern` table extraction.

Important `.otf` note: CFF-based fonts may not have `post` format 2 glyph names. In that case, glyph names can fall back to names like `glyph00042`.

## Safety Limits

The parser has configurable limits to avoid huge expansions from malformed or hostile fonts.

OpenType limits are in `github.com/kamiwanai/fonttools/opentype`:

- `MaxCmapGroupCodepoints`
- `MaxCmapFormat4Segments`
- `MaxPostCustomNames`
- `MaxNameRecords`

GPOS limits are in `github.com/kamiwanai/fonttools/gpos`:

- `MaxCoverageGlyphs`
- `MaxClassDefGlyphs`
- `MaxClassPairExpansion`
- `MaxLookupCount`

Most users should keep the defaults.

## License

This project is licensed under the GNU Affero General Public License v3.0 or later. See `LICENSE`.

You may use it under the AGPL, including for commercial work, if you follow the AGPL terms.

If you want to use this library in a closed-source product, proprietary service, or another project where AGPL compliance is not a good fit, contact `kamiwanaiii@gmail.com` about a commercial license.

This section is a plain-language summary, not legal advice.

## Development

Useful commands:

```bash
go test ./...
go vet ./...
gofmt -w .
go test -bench=. ./...
```

More development notes are in `DEVELOPMENT.md`.
