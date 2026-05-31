// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package mvar

import (
	"encoding/binary"
	"testing"

	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

func putU16(data []byte, offset int, v uint16) {
	binary.BigEndian.PutUint16(data[offset:offset+2], v)
}

func putU32(data []byte, offset int, v uint32) {
	binary.BigEndian.PutUint32(data[offset:offset+4], v)
}

func putTag(data []byte, offset int, tag string) {
	copy(data[offset:offset+4], tag)
}

// buildMinimalItemVariationStore builds an ItemVariationStore with 1 region, 1 IVD, 1 item.
// The IVS is placed at the given offset within 'data'.
// Returns the byte offset where the IVS starts.
func buildItemVariationStore(data []byte, offset int) {
	// VariationRegionList at offset+12 (relative to IVS start)
	putU16(data, offset, 1)       // format
	putU32(data, offset+2, 12)    // varRegionListOffset
	putU16(data, offset+6, 1)     // itemVarDataCount
	putU32(data, offset+8, 20)    // ivdOffsets[0]

	// VariationRegionList at offset+12:
	regOff := offset + 12
	putU16(data, regOff, 1)       // axisCount
	putU16(data, regOff+2, 1)     // regionCount
	putU16(data, regOff+4, 0xC000)   // startCoord = -1.0 in F2DOT14
	putU16(data, regOff+6, 0x4000)   // peakCoord = 1.0
	putU16(data, regOff+8, 0x0000)   // endCoord = 0.0

	// ItemVariationData at offset+20:
	ivdOff := offset + 20
	putU16(data, ivdOff, 1)       // itemCount
	putU16(data, ivdOff+2, 0)     // wordDeltaCount (all 8-bit deltas)
	putU16(data, ivdOff+4, 1)     // regionIdxCount
	putU16(data, ivdOff+6, 0)     // regionIndexes[0] = 0
	data[ivdOff+8] = 50           // delta[0] = 50
}

func buildMinimalMVAR() []byte {
	// MVAR header: 12 bytes + 3 value records (24 bytes) = 36
	// IVS starts at offset 36
	data := make([]byte, 80)

	// Header
	putU16(data, 0, 1)   // majorVersion
	putU16(data, 2, 0)   // minorVersion
	putU16(data, 4, 0)   // reserved
	putU16(data, 6, 8)   // valueRecordSize
	putU16(data, 8, 3)   // valueRecordCount
	putU16(data, 10, 36) // itemVariationStoreOffset

	// Value records (sorted alphabetically per spec)
	// "cpht" = 0x63706874
	putTag(data, 12, "cpht")
	putU16(data, 16, 0) // outer
	putU16(data, 18, 0) // inner

	// "hasc" = 0x68617363
	putTag(data, 20, "hasc")
	putU16(data, 24, 0) // outer
	putU16(data, 26, 0) // inner

	// "xhgt" = 0x78686774
	putTag(data, 28, "xhgt")
	putU16(data, 32, 0) // outer
	putU16(data, 34, 0) // inner

	// ItemVariationStore at offset 36
	buildItemVariationStore(data, 36)

	return data
}

func TestDecodeBasicMVAR(t *testing.T) {
	data := buildMinimalMVAR()
	mv, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if mv.MajorVersion != 1 {
		t.Fatalf("MajorVersion = %d, want 1", mv.MajorVersion)
	}
	if len(mv.ValueRecords) != 3 {
		t.Fatalf("ValueRecords = %d, want 3", len(mv.ValueRecords))
	}
	if mv.ValueRecords[0].ValueTag != "cpht" {
		t.Fatalf("ValueRecords[0].ValueTag = %q, want %q", mv.ValueRecords[0].ValueTag, "cpht")
	}
	if mv.ValueRecords[1].ValueTag != "hasc" {
		t.Fatalf("ValueRecords[1].ValueTag = %q, want %q", mv.ValueRecords[1].ValueTag, "hasc")
	}
	if mv.ValueRecords[2].ValueTag != "xhgt" {
		t.Fatalf("ValueRecords[2].ValueTag = %q, want %q", mv.ValueRecords[2].ValueTag, "xhgt")
	}
}

func TestRecordByTag(t *testing.T) {
	data := buildMinimalMVAR()
	mv, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	// Exact match
	rec := mv.RecordByTag("hasc")
	if rec == nil {
		t.Fatal("RecordByTag(hasc) returned nil")
	}
	if rec.ValueTag != "hasc" {
		t.Fatalf("RecordByTag(hasc).ValueTag = %q", rec.ValueTag)
	}

	// Check HasTag
	if !mv.HasTag("xhgt") {
		t.Fatal("HasTag(xhgt) should be true")
	}
	if mv.HasTag("none") {
		t.Fatal("HasTag(none) should be false")
	}

	// Missing tag
	if mv.RecordByTag("none") != nil {
		t.Fatal("RecordByTag(none) should be nil")
	}

	// First and last
	if mv.RecordByTag("cpht") == nil {
		t.Fatal("RecordByTag(cpht) returned nil")
	}
	if mv.RecordByTag("xhgt") == nil {
		t.Fatal("RecordByTag(xhgt) returned nil")
	}
}

func TestDecodeMVARNoRecords(t *testing.T) {
	data := make([]byte, 12)
	putU16(data, 0, 1)   // majorVersion
	putU16(data, 2, 0)   // minorVersion
	putU16(data, 4, 0)   // reserved
	putU16(data, 6, 8)   // valueRecordSize
	putU16(data, 8, 0)   // valueRecordCount = 0
	putU16(data, 10, 0)  // itemVariationStoreOffset = 0

	mv, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(mv.ValueRecords) != 0 {
		t.Fatalf("ValueRecords = %d, want 0", len(mv.ValueRecords))
	}
	if mv.RecordByTag("hasc") != nil {
		t.Fatal("RecordByTag should return nil for empty table")
	}
}

func TestDecodeMVARTruncatedHeader(t *testing.T) {
	_, err := Decode(make([]byte, 4))
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestDecodeMVARBadVersion(t *testing.T) {
	data := make([]byte, 12)
	putU16(data, 0, 2) // bad version
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
}

func TestDecodeMVARRecordsExceedTable(t *testing.T) {
	data := make([]byte, 14)
	putU16(data, 0, 1)   // majorVersion
	putU16(data, 2, 0)   // minorVersion
	putU16(data, 4, 0)   // reserved
	putU16(data, 6, 8)   // valueRecordSize
	putU16(data, 8, 5)   // valueRecordCount (5 records = 40 bytes, but data is short)
	putU16(data, 10, 0)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for records exceeding table")
	}
}

func TestDecodeMVARRecordsWithNoIVS(t *testing.T) {
	data := make([]byte, 20)
	putU16(data, 0, 1)   // majorVersion
	putU16(data, 2, 0)   // minorVersion
	putU16(data, 4, 0)   // reserved
	putU16(data, 6, 8)   // valueRecordSize
	putU16(data, 8, 1)   // valueRecordCount = 1
	putU16(data, 10, 0)  // itemVariationStoreOffset = 0
	putTag(data, 12, "hasc")
	putU16(data, 16, 0)
	putU16(data, 18, 0)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for records with zero IVS offset")
	}
}

func TestMVARFromRealFont(t *testing.T) {
	font, err := opentype.ParseFile("C:/Windows/Fonts/SegUIVar.ttf")
	if err != nil {
		t.Skipf("cannot open SegUIVar: %v", err)
	}
	mvarData, err := font.TableData("MVAR")
	if err != nil {
		t.Skipf("no MVAR: %v", err)
	}
	if len(mvarData) == 0 {
		t.Skip("MVAR data empty")
	}
	mv, err := Decode(mvarData)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	t.Logf("MVAR v%d.%d", mv.MajorVersion, mv.MinorVersion)
	t.Logf("ValueRecords: %d", len(mv.ValueRecords))
	t.Logf("VariationRegions: %d", len(mv.ItemVariationStore.VariationRegions))
	t.Logf("ItemVariationDatas: %d", len(mv.ItemVariationStore.ItemVariationDatas))
	// Check known tags
	for _, tag := range []string{"hasc", "hdsc", "hlgp"} {
		if mv.HasTag(tag) {
			t.Logf("Has %s: yes", tag)
		}
	}
}

func FuzzMVARDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 12))
	f.Fuzz(func(t *testing.T, data []byte) {
		mv, err := Decode(data)
		if err != nil {
			return
		}
		_ = mv
	})
}
