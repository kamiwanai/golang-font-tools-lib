// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// Head contains the parsed fields from the OpenType 'head' table.
type Head struct {
	// MajorVersion is the major version number.
	MajorVersion uint16
	// MinorVersion is the minor version number.
	MinorVersion uint16
	// FontRevision is the font revision as a Fixed-point number.
	FontRevision uint32
	// ChecksumAdjustment is the checksum adjustment.
	ChecksumAdjustment uint32
	// MagicNumber should be 0x5F0F3CF5.
	MagicNumber uint32
	// Flags is the font flags bitmask.
	Flags uint16
	// UnitsPerEm is the number of font design units per em.
	UnitsPerEm uint16
	// Created is the number of seconds since 1904-01-01 00:00:00.
	Created uint64
	// Modified is the number of seconds since 1904-01-01 00:00:00.
	Modified uint64
	// XMin is the minimum x coordinate across all glyph bounding boxes.
	XMin int16
	// YMin is the minimum y coordinate across all glyph bounding boxes.
	YMin int16
	// XMax is the maximum x coordinate across all glyph bounding boxes.
	XMax int16
	// YMax is the maximum y coordinate across all glyph bounding boxes.
	YMax int16
	// MacStyle is the Mac style bitmask.
	MacStyle uint16
	// LowestRecPPEM is the smallest readable size in pixels per em.
	LowestRecPPEM uint16
	// FontDirectionHint is the font direction hint.
	FontDirectionHint int16
	// IndexToLocFormat is 0 for short offsets, 1 for long.
	IndexToLocFormat int16
	// GlyphDataFormat is 0 for TrueType, 1 for CFF.
	GlyphDataFormat int16
}

// Head returns the parsed 'head' table.
func (font *Font) Head() (Head, error) {
	data, err := font.TableData("head")
	if err != nil {
		return Head{}, fmt.Errorf("head: %w", err)
	}
	if len(data) < 54 {
		return Head{}, fmt.Errorf("head table too short: %d bytes", len(data))
	}
	return Head{
		MajorVersion:       readU16(data, 0),
		MinorVersion:       readU16(data, 2),
		FontRevision:        readU32(data, 4),
		ChecksumAdjustment: readU32(data, 8),
		MagicNumber:         readU32(data, 12),
		Flags:               readU16(data, 16),
		UnitsPerEm:          readU16(data, 18),
		Created:             binary.BigEndian.Uint64(data[20:28]),
		Modified:            binary.BigEndian.Uint64(data[28:36]),
		XMin:                int16(readU16(data, 36)),
		YMin:                int16(readU16(data, 38)),
		XMax:                int16(readU16(data, 40)),
		YMax:                int16(readU16(data, 42)),
		MacStyle:            readU16(data, 44),
		LowestRecPPEM:       readU16(data, 46),
		FontDirectionHint:   int16(readU16(data, 48)),
		IndexToLocFormat:    int16(readU16(data, 50)),
		GlyphDataFormat:     int16(readU16(data, 52)),
	}, nil
}

// Hhea contains the parsed fields from the OpenType 'hhea' table.
type Hhea struct {
	// MajorVersion is the major version number.
	MajorVersion uint16
	// MinorVersion is the minor version number.
	MinorVersion uint16
	// Ascender is the typographic ascent in font units.
	Ascender int16
	// Descender is the typographic descent in font units.
	Descender int16
	// LineGap is the typographic line gap in font units.
	LineGap int16
	// AdvanceWidthMax is the maximum advance width in font units.
	AdvanceWidthMax uint16
	// MinLeftSideBearing is the minimum left side bearing in font units.
	MinLeftSideBearing int16
	// MinRightSideBearing is the minimum right side bearing in font units.
	MinRightSideBearing int16
	// XMaxExtent is the maximum horizontal extent in font units.
	XMaxExtent int16
	// CaretSlopeRise is the caret slope rise.
	CaretSlopeRise int16
	// CaretSlopeRun is the caret slope run.
	CaretSlopeRun int16
	// CaretOffset is the caret offset.
	CaretOffset int16
	// NumberOfHMetrics is the number of hMetric entries in hmtx.
	NumberOfHMetrics uint16
}

// Hhea returns the parsed 'hhea' table.
func (font *Font) Hhea() (Hhea, error) {
	data, err := font.TableData("hhea")
	if err != nil {
		return Hhea{}, fmt.Errorf("hhea: %w", err)
	}
	if len(data) < 36 {
		return Hhea{}, fmt.Errorf("hhea table too short: %d bytes", len(data))
	}
	return Hhea{
		MajorVersion:       readU16(data, 0),
		MinorVersion:       readU16(data, 2),
		Ascender:           int16(readU16(data, 4)),
		Descender:          int16(readU16(data, 6)),
		LineGap:            int16(readU16(data, 8)),
		AdvanceWidthMax:    readU16(data, 10),
		MinLeftSideBearing: int16(readU16(data, 12)),
		MinRightSideBearing: int16(readU16(data, 14)),
		XMaxExtent:         int16(readU16(data, 16)),
		CaretSlopeRise:     int16(readU16(data, 18)),
		CaretSlopeRun:      int16(readU16(data, 20)),
		CaretOffset:        int16(readU16(data, 22)),
		NumberOfHMetrics:   readU16(data, 34),
	}, nil
}

// HMetric is one entry from the 'hmtx' table: advance width + left side bearing.
type HMetric struct {
	// AdvanceWidth is the horizontal advance width in font units.
	AdvanceWidth uint16
	// LeftSideBearing is the left side bearing in font units.
	LeftSideBearing int16
}

// Hmtx returns the parsed 'hmtx' table as a slice of horizontal metrics.
//
// The hmtx table contains NumberOfHMetrics entries (from hhea) with full
// advanceWidth+lsb pairs, followed by leftSideBearing entries for remaining
// glyphs that share the last advance width.
func (font *Font) Hmtx() ([]HMetric, error) {
	hheaData, err := font.TableData("hhea")
	if err != nil {
		return nil, fmt.Errorf("hmtx: need hhea table: %w", err)
	}
	if len(hheaData) < 36 {
		return nil, fmt.Errorf("hhea table too short: %d bytes", len(hheaData))
	}
	numberOfHMetrics := int(readU16(hheaData, 34))

	maxpData, err := font.TableData("maxp")
	if err != nil {
		return nil, fmt.Errorf("hmtx: need maxp table: %w", err)
	}
	if len(maxpData) < 6 {
		return nil, fmt.Errorf("maxp table too short: %d bytes", len(maxpData))
	}
	numGlyphs := int(readU16(maxpData, 4))

	if numberOfHMetrics > numGlyphs {
		return nil, fmt.Errorf("hmtx: numberOfHMetrics %d exceeds numGlyphs %d", numberOfHMetrics, numGlyphs)
	}

	data, err := font.TableData("hmtx")
	if err != nil {
		return nil, fmt.Errorf("hmtx: %w", err)
	}

	expectedLen := numberOfHMetrics*4 + (numGlyphs-numberOfHMetrics)*2
	if len(data) < expectedLen {
		return nil, fmt.Errorf("hmtx table too short: need %d bytes, got %d", expectedLen, len(data))
	}

	metrics := make([]HMetric, numGlyphs)
	for i := 0; i < numberOfHMetrics; i++ {
		offset := i * 4
		metrics[i] = HMetric{
			AdvanceWidth:    readU16(data, offset),
			LeftSideBearing: int16(readU16(data, offset+2)),
		}
	}
	// Remaining glyphs share the last advance width.
	if numberOfHMetrics > 0 {
		lastAdvanceWidth := metrics[numberOfHMetrics-1].AdvanceWidth
		lsbOffset := numberOfHMetrics * 4
		for i := numberOfHMetrics; i < numGlyphs; i++ {
			metrics[i] = HMetric{
				AdvanceWidth:    lastAdvanceWidth,
				LeftSideBearing: int16(readU16(data, lsbOffset)),
			}
			lsbOffset += 2
		}
	}
	return metrics, nil
}

// AdvanceWidths returns a slice of advance widths for all glyphs in font units.
//
// This is a convenience method that reads hhea and hmtx and returns only the
// advance widths. The returned slice is indexed by glyph ID.
func (font *Font) AdvanceWidths() ([]uint16, error) {
	metrics, err := font.Hmtx()
	if err != nil {
		return nil, err
	}
	widths := make([]uint16, len(metrics))
	for i, m := range metrics {
		widths[i] = m.AdvanceWidth
	}
	return widths, nil
}

// KernPair is one kerning pair from the legacy 'kern' table.
type KernPair struct {
	// Left is the left-hand (first) glyph ID.
	Left uint16
	// Right is the right-hand (second) glyph ID.
	Right uint16
	// Value is the kerning value in font units (negative = move glyphs closer).
	Value int16
}

// KernSubtable is one subtable from the legacy 'kern' table.
type KernSubtable struct {
	// Format is the subtable format (0 = format 0, others parsed as unsupported).
	Format uint8
	// Horizontal is true when the subtable contains horizontal kerning values.
	Horizontal bool
	// HasMinimum is true when the subtable has minimum values (Apple AAT).
	HasMinimum bool
	// CrossStream is true when the subtable has cross-stream kerning values.
	CrossStream bool
	// Override is true when the subtable overrides previous subtable values.
	Override bool
	// Pairs contains the kerning pairs for format 0 subtables.
	// Non-format-0 subtables have an empty Pairs slice.
	Pairs []KernPair
}

// Kern returns the parsed 'kern' table as a slice of subtables.
//
// It supports Microsoft-style kern table version 0 with format 0 subtables.
// Apple AAT kern table version 1.0 and format 2 (class-based) subtables are
// skipped. The returned subtables are in the order they appear in the table.
func (font *Font) Kern() ([]KernSubtable, error) {
	data, err := font.TableData("kern")
	if err != nil {
		return nil, fmt.Errorf("kern: %w", err)
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("kern table too short: %d bytes", len(data))
	}

	version := readU16(data, 0)
	nTables := int(readU16(data, 2))

	// Apple AAT kern table version 1.0 has uint32 nTables at offset 4.
	// We only support Microsoft/OpenType version 0.
	if version != 0 {
		return nil, nil
	}

	subtables := make([]KernSubtable, 0, nTables)
	pos := 4
	for i := 0; i < nTables; i++ {
		if pos+6 > len(data) {
			return nil, fmt.Errorf("kern subtable %d header exceeds table", i)
		}
		subVersion := readU16(data, pos)
		subLength := int(readU16(data, pos+2))
		coverage := readU16(data, pos+4)

		subFormat := uint8(coverage & 0xFF)
		horizontal := coverage&0x8000 != 0
		hasMinimum := coverage&0x4000 != 0
		crossStream := coverage&0x2000 != 0
		override := coverage&0x1000 != 0

		subtableEnd := pos + subLength
		if subtableEnd > len(data) {
			return nil, fmt.Errorf("kern subtable %d exceeds table bounds", i)
		}

		sub := KernSubtable{
			Format:      subFormat,
			Horizontal:  horizontal,
			HasMinimum:  hasMinimum,
			CrossStream:  crossStream,
			Override:    override,
		}

		if subFormat == 0 && subVersion == 0 {
			pairsData := data[pos+6 : subtableEnd]
			if len(pairsData) < 8 {
				return nil, fmt.Errorf("kern subtable %d format 0 header too short", i)
			}
			nPairs := int(readU16(pairsData, 0))
			pairStart := 8 // skip searchRange, entrySelector, rangeShift
			if pairStart+nPairs*6 > len(pairsData) {
				return nil, fmt.Errorf("kern subtable %d pairs exceed subtable", i)
			}
			sub.Pairs = make([]KernPair, nPairs)
			for j := 0; j < nPairs; j++ {
				off := pairStart + j*6
				sub.Pairs[j] = KernPair{
					Left:  readU16(pairsData, off),
					Right: readU16(pairsData, off+2),
					Value: int16(readU16(pairsData, off+4)),
				}
			}
		}

		subtables = append(subtables, sub)
		pos = subtableEnd
	}
	return subtables, nil
}

// Panose is the 10-byte PANOSE classification from the OS/2 table.
type Panose struct {
	FamilyType      uint8
	SerifStyle      uint8
	Weight          uint8
	Proportion      uint8
	Contrast        uint8
	StrokeVariation uint8
	ArmStyle        uint8
	LetterForm      uint8
	Midline         uint8
	XHeight         uint8
}

// OS2 contains the parsed fields from the OpenType 'OS/2' table.
//
// Only fields present in the table version are populated; fields from newer
// versions are left at zero if the table is too short. The minimum
// parseable size is 78 bytes (version 0).
type OS2 struct {
	// Version is the OS/2 table version (0–5).
	Version uint16
	// XAvgCharWidth is the average character width.
	XAvgCharWidth int16
	// WeightClass is the font weight class (100–900).
	WeightClass uint16
	// WidthClass is the font width class (1–9).
	WidthClass uint16
	// Type is the embedding/licensing flags.
	Type uint16
	// SubscriptXSize is the subscript x size.
	SubscriptXSize int16
	// SubscriptYSize is the subscript y size.
	SubscriptYSize int16
	// SubscriptXOffset is the subscript x offset.
	SubscriptXOffset int16
	// SubscriptYOffset is the subscript y offset.
	SubscriptYOffset int16
	// SuperscriptXSize is the superscript x size.
	SuperscriptXSize int16
	// SuperscriptYSize is the superscript y size.
	SuperscriptYSize int16
	// SuperscriptXOffset is the superscript x offset.
	SuperscriptXOffset int16
	// SuperscriptYOffset is the superscript y offset.
	SuperscriptYOffset int16
	// StrikeoutSize is the strikeout stroke width.
	StrikeoutSize int16
	// StrikeoutPosition is the strikeout position.
	StrikeoutPosition int16
	// FamilyClass is the sFamilyClass (IBM font class + subclass).
	FamilyClass int16
	// Panose is the PANOSE classification.
	Panose Panose
	// UnicodeRange is the Unicode character range bits [4]uint32.
	UnicodeRange [4]uint32
	// VendorID is the font vendor tag (4 bytes).
	VendorID string
	// Selection is the fsSelection flags.
	Selection uint16
	// FirstCharIndex is the first Unicode character index.
	FirstCharIndex uint16
	// LastCharIndex is the last Unicode character index.
	LastCharIndex uint16
	// TypoAscender is the typographic ascent (version 1+).
	TypoAscender int16
	// TypoDescender is the typographic descent (version 1+).
	TypoDescender int16
	// TypoLineGap is the typographic line gap (version 1+).
	TypoLineGap int16
	// WinAscent is the Windows ascent.
	WinAscent uint16
	// WinDescent is the Windows descent.
	WinDescent uint16
	// CodePageRange is the code page range bits [2]uint32 (version 1+).
	CodePageRange [2]uint32
	// XHeight is the x-height (version 2+).
	XHeight int16
	// CapHeight is the cap height (version 2+).
	CapHeight int16
	// DefaultChar is the default character (version 2+).
	DefaultChar uint16
	// BreakChar is the word-break character (version 2+).
	BreakChar uint16
	// MaxContext is the max context (version 2+).
	MaxContext uint16
}

// OS2 returns the parsed 'OS/2' table.
//
// It supports versions 0 through 5. Fields introduced in later versions
// are left at their zero values when the table is too short to contain them.
func (font *Font) OS2() (OS2, error) {
	data, err := font.TableData("OS/2")
	if err != nil {
		return OS2{}, fmt.Errorf("OS/2: %w", err)
	}
	// Version 0 minimum: 78 bytes.
	if len(data) < 78 {
		return OS2{}, fmt.Errorf("OS/2 table too short: %d bytes", len(data))
	}

	os2 := OS2{
		Version:             readU16(data, 0),
		XAvgCharWidth:       int16(readU16(data, 2)),
		WeightClass:         readU16(data, 4),
		WidthClass:          readU16(data, 6),
		Type:                readU16(data, 8),
		SubscriptXSize:      int16(readU16(data, 10)),
		SubscriptYSize:      int16(readU16(data, 12)),
		SubscriptXOffset:    int16(readU16(data, 14)),
		SubscriptYOffset:    int16(readU16(data, 16)),
		SuperscriptXSize:    int16(readU16(data, 18)),
		SuperscriptYSize:    int16(readU16(data, 20)),
		SuperscriptXOffset:  int16(readU16(data, 22)),
		SuperscriptYOffset:  int16(readU16(data, 24)),
		StrikeoutSize:       int16(readU16(data, 26)),
		StrikeoutPosition:   int16(readU16(data, 28)),
		FamilyClass:         int16(readU16(data, 30)),
		Panose: Panose{
			FamilyType:      data[32],
			SerifStyle:      data[33],
			Weight:          data[34],
			Proportion:      data[35],
			Contrast:        data[36],
			StrokeVariation: data[37],
			ArmStyle:        data[38],
			LetterForm:      data[39],
			Midline:         data[40],
			XHeight:         data[41],
		},
		UnicodeRange: [4]uint32{
			readU32(data, 42),
			readU32(data, 46),
			readU32(data, 50),
			readU32(data, 54),
		},
		VendorID:       string(data[58:62]),
		Selection:      readU16(data, 62),
		FirstCharIndex: readU16(data, 64),
		LastCharIndex:  readU16(data, 66),
		// Offsets 68-77: present in all versions (v0+).
		TypoAscender:  int16(readU16(data, 68)),
		TypoDescender: int16(readU16(data, 70)),
		TypoLineGap:   int16(readU16(data, 72)),
		WinAscent:     readU16(data, 74),
		WinDescent:    readU16(data, 76),
	}

	// Code page range: version 1+ (offset 78-85, need 86 bytes).
	if len(data) >= 86 {
		os2.CodePageRange = [2]uint32{
			readU32(data, 78),
			readU32(data, 82),
		}
	}

	// Version 2+: xHeight, capHeight, defaultChar, breakChar, maxContext.
	// (offsets 86-95, need 96 bytes).
	if len(data) >= 96 {
		os2.XHeight = int16(readU16(data, 86))
		os2.CapHeight = int16(readU16(data, 88))
		os2.DefaultChar = readU16(data, 90)
		os2.BreakChar = readU16(data, 92)
		os2.MaxContext = readU16(data, 94)
	}

	return os2, nil
}

// GSUB returns the raw GSUB table data. Returns an error if the font has no
// GSUB table.
func (font *Font) GSUB() ([]byte, error) {
	return font.TableData("GSUB")
}

// GPOS returns the raw GPOS table data. Returns an error if the font has no
// GPOS table.
func (font *Font) GPOS() ([]byte, error) {
	return font.TableData("GPOS")
}

// GDEF returns the raw GDEF table data. Returns an error if the font has no
// GDEF table.
func (font *Font) GDEF() ([]byte, error) {
	return font.TableData("GDEF")
}

// FVAR returns the raw fvar (Font Variations) table data. Returns an error
// if the font has no fvar table.
func (font *Font) FVAR() ([]byte, error) {
	return font.TableData("fvar")
}

// Post contains the parsed fields from the OpenType 'post' table.
//
// ItalicAngle is a Fixed 16.16 value (signed int32 / 65536.0) that indicates
// the font's italic slant in degrees counter-clockwise from vertical.
// Zero means upright; negative means right-leaning italic (e.g., -12.0).
type Post struct {
	// ItalicAngle is the italic slant in degrees (Fixed 16.16).
	ItalicAngle float64
	// UnderlinePosition is the suggested top of underline in font units.
	UnderlinePosition int16
	// UnderlineThickness is the suggested underline thickness in font units.
	UnderlineThickness int16
	// IsFixedPitch is 1 if the font is monospaced, 0 otherwise.
	IsFixedPitch uint32
	// MinMemType42 is the minimum memory usage for Type42 download (version 2.0+).
	MinMemType42 uint32
	// MaxMemType42 is the maximum memory usage for Type42 download (version 2.0+).
	MaxMemType42 uint32
	// MinMemType1 is the minimum memory usage for Type1 download (version 2.0+).
	MinMemType1 uint32
	// MaxMemType1 is the maximum memory usage for Type1 download (version 2.0+).
	MaxMemType1 uint32
}

// Post returns the parsed 'post' table.
func (font *Font) Post() (Post, error) {
	data, err := font.TableData("post")
	if err != nil {
		return Post{}, fmt.Errorf("post: %w", err)
	}
	if len(data) < 32 {
		return Post{}, fmt.Errorf("post table too short: %d bytes", len(data))
	}

	// italicAngle is a Fixed 16.16 at offset 4: signed int32 / 65536.0.
	italicRaw := int32(binary.BigEndian.Uint32(data[4:8]))
	italicAngle := float64(italicRaw) / 65536.0

	p := Post{
		ItalicAngle:        italicAngle,
		UnderlinePosition:  int16(binary.BigEndian.Uint16(data[8:10])),
		UnderlineThickness: int16(binary.BigEndian.Uint16(data[10:12])),
		IsFixedPitch:       binary.BigEndian.Uint32(data[12:16]),
	}

	// Version 2.0+ fields start at offset 16.
	if len(data) >= 32 {
		p.MinMemType42 = binary.BigEndian.Uint32(data[16:20])
		p.MaxMemType42 = binary.BigEndian.Uint32(data[20:24])
		p.MinMemType1 = binary.BigEndian.Uint32(data[24:28])
		p.MaxMemType1 = binary.BigEndian.Uint32(data[28:32])
	}

	return p, nil
}

// MVAR returns the raw 'MVAR' (Metrics Variations) table bytes.
// Returns the table data and nil error on success.
func (font *Font) MVAR() ([]byte, error) {
	return font.TableData("MVAR")
}
