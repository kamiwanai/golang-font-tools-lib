// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gpos

import (
	"encoding/binary"
	"testing"
)

func TestDecodePairPosFormat1CoveragePairs(t *testing.T) {
	data := make([]byte, 28)
	binary.BigEndian.PutUint16(data[0:2], 1)        // PairPos format 1
	binary.BigEndian.PutUint16(data[2:4], 12)       // Coverage offset
	binary.BigEndian.PutUint16(data[4:6], 0x0004)   // ValueFormat1: XAdvance
	binary.BigEndian.PutUint16(data[6:8], 0)        // ValueFormat2
	binary.BigEndian.PutUint16(data[8:10], 1)       // PairSetCount
	binary.BigEndian.PutUint16(data[10:12], 18)     // PairSet offset for A
	binary.BigEndian.PutUint16(data[12:14], 1)      // Coverage format 1
	binary.BigEndian.PutUint16(data[14:16], 1)      // GlyphCount
	binary.BigEndian.PutUint16(data[16:18], 1)      // A glyph ID
	binary.BigEndian.PutUint16(data[18:20], 2)      // PairValueCount
	binary.BigEndian.PutUint16(data[20:22], 2)      // V glyph ID
	binary.BigEndian.PutUint16(data[22:24], 0xffb0) // -80
	binary.BigEndian.PutUint16(data[24:26], 3)      // T glyph ID
	binary.BigEndian.PutUint16(data[26:28], 0)

	pairs, err := DecodePairPosFormat1(data, []string{".notdef", "A", "V", "T"})
	if err != nil {
		t.Fatalf("DecodePairPosFormat1 returned error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("pair count = %d, want 1", len(pairs))
	}
	got := pairs[0]
	if got.LeftGlyph != "A" || got.RightGlyph != "V" || got.Value != -80 {
		t.Fatalf("pair = %+v, want A/V -80", got)
	}
}

func TestDecodePairPosFormat1AddsValue2XAdvance(t *testing.T) {
	data := make([]byte, 24)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 12)
	binary.BigEndian.PutUint16(data[4:6], 0)
	binary.BigEndian.PutUint16(data[6:8], 0x0004) // ValueFormat2: XAdvance
	binary.BigEndian.PutUint16(data[8:10], 1)
	binary.BigEndian.PutUint16(data[10:12], 18)
	binary.BigEndian.PutUint16(data[12:14], 1)
	binary.BigEndian.PutUint16(data[14:16], 1)
	binary.BigEndian.PutUint16(data[16:18], 1)
	binary.BigEndian.PutUint16(data[18:20], 1)
	binary.BigEndian.PutUint16(data[20:22], 2)
	binary.BigEndian.PutUint16(data[22:24], 0xfff1) // -15

	pairs, err := DecodePairPosFormat1(data, []string{".notdef", "A", "V"})
	if err != nil {
		t.Fatalf("DecodePairPosFormat1 returned error: %v", err)
	}

	if len(pairs) != 1 {
		t.Fatalf("pair count = %d, want 1", len(pairs))
	}
	if pairs[0].Value != -15 {
		t.Fatalf("value = %d, want -15", pairs[0].Value)
	}
}

func TestDecodePairPosLookupsReadsLookupType2Format1(t *testing.T) {
	gpos := make([]byte, 84)
	binary.BigEndian.PutUint16(gpos[0:2], 1)   // major
	binary.BigEndian.PutUint16(gpos[2:4], 0)   // minor
	binary.BigEndian.PutUint16(gpos[8:10], 10) // LookupList offset
	binary.BigEndian.PutUint16(gpos[10:12], 1) // LookupCount
	binary.BigEndian.PutUint16(gpos[12:14], 4) // Lookup offset from LookupList
	binary.BigEndian.PutUint16(gpos[14:16], 2) // LookupType PairPos
	binary.BigEndian.PutUint16(gpos[18:20], 1) // SubTableCount
	binary.BigEndian.PutUint16(gpos[20:22], 8) // Subtable offset from Lookup

	subtable := gpos[22:]
	binary.BigEndian.PutUint16(subtable[0:2], 1)
	binary.BigEndian.PutUint16(subtable[2:4], 12)
	binary.BigEndian.PutUint16(subtable[4:6], 0x0004)
	binary.BigEndian.PutUint16(subtable[8:10], 1)
	binary.BigEndian.PutUint16(subtable[10:12], 18)
	binary.BigEndian.PutUint16(subtable[12:14], 1)
	binary.BigEndian.PutUint16(subtable[14:16], 1)
	binary.BigEndian.PutUint16(subtable[16:18], 1)
	binary.BigEndian.PutUint16(subtable[18:20], 1)
	binary.BigEndian.PutUint16(subtable[20:22], 2)
	binary.BigEndian.PutUint16(subtable[22:24], 0xffb0)

	pairs, err := DecodePairPosLookups(gpos, []string{".notdef", "A", "V"})
	if err != nil {
		t.Fatalf("DecodePairPosLookups returned error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("pair count = %d, want 1", len(pairs))
	}
	if pairs[0].LeftGlyph != "A" || pairs[0].RightGlyph != "V" || pairs[0].Value != -80 {
		t.Fatalf("pair = %+v, want A/V -80", pairs[0])
	}
}

func TestDecodePairPosFormat2ExpandsClassPairs(t *testing.T) {
	data := buildPairPosFormat2Fixture()

	pairs, err := DecodePairPosFormat2(data, []string{".notdef", "A", "T", "V", "W"})
	if err != nil {
		t.Fatalf("DecodePairPosFormat2 returned error: %v", err)
	}

	got := map[string]int{}
	for _, pair := range pairs {
		got[pair.LeftGlyph+"/"+pair.RightGlyph] = pair.Value
	}
	want := map[string]int{"A/V": -70, "A/W": -70, "T/V": -10, "T/W": -10}
	if len(got) != len(want) {
		t.Fatalf("pair count = %d, want %d: %v", len(got), len(want), got)
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("%s = %d, want %d", key, got[key], value)
		}
	}
}

func TestDecodePairPosLookupsReadsLookupType2Format2(t *testing.T) {
	gpos := make([]byte, 22)
	binary.BigEndian.PutUint16(gpos[0:2], 1)
	binary.BigEndian.PutUint16(gpos[8:10], 10)
	binary.BigEndian.PutUint16(gpos[10:12], 1)
	binary.BigEndian.PutUint16(gpos[12:14], 4)
	binary.BigEndian.PutUint16(gpos[14:16], 2)
	binary.BigEndian.PutUint16(gpos[18:20], 1)
	binary.BigEndian.PutUint16(gpos[20:22], 8)
	gpos = append(gpos, buildPairPosFormat2Fixture()...)

	pairs, err := DecodePairPosLookups(gpos, []string{".notdef", "A", "T", "V", "W"})
	if err != nil {
		t.Fatalf("DecodePairPosLookups returned error: %v", err)
	}
	if len(pairs) != 4 {
		t.Fatalf("pair count = %d, want 4", len(pairs))
	}
}

func TestDecodePairPosLookupsReadsExtensionLookupType9(t *testing.T) {
	gpos := make([]byte, 82)
	binary.BigEndian.PutUint16(gpos[0:2], 1)
	binary.BigEndian.PutUint16(gpos[8:10], 10)
	binary.BigEndian.PutUint16(gpos[10:12], 1)
	binary.BigEndian.PutUint16(gpos[12:14], 4)
	binary.BigEndian.PutUint16(gpos[14:16], 9) // ExtensionPos
	binary.BigEndian.PutUint16(gpos[18:20], 1)
	binary.BigEndian.PutUint16(gpos[20:22], 8)

	extension := gpos[22:]
	binary.BigEndian.PutUint16(extension[0:2], 1) // ExtensionPos format 1
	binary.BigEndian.PutUint16(extension[2:4], 2) // ExtensionLookupType PairPos
	binary.BigEndian.PutUint32(extension[4:8], 8) // ExtensionOffset
	copy(extension[8:], buildPairPosFormat2Fixture())

	pairs, err := DecodePairPosLookups(gpos, []string{".notdef", "A", "T", "V", "W"})
	if err != nil {
		t.Fatalf("DecodePairPosLookups returned error: %v", err)
	}
	if len(pairs) != 4 {
		t.Fatalf("pair count = %d, want 4", len(pairs))
	}
	if pairs[0].Value != -10 && pairs[0].Value != -70 {
		t.Fatalf("unexpected extension pair value %d", pairs[0].Value)
	}
}

func TestDecodePairPosLookupsSkipsExtensionLookupType9NonPairPosTarget(t *testing.T) {
	gpos := make([]byte, 30)
	binary.BigEndian.PutUint16(gpos[0:2], 1)
	binary.BigEndian.PutUint16(gpos[8:10], 10)
	binary.BigEndian.PutUint16(gpos[10:12], 1)
	binary.BigEndian.PutUint16(gpos[12:14], 4)
	binary.BigEndian.PutUint16(gpos[14:16], 9)
	binary.BigEndian.PutUint16(gpos[18:20], 1)
	binary.BigEndian.PutUint16(gpos[20:22], 8)

	extension := gpos[22:]
	binary.BigEndian.PutUint16(extension[0:2], 1)
	binary.BigEndian.PutUint16(extension[2:4], 4) // not PairPos
	binary.BigEndian.PutUint32(extension[4:8], 8)

	pairs, err := DecodePairPosLookups(gpos, []string{".notdef", "A", "V"})
	if err != nil {
		t.Fatalf("DecodePairPosLookups returned error: %v", err)
	}
	if len(pairs) != 0 {
		t.Fatalf("pair count = %d, want 0", len(pairs))
	}
}

func TestDecodePairPosLookupsAggregatesRepeatedPairs(t *testing.T) {
	gpos := make([]byte, 16)
	binary.BigEndian.PutUint16(gpos[0:2], 1)
	binary.BigEndian.PutUint16(gpos[8:10], 10)
	binary.BigEndian.PutUint16(gpos[10:12], 2)
	binary.BigEndian.PutUint16(gpos[12:14], 6)
	gpos = append(gpos, buildLookupType2Format1Fixture(-80)...)
	binary.BigEndian.PutUint16(gpos[14:16], uint16(len(gpos)-10))
	gpos = append(gpos, buildLookupType2Format1Fixture(-20)...)

	pairs, err := DecodePairPosLookups(gpos, []string{".notdef", "A", "V"})
	if err != nil {
		t.Fatalf("DecodePairPosLookups returned error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("pair count = %d, want 1", len(pairs))
	}
	if pairs[0].LeftGlyph != "A" || pairs[0].RightGlyph != "V" || pairs[0].Value != -100 {
		t.Fatalf("pair = %+v, want A/V -100", pairs[0])
	}
}

func TestDecodePairPosLookupsAggregatesPairSources(t *testing.T) {
	gpos := make([]byte, 16)
	binary.BigEndian.PutUint16(gpos[0:2], 1)
	binary.BigEndian.PutUint16(gpos[8:10], 10)
	binary.BigEndian.PutUint16(gpos[10:12], 2)
	binary.BigEndian.PutUint16(gpos[12:14], 6)
	gpos = append(gpos, buildLookupType2Format1FixtureWithRightGlyph(-80, 3)...)
	binary.BigEndian.PutUint16(gpos[14:16], uint16(len(gpos)-10))
	gpos = append(gpos, buildExtensionLookupType9Format2Fixture()...)

	pairs, err := DecodePairPosLookups(gpos, []string{".notdef", "A", "T", "V", "W"})
	if err != nil {
		t.Fatalf("DecodePairPosLookups returned error: %v", err)
	}
	for _, pair := range pairs {
		if pair.LeftGlyph == "A" && pair.RightGlyph == "V" {
			if pair.Value != -150 {
				t.Fatalf("A/V value = %d, want -150", pair.Value)
			}
			if len(pair.Sources) != 2 {
				t.Fatalf("A/V source count = %d, want 2", len(pair.Sources))
			}
			if pair.Sources[0].LookupIndex != 0 || pair.Sources[0].Format != 1 || pair.Sources[0].Value != -80 {
				t.Fatalf("first source = %+v, want lookup 0 format 1 value -80", pair.Sources[0])
			}
			if pair.Sources[1].LookupIndex != 1 || pair.Sources[1].Format != 2 || pair.Sources[1].Value != -70 {
				t.Fatalf("second source = %+v, want lookup 1 format 2 value -70", pair.Sources[1])
			}
			return
		}
	}
	t.Fatal("missing aggregated A/V pair")
}

func TestDecodeClassDefRejectsTruncatedFormat1(t *testing.T) {
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 1)

	if _, err := decodeClassDef(data, 0); err == nil {
		t.Fatal("decodeClassDef succeeded for truncated format 1")
	}
}

func buildPairPosFormat2Fixture() []byte {
	data := make([]byte, 52)
	binary.BigEndian.PutUint16(data[0:2], 2)      // PairPos format 2
	binary.BigEndian.PutUint16(data[2:4], 24)     // Coverage offset
	binary.BigEndian.PutUint16(data[4:6], 0x0004) // ValueFormat1: XAdvance
	binary.BigEndian.PutUint16(data[6:8], 0)
	binary.BigEndian.PutUint16(data[8:10], 32)  // ClassDef1 offset
	binary.BigEndian.PutUint16(data[10:12], 42) // ClassDef2 offset
	binary.BigEndian.PutUint16(data[12:14], 2)  // Class1Count
	binary.BigEndian.PutUint16(data[14:16], 2)  // Class2Count
	binary.BigEndian.PutUint16(data[16:18], 0)
	binary.BigEndian.PutUint16(data[18:20], 0xfff6) // class0/class1 = -10
	binary.BigEndian.PutUint16(data[20:22], 0)
	binary.BigEndian.PutUint16(data[22:24], 0xffba) // class1/class1 = -70
	binary.BigEndian.PutUint16(data[24:26], 1)
	binary.BigEndian.PutUint16(data[26:28], 2)
	binary.BigEndian.PutUint16(data[28:30], 1) // A
	binary.BigEndian.PutUint16(data[30:32], 2) // T
	binary.BigEndian.PutUint16(data[32:34], 1)
	binary.BigEndian.PutUint16(data[34:36], 1) // start A
	binary.BigEndian.PutUint16(data[36:38], 2)
	binary.BigEndian.PutUint16(data[38:40], 1) // A class 1
	binary.BigEndian.PutUint16(data[40:42], 0) // T class 0
	binary.BigEndian.PutUint16(data[42:44], 1)
	binary.BigEndian.PutUint16(data[44:46], 3) // start V
	binary.BigEndian.PutUint16(data[46:48], 2)
	binary.BigEndian.PutUint16(data[48:50], 1) // V class 1
	binary.BigEndian.PutUint16(data[50:52], 1) // W class 1
	return data
}

func buildLookupType2Format1Fixture(value int16) []byte {
	return buildLookupType2Format1FixtureWithRightGlyph(value, 2)
}

func buildLookupType2Format1FixtureWithRightGlyph(value int16, rightGlyphID uint16) []byte {
	lookup := make([]byte, 32)
	binary.BigEndian.PutUint16(lookup[0:2], 2)
	binary.BigEndian.PutUint16(lookup[4:6], 1)
	binary.BigEndian.PutUint16(lookup[6:8], 8)
	subtable := lookup[8:]
	binary.BigEndian.PutUint16(subtable[0:2], 1)
	binary.BigEndian.PutUint16(subtable[2:4], 12)
	binary.BigEndian.PutUint16(subtable[4:6], 0x0004)
	binary.BigEndian.PutUint16(subtable[8:10], 1)
	binary.BigEndian.PutUint16(subtable[10:12], 18)
	binary.BigEndian.PutUint16(subtable[12:14], 1)
	binary.BigEndian.PutUint16(subtable[14:16], 1)
	binary.BigEndian.PutUint16(subtable[16:18], 1)
	binary.BigEndian.PutUint16(subtable[18:20], 1)
	binary.BigEndian.PutUint16(subtable[20:22], rightGlyphID)
	binary.BigEndian.PutUint16(subtable[22:24], uint16(value))
	return lookup
}

func buildExtensionLookupType9Format2Fixture() []byte {
	lookup := make([]byte, 68)
	binary.BigEndian.PutUint16(lookup[0:2], 9)
	binary.BigEndian.PutUint16(lookup[4:6], 1)
	binary.BigEndian.PutUint16(lookup[6:8], 8)
	extension := lookup[8:]
	binary.BigEndian.PutUint16(extension[0:2], 1)
	binary.BigEndian.PutUint16(extension[2:4], 2)
	binary.BigEndian.PutUint32(extension[4:8], 8)
	copy(extension[8:], buildPairPosFormat2Fixture())
	return lookup
}
