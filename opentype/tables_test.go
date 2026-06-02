// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"testing"
)

func TestHeadParsesAllFields(t *testing.T) {
	data := buildFont(map[string][]byte{
		"head": buildHeadTable(42, 1, 2048),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	head, err := font.Head()
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}

	if head.UnitsPerEm != 2048 {
		t.Fatalf("UnitsPerEm = %d, want 2048", head.UnitsPerEm)
	}
	if head.Flags != 42 {
		t.Fatalf("Flags = %d, want 42", head.Flags)
	}
	if head.IndexToLocFormat != 1 {
		t.Fatalf("IndexToLocFormat = %d, want 1", head.IndexToLocFormat)
	}
	if head.XMin != -100 {
		t.Fatalf("XMin = %d, want -100", head.XMin)
	}
	if head.YMax != 900 {
		t.Fatalf("YMax = %d, want 900", head.YMax)
	}
}

func TestHeadRejectsMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.Head()
	if err == nil {
		t.Fatal("Head succeeded for missing table")
	}
}

func TestHeadRejectsTruncatedTable(t *testing.T) {
	data := buildFont(map[string][]byte{
		"head": []byte{0, 1, 0, 0}, // 4 bytes, too short
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.Head()
	if err == nil {
		t.Fatal("Head succeeded for truncated table")
	}
}

func TestHheaParsesAllFields(t *testing.T) {
	data := buildFont(map[string][]byte{
		"hhea": buildHheaTable(800, -200, 50, 1200, 350),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	hhea, err := font.Hhea()
	if err != nil {
		t.Fatalf("Hhea returned error: %v", err)
	}

	if hhea.Ascender != 800 {
		t.Fatalf("Ascender = %d, want 800", hhea.Ascender)
	}
	if hhea.Descender != -200 {
		t.Fatalf("Descender = %d, want -200", hhea.Descender)
	}
	if hhea.LineGap != 50 {
		t.Fatalf("LineGap = %d, want 50", hhea.LineGap)
	}
	if hhea.AdvanceWidthMax != 1200 {
		t.Fatalf("AdvanceWidthMax = %d, want 1200", hhea.AdvanceWidthMax)
	}
	if hhea.NumberOfHMetrics != 350 {
		t.Fatalf("NumberOfHMetrics = %d, want 350", hhea.NumberOfHMetrics)
	}
}

func TestHheaRejectsMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.Hhea()
	if err == nil {
		t.Fatal("Hhea succeeded for missing table")
	}
}

func TestHmtxParsesMetrics(t *testing.T) {
	numGlyphs := uint16(5)
	numHMetrics := uint16(3)
	// 3 full metrics: {500, 10}, {600, -5}, {700, 20}
	// 2 LSB-only entries: {15, 25} (use last advance=700)
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(numGlyphs),
		"hhea": buildHheaWithHMetrics(800, -200, 50, numHMetrics),
		"hmtx": buildHmtxTable(
			[]hmetricEntry{{adv: 500, lsb: 10}, {adv: 600, lsb: -5}, {adv: 700, lsb: 20}},
			[]int16{15, 25},
		),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	metrics, err := font.Hmtx()
	if err != nil {
		t.Fatalf("Hmtx returned error: %v", err)
	}

	if len(metrics) != 5 {
		t.Fatalf("metrics count = %d, want 5", len(metrics))
	}
	// Full entries
	if metrics[0].AdvanceWidth != 500 || metrics[0].LeftSideBearing != 10 {
		t.Fatalf("metrics[0] = %+v, want {500, 10}", metrics[0])
	}
	if metrics[1].AdvanceWidth != 600 || metrics[1].LeftSideBearing != -5 {
		t.Fatalf("metrics[1] = %+v, want {600, -5}", metrics[1])
	}
	if metrics[2].AdvanceWidth != 700 || metrics[2].LeftSideBearing != 20 {
		t.Fatalf("metrics[2] = %+v, want {700, 20}", metrics[2])
	}
	// LSB-only entries share last advance width
	if metrics[3].AdvanceWidth != 700 || metrics[3].LeftSideBearing != 15 {
		t.Fatalf("metrics[3] = %+v, want {700, 15}", metrics[3])
	}
	if metrics[4].AdvanceWidth != 700 || metrics[4].LeftSideBearing != 25 {
		t.Fatalf("metrics[4] = %+v, want {700, 25}", metrics[4])
	}
}

func TestHmtxAllGlyphsHaveFullMetrics(t *testing.T) {
	// When numberOfHMetrics == numGlyphs, no LSB-only entries
	numGlyphs := uint16(2)
	numHMetrics := uint16(2)
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(numGlyphs),
		"hhea": buildHheaWithHMetrics(800, -200, 50, numHMetrics),
		"hmtx": buildHmtxTable(
			[]hmetricEntry{{adv: 500, lsb: 10}, {adv: 600, lsb: -5}},
			nil,
		),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	metrics, err := font.Hmtx()
	if err != nil {
		t.Fatalf("Hmtx returned error: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("metrics count = %d, want 2", len(metrics))
	}
	if metrics[0].AdvanceWidth != 500 {
		t.Fatalf("metrics[0].AdvanceWidth = %d, want 500", metrics[0].AdvanceWidth)
	}
	if metrics[1].AdvanceWidth != 600 {
		t.Fatalf("metrics[1].AdvanceWidth = %d, want 600", metrics[1].AdvanceWidth)
	}
}

func TestAdvanceWidthsConvenience(t *testing.T) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(3),
		"hhea": buildHheaWithHMetrics(800, -200, 50, 3),
		"hmtx": buildHmtxTable(
			[]hmetricEntry{{adv: 500, lsb: 10}, {adv: 600, lsb: -5}, {adv: 700, lsb: 20}},
			nil,
		),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	widths, err := font.AdvanceWidths()
	if err != nil {
		t.Fatalf("AdvanceWidths returned error: %v", err)
	}
	want := []uint16{500, 600, 700}
	for i, w := range want {
		if widths[i] != w {
			t.Fatalf("widths[%d] = %d, want %d", i, widths[i], w)
		}
	}
}

func TestHmtxRejectsMissingHhea(t *testing.T) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(2),
		"hmtx": buildHmtxTable(
			[]hmetricEntry{{adv: 500, lsb: 10}, {adv: 600, lsb: -5}},
			nil,
		),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.Hmtx()
	if err == nil {
		t.Fatal("Hmtx succeeded without hhea table")
	}
}

func TestHmtxRejectsMissingMaxp(t *testing.T) {
	data := buildFont(map[string][]byte{
		"hhea": buildHheaWithHMetrics(800, -200, 50, 2),
		"hmtx": buildHmtxTable(
			[]hmetricEntry{{adv: 500, lsb: 10}, {adv: 600, lsb: -5}},
			nil,
		),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.Hmtx()
	if err == nil {
		t.Fatal("Hmtx succeeded without maxp table")
	}
}

// --- test helpers for tables ---

type hmetricEntry struct {
	adv uint16
	lsb int16
}

func buildHeadTable(flags uint16, indexToLocFormat int16, unitsPerEm uint16) []byte {
	data := make([]byte, 54)
	binary.BigEndian.PutUint16(data[0:2], 1)     // majorVersion
	binary.BigEndian.PutUint16(data[2:4], 0)      // minorVersion
	binary.BigEndian.PutUint32(data[4:8], 0x00010000) // fontRevision
	binary.BigEndian.PutUint32(data[8:12], 0)     // checksumAdjustment
	binary.BigEndian.PutUint32(data[12:16], 0x5F0F3CF5) // magicNumber
	binary.BigEndian.PutUint16(data[16:18], flags)
	binary.BigEndian.PutUint16(data[18:20], unitsPerEm)
	// created/modified = 0 (8 bytes each, offsets 20-28, 28-36)
	putInt16(data, 36, -100) // xMin
	putInt16(data, 38, -200) // yMin
	putInt16(data, 40, 800)  // xMax
	putInt16(data, 42, 900)  // yMax
	binary.BigEndian.PutUint16(data[44:46], 0)    // macStyle
	binary.BigEndian.PutUint16(data[46:48], 8)    // lowestRecPPEM
	binary.BigEndian.PutUint16(data[48:50], 2)    // fontDirectionHint
	binary.BigEndian.PutUint16(data[50:52], uint16(indexToLocFormat))
	binary.BigEndian.PutUint16(data[52:54], 0)    // glyphDataFormat
	return data
}

func putInt16(data []byte, offset int, value int16) {
	binary.BigEndian.PutUint16(data[offset:offset+2], uint16(value))
}

func buildHheaTable(ascender, descender, lineGap int16, advanceWidthMax uint16, numberOfHMetrics uint16) []byte {
	data := make([]byte, 36)
	binary.BigEndian.PutUint16(data[0:2], 1) // majorVersion
	binary.BigEndian.PutUint16(data[2:4], 0)  // minorVersion
	putInt16(data, 4, ascender)
	putInt16(data, 6, descender)
	putInt16(data, 8, lineGap)
	binary.BigEndian.PutUint16(data[10:12], advanceWidthMax)
	putInt16(data, 12, 0) // minLeftSideBearing
	putInt16(data, 14, 0) // minRightSideBearing
	putInt16(data, 16, 0) // xMaxExtent
	binary.BigEndian.PutUint16(data[18:20], 1)  // caretSlopeRise
	binary.BigEndian.PutUint16(data[20:22], 0)  // caretSlopeRun
	// 22-32 reserved, zeros
	binary.BigEndian.PutUint16(data[34:36], numberOfHMetrics)
	return data
}

func buildHheaWithHMetrics(ascender, descender, lineGap int16, numberOfHMetrics uint16) []byte {
	data := make([]byte, 36)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 0)
	putInt16(data, 4, ascender)
	putInt16(data, 6, descender)
	putInt16(data, 8, lineGap)
	binary.BigEndian.PutUint16(data[34:36], numberOfHMetrics)
	return data
}

func buildHmtxTable(fullMetrics []hmetricEntry, trailingLSB []int16) []byte {
	size := len(fullMetrics)*4 + len(trailingLSB)*2
	data := make([]byte, 0, size)
	for _, m := range fullMetrics {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint16(buf[0:2], m.adv)
		putInt16(buf, 2, m.lsb)
		data = append(data, buf...)
	}
	for _, lsb := range trailingLSB {
		buf := make([]byte, 2)
		putInt16(buf, 0, lsb)
		data = append(data, buf...)
	}
	return data
}
// --- kern table tests ---

type kernPairEntry struct {
	left  uint16
	right uint16
	value int16
}

type kernSubtableDef struct {
	format      uint8
	horizontal  bool
	crossStream bool
	override    bool
	pairs       []kernPairEntry
}

func buildKernTable(subtables []kernSubtableDef) []byte {
	nTables := len(subtables)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint16(buf[0:2], 0) // version
	binary.BigEndian.PutUint16(buf[2:4], uint16(nTables))
	for _, sub := range subtables {
		nPairs := len(sub.pairs)
		subLength := uint16(6 + 8 + nPairs*6)
		coverage := uint16(sub.format)
		if sub.horizontal {
			coverage |= 0x8000
		}
		if sub.crossStream {
			coverage |= 0x2000
		}
		if sub.override {
			coverage |= 0x1000
		}

		subBuf := make([]byte, 6+8+nPairs*6)
		binary.BigEndian.PutUint16(subBuf[0:2], 0)
		binary.BigEndian.PutUint16(subBuf[2:4], subLength)
		binary.BigEndian.PutUint16(subBuf[4:6], coverage)
		binary.BigEndian.PutUint16(subBuf[6:8], uint16(nPairs))
		// searchRange, entrySelector, rangeShift left as 0
		for j, p := range sub.pairs {
			off := 14 + j*6
			binary.BigEndian.PutUint16(subBuf[off:off+2], p.left)
			binary.BigEndian.PutUint16(subBuf[off+2:off+4], p.right)
			putInt16(subBuf, off+4, p.value)
		}
		buf = append(buf, subBuf...)
	}
	return buf
}

func TestKernParsesFormat0Pairs(t *testing.T) {
	data := buildFont(map[string][]byte{
		"kern": buildKernTable([]kernSubtableDef{
			{
				format:     0,
				horizontal: true,
				pairs: []kernPairEntry{
					{left: 1, right: 2, value: -50},
					{left: 3, right: 4, value: -30},
					{left: 5, right: 6, value: 10},
				},
			},
		}),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	subs, err := font.Kern()
	if err != nil {
		t.Fatalf("Kern returned error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("subtables count = %d, want 1", len(subs))
	}
	sub := subs[0]
	if sub.Format != 0 {
		t.Fatalf("Format = %d, want 0", sub.Format)
	}
	if !sub.Horizontal {
		t.Fatal("Horizontal = false, want true")
	}
	if len(sub.Pairs) != 3 {
		t.Fatalf("pairs count = %d, want 3", len(sub.Pairs))
	}
	if sub.Pairs[0] != (KernPair{Left: 1, Right: 2, Value: -50}) {
		t.Fatalf("pairs[0] = %+v, want {1, 2, -50}", sub.Pairs[0])
	}
	if sub.Pairs[1] != (KernPair{Left: 3, Right: 4, Value: -30}) {
		t.Fatalf("pairs[1] = %+v, want {3, 4, -30}", sub.Pairs[1])
	}
	if sub.Pairs[2] != (KernPair{Left: 5, Right: 6, Value: 10}) {
		t.Fatalf("pairs[2] = %+v, want {5, 6, 10}", sub.Pairs[2])
	}
}

func TestKernMultipleSubtables(t *testing.T) {
	data := buildFont(map[string][]byte{
		"kern": buildKernTable([]kernSubtableDef{
			{
				format:     0,
				horizontal: true,
				pairs: []kernPairEntry{
					{left: 1, right: 2, value: -40},
				},
			},
			{
				format:     0,
				horizontal: true,
				override:   true,
				pairs: []kernPairEntry{
					{left: 3, right: 4, value: -20},
				},
			},
		}),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	subs, err := font.Kern()
	if err != nil {
		t.Fatalf("Kern returned error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("subtables count = %d, want 2", len(subs))
	}
	if !subs[0].Horizontal {
		t.Fatal("subtable 0 Horizontal = false, want true")
	}
	if subs[1].Override != true {
		t.Fatal("subtable 1 Override = false, want true")
	}
}

func TestKernRejectsMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.Kern()
	if err == nil {
		t.Fatal("Kern succeeded for missing table")
	}
}

func TestKernRejectsTruncatedSubtable(t *testing.T) {
	kernData := make([]byte, 4)
	binary.BigEndian.PutUint16(kernData[0:2], 0) // version
	binary.BigEndian.PutUint16(kernData[2:4], 1)  // nTables = 1

	data := buildFont(map[string][]byte{
		"kern": kernData,
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.Kern()
	if err == nil {
		t.Fatal("Kern succeeded for truncated subtable")
	}
}

func TestKernSkipsAppleAATVersion(t *testing.T) {
	kernData := []byte{0, 1, 0, 0} // version=1, nTables=0
	data := buildFont(map[string][]byte{
		"kern": kernData,
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	subs, err := font.Kern()
	if err != nil {
		t.Fatalf("Kern returned error: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("Apple AAT kern should return empty subtables, got %d", len(subs))
	}
}

func TestKernSkipsFormat2Subtable(t *testing.T) {
	kernData := make([]byte, 18)
	binary.BigEndian.PutUint16(kernData[0:2], 0) // version
	binary.BigEndian.PutUint16(kernData[2:4], 1)  // nTables = 1
	binary.BigEndian.PutUint16(kernData[4:6], 0)  // subtable version
	binary.BigEndian.PutUint16(kernData[6:8], 14) // subtable length
	binary.BigEndian.PutUint16(kernData[8:10], 0x8002) // coverage: horizontal + format 2
	binary.BigEndian.PutUint16(kernData[10:12], 0) // nPairs
	binary.BigEndian.PutUint16(kernData[12:14], 0) // searchRange
	binary.BigEndian.PutUint16(kernData[14:16], 0) // entrySelector
	binary.BigEndian.PutUint16(kernData[16:18], 0) // rangeShift

	data := buildFont(map[string][]byte{
		"kern": kernData,
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	subs, err := font.Kern()
	if err != nil {
		t.Fatalf("Kern returned error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 subtable, got %d", len(subs))
	}
	if subs[0].Format != 2 {
		t.Fatalf("Format = %d, want 2", subs[0].Format)
	}
	if len(subs[0].Pairs) != 0 {
		t.Fatalf("format 2 subtable should have empty Pairs, got %d", len(subs[0].Pairs))
	}
}

func TestKernCoverageFlags(t *testing.T) {
	data := buildFont(map[string][]byte{
		"kern": buildKernTable([]kernSubtableDef{
			{
				format:      0,
				horizontal:  true,
				crossStream: true,
				override:    true,
				pairs: []kernPairEntry{
					{left: 1, right: 2, value: -5},
				},
			},
		}),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	subs, err := font.Kern()
	if err != nil {
		t.Fatalf("Kern returned error: %v", err)
	}
	sub := subs[0]
	if !sub.Horizontal {
		t.Fatal("Horizontal = false, want true")
	}
	if !sub.CrossStream {
		t.Fatal("CrossStream = false, want true")
	}
	if !sub.Override {
		t.Fatal("Override = false, want true")
	}
}

// --- OS/2 table tests ---

func buildOS2Table(weightClass, widthClass uint16, panoseByte byte, typoAscender, typoDescender, typoLineGap int16, winAscent, winDescent uint16) []byte {
	// Build a version 4 OS/2 table (96 bytes)
	data := make([]byte, 96)
	putInt16(data, 0, 4)    // version
	putInt16(data, 2, 500)  // xAvgCharWidth
	binary.BigEndian.PutUint16(data[4:6], weightClass)
	binary.BigEndian.PutUint16(data[6:8], widthClass)
	putInt16(data, 8, 0)    // fsType
	putInt16(data, 10, 650) // ySubscriptXSize
	putInt16(data, 12, 600) // ySubscriptYSize
	putInt16(data, 14, 0)   // ySubscriptXOffset
	putInt16(data, 16, 50)  // ySubscriptYOffset
	putInt16(data, 18, 650) // ySuperscriptXSize
	putInt16(data, 20, 600) // ySuperscriptYSize
	putInt16(data, 22, 0)   // ySuperscriptXOffset
	putInt16(data, 24, 150) // ySuperscriptYOffset
	putInt16(data, 26, 50)  // yStrikeoutSize
	putInt16(data, 28, 300) // yStrikeoutPosition
	putInt16(data, 30, 0)   // sFamilyClass
	// Panose: 10 bytes at offset 32
	for i := 32; i < 42; i++ {
		data[i] = panoseByte + byte(i-32)
	}
	// UnicodeRange: 16 bytes at offset 42
	binary.BigEndian.PutUint32(data[42:46], 0x00000001) // bit 0 = Basic Latin
	binary.BigEndian.PutUint32(data[46:50], 0)
	binary.BigEndian.PutUint32(data[50:54], 0)
	binary.BigEndian.PutUint32(data[54:58], 0)
	// achVendID: 4 bytes at offset 58
	copy(data[58:62], "GOOG")
	binary.BigEndian.PutUint16(data[62:64], 0x0040) // fsSelection: REGULAR
	binary.BigEndian.PutUint16(data[64:66], 0x0020) // usFirstCharIndex
	binary.BigEndian.PutUint16(data[66:68], 0xFFFD) // usLastCharIndex
	// Version 1+ fields at offset 68+
	putInt16(data, 68, typoAscender)
	putInt16(data, 70, typoDescender)
	putInt16(data, 72, typoLineGap)
	binary.BigEndian.PutUint16(data[74:76], winAscent)
	binary.BigEndian.PutUint16(data[76:78], winDescent)
	// CodePageRange: 8 bytes at offset 78
	binary.BigEndian.PutUint32(data[78:82], 0x00000001) // Latin 1
	binary.BigEndian.PutUint32(data[82:86], 0)
	// Version 2+ fields at offset 86+
	putInt16(data, 86, 530) // sxHeight
	putInt16(data, 88, 720)  // sCapHeight
	binary.BigEndian.PutUint16(data[90:92], 0x0020) // usDefaultChar
	binary.BigEndian.PutUint16(data[92:94], 0x0020) // usBreakChar
	binary.BigEndian.PutUint16(data[94:96], 0)      // usMaxContext
	return data
}

func buildOS2TableV0(weightClass, widthClass uint16) []byte {
	// Build a version 0 OS/2 table (78 bytes)
	// v0 has all fields through offset 77: typo metrics and win ascent/descent.
	// Only ulCodePageRange (v1+) and v2+ fields are absent.
	data := make([]byte, 78)
	putInt16(data, 0, 0) // version 0
	putInt16(data, 2, 500)
	binary.BigEndian.PutUint16(data[4:6], weightClass)
	binary.BigEndian.PutUint16(data[6:8], widthClass)
	putInt16(data, 8, 0)
	for i := 32; i < 42; i++ {
		data[i] = 0
	}
	copy(data[58:62], "MS  ")
	binary.BigEndian.PutUint16(data[62:64], 0x0040)
	binary.BigEndian.PutUint16(data[64:66], 0x0020)
	binary.BigEndian.PutUint16(data[66:68], 0xFFFD)
	// Offsets 68-77: present in all versions
	putInt16(data, 68, 900)  // sTypoAscender
	putInt16(data, 70, -100) // sTypoDescender
	putInt16(data, 72, 0)    // sTypoLineGap
	binary.BigEndian.PutUint16(data[74:76], 900) // usWinAscent
	binary.BigEndian.PutUint16(data[76:78], 100) // usWinDescent
	return data
}

func TestOS2ParsesVersion4(t *testing.T) {
	data := buildFont(map[string][]byte{
		"OS/2": buildOS2Table(400, 5, 2, 800, -200, 50, 900, 100),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	os2, err := font.OS2()
	if err != nil {
		t.Fatalf("OS2 returned error: %v", err)
	}
	if os2.Version != 4 {
		t.Fatalf("Version = %d, want 4", os2.Version)
	}
	if os2.WeightClass != 400 {
		t.Fatalf("WeightClass = %d, want 400", os2.WeightClass)
	}
	if os2.WidthClass != 5 {
		t.Fatalf("WidthClass = %d, want 5", os2.WidthClass)
	}
	if os2.Panose.FamilyType != 2 {
		t.Fatalf("Panose.FamilyType = %d, want 2", os2.Panose.FamilyType)
	}
	if os2.Panose.Weight != 4 {
		t.Fatalf("Panose.Weight = %d, want 4", os2.Panose.Weight)
	}
	if os2.VendorID != "GOOG" {
		t.Fatalf("VendorID = %q, want \"GOOG\"", os2.VendorID)
	}
	if os2.Selection != 0x0040 {
		t.Fatalf("Selection = %#x, want 0x0040", os2.Selection)
	}
	if os2.TypoAscender != 800 {
		t.Fatalf("TypoAscender = %d, want 800", os2.TypoAscender)
	}
	if os2.TypoDescender != -200 {
		t.Fatalf("TypoDescender = %d, want -200", os2.TypoDescender)
	}
	if os2.TypoLineGap != 50 {
		t.Fatalf("TypoLineGap = %d, want 50", os2.TypoLineGap)
	}
	if os2.WinAscent != 900 {
		t.Fatalf("WinAscent = %d, want 900", os2.WinAscent)
	}
	if os2.WinDescent != 100 {
		t.Fatalf("WinDescent = %d, want 100", os2.WinDescent)
	}
	if os2.UnicodeRange[0] != 1 {
		t.Fatalf("UnicodeRange[0] = %d, want 1", os2.UnicodeRange[0])
	}
	if os2.CodePageRange[0] != 1 {
		t.Fatalf("CodePageRange[0] = %d, want 1", os2.CodePageRange[0])
	}
	if os2.XHeight != 530 {
		t.Fatalf("XHeight = %d, want 530", os2.XHeight)
	}
	if os2.CapHeight != 720 {
		t.Fatalf("CapHeight = %d, want 720", os2.CapHeight)
	}
	if os2.DefaultChar != 0x0020 {
		t.Fatalf("DefaultChar = %#x, want 0x0020", os2.DefaultChar)
	}
	if os2.MaxContext != 0 {
		t.Fatalf("MaxContext = %d, want 0", os2.MaxContext)
	}
}

func TestOS2RejectsMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.OS2()
	if err == nil {
		t.Fatal("OS2 succeeded for missing table")
	}
}

func TestOS2RejectsTruncatedTable(t *testing.T) {
	shortOS2 := make([]byte, 40) // too short for version 0 (need 78)
	data := buildFont(map[string][]byte{
		"OS/2": shortOS2,
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.OS2()
	if err == nil {
		t.Fatal("OS2 succeeded for truncated table")
	}
}

func TestOS2Version0GracefulFallback(t *testing.T) {
	// Version 0 OS/2 table (78 bytes): has no typo metrics, code pages, or xHeight.
	data := buildFont(map[string][]byte{
		"OS/2": buildOS2TableV0(700, 3),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	os2, err := font.OS2()
	if err != nil {
		t.Fatalf("OS2 returned error: %v", err)
	}
	if os2.Version != 0 {
		t.Fatalf("Version = %d, want 0", os2.Version)
	}
	if os2.WeightClass != 700 {
		t.Fatalf("WeightClass = %d, want 700", os2.WeightClass)
	}
	if os2.WidthClass != 3 {
		t.Fatalf("WidthClass = %d, want 3", os2.WidthClass)
	}
	// v0 fields at offsets 68-77 are present in all versions
	if os2.TypoAscender != 900 {
		t.Fatalf("TypoAscender = %d, want 900", os2.TypoAscender)
	}
	if os2.WinAscent != 900 {
		t.Fatalf("WinAscent = %d, want 900", os2.WinAscent)
	}
	// Code page range is v1+ only (offsets 78-85)
	if os2.CodePageRange[0] != 0 {
		t.Fatalf("CodePageRange[0] = %d, want 0 (version 0 has no code pages)", os2.CodePageRange[0])
	}
	// v2+ fields (xHeight, capHeight, etc.) not present in 78-byte v0 table
	if os2.XHeight != 0 {
		t.Fatalf("XHeight = %d, want 0 (version 0 has no v2+ fields)", os2.XHeight)
	}
}

func buildPostTable(italicAngle float64, underlinePosition, underlineThickness int16, isFixedPitch uint32) []byte {
	// Build a version 2.0 post table (32 bytes header).
	data := make([]byte, 32)
	binary.BigEndian.PutUint32(data[0:4], 0x00020000) // version 2.0
	// italicAngle: Fixed 16.16
	italicRaw := int32(italicAngle * 65536.0)
	binary.BigEndian.PutUint32(data[4:8], uint32(italicRaw))
	putInt16(data, 8, underlinePosition)
	putInt16(data, 10, underlineThickness)
	binary.BigEndian.PutUint32(data[12:16], isFixedPitch)
	binary.BigEndian.PutUint32(data[16:20], 0) // minMemType42
	binary.BigEndian.PutUint32(data[20:24], 0) // maxMemType42
	binary.BigEndian.PutUint32(data[24:28], 0) // minMemType1
	binary.BigEndian.PutUint32(data[28:32], 0) // maxMemType1
	return data
}

func TestPostParsesItalicAngle(t *testing.T) {
	data := buildFont(map[string][]byte{
		"post": buildPostTable(-12.0, -150, 50, 0),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	post, err := font.Post()
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	// italicAngle = -12.0 with Fixed 16.16 rounding
	expected := -12.0
	if post.ItalicAngle < expected-0.001 || post.ItalicAngle > expected+0.001 {
		t.Fatalf("ItalicAngle = %f, want ~%f", post.ItalicAngle, expected)
	}
	if post.UnderlinePosition != -150 {
		t.Fatalf("UnderlinePosition = %d, want -150", post.UnderlinePosition)
	}
	if post.UnderlineThickness != 50 {
		t.Fatalf("UnderlineThickness = %d, want 50", post.UnderlineThickness)
	}
	if post.IsFixedPitch != 0 {
		t.Fatalf("IsFixedPitch = %d, want 0", post.IsFixedPitch)
	}
}

func TestPostParsesUprightFont(t *testing.T) {
	data := buildFont(map[string][]byte{
		"post": buildPostTable(0.0, -200, 40, 1),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	post, err := font.Post()
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}

	if post.ItalicAngle != 0.0 {
		t.Fatalf("ItalicAngle = %f, want 0.0 (upright)", post.ItalicAngle)
	}
	if post.IsFixedPitch != 1 {
		t.Fatalf("IsFixedPitch = %d, want 1 (monospaced)", post.IsFixedPitch)
	}
}

func TestPostMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	_, err = font.Post()
	if err == nil {
		t.Fatal("Post should return error for font without post table")
	}
}
