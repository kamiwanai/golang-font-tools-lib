// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"testing"
)

// --- CFF glyph names tests ---

// buildCFFData constructs a minimal valid CFF table with:
// - 3 glyphs: .notdef, "A" (SID 34), "V" (SID 56) using charset format 0
// - A custom string "dotaccent" (SID 391) to test String INDEX resolution
func buildCFFDataWithCharset0() []byte {
	// CFF structure:
	// Header (4 bytes) → Name INDEX → Top DICT INDEX → String INDEX → Global Subr INDEX
	// Then at offsets referenced by Top DICT: charset, CharStrings INDEX

	// We'll build the CFF data in pieces and compute offsets

	// Header: major=1, minor=0, hdrSize=4
	header := []byte{1, 0, 4, 1}

	// Name INDEX: one entry "TestFont"
	nameIndex := buildCFFIndexBytes([][]byte{[]byte("TestFont")})

	// Top DICT: we need charset offset and CharStrings offset
	// We'll compute these after building the table structure
	// Top DICT uses DICT encoding: operands come before operators
	// Operator 15 = charset, 17 = CharStrings
	// For now, use placeholder offsets; we'll patch them after
	topDictData := buildTopDICT(0, 0) // placeholder offsets
	topDictIndex := buildCFFIndexBytes([][]byte{topDictData})

	// String INDEX: one custom string "dotaccent" (SID 391 → index 0 in String INDEX)
	stringIndex := buildCFFIndexBytes([][]byte{[]byte("dotaccent")})

	// Global Subr INDEX: empty
	globalSubrIndex := []byte{0, 0} // count=0

	prefixLen := len(header) + len(nameIndex) + len(topDictIndex) + len(stringIndex) + len(globalSubrIndex)

	// CharStrings INDEX: 3 entries (one per glyph), each is a minimal charstring
	// Minimal Type 2 charstring: endchar (opcode 14)
	charStrings := buildCFFIndexBytes([][]byte{{14}, {14}, {14}})
	charsetOffset := prefixLen
	charstringsOffset := prefixLen + len(charStrings)

	// Now rebuild Top DICT with real offsets
	topDictData = buildTopDICT(charsetOffset, charstringsOffset)
	topDictIndex = buildCFFIndexBytes([][]byte{topDictData})

	// Assemble
	prefix := append(header, nameIndex...)
	prefix = append(prefix, topDictIndex...)
	prefix = append(prefix, stringIndex...)
	prefix = append(prefix, globalSubrIndex...)

	result := make([]byte, len(prefix))
	copy(result, prefix)
	result = append(result, charStrings...)

	// Charset format 0: format byte + (nGlyphs-1) SIDs
	// .notdef=SID 0 (implicit), glyph1=SID 34 ("A"), glyph2=SID 55 ("V")
	charset := []byte{0}
	charset = append(charset, []byte{0, 34}...) // SID 34 = "A"
	charset = append(charset, []byte{0, 55}...) // SID 56 = "V"

	// Patch: we need to insert the charset between the prefix and charstrings
	// Rebuild: prefix + charset + charstrings, and adjust CharStrings offset
	charstringsOffsetNew := charsetOffset + len(charset)

	// Re-rebuild everything with the adjusted offset
	topDictData = buildTopDICT(charsetOffset, charstringsOffsetNew)
	topDictIndex = buildCFFIndexBytes([][]byte{topDictData})

	prefix = append(header, nameIndex...)
	prefix = append(prefix, topDictIndex...)
	prefix = append(prefix, stringIndex...)
	prefix = append(prefix, globalSubrIndex...)

	// Verify charset offset matches
	if len(prefix) != charsetOffset {
		// If prefix length changed, recalculate again
		charsetOffset = len(prefix)
		charstringsOffsetNew = charsetOffset + len(charset)
		topDictData = buildTopDICT(charsetOffset, charstringsOffsetNew)
		topDictIndex = buildCFFIndexBytes([][]byte{topDictData})

		prefix = append(header, nameIndex...)
		prefix = append(prefix, topDictIndex...)
		prefix = append(prefix, stringIndex...)
		prefix = append(prefix, globalSubrIndex...)

		if len(prefix) != charsetOffset {
			panic("CFF offset calculation unstable")
		}
	}

	result = make([]byte, len(prefix))
	copy(result, prefix)
	result = append(result, charset...)
	result = append(result, charStrings...)

	return result
}

// buildTopDICT encodes a CFF Top DICT with charset and CharStrings offsets.
func buildTopDICT(charsetOffset, charstringsOffset int) []byte {
	var buf []byte
	// charset (operator 15)
	buf = append(buf, encodeCFFDictInt(charsetOffset)...)
	buf = append(buf, 15)
	// CharStrings (operator 17)
	buf = append(buf, encodeCFFDictInt(charstringsOffset)...)
	buf = append(buf, 17)
	return buf
}

// encodeCFFDictInt encodes an integer as a CFF DICT operand.
func encodeCFFDictInt(val int) []byte {
	if val >= -107 && val <= 107 {
		return []byte{byte(val + 139)}
	}
	if val >= 108 && val <= 1131 {
		v := val - 108
		return []byte{byte(247 + v/256), byte(v % 256)}
	}
	if val >= -1131 && val <= -108 {
		v := -val - 108
		return []byte{byte(251 + v/256), byte(v % 256)}
	}
	if val >= -32768 && val <= 32767 {
		b := make([]byte, 3)
		b[0] = 28
		binary.BigEndian.PutUint16(b[1:3], uint16(int16(val)))
		return b
	}
	b := make([]byte, 5)
	b[0] = 29
	binary.BigEndian.PutUint32(b[1:5], uint32(int32(val)))
	return b
}

// buildCFFIndexBytes builds a CFF INDEX structure from elements.
func buildCFFIndexBytes(elements [][]byte) []byte {
	count := len(elements)
	if count == 0 {
		return []byte{0, 0} // empty INDEX
	}

	// Determine offSize
	totalDataLen := 0
	for _, e := range elements {
		totalDataLen += len(e)
	}
	offSize := 1
	if totalDataLen+count+1 > 255 {
		offSize = 2
	}
	if totalDataLen+count+1 > 65535 {
		offSize = 3
	}
	if totalDataLen+count+1 > 16777215 {
		offSize = 4
	}

	// Build offset array (1-based offsets)
	offsets := make([]int, count+1)
	off := 1
	for i, e := range elements {
		offsets[i] = off
		off += len(e)
	}
	offsets[count] = off

	// Encode
	result := make([]byte, 0, 3+count*offSize+totalDataLen+1)
	result = append(result, byte(count>>8), byte(count))
	result = append(result, byte(offSize))

	for _, o := range offsets {
		result = appendCFFOffset(result, o, offSize)
	}

	for _, e := range elements {
		result = append(result, e...)
	}

	return result
}

func appendCFFOffset(buf []byte, offset, offSize int) []byte {
	switch offSize {
	case 1:
		return append(buf, byte(offset))
	case 2:
		return append(buf, byte(offset>>8), byte(offset))
	case 3:
		return append(buf, byte(offset>>16), byte(offset>>8), byte(offset))
	case 4:
		return append(buf, byte(offset>>24), byte(offset>>16), byte(offset>>8), byte(offset))
	}
	return buf
}

func TestCFFGlyphNamesCharset0(t *testing.T) {
	cffData := buildCFFDataWithCharset0()
	fontData := buildFont(map[string][]byte{
		"CFF ":  cffData,
		"maxp":  buildMaxp(3),
	})
	font, err := Parse(fontData)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	names, err := font.CFFGlyphNames()
	if err != nil {
		t.Fatalf("CFFGlyphNames returned error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("names count = %d, want 3", len(names))
	}
	if names[0] != ".notdef" {
		t.Fatalf("names[0] = %q, want .notdef", names[0])
	}
	if names[1] != "A" {
		t.Fatalf("names[1] = %q, want A", names[1])
	}
	if names[2] != "V" {
		t.Fatalf("names[2] = %q, want V", names[2])
	}
}

func TestCFFGlyphNamesWithCustomSID(t *testing.T) {
	// Build a CFF with a custom string in the String INDEX (SID 391 = "dotaccent")
	cffData := buildCFFDataWithCustomString()
	fontData := buildFont(map[string][]byte{
		"CFF ":  cffData,
		"maxp":  buildMaxp(3),
	})
	font, err := Parse(fontData)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	names, err := font.CFFGlyphNames()
	if err != nil {
		t.Fatalf("CFFGlyphNames returned error: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("names count = %d, want 3", len(names))
	}
	if names[0] != ".notdef" {
		t.Fatalf("names[0] = %q, want .notdef", names[0])
	}
	// SID 391 = first custom string = "dotaccent"
	if names[1] != "dotaccent" {
		t.Fatalf("names[1] = %q, want dotaccent", names[1])
	}
	if names[2] != "V" {
		t.Fatalf("names[2] = %q, want V", names[2])
	}
}

func buildCFFDataWithCustomString() []byte {
	header := []byte{1, 0, 4, 1}
	nameIndex := buildCFFIndexBytes([][]byte{[]byte("TestFont")})

	// String INDEX with one entry "dotaccent"
	stringIndex := buildCFFIndexBytes([][]byte{[]byte("dotaccent")})
	globalSubrIndex := []byte{0, 0}

	// Top DICT placeholder
	topDictData := buildTopDICT(0, 0)
	topDictIndex := buildCFFIndexBytes([][]byte{topDictData})

	prefixLen := len(header) + len(nameIndex) + len(topDictIndex) + len(stringIndex) + len(globalSubrIndex)

	charStrings := buildCFFIndexBytes([][]byte{{14}, {14}, {14}})

	// Charset format 0: SID 391 (custom "dotaccent"), SID 55 ("V")
	charset := []byte{0, 0x01, 0x87, 0, 55} // SID 391 = 0x0187

	charsetOffset := prefixLen
	charstringsOffsetNew := charsetOffset + len(charset)

	topDictData = buildTopDICT(charsetOffset, charstringsOffsetNew)
	topDictIndex = buildCFFIndexBytes([][]byte{topDictData})

	prefix := append(header, nameIndex...)
	prefix = append(prefix, topDictIndex...)
	prefix = append(prefix, stringIndex...)
	prefix = append(prefix, globalSubrIndex...)

	if len(prefix) != charsetOffset {
		charsetOffset = len(prefix)
		charstringsOffsetNew = charsetOffset + len(charset)
		topDictData = buildTopDICT(charsetOffset, charstringsOffsetNew)
		topDictIndex = buildCFFIndexBytes([][]byte{topDictData})
		prefix = append(header, nameIndex...)
		prefix = append(prefix, topDictIndex...)
		prefix = append(prefix, stringIndex...)
		prefix = append(prefix, globalSubrIndex...)
	}

	result := make([]byte, len(prefix))
	copy(result, prefix)
	result = append(result, charset...)
	result = append(result, charStrings...)
	return result
}

func TestCFFGlyphNamesRejectsMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.CFFGlyphNames()
	if err == nil {
		t.Fatal("CFFGlyphNames succeeded for missing CFF table")
	}
}

func TestCFFGlyphNamesRejectsTruncatedTable(t *testing.T) {
	data := buildFont(map[string][]byte{
		"CFF ": []byte{1, 0}, // too short
	})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	_, err = font.CFFGlyphNames()
	if err == nil {
		t.Fatal("CFFGlyphNames succeeded for truncated CFF table")
	}
}

func TestCFFGlyphNamesDefaultCharset(t *testing.T) {
	// Build CFF with charset offset 0 (default ISOAdobe charset)
	header := []byte{1, 0, 4, 1}
	nameIndex := buildCFFIndexBytes([][]byte{[]byte("TestFont")})
	stringIndex := buildCFFIndexBytes(nil) // empty
	globalSubrIndex := []byte{0, 0}

	charStrings := buildCFFIndexBytes([][]byte{{14}, {14}}) // 2 glyphs

	// charset offset 0 = default
	topDictData := buildTopDICT(0, 0)
	topDictIndex := buildCFFIndexBytes([][]byte{topDictData})

	prefix := append(header, nameIndex...)
	prefix = append(prefix, topDictIndex...)
	prefix = append(prefix, stringIndex...)
	prefix = append(prefix, globalSubrIndex...)

	charstringsOffset := len(prefix)
	topDictData = buildTopDICT(0, charstringsOffset)
	topDictIndex = buildCFFIndexBytes([][]byte{topDictData})

	prefix = append(header, nameIndex...)
	prefix = append(prefix, topDictIndex...)
	prefix = append(prefix, stringIndex...)
	prefix = append(prefix, globalSubrIndex...)

	if len(prefix) != charstringsOffset {
		charstringsOffset = len(prefix)
		topDictData = buildTopDICT(0, charstringsOffset)
		topDictIndex = buildCFFIndexBytes([][]byte{topDictData})
		prefix = append(header, nameIndex...)
		prefix = append(prefix, topDictIndex...)
		prefix = append(prefix, stringIndex...)
		prefix = append(prefix, globalSubrIndex...)
	}

	result := make([]byte, len(prefix))
	copy(result, prefix)
	result = append(result, charStrings...)

	fontData := buildFont(map[string][]byte{
		"CFF ":  result,
		"maxp":  buildMaxp(2),
	})
	font, err := Parse(fontData)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	names, err := font.CFFGlyphNames()
	if err != nil {
		t.Fatalf("CFFGlyphNames returned error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("names count = %d, want 2", len(names))
	}
	if names[0] != ".notdef" {
		t.Fatalf("names[0] = %q, want .notdef", names[0])
	}
	// Default charset SID 1 = "space"
	if names[1] != "space" {
		t.Fatalf("names[1] = %q, want space", names[1])
	}
}

func TestCFFIndexParsing(t *testing.T) {
	// Test CFF INDEX parsing directly
	elements := buildCFFIndexBytes([][]byte{[]byte("hello"), []byte("world")})
	count, result, next, err := parseCFFIndex(elements, 0)
	if err != nil {
		t.Fatalf("parseCFFIndex returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if string(result[0]) != "hello" {
		t.Fatalf("result[0] = %q, want hello", string(result[0]))
	}
	if string(result[1]) != "world" {
		t.Fatalf("result[1] = %q, want world", string(result[1]))
	}
	_ = next
}

func TestCFFIndexEmpty(t *testing.T) {
	empty := buildCFFIndexBytes(nil)
	// Should be just count=0 (2 bytes)
	if len(empty) != 2 || empty[0] != 0 || empty[1] != 0 {
		t.Fatalf("empty INDEX = %v, want [0 0]", empty)
	}
	count, result, next, err := parseCFFIndex(empty, 0)
	if err != nil {
		t.Fatalf("parseCFFIndex empty returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("empty count = %d, want 0", count)
	}
	if len(result) != 0 {
		t.Fatalf("empty result = %v, want empty", result)
	}
	_ = next
}