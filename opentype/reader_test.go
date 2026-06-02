// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"sort"
	"testing"
)

func TestParseTableDirectoryAndReadTableData(t *testing.T) {
	data := make([]byte, 64)
	binary.BigEndian.PutUint32(data[0:4], 0x00010000)
	binary.BigEndian.PutUint16(data[4:6], 2)
	writeTableRecord(data[12:28], "head", 0x01020304, 44, 4)
	writeTableRecord(data[28:44], "GPOS", 0x05060708, 48, 6)
	copy(data[44:48], []byte{0xde, 0xad, 0xbe, 0xef})
	copy(data[48:54], []byte{1, 2, 3, 4, 5, 6})

	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if font.ScalerType != 0x00010000 {
		t.Fatalf("ScalerType = %#x, want 0x00010000", font.ScalerType)
	}
	if len(font.Tables) != 2 {
		t.Fatalf("table count = %d, want 2", len(font.Tables))
	}
	got, err := font.TableData("GPOS")
	if err != nil {
		t.Fatalf("TableData returned error: %v", err)
	}
	if string(got) != string([]byte{1, 2, 3, 4, 5, 6}) {
		t.Fatalf("GPOS bytes = %v", got)
	}
}

func TestParseRejectsTruncatedTableRecord(t *testing.T) {
	data := make([]byte, 20)
	binary.BigEndian.PutUint32(data[0:4], 0x00010000)
	binary.BigEndian.PutUint16(data[4:6], 1)

	if _, err := Parse(data); err == nil {
		t.Fatal("Parse succeeded for truncated table record")
	}
}

func TestNumGlyphsFromMaxp(t *testing.T) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(4),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	got, err := font.NumGlyphs()
	if err != nil {
		t.Fatalf("NumGlyphs returned error: %v", err)
	}
	if got != 4 {
		t.Fatalf("NumGlyphs = %d, want 4", got)
	}
}

func TestGlyphOrderFromPostFormat2CustomNames(t *testing.T) {
	data := buildFont(map[string][]byte{
		"maxp": buildMaxp(3),
		"post": buildPostFormat2([]string{"A", "V", "T"}),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	got, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder returned error: %v", err)
	}
	want := []string{"A", "V", "T"}
	if len(got) != len(want) {
		t.Fatalf("GlyphOrder length = %d, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("GlyphOrder[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestNameReadsWindowsUTF16Records(t *testing.T) {
	data := buildFont(map[string][]byte{
		"name": buildNameTable(map[uint16]string{1: "Fixture Family", 2: "Bold"}),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got := font.Name(16, 1); got != "Fixture Family" {
		t.Fatalf("family name = %q, want Fixture Family", got)
	}
	if got := font.Name(17, 2); got != "Bold" {
		t.Fatalf("style name = %q, want Bold", got)
	}
}

func TestBestCmapFormat12(t *testing.T) {
	data := buildFont(map[string][]byte{
		"cmap": buildCmapFormat12(map[uint32]uint16{0x41: 1, 0x56: 2}),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	cmap, err := font.BestCmap()
	if err != nil {
		t.Fatalf("BestCmap returned error: %v", err)
	}
	if cmap[0x41] != 1 || cmap[0x56] != 2 {
		t.Fatalf("cmap = %v, want A->1 V->2", cmap)
	}
}

func TestReverseCmapInvertsBestCmap(t *testing.T) {
	data := buildFont(map[string][]byte{
		"cmap": buildCmapFormat12(map[uint32]uint16{0x41: 1, 0x42: 1, 0x56: 2}),
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	rev, err := font.ReverseCmap()
	if err != nil {
		t.Fatalf("ReverseCmap returned error: %v", err)
	}
	if len(rev[1]) != 2 {
		t.Fatalf("glyph 1 codepoints = %v, want 2 entries", rev[1])
	}
	if len(rev[2]) != 1 || rev[2][0] != 0x56 {
		t.Fatalf("glyph 2 codepoints = %v, want [0x56]", rev[2])
	}
	// Verify round-trip: every forward entry appears in reverse
	forward, _ := font.BestCmap()
	for cp, gid := range forward {
		found := false
		for _, cp2 := range rev[gid] {
			if cp2 == cp {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("codepoint %x with glyph %d not found in reverse", cp, gid)
		}
	}
}

func writeTableRecord(dst []byte, tag string, checksum uint32, offset uint32, length uint32) {
	copy(dst[0:4], []byte(tag))
	binary.BigEndian.PutUint32(dst[4:8], checksum)
	binary.BigEndian.PutUint32(dst[8:12], offset)
	binary.BigEndian.PutUint32(dst[12:16], length)
}

func buildFont(tables map[string][]byte) []byte {
	tags := make([]string, 0, len(tables))
	for tag := range tables {
		tags = append(tags, tag)
	}
	dataOffset := 12 + len(tags)*16
	data := make([]byte, dataOffset)
	binary.BigEndian.PutUint32(data[0:4], 0x00010000)
	binary.BigEndian.PutUint16(data[4:6], uint16(len(tags)))
	currentOffset := dataOffset
	for index, tag := range tags {
		body := tables[tag]
		writeTableRecord(data[12+index*16:12+(index+1)*16], tag, 0, uint32(currentOffset), uint32(len(body)))
		data = append(data, body...)
		currentOffset += len(body)
	}
	return data
}

func buildMaxp(numGlyphs uint16) []byte {
	data := make([]byte, 6)
	binary.BigEndian.PutUint32(data[0:4], 0x00010000)
	binary.BigEndian.PutUint16(data[4:6], numGlyphs)
	return data
}

func buildPostFormat2(names []string) []byte {
	data := make([]byte, 34+len(names)*2)
	binary.BigEndian.PutUint32(data[0:4], 0x00020000)
	binary.BigEndian.PutUint16(data[32:34], uint16(len(names)))
	for index := range names {
		binary.BigEndian.PutUint16(data[34+index*2:36+index*2], uint16(258+index))
	}
	for _, name := range names {
		data = append(data, byte(len(name)))
		data = append(data, []byte(name)...)
	}
	return data
}

func buildCmapFormat12(mapping map[uint32]uint16) []byte {
	groupCount := len(mapping)
	subtableLength := 16 + groupCount*12
	data := make([]byte, 12+subtableLength)
	binary.BigEndian.PutUint16(data[2:4], 1)
	binary.BigEndian.PutUint16(data[4:6], 3)
	binary.BigEndian.PutUint16(data[6:8], 10)
	binary.BigEndian.PutUint32(data[8:12], 12)
	sub := data[12:]
	binary.BigEndian.PutUint16(sub[0:2], 12)
	binary.BigEndian.PutUint32(sub[4:8], uint32(subtableLength))
	binary.BigEndian.PutUint32(sub[12:16], uint32(groupCount))
	index := 0
	for codepoint, glyphID := range mapping {
		offset := 16 + index*12
		binary.BigEndian.PutUint32(sub[offset:offset+4], codepoint)
		binary.BigEndian.PutUint32(sub[offset+4:offset+8], codepoint)
		binary.BigEndian.PutUint32(sub[offset+8:offset+12], uint32(glyphID))
		index++
	}
	return data
}

func buildNameTable(values map[uint16]string) []byte {
	nameIDs := make([]uint16, 0, len(values))
	for nameID := range values {
		nameIDs = append(nameIDs, nameID)
	}
	sort.Slice(nameIDs, func(i, j int) bool { return nameIDs[i] < nameIDs[j] })

	stringOffset := 6 + len(nameIDs)*12
	data := make([]byte, stringOffset)
	binary.BigEndian.PutUint16(data[2:4], uint16(len(nameIDs)))
	binary.BigEndian.PutUint16(data[4:6], uint16(stringOffset))
	currentStringOffset := 0
	for index, nameID := range nameIDs {
		encoded := encodeUTF16BE(values[nameID])
		recordOffset := 6 + index*12
		binary.BigEndian.PutUint16(data[recordOffset:recordOffset+2], 3)
		binary.BigEndian.PutUint16(data[recordOffset+2:recordOffset+4], 1)
		binary.BigEndian.PutUint16(data[recordOffset+4:recordOffset+6], 0x0409)
		binary.BigEndian.PutUint16(data[recordOffset+6:recordOffset+8], nameID)
		binary.BigEndian.PutUint16(data[recordOffset+8:recordOffset+10], uint16(len(encoded)))
		binary.BigEndian.PutUint16(data[recordOffset+10:recordOffset+12], uint16(currentStringOffset))
		data = append(data, encoded...)
		currentStringOffset += len(encoded)
	}
	return data
}

func encodeUTF16BE(value string) []byte {
	data := make([]byte, 0, len(value)*2)
	for _, char := range value {
		if char > 0xffff {
			continue
		}
		data = append(data, byte(char>>8), byte(char))
	}
	return data
}

// --- Font Collection (.ttc) tests ---

func buildTTC(version uint16, fonts [][]byte) []byte {
	// TTC header: tag(4) + version(2) + numFonts(4) + offsets(4*n) [+ DSIG if v2]
	numFonts := len(fonts)
	headerSize := 12 + numFonts*4
	if version == 2 {
		headerSize += 12 // DSIG offset + length fields
	}
	data := make([]byte, headerSize)
	copy(data[0:4], "ttcf")
	binary.BigEndian.PutUint16(data[4:6], version)
	binary.BigEndian.PutUint32(data[8:12], uint32(numFonts))

	// Calculate font offsets (after TTC header)
	offset := headerSize
	for i, fontData := range fonts {
		binary.BigEndian.PutUint32(data[12+i*4:12+i*4+4], uint32(offset))
		offset += len(fontData)
	}
	for _, fontData := range fonts {
		data = append(data, fontData...)
	}
	return data
}

func TestParseCollectionTwoFonts(t *testing.T) {
	font1 := buildFont(map[string][]byte{
		"maxp": buildMaxp(100),
		"name": buildNameTable(map[uint16]string{1: "FontOne"}),
	})
	font2 := buildFont(map[string][]byte{
		"maxp": buildMaxp(200),
		"name": buildNameTable(map[uint16]string{1: "FontTwo"}),
	})

	ttcData := buildTTC(1, [][]byte{font1, font2})
	coll, err := ParseCollection(ttcData)
	if err != nil {
		t.Fatalf("ParseCollection returned error: %v", err)
	}
	if len(coll.Fonts) != 2 {
		t.Fatalf("collection has %d fonts, want 2", len(coll.Fonts))
	}
	n1 := coll.Fonts[0].Name(1)
	if n1 != "FontOne" {
		t.Fatalf("font 0 name = %q, want FontOne", n1)
	}
	n2 := coll.Fonts[1].Name(1)
	if n2 != "FontTwo" {
		t.Fatalf("font 1 name = %q, want FontTwo", n2)
	}
}

func TestParseCollectionVersion2(t *testing.T) {
	font1 := buildFont(map[string][]byte{
		"maxp": buildMaxp(50),
	})
	ttcData := buildTTC(2, [][]byte{font1})
	coll, err := ParseCollection(ttcData)
	if err != nil {
		t.Fatalf("ParseCollection v2 returned error: %v", err)
	}
	if len(coll.Fonts) != 1 {
		t.Fatalf("collection has %d fonts, want 1", len(coll.Fonts))
	}
}

func TestParseCollectionRejectsInvalidTag(t *testing.T) {
	data := make([]byte, 16)
	copy(data[0:4], "ffff")
	_, err := ParseCollection(data)
	if err == nil {
		t.Fatal("ParseCollection succeeded for invalid tag")
	}
}

func TestParseCollectionRejectsTruncatedHeader(t *testing.T) {
	data := make([]byte, 8) // too short for TTC header
	_, err := ParseCollection(data)
	if err == nil {
		t.Fatal("ParseCollection succeeded for truncated header")
	}
}

func TestParseCollectionRejectsOffsetExceedsData(t *testing.T) {
	font1 := buildFont(map[string][]byte{
		"maxp": buildMaxp(1),
	})
	ttcData := buildTTC(1, [][]byte{font1})
	// Corrupt the font offset to point beyond data
	binary.BigEndian.PutUint32(ttcData[12:16], uint32(len(ttcData)+100))
	_, err := ParseCollection(ttcData)
	if err == nil {
		t.Fatal("ParseCollection succeeded for offset exceeding data")
	}
}

func TestParseCollectionRejectsTooManyFonts(t *testing.T) {
	// Save and restore the limit
	origLimit := MaxCollectionFonts
	MaxCollectionFonts = 2
	defer func() { MaxCollectionFonts = origLimit }()

	fonts := make([][]byte, 3)
	for i := range fonts {
		fonts[i] = buildFont(map[string][]byte{
			"maxp": buildMaxp(1),
		})
	}
	ttcData := buildTTC(1, fonts)
	_, err := ParseCollection(ttcData)
	if err == nil {
		t.Fatal("ParseCollection succeeded for too many fonts")
	}
}

func TestParseCollectionRejectsBadFont(t *testing.T) {
	// Build a TTC where one font data is invalid (too short for a font header)
	data := make([]byte, 12+4+4) // header + 1 offset + 4 bytes of invalid font data
	copy(data[0:4], "ttcf")
	binary.BigEndian.PutUint16(data[4:6], 1)
	binary.BigEndian.PutUint32(data[8:12], 1)
	binary.BigEndian.PutUint32(data[12:16], 16) // offset past header
	// Only 4 bytes for the "font" — not enough for a valid font
	_, err := ParseCollection(data)
	if err == nil {
		t.Fatal("ParseCollection succeeded for invalid font data")
	}
}

func TestParseCollectionRejectsUnsupportedVersion(t *testing.T) {
	data := make([]byte, 16)
	copy(data[0:4], "ttcf")
	binary.BigEndian.PutUint16(data[4:6], 3) // unsupported version
	binary.BigEndian.PutUint32(data[8:12], 1)
	_, err := ParseCollection(data)
	if err == nil {
		t.Fatal("ParseCollection succeeded for unsupported version")
	}
}

func TestParseCollectionFontTableAccess(t *testing.T) {
	// Verify that table data in collection fonts is accessible
	fontData := buildFont(map[string][]byte{
		"maxp": buildMaxp(42),
	})
	ttcData := buildTTC(1, [][]byte{fontData})
	coll, err := ParseCollection(ttcData)
	if err != nil {
		t.Fatalf("ParseCollection returned error: %v", err)
	}
	num, err := coll.Fonts[0].NumGlyphs()
	if err != nil {
		t.Fatalf("NumGlyphs returned error: %v", err)
	}
	if num != 42 {
		t.Fatalf("NumGlyphs = %d, want 42", num)
	}
}
