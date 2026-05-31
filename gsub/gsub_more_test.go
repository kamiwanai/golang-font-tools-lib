package gsub

import (
	"testing"
)

// --- AlternateSubst tests ---

func TestDecodeAlternateSubstFormat1(t *testing.T) {
	// AlternateSubst Format 1: glyph A(1) → alternates [C(3), D(4)]
	glyphOrder := []string{".notdef", "A", "B", "C", "D"}

	data := buildGSUBHeader(80, 14)
	putU16(data, 14, 1) // lookupCount
	putU16(data, 16, 4) // offset to lookup = 18

	// Lookup at 18
	putU16(data, 18, 3) // AlternateSubst
	putU16(data, 20, 0)  // flag
	putU16(data, 22, 1)  // subtableCount
	putU16(data, 24, 8)  // offset from lookup start → subtable at 26

	// Subtable (AlternateSubst format 1) at 26
	// Layout:
	//   26: format=1
	//   28: coverageOffset
	//   30: altSetCount=1
	//   32: altSetOffset[0]
	// Coverage at offset from subtable
	// AlternateSet at offset from subtable

	subtableBase := 26
	// Place coverage after header: subtable header is 2+2+2+2=8 bytes → header ends at 34
	// AlternateSet after coverage
	// Coverage format 1: 2+2+1*2 = 6 bytes
	// AlternateSet: 2 + 2*2 = 6 bytes

	// Step 1: compute offsets
	// header: 8 bytes (26-33)
	// coverage at offset 8 from subtable (34)
	// coverage is 6 bytes (34-39)
	// altSet at offset 14 from subtable (40)
	// altSet is 6 bytes (40-45)

	putU16(data, subtableBase, 1)    // format
	putU16(data, subtableBase+2, 8)  // coverageOffset → 34
	putU16(data, subtableBase+4, 1)  // altSetCount
	putU16(data, subtableBase+6, 14) // altSetOffset[0] → 40

	// Coverage at 34
	putU16(data, 34, 1) // format
	putU16(data, 36, 1) // count
	putU16(data, 38, 1) // A

	// AlternateSet at 40
	putU16(data, 40, 2) // glyphCount=2
	putU16(data, 42, 3) // C
	putU16(data, 44, 4) // D

	alts, err := DecodeAlternateSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(alts) != 1 {
		t.Fatalf("expected 1 alternate subst, got %d", len(alts))
	}
	if alts[0].From != "A" {
		t.Fatalf("expected From=A, got %s", alts[0].From)
	}
	if len(alts[0].Alternates) != 2 {
		t.Fatalf("expected 2 alternates, got %d", len(alts[0].Alternates))
	}
	if alts[0].Alternates[0] != "C" || alts[0].Alternates[1] != "D" {
		t.Fatalf("expected alternates [C, D], got %v", alts[0].Alternates)
	}
}

func TestDecodeAlternateSubstMultipleGlyphs(t *testing.T) {
	// A→[C,D], B→[E]
	glyphOrder := []string{".notdef", "A", "B", "C", "D", "E"}

	data := buildGSUBHeader(120, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	putU16(data, 18, 3) // AlternateSubst
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable at 26

	subtableBase := 26
	// header: fmt(2) + covOff(2) + altSetCount(2) + 2*altSetOff(4) = 10 bytes (26-35)
	// Coverage at offset 10 → 36: fmt(2)+count(2)+2*glyphs(4)=8 bytes → 36-43
	// AltSet0 at offset 18 → 44: count(2)+2*gids(4)=6 bytes → 44-49
	// AltSet1 at offset 24 → 50: count(2)+1*gid(2)=4 bytes → 50-53

	putU16(data, subtableBase, 1)    // format
	putU16(data, subtableBase+2, 10) // coverageOffset → 36
	putU16(data, subtableBase+4, 2)  // altSetCount
	putU16(data, subtableBase+6, 18) // altSetOff[0] → 44
	putU16(data, subtableBase+8, 24) // altSetOff[1] → 50

	// Coverage at 36
	putU16(data, 36, 1) // format
	putU16(data, 38, 2) // count
	putU16(data, 40, 1) // A
	putU16(data, 42, 2) // B

	// AlternateSet 0 at 44: A→[C, D]
	putU16(data, 44, 2) // count
	putU16(data, 46, 3) // C
	putU16(data, 48, 4) // D

	// AlternateSet 1 at 50: B→[E]
	putU16(data, 50, 1) // count
	putU16(data, 52, 5) // E

	alts, err := DecodeAlternateSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(alts) != 2 {
		t.Fatalf("expected 2 alternate substs, got %d", len(alts))
	}
	if alts[0].From != "A" {
		t.Fatalf("expected From=A, got %s", alts[0].From)
	}
	if len(alts[0].Alternates) != 2 || alts[0].Alternates[0] != "C" || alts[0].Alternates[1] != "D" {
		t.Fatalf("expected A→[C,D], got %v", alts[0].Alternates)
	}
	if alts[1].From != "B" {
		t.Fatalf("expected From=B, got %s", alts[1].From)
	}
	if len(alts[1].Alternates) != 1 || alts[1].Alternates[0] != "E" {
		t.Fatalf("expected B→[E], got %v", alts[1].Alternates)
	}
}

func TestDecodeAlternateSubstExtensionLookup(t *testing.T) {
	// Extension lookup type 9 → type 3 (AlternateSubst)
	glyphOrder := []string{".notdef", "A", "B", "C"}

	data := buildGSUBHeader(80, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	// Lookup type 9 at 18
	putU16(data, 18, 9)
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable at 26

	// Extension subtable at 26
	putU16(data, 26, 1) // format 1
	putU16(data, 28, 3)  // extensionType = AlternateSubst
	putU32(data, 30, 8)  // offset → 34

	// Inner AlternateSubst format 1 at 34
	// header: fmt(2)+covOff(2)+altSetCount(2)+altSetOff(2)=8 bytes → 34-41
	// Coverage at offset 8 → 42: fmt(2)+count(2)+gid(2)=6 bytes → 42-47
	// AlternateSet at offset 14 → 48: count(2)+gid(2)=4 bytes → 48-51

	putU16(data, 34, 1)  // format
	putU16(data, 36, 8)  // coverageOffset → 42
	putU16(data, 38, 1)  // altSetCount
	putU16(data, 40, 14) // altSetOffset → 48

	// Coverage at 42
	putU16(data, 42, 1) // format
	putU16(data, 44, 1) // count
	putU16(data, 46, 1) // A

	// AlternateSet at 48
	putU16(data, 48, 1) // count
	putU16(data, 50, 3) // C

	alts, err := DecodeAlternateSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("extension lookup error: %v", err)
	}
	if len(alts) != 1 {
		t.Fatalf("expected 1 alt, got %d", len(alts))
	}
	if alts[0].From != "A" {
		t.Fatalf("expected From=A, got %s", alts[0].From)
	}
	if len(alts[0].Alternates) != 1 || alts[0].Alternates[0] != "C" {
		t.Fatalf("expected A→[C], got %v", alts[0].Alternates)
	}
}

func TestDecodeAlternateSubstTruncated(t *testing.T) {
	_, err := DecodeAlternateSubstLookups(make([]byte, 5), []string{".notdef"})
	if err == nil {
		t.Fatal("expected error for truncated GSUB header")
	}
}

// --- Extension path tests for LigatureSubst and MultipleSubst ---

func TestDecodeLigatureSubstExtensionLookup(t *testing.T) {
	// Extension lookup type 9 → type 4 (LigatureSubst)
	glyphOrder := []string{".notdef", "space", "A", "B", "C", "D", "f", "i", "fi"}

	data := buildGSUBHeader(100, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	// Lookup type 9 at 18
	putU16(data, 18, 9)
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable at 26

	// Extension at 26
	putU16(data, 26, 1) // format
	putU16(data, 28, 4) // extensionType = LigatureSubst
	putU32(data, 30, 8) // offset → 34

	// Inner LigatureSubst format 1 at 34
	// subtable header: fmt(2)+covOff(2)+ligSetCount(2)+ligSetOff(2)=8 bytes → 34-41
	// LigSet at offset 8 → 42: count(2)+ligOff(2)=4 bytes → 42-45
	// Ligature at offset 12 → 46: ligGlyph(2)+compCount(2)+comp(2)=6 bytes → 46-51
	// Coverage at offset 18 → 52: fmt(2)+count(2)+gid(2)=6 bytes → 52-57

	putU16(data, 34, 1)  // format
	putU16(data, 36, 18) // coverageOffset → 52
	putU16(data, 38, 1)  // ligSetCount
	putU16(data, 40, 8)  // ligSetOff[0] → 42

	// LigSet at 42
	putU16(data, 42, 1) // ligCount
	putU16(data, 44, 4) // ligOff → 46

	// Ligature at 46
	putU16(data, 46, 8) // fi
	putU16(data, 48, 2) // compCount
	putU16(data, 50, 7) // i

	// Coverage at 52
	putU16(data, 52, 1) // format
	putU16(data, 54, 1) // count
	putU16(data, 56, 6) // f

	ligs, err := DecodeLigatureSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("extension lookup error: %v", err)
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

func TestDecodeMultipleSubstExtensionLookup(t *testing.T) {
	// Extension lookup type 9 → type 2 (MultipleSubst)
	glyphOrder := []string{".notdef", "A", "B", "C", "D", "E", "F", "G"}

	data := buildGSUBHeader(100, 14)
	putU16(data, 14, 1)
	putU16(data, 16, 4)

	// Lookup type 9 at 18
	putU16(data, 18, 9)
	putU16(data, 20, 0)
	putU16(data, 22, 1)
	putU16(data, 24, 8) // subtable at 26

	// Extension at 26
	putU16(data, 26, 1) // format
	putU16(data, 28, 2) // extensionType = MultipleSubst
	putU32(data, 30, 8) // offset → 34

	// Inner MultipleSubst format 1 at 34
	// header: fmt(2)+covOff(2)+seqCount(2)+seqOff(2)=8 bytes → 34-41
	// Coverage at offset 8 → 42: fmt(2)+count(2)+gid(2)=6 bytes → 42-47
	// Sequence at offset 14 → 48: count(2)+2*gids(4)=6 bytes → 48-53

	putU16(data, 34, 1)  // format
	putU16(data, 36, 8)  // coverageOffset → 42
	putU16(data, 38, 1)  // sequenceCount
	putU16(data, 40, 14) // seqOffset[0] → 48

	// Coverage at 42
	putU16(data, 42, 1) // format
	putU16(data, 44, 1) // count
	putU16(data, 46, 5) // E

	// Sequence at 48
	putU16(data, 48, 2) // glyphCount
	putU16(data, 50, 6) // F
	putU16(data, 52, 7) // G

	subs, err := DecodeMultipleSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("extension lookup error: %v", err)
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