// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gsub

import (
	"encoding/binary"
	"testing"
)

func putU16(buf []byte, offset int, v uint16) {
	binary.BigEndian.PutUint16(buf[offset:offset+2], v)
}

func putU32(buf []byte, offset int, v uint32) {
	binary.BigEndian.PutUint32(buf[offset:offset+4], v)
}

func putInt16(buf []byte, offset int, v int16) {
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(v))
}

// buildGSUBHeader writes version 1.0 header with empty ScriptList/FeatureList.
// The LookupList is placed at lookupListOffset. Returns the data slice.
func buildGSUBHeader(allocLen int, lookupListOffset uint16) []byte {
	data := make([]byte, allocLen)
	putU16(data, 0, 1) // majorVersion
	putU16(data, 2, 0) // minorVersion
	putU16(data, 4, 10) // scriptListOffset (empty)
	putU16(data, 6, 12) // featureListOffset (empty)
	putU16(data, 8, lookupListOffset)
	// Empty ScriptList at 10: scriptCount=0
	putU16(data, 10, 0)
	// Empty FeatureList at 12: featureCount=0
	putU16(data, 12, 0)
	return data
}

// --- SingleSubst tests ---

func TestDecodeSingleSubstFormat1(t *testing.T) {
	// SingleSubst Format 1: deltaGlyphID
	// Coverage: A(1). Delta = +2 → A→C.
	glyphOrder := []string{".notdef", "A", "B", "C"}

	data := buildGSUBHeader(50, 14)
	// LookupList at 14
	putU16(data, 14, 1) // count
	putU16(data, 16, 4) // offset to lookup (14+4=18)
	// Lookup at 18
	putU16(data, 18, 1) // lookupType = SingleSubst
	putU16(data, 20, 0) // flag
	putU16(data, 22, 1) // subtableCount
	putU16(data, 24, 8) // offset from lookup start = 18+8=26
	// Subtable at 26: SingleSubst Format 1
	putU16(data, 26, 1)   // format
	putU16(data, 28, 8)   // coverageOffset relative to subtable → 26+8=34
	putInt16(data, 30, 2) // deltaGlyphID = +2
	// Coverage at 34
	putU16(data, 34, 1) // format 1
	putU16(data, 36, 1) // glyphCount
	putU16(data, 38, 1) // glyphID A

	subs, err := DecodeSingleSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 sub, got %d", len(subs))
	}
	if subs[0].From != "A" || subs[0].To != "C" {
		t.Fatalf("expected A→C, got %s→%s", subs[0].From, subs[0].To)
	}
}

func TestDecodeSingleSubstFormat2(t *testing.T) {
	// SingleSubst Format 2: substitute array
	glyphOrder := []string{".notdef", "A", "B", "C", "D"}

	data := buildGSUBHeader(60, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)
	// Lookup at 18
	putU16(data, 18, 1) // SingleSubst
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable at 26
	// Subtable at 26: format 2
	putU16(data, 26, 2)  // format
	putU16(data, 28, 10) // coverageOffset → 26+10=36
	putU16(data, 30, 2)  // glyphCount
	putU16(data, 32, 3)  // substitute[0] = C
	putU16(data, 34, 4)  // substitute[1] = D
	// Coverage at 36
	putU16(data, 36, 1) // format
	putU16(data, 38, 2) // count
	putU16(data, 40, 1) // A
	putU16(data, 42, 2) // B

	subs, err := DecodeSingleSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subs, got %d", len(subs))
	}
	if subs[0].From != "A" || subs[0].To != "C" {
		t.Fatalf("expected A→C, got %s→%s", subs[0].From, subs[0].To)
	}
	if subs[1].From != "B" || subs[1].To != "D" {
		t.Fatalf("expected B→D, got %s→%s", subs[1].From, subs[1].To)
	}
}

// --- LigatureSubst tests ---

func TestDecodeLigatureSubstFormat1(t *testing.T) {
	// LigatureSubst: f(6) has ligature set with:
	//   f+i → fi(8), compCount=2
	glyphOrder := []string{".notdef", "space", "A", "B", "C", "D", "f", "i", "fi"}

	data := buildGSUBHeader(80, 14)
	putU16(data, 14, 1) // lookupCount
	putU16(data, 16, 4) // offset to lookup = 18

	// Lookup at 18
	putU16(data, 18, 4) // LigatureSubst
	putU16(data, 20, 0) // flag
	putU16(data, 22, 1) // subtableCount
	putU16(data, 24, 8) // offset from lookup start → 26

	// Subtable (LigatureSubst format 1) at 26
	// Layout:
	//   26: format=1
	//   28: coverageOffset
	//   30: ligSetCount=1
	//   32: ligSetOffset[0]
	// Coverage and LigatureSet need specific offsets from subtable base.
	//
	// Subtable body: 6 bytes (fmt+covOff+count) + 2 bytes (offset)
	// = 8 bytes → subtable header ends at 34
	// Put LigatureSet right after at offset 8 from subtable base (=34)
	// Put Coverage after that.
	//
	// LigatureSet at 34 (offset 8 from 26):
	//   34: ligCount=1
	//   36: ligOffset[0] = 4 (relative to LigSet start → 34+4=38)
	// Ligature at 38:
	//   38: ligGlyph=8 (fi)
	//   40: compCount=2
	//   42: comp[0]=7 (i)
	// Coverage after that at offset 20 from subtable base (=46):
	//   46: format=1
	//   48: count=1
	//   50: glyph=6 (f)

	subtableBase := 26
	putU16(data, subtableBase, 1)    // format
	putU16(data, subtableBase+2, 20) // coverageOffset (=46-26=20)
	putU16(data, subtableBase+4, 1) // ligSetCount
	putU16(data, subtableBase+6, 8)  // ligSetOffset[0] (=34-26=8)

	// LigatureSet at 34
	putU16(data, 34, 1) // ligCount
	putU16(data, 36, 4) // ligOffset[0] → 34+4=38

	// Ligature at 38
	putU16(data, 38, 8) // ligGlyph = fi
	putU16(data, 40, 2) // compCount
	putU16(data, 42, 7) // comp[0] = i

	// Coverage at 46
	putU16(data, 46, 1) // format
	putU16(data, 48, 1) // count
	putU16(data, 50, 6) // glyph f

	ligs, err := DecodeLigatureSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(ligs) != 1 {
		t.Fatalf("expected 1 ligature, got %d", len(ligs))
	}
	if ligs[0].Ligature != "fi" {
		t.Fatalf("expected fi, got %s", ligs[0].Ligature)
	}
	if len(ligs[0].Components) != 2 || ligs[0].Components[0] != "f" || ligs[0].Components[1] != "i" {
		t.Fatalf("expected [f, i], got %v", ligs[0].Components)
	}
}

func TestDecodeLigatureSubstThreeComponent(t *testing.T) {
	// f_f_i: first glyph f(6), components=[f(6), i(7)], ligGlyph=f_f_i(8)
	// compCount=3 means 3 components (first is from coverage)
	glyphOrder := []string{".notdef", "space", "A", "B", "C", "D", "f", "i", "f_f_i"}

	data := buildGSUBHeader(80, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	putU16(data, 18, 4) // LigatureSubst
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable at 26

	// Subtable at 26
	putU16(data, 26, 1)  // format
	putU16(data, 28, 22) // coverageOffset (26+22=48)
	putU16(data, 30, 1)  // ligSetCount
	putU16(data, 32, 8)  // ligSetOffset (26+8=34)

	// LigatureSet at 34
	putU16(data, 34, 1) // ligCount
	putU16(data, 36, 6) // ligOffset → 34+6=40

	// Ligature at 40: ligGlyph=8(f_f_i), compCount=3, comp[0]=6(f), comp[1]=7(i)
	putU16(data, 40, 8) // ligGlyph
	putU16(data, 42, 3) // compCount
	putU16(data, 44, 6) // comp[0]=f
	putU16(data, 46, 7) // comp[1]=i

	// Coverage at 48
	putU16(data, 48, 1) // format
	putU16(data, 50, 1) // count
	putU16(data, 52, 6) // glyph f

	ligs, err := DecodeLigatureSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(ligs) != 1 {
		t.Fatalf("expected 1 ligature, got %d", len(ligs))
	}
	if ligs[0].Ligature != "f_f_i" {
		t.Fatalf("expected f_f_i, got %s", ligs[0].Ligature)
	}
	if len(ligs[0].Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(ligs[0].Components))
	}
	if ligs[0].Components[0] != "f" || ligs[0].Components[1] != "f" || ligs[0].Components[2] != "i" {
		t.Fatalf("expected [f, f, i], got %v", ligs[0].Components)
	}
}

// --- MultipleSubst tests ---

func TestDecodeMultipleSubstFormat1(t *testing.T) {
	glyphOrder := []string{".notdef", "A", "B", "C", "D", "E", "F", "G"}

	data := buildGSUBHeader(80, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	putU16(data, 18, 2) // MultipleSubst
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable at 26

	// Subtable at 26: MultipleSubst format 1
	//  26: format=1
	//  28: coverageOffset (to coverage relative to 26)
	//  30: sequenceCount=1
	//  32: seqOffset[0] (to sequence relative to 26)
	//
	// Coverage at 34 (offset 8 from 26):
	//   34: format=1, 36: count=1, 38: glyph=5(E)
	// Sequence at 40 (offset 14 from 26):
	//   40: glyphCount=2, 42: gid6(F), 44: gid7(G)

	putU16(data, 26, 1)  // format
	putU16(data, 28, 8)  // coverageOffset
	putU16(data, 30, 1)  // sequenceCount
	putU16(data, 32, 14) // seqOffset[0]

	// Coverage at 34
	putU16(data, 34, 1) // format
	putU16(data, 36, 1) // count
	putU16(data, 38, 5) // glyph E

	// Sequence at 40
	putU16(data, 40, 2) // glyphCount
	putU16(data, 42, 6) // F
	putU16(data, 44, 7) // G

	subs, err := DecodeMultipleSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 substitution, got %d", len(subs))
	}
	if subs[0].From != "E" {
		t.Fatalf("expected From=E, got %s", subs[0].From)
	}
	if len(subs[0].To) != 2 || subs[0].To[0] != "F" || subs[0].To[1] != "G" {
		t.Fatalf("expected [F, G], got %v", subs[0].To)
	}
}

// --- Filtering: wrong lookup type skipped ---

func TestDecodeLookupsSkipsWrongType(t *testing.T) {
	data := buildGSUBHeader(30, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)
	putU16(data, 18, 2) // type 2 = MultipleSubst, not type 1
	putU16(data, 20, 0)
	putU16(data, 22, 0)
	// No subtables needed

	subs, err := DecodeSingleSubstLookups(data, []string{".notdef"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("expected 0 subs for wrong type, got %d", len(subs))
	}
}

// --- Error cases ---

func TestDecodeSingleSubstTruncated(t *testing.T) {
	_, err := DecodeSingleSubstLookups(make([]byte, 5), []string{".notdef"})
	if err == nil {
		t.Fatal("expected error for truncated GSUB header")
	}
}

func TestDecodeLigatureSubstTruncated(t *testing.T) {
	_, err := DecodeLigatureSubstLookups(make([]byte, 5), []string{".notdef"})
	if err == nil {
		t.Fatal("expected error for truncated GSUB header")
	}
}

func TestDecodeMultipleSubstTruncated(t *testing.T) {
	_, err := DecodeMultipleSubstLookups(make([]byte, 5), []string{".notdef"})
	if err == nil {
		t.Fatal("expected error for truncated GSUB header")
	}
}

// --- Extension lookup type 9 for SingleSubst ---

func TestDecodeSingleSubstExtensionLookup(t *testing.T) {
	glyphOrder := []string{".notdef", "A", "B", "C"}

	data := buildGSUBHeader(80, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	// Lookup type 9 (extension) at 18
	putU16(data, 18, 9) // extension
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable offset → 26

	// Extension subtable at 26
	putU16(data, 26, 1) // format 1
	putU16(data, 28, 1) // extensionType = SingleSubst
	putU32(data, 30, 8) // offset to inner subtable (26+8=34)

	// Inner SingleSubst format 1 at 34
	putU16(data, 34, 1)   // format
	putU16(data, 36, 8)   // coverageOffset (34+8=42)
	putInt16(data, 38, 2) // deltaGlyphID = +2

	// Coverage at 42
	putU16(data, 42, 1) // format
	putU16(data, 44, 1) // count
	putU16(data, 46, 1) // glyph A

	subs, err := DecodeSingleSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("extension lookup error: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 sub, got %d", len(subs))
	}
	if subs[0].From != "A" || subs[0].To != "C" {
		t.Fatalf("expected A→C, got %s→%s", subs[0].From, subs[0].To)
	}
}

// --- Coverage format 2 test ---

func TestDecodeSingleSubstCoverageFormat2(t *testing.T) {
	glyphOrder := []string{".notdef", "A", "B", "C", "D"}

	data := buildGSUBHeader(80, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	putU16(data, 18, 1) // SingleSubst
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8)

	// Subtable at 26
	putU16(data, 26, 1)  // format
	putU16(data, 28, 10) // coverageOffset (36)
	putInt16(data, 30, 1) // delta +1

	// Coverage format 2 at 36
	putU16(data, 36, 2) // format 2
	putU16(data, 38, 1) // rangeCount
	putU16(data, 40, 1) // startGlyphID (A)
	putU16(data, 42, 3) // endGlyphID (A..D → glyphs 1-3)
	putU16(data, 44, 0) // startCoverageIndex

	subs, err := DecodeSingleSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// Coverage range 1-3: A,B,C with delta+1: A→B, B→C, C→D
	if len(subs) != 3 {
		t.Fatalf("expected 3 subs, got %d: %v", len(subs), subs)
	}
	for _, s := range subs {
		if s.From == "A" && s.To != "B" {
			t.Fatalf("A should map to B, got %s", s.To)
		}
		if s.From == "B" && s.To != "C" {
			t.Fatalf("B should map to C, got %s", s.To)
		}
		if s.From == "C" && s.To != "D" {
			t.Fatalf("C should map to D, got %s", s.To)
		}
	}
}

// --- Multiple lookups aggregation ---

func TestDecodeSingleSubstMultipleLookups(t *testing.T) {
	// Two lookups of type 1 (SingleSubst), each with one subtable.
	// First: A→C (delta +2). Second: B→C (delta +1).
	glyphOrder := []string{".notdef", "A", "B", "C"}

	data := buildGSUBHeader(120, 14)
	// LookupList at 14: 2 lookups
	putU16(data, 14, 2) // count
	putU16(data, 16, 6) // offset to lookup 0 (14+6=20)
	putU16(data, 18, 22) // offset to lookup 1 (14+22=36) - will adjust

	// Need to calculate sizes properly.
	// Lookup 0 at offset (from lookupList start 14) = 6+2 = 8 bytes header
	// Actually, let me just place them sequentially.

	// Lookup 0 at byte 20 (lookupList offset +6)
	// Lookup 1 at byte 40 (after lookup 0 + subtable)
	// Better approach: build lookup by lookup

	// Lookup 0 starts at offset 6 from lookupList start
	// But offsets in the lookup list are from lookupList start.
	// lookupListOffset = 14
	// lookupList: count(2) + offset0(2) + offset1(2) = 6 bytes → ends at 20
	// offsets are relative to lookupList start

	// Lookup 0 at lookupListOffset+6 = 20:
	// type(2)+flag(2)+count(2)+offset(2) = 8 bytes → 20-27
	// Subtable for lookup 0 at 20+8=28: SingleSubst fmt1
	// subtable: fmt(2)+covOff(2)+delta(2) = 6 bytes + Coverage

	// Let me compute the exact layout dynamically.
	// I'll use a simple approach: lay everything out in known positions.

	// Layout:
	// 00-13: GSUB header
	// 14-15: lookupList count=2
	// 16-17: offset to lookup 0 (relative to lookupList start 14) = 6 → lookup0 at 20
	// 18-19: offset to lookup 1 = 24 → lookup1 at 38
	//   Lookup0 at 20: type=1,flag=0,count=1,subtableOff=8 → sub at 28
	//     Subtable at 28: fmt1, covOff=8, delta=+2
	//       Coverage at 36: fmt1, count=1, glyph A(1)
	//   Lookup1 at 38: type=1,flag=0,count=1,subtableOff=8 → sub at 46
	//     Subtable at 46: fmt1, covOff=8, delta=+1
	//       Coverage at 54: fmt1, count=1, glyph B(2)

	// But we need lookup 1 offset relative to lookupList (14).
	// lookup 1 at byte 38 → offset = 38-14 = 24

	putU16(data, 14, 2)  // lookupCount
	putU16(data, 16, 6)  // offset0 → lookup0 at 20
	putU16(data, 18, 24) // offset1 → lookup1 at 38

	// Lookup 0 at 20
	putU16(data, 20, 1) // type = SingleSubst
	putU16(data, 22, 0) // flag
	putU16(data, 24, 1) // subtableCount
	putU16(data, 26, 8) // subtableOffset from lookup start → 28

	// Subtable at 28: fmt1
	putU16(data, 28, 1)   // format
	putU16(data, 30, 8)   // coverageOffset → 36
	putInt16(data, 32, 2) // delta +2

	// Coverage at 36
	putU16(data, 36, 1) // format
	putU16(data, 38, 1) // count
	putU16(data, 40, 1) // A

	// Lookup 1 at 38 → but 36-41 is coverage... conflict!
	// I need to place lookup1 after coverage.
	// Let me recompute with more space.

	data = buildGSUBHeader(120, 14)

	putU16(data, 14, 2)  // lookupCount
	putU16(data, 16, 6)  // offset0 → 20
	putU16(data, 18, 26) // offset1 → 40

	// Lookup 0 at 20
	putU16(data, 20, 1) // type
	putU16(data, 22, 0) // flag
	putU16(data, 24, 1) // subtableCount
	putU16(data, 26, 8) // subtableOff → 28

	// Subtable at 28
	putU16(data, 28, 1)   // format
	putU16(data, 30, 8)   // coverageOffset → 36
	putInt16(data, 32, 2) // delta +2

	// Coverage at 36
	putU16(data, 36, 1) // format
	putU16(data, 38, 1) // count
	putU16(data, 40, 1) // A

	// Lookup 1 at 40 (offset from lookupList=26)
	// Wait, 14+26=40. But coverage for lookup0 uses bytes 36-41.
	// Coverage at 36: 36(hdr)+38(count)+40(glyph) = 36+6=42
	// So 40 is used by coverage. Need to push lookup1 further out.

	data = buildGSUBHeader(120, 14)

	// Let me be really explicit about byte layout:
	// GSUB header: 0-13 (14 bytes)
	// LookupList at 14: count(2)=2, off0(2), off1(2) = 14-19
	// So lookup offsets start at byte 16 (off0) and 18 (off1).
	// Offsets are relative to lookupList start (14).
	// off0 = 6 → lookup0 at 14+6 = 20
	// Need lookup1 after all of lookup0's data.

	// Lookup0 at 20: type(2)+flag(2)+count(2)+off0(2) = 20-27 (8 bytes)
	// Subtable0 at 28: fmt(2)+covOff(2)+delta(2) = 28-33 (6 bytes)
	// Coverage0 at 36 (28+8): fmt(2)+count(2)+gid(2) = 36-41 (6 bytes)
	// → lookup0 data ends at 42.

	// off1 = 42-14 = 28

	putU16(data, 14, 2)  // lookupCount
	putU16(data, 16, 6)  // off0
	putU16(data, 18, 28) // off1 → 14+28=42

	// Lookup0 at 20
	putU16(data, 20, 1) // type
	putU16(data, 22, 0)
	putU16(data, 24, 1)  // subtableCount
	putU16(data, 26, 8)  // subtableOff → 28

	// Subtable0 at 28
	putU16(data, 28, 1)   // format
	putU16(data, 30, 8)   // covOff → 36
	putInt16(data, 32, 2) // delta +2

	// Coverage0 at 36
	putU16(data, 36, 1) // format
	putU16(data, 38, 1) // count
	putU16(data, 40, 1) // A

	// Lookup1 at 42
	putU16(data, 42, 1) // type
	putU16(data, 44, 0)
	putU16(data, 46, 1)  // subtableCount
	putU16(data, 48, 8)  // subtableOff → 50

	// Subtable1 at 50
	putU16(data, 50, 1)   // format
	putU16(data, 52, 8)   // covOff → 58
	putInt16(data, 54, 1) // delta +1

	// Coverage1 at 58
	putU16(data, 58, 1) // format
	putU16(data, 60, 1) // count
	putU16(data, 62, 2) // B

	subs, err := DecodeSingleSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subs, got %d", len(subs))
	}
	// A→C (delta+2) and B→C (delta+1)
	foundAC := false
	foundBC := false
	for _, s := range subs {
		if s.From == "A" && s.To == "C" {
			foundAC = true
		}
		if s.From == "B" && s.To == "C" {
			foundBC = true
		}
	}
	if !foundAC {
		t.Fatal("missing A→C substitution")
	}
	if !foundBC {
		t.Fatal("missing B→C substitution")
	}
}