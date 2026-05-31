// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gpos

import (
	"encoding/binary"
	"testing"

	"github.com/kamiwanai/golang-font-tools-lib/opentype"
	"golang.org/x/image/font/gofont/goregular"
)

func buildMinimalGPOS(subtable []byte, lookupType uint16) []byte {
	headerSize := 10
	lookupListSize := 2 + 2
	lookupSize := 6 + 2
	lookupListOff := headerSize
	lookupOff := headerSize + lookupListSize
	subtableOff := headerSize + lookupListSize + lookupSize
	total := subtableOff + len(subtable)
	data := make([]byte, total)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 0)
	binary.BigEndian.PutUint16(data[4:6], 0)
	binary.BigEndian.PutUint16(data[6:8], 0)
	binary.BigEndian.PutUint16(data[8:10], uint16(lookupListOff))
	binary.BigEndian.PutUint16(data[lookupListOff:lookupListOff+2], 1)
	binary.BigEndian.PutUint16(data[lookupListOff+2:lookupListOff+4], uint16(lookupOff-lookupListOff))
	binary.BigEndian.PutUint16(data[lookupOff:lookupOff+2], lookupType)
	binary.BigEndian.PutUint16(data[lookupOff+2:lookupOff+4], 0)
	binary.BigEndian.PutUint16(data[lookupOff+4:lookupOff+6], 1)
	binary.BigEndian.PutUint16(data[lookupOff+6:lookupOff+8], uint16(subtableOff-lookupOff))
	copy(data[subtableOff:], subtable)
	return data
}


func TestDecodeSinglePosFormat1(t *testing.T) {
	vf := uint16(0x0004) // XAdvance only
	vrSize := valueRecordSize(vf) // = 2
	covOff := 6 + vrSize         // coverage after ValueRecord

	sub := make([]byte, covOff+10)
	binary.BigEndian.PutUint16(sub[0:2], 1)       // PosFormat 1
	binary.BigEndian.PutUint16(sub[2:4], uint16(covOff)) // Coverage offset
	binary.BigEndian.PutUint16(sub[4:6], vf)       // ValueFormat
	// ValueRecord at offset 6
	binary.BigEndian.PutUint16(sub[6:8], 0xfff0)   // -16 XAdvance
	// Coverage at offset covOff=8
	binary.BigEndian.PutUint16(sub[8:10], 1)       // Coverage format 1
	binary.BigEndian.PutUint16(sub[10:12], 3)      // GlyphCount = 3
	binary.BigEndian.PutUint16(sub[12:14], 1)      // glyph A
	binary.BigEndian.PutUint16(sub[14:16], 2)      // glyph V
	binary.BigEndian.PutUint16(sub[16:18], 3)      // glyph T

	data := buildMinimalGPOS(sub, 1)
	vals, err := DecodeSinglePosLookups(data, []string{".notdef", "A", "V", "T"})
	if err != nil {
		t.Fatalf("DecodeSinglePosLookups error: %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("got %d values, want 3", len(vals))
	}
	for _, v := range vals {
		if v.XAdvance != -16 {
			t.Fatalf("XAdvance=%d, want -16 in %+v", v.XAdvance, v)
		}
	}
}

func TestDecodeSinglePosFormat1AllFields(t *testing.T) {
	vf := uint16(0x000F) // all 4 fields
	vrSize := valueRecordSize(vf) // = 8
	covOff := 6 + vrSize           // coverage after ValueRecord

	sub := make([]byte, covOff+6)
	binary.BigEndian.PutUint16(sub[0:2], 1)       // PosFormat 1
	binary.BigEndian.PutUint16(sub[2:4], uint16(covOff)) // Coverage offset
	binary.BigEndian.PutUint16(sub[4:6], vf)       // ValueFormat
	// ValueRecord at offset 6 (8 bytes)
	binary.BigEndian.PutUint16(sub[6:8], 10)       // XPlacement
	binary.BigEndian.PutUint16(sub[8:10], 20)      // YPlacement
	binary.BigEndian.PutUint16(sub[10:12], 0xfff0) // XAdvance = -16
	binary.BigEndian.PutUint16(sub[12:14], 0xffe0) // YAdvance = -32
	// Coverage at offset covOff=14
	binary.BigEndian.PutUint16(sub[14:16], 1)      // Coverage format 1
	binary.BigEndian.PutUint16(sub[16:18], 1)      // GlyphCount = 1
	binary.BigEndian.PutUint16(sub[18:20], 1)      // glyph A

	data := buildMinimalGPOS(sub, 1)
	vals, err := DecodeSinglePosLookups(data, []string{".notdef", "A"})
	if err != nil {
		t.Fatalf("DecodeSinglePosLookups error: %v", err)
	}
	if len(vals) != 1 {
		t.Fatalf("got %d values, want 1", len(vals))
	}
	v := vals[0]
	if v.XPlacement != 10 {
		t.Fatalf("XPlacement=%d, want 10", v.XPlacement)
	}
	if v.YPlacement != 20 {
		t.Fatalf("YPlacement=%d, want 20", v.YPlacement)
	}
	if v.XAdvance != -16 {
		t.Fatalf("XAdvance=%d, want -16", v.XAdvance)
	}
	if v.YAdvance != -32 {
		t.Fatalf("YAdvance=%d, want -32", v.YAdvance)
	}
}

func TestDecodeSinglePosFormat2(t *testing.T) {
	vf := uint16(0x0004) // XAdvance only
	vrSize := valueRecordSize(vf) // = 2
	valueCount := 3
	covOff := 8 + valueCount*vrSize // coverage after ValueRecords

	sub := make([]byte, covOff+10)
	binary.BigEndian.PutUint16(sub[0:2], 2)       // PosFormat 2
	binary.BigEndian.PutUint16(sub[2:4], uint16(covOff)) // Coverage offset
	binary.BigEndian.PutUint16(sub[4:6], vf)       // ValueFormat
	binary.BigEndian.PutUint16(sub[6:8], uint16(valueCount)) // ValueCount
	// ValueRecords at offset 8
	binary.BigEndian.PutUint16(sub[8:10], 0xfff0)  // A: -16
	binary.BigEndian.PutUint16(sub[10:12], 0xffe0) // V: -32
	binary.BigEndian.PutUint16(sub[12:14], 10)     // T: +10
	// Coverage at offset covOff=14
	binary.BigEndian.PutUint16(sub[14:16], 1)      // Coverage format 1
	binary.BigEndian.PutUint16(sub[16:18], 3)      // GlyphCount = 3
	binary.BigEndian.PutUint16(sub[18:20], 1)      // glyph A
	binary.BigEndian.PutUint16(sub[20:22], 2)      // glyph V
	binary.BigEndian.PutUint16(sub[22:24], 3)      // glyph T

	data := buildMinimalGPOS(sub, 1)
	vals, err := DecodeSinglePosLookups(data, []string{".notdef", "A", "V", "T"})
	if err != nil {
		t.Fatalf("DecodeSinglePosLookups error: %v", err)
	}
	if len(vals) != 3 {
		t.Fatalf("got %d values, want 3", len(vals))
	}
	if vals[0].Glyph != "A" || vals[0].XAdvance != -16 {
		t.Fatalf("A: XAdvance=%d want -16", vals[0].XAdvance)
	}
	if vals[1].Glyph != "V" || vals[1].XAdvance != -32 {
		t.Fatalf("V: XAdvance=%d want -32", vals[1].XAdvance)
	}
	if vals[2].Glyph != "T" || vals[2].XAdvance != 10 {
		t.Fatalf("T: XAdvance=%d want 10", vals[2].XAdvance)
	}
}

func TestDecodeSinglePosExtensionType9(t *testing.T) {
	// SinglePosFormat1 inside Extension
	vf := uint16(0x0004)
	vrSize := valueRecordSize(vf)
	innerCovOff := 6 + vrSize // = 8

	// Extension subtable: 8 bytes header + target subtable
	extHeaderSize := 8
	singlePosSize := innerCovOff + 8 // 8 + 8 = 16 (2 format + 2 count + 4 glyphIDs)
	extSub := make([]byte, extHeaderSize + singlePosSize)

	// Extension header
	binary.BigEndian.PutUint16(extSub[0:2], 1)           // Extension format 1
	binary.BigEndian.PutUint16(extSub[2:4], 1)           // extensionType = SinglePos
	binary.BigEndian.PutUint32(extSub[4:8], uint32(extHeaderSize)) // extensionOffset

	// Target SinglePosFormat1 at offset 8
	binary.BigEndian.PutUint16(extSub[8:10], 1)          // PosFormat 1
	binary.BigEndian.PutUint16(extSub[10:12], uint16(innerCovOff)) // Coverage offset
	binary.BigEndian.PutUint16(extSub[12:14], vf)         // ValueFormat
	binary.BigEndian.PutUint16(extSub[14:16], 0xffce)     // ValueRecord: -50
	// Coverage at offset 8+innerCovOff=16
	binary.BigEndian.PutUint16(extSub[16:18], 1)          // Coverage format 1
	binary.BigEndian.PutUint16(extSub[18:20], 2)          // GlyphCount = 2
	binary.BigEndian.PutUint16(extSub[20:22], 1)          // glyph A
	binary.BigEndian.PutUint16(extSub[22:24], 2)          // glyph V

	data := buildMinimalGPOS(extSub, 9)
	vals, err := DecodeSinglePosLookups(data, []string{".notdef", "A", "V"})
	if err != nil {
		t.Fatalf("DecodeSinglePosLookups error: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("got %d values, want 2", len(vals))
	}
	if vals[0].Glyph != "A" || vals[0].XAdvance != -50 {
		t.Fatalf("A: got %+v", vals[0])
	}
	if vals[1].Glyph != "V" || vals[1].XAdvance != -50 {
		t.Fatalf("V: got %+v", vals[1])
	}
}

func TestDecodeSinglePosTruncated(t *testing.T) {
	_, err := DecodeSinglePosLookups([]byte{}, []string{})
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestDecodeSinglePosIgnoredLookupType(t *testing.T) {
	sub := make([]byte, 4)
	data := buildMinimalGPOS(sub, 2) // type 2 = PairPos, should be ignored
	vals, err := DecodeSinglePosLookups(data, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vals) != 0 {
		t.Fatalf("got %d values, want 0", len(vals))
	}
}

func TestDecodeSinglePosFormat2CoverageCountMismatch(t *testing.T) {
	// ValueCount > Coverage GlyphCount should error
	vf := uint16(0x0004)
	covOff := 8 + 2*valueRecordSize(vf) // 12
	sub := make([]byte, covOff+2)
	binary.BigEndian.PutUint16(sub[0:2], 2)
	binary.BigEndian.PutUint16(sub[2:4], uint16(covOff))
	binary.BigEndian.PutUint16(sub[4:6], vf)
	binary.BigEndian.PutUint16(sub[6:8], 2) // ValueCount=2
	binary.BigEndian.PutUint16(sub[8:10], 0)
	binary.BigEndian.PutUint16(sub[10:12], 0)
	binary.BigEndian.PutUint16(sub[12:14], 1) // Coverage: only 1 glyph

	data := buildMinimalGPOS(sub, 1)
	_, err := DecodeSinglePosLookups(data, []string{})
	if err == nil {
		t.Fatal("expected error for valueCount > coverage count")
	}
}

func TestDecodeSinglePosFromGoRegular(t *testing.T) {
	font, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder: %v", err)
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		t.Skip("no GPOS table in Go Regular")
	}
	vals, err := DecodeSinglePosLookups(gposData, glyphOrder)
	if err != nil {
		t.Fatalf("DecodeSinglePosLookups: %v", err)
	}
	t.Logf("DecodeSinglePosLookups returned %d single position values", len(vals))
}

func FuzzSinglePosDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 16))
	f.Fuzz(func(t *testing.T, data []byte) {
		vals, err := DecodeSinglePosLookups(data, []string{})
		if err != nil {
			return
		}
		_ = vals
	})
}
