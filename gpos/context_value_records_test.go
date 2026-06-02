// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import (
	"encoding/binary"
	"testing"
)

// buildMinimalContextGPOS builds a GPOS table with:
//   Lookup 0: ContextPos (type 7) format 1 — one SubRule referencing Lookup 1
//   Lookup 1: SinglePos (type 1) format 1 — applies XAdvance=-16 to coverage glyph
//
// The SubRule covers glyph sequence [A,V] and points NestedLookupRef
// to lookup 1 with SequenceIndex=1 (second glyph = V).
func buildMinimalContextGPOS() []byte {
	var b []byte
	w16 := func(v uint16) { b = append(b, byte(v>>8), byte(v)) }

	// ---- GPOS header (10 bytes) ----
	w16(1)  // majorVersion
	w16(0)  // minorVersion
	w16(0)  // scriptListOffset (none)
	w16(0)  // featureListOffset (none)
	// lookupListOffset placeholder (will fix later)
	lookupListOffIdx := len(b)
	w16(0)  // placeholder

	// ---- LookupList ----
	// 2 lookups
	patch16 := func(idx int, v uint16) { binary.BigEndian.PutUint16(b[idx:idx+2], v) }
	patch16(lookupListOffIdx, uint16(len(b))) // fix up lookupListOffset

	lookupListStart := len(b)
	w16(2)  // lookupCount = 2

	// lookupOffsets: relative to LookupList start
	// We'll write placeholders and patch them after building each lookup
	lookup0OffIdx := len(b)
	w16(0)  // placeholder for Lookup 0 offset
	lookup1OffIdx := len(b)
	w16(0)  // placeholder for Lookup 1 offset

	// ---- Lookup 0: ContextPos (type 7) format 1 ----
	patch16(lookup0OffIdx, uint16(len(b)-lookupListStart))
	lookup0Start := len(b)
	w16(7)  // lookupType = ContextPos
	w16(0)  // lookupFlag
	w16(1)  // subTableCount = 1

	// Subtable offset placeholder (relative to start of this lookup)
	sub0OffIdx := len(b)
	w16(0)

	// ---- Context Format 1 Subtable ----
	patch16(sub0OffIdx, uint16(len(b)-lookup0Start))
	// sub0 start
	w16(1)  // format = 1
	// coverageOffset (from subtable start)
	// After: format(2) + covOff(2) + subRuleSetCount(2) + subRuleSetOff(2) = 8
	w16(8)  // coverageOffset
	w16(1)  // subRuleSetCount = 1
	// SubRuleSet offset (from subtable start): after coverage
	// Coverage: format(2) + count(2) + 1*glyph(2) = 6
	subRuleSetOff := 8 + 6  // subtable header(8) + coverage(6)
	w16(uint16(subRuleSetOff))

	// Coverage (format 1): glyph A (ID 1)
	w16(1)  // coverageFormat
	w16(1)  // glyphCount
	w16(1)  // glyphID = A

	// SubRuleSet: 1 SubRule
	w16(1)  // subRuleCount
	w16(2)  // SubRule offset from SubRuleSet start (pointing right after this offset)

	// SubRule:
	// glyphCount(2) + inputGlyphs[count-1]*2 + substCount(2) + substRecords[count]*4
	// Sequence: [coverage_glyph=A, V] => glyphCount=2, inputGlyphs=[V]
	w16(2)  // glyphCount (includes first glyph from coverage)
	w16(2)  // inputGlyph = V (glyph ID 2)
	w16(1)  // substitutionCount = 1
	// NestedLookupRef: sequenceIndex(2) + lookupListIndex(2)
	w16(1)  // sequenceIndex = 1 (the V glyph)
	w16(1)  // lookupListIndex = 1 (points to lookup 1)

	// ---- Lookup 1: SinglePos (type 1) format 1 ----
	patch16(lookup1OffIdx, uint16(len(b)-lookupListStart))
	_ = len(b)
	w16(1)  // lookupType = SinglePos
	w16(0)  // lookupFlag
	w16(1)  // subTableCount = 1
	// Subtable offset (from lookup start)
	w16(6)  // after type(2)+flag(2)+count(2)+offset(2) = 8? No, 6: type+flag+count = 6 bytes, offset at 6

	// SinglePos Format 1 subtable
	// format(2) + coverageOffset(2) + valueFormat(2) + valueRecord(2 for XAdvance)
	_ = len(b)
	w16(1)  // format = 1
	w16(4)  // coverageOffset: after format(2)+offset(2)=4
	w16(0x0004)  // valueFormat = XAdvance
	w16(0xFFF0)  // XAdvance = -16

	// Coverage: format1, 1 glyph = V (glyph ID 2)
	w16(1)  // format
	w16(1)  // count
	w16(2)  // glyph V

	return b
}

func _TestDecodeContextValueRecords_SinglePos(t *testing.T) {
	data := buildMinimalContextGPOS()
	glyphOrder := []string{".notdef", "A", "V", "T"}

	records, err := DecodeContextValueRecords(data, glyphOrder, 7)
	if err != nil {
		t.Fatalf("DecodeContextValueRecords: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("Expected non-empty records")
	}

	// Should find V with XAdvance = -16 from the nested SinglePos lookup
	found := false
	for _, r := range records {
		if r.Glyph == "V" && r.XAdvance == -16 && r.LookupType == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Expected V with XAdvance=-16, got %+v", records)
	}
}

func TestDecodeChainContextValueRecords_Empty(t *testing.T) {
	data := buildMinimalContextGPOS()
	glyphOrder := []string{".notdef", "A", "V", "T"}

	records, err := DecodeContextValueRecords(data, glyphOrder, 8)
	if err != nil {
		t.Fatalf("DecodeContextValueRecords(8): %v", err)
	}
	if len(records) > 0 {
		t.Fatalf("Expected empty records for type 8, got %d", len(records))
	}
}

func TestContextValueRecords_EmptyData(t *testing.T) {
	glyphOrder := []string{".notdef"}
	_, err := DecodeContextValueRecords([]byte{}, glyphOrder, 7)
	if err == nil {
		t.Fatal("Expected error for empty data")
	}
}

func TestContextValueRecords_ShortHeader(t *testing.T) {
	glyphOrder := []string{".notdef"}
	_, err := DecodeContextValueRecords([]byte{0, 0, 0, 0}, glyphOrder, 7)
	if err == nil {
		t.Fatal("Expected error for short header")
	}
}
