// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gpos

import (
	"encoding/binary"
	"testing"
)

func buildMarkBasePosFixture() ([]byte, []string) {
	// Glyph order: .notdef=0, a=1, acutecomb=2
	glyphOrder := []string{".notdef", "a", "acutecomb"}

	// Build a minimal MarkBasePosFormat1 subtable.
	// Subtable starts at offset 0 of this buffer.
	//
	// Layout:
	//   +0:  Format=1
	//   +2:  MarkCoverageOffset
	//   +4:  BaseCoverageOffset
	//   +6:  ClassCount=1
	//   +8:  MarkArrayOffset
	//   +10: BaseArrayOffset
	//
	// MarkCoverage (format 1): 1 glyph, glyphID=2 (acutecomb)
	// BaseCoverage (format 1): 1 glyph, glyphID=1 (a)
	// MarkArray: 1 record, class=0, anchor(200, 400)
	// BaseArray: 1 base, class 0 anchor(200, 600)

	data := make([]byte, 100)

	// Subtable header
	binary.BigEndian.PutUint16(data[0:2], 1)   // format
	binary.BigEndian.PutUint16(data[2:4], 12)  // MarkCoverageOffset
	binary.BigEndian.PutUint16(data[4:6], 22)  // BaseCoverageOffset
	binary.BigEndian.PutUint16(data[6:8], 1)   // ClassCount
	binary.BigEndian.PutUint16(data[8:10], 32) // MarkArrayOffset
	binary.BigEndian.PutUint16(data[10:12], 44) // BaseArrayOffset

	// MarkCoverage at +12: format 1, 1 glyph
	binary.BigEndian.PutUint16(data[12:14], 1) // format 1
	binary.BigEndian.PutUint16(data[14:16], 1) // glyph count
	binary.BigEndian.PutUint16(data[16:18], 2) // glyphID 2 = acutecomb

	// BaseCoverage at +22: format 1, 1 glyph
	binary.BigEndian.PutUint16(data[22:24], 1) // format 1
	binary.BigEndian.PutUint16(data[24:26], 1) // glyph count
	binary.BigEndian.PutUint16(data[26:28], 1) // glyphID 1 = a

	// MarkArray at +32: 1 record
	binary.BigEndian.PutUint16(data[32:34], 1) // MarkCount
	binary.BigEndian.PutUint16(data[34:36], 0) // MarkClass=0
	binary.BigEndian.PutUint16(data[36:38], 12) // MarkAnchorOffset from MarkArray start

	// Mark Anchor at +32+12=44: format 1
	binary.BigEndian.PutUint16(data[44:46], 1)    // anchor format 1
	binary.BigEndian.PutUint16(data[46:48], 200)   // X
	binary.BigEndian.PutUint16(data[48:50], 400)   // Y

	// BaseArray at +44: 1 base, 1 class
	binary.BigEndian.PutUint16(data[44:46], 1) // BaseCount (overlaps with anchor — we need to fix layout)

	// Fix: let's recompute with non-overlapping offsets.
	// Subtable header: 0-11 (12 bytes)
	// MarkCoverage: 12-21 (10 bytes)
	// BaseCoverage: 22-31 (10 bytes)
	// MarkArray: 32-43 (2 + 1*4 = 6 bytes, fits in 12)
	// MarkAnchor: 44-49 (6 bytes)
	// BaseArray: 50+ (2 + 1*2 = 4 bytes)
	// BaseAnchor: 54+ (6 bytes)

	// Actually let me rebuild properly with exact offsets.
	data = make([]byte, 60)

	// Header: 12 bytes
	binary.BigEndian.PutUint16(data[0:2], 1)    // format
	binary.BigEndian.PutUint16(data[2:4], 12)   // MarkCoverageOffset (subtable-relative)
	binary.BigEndian.PutUint16(data[4:6], 22)   // BaseCoverageOffset
	binary.BigEndian.PutUint16(data[6:8], 1)    // ClassCount
	binary.BigEndian.PutUint16(data[8:10], 32)  // MarkArrayOffset
	binary.BigEndian.PutUint16(data[10:12], 50) // BaseArrayOffset

	// MarkCoverage at 12: format1, 1 glyph, glyphID=2
	binary.BigEndian.PutUint16(data[12:14], 1)
	binary.BigEndian.PutUint16(data[14:16], 1)
	binary.BigEndian.PutUint16(data[16:18], 2) // acutecomb

	// BaseCoverage at 22: format1, 1 glyph, glyphID=1
	binary.BigEndian.PutUint16(data[22:24], 1)
	binary.BigEndian.PutUint16(data[24:26], 1)
	binary.BigEndian.PutUint16(data[26:28], 1) // a

	// MarkArray at 32: MarkCount=1
	binary.BigEndian.PutUint16(data[32:34], 1)   // MarkCount
	binary.BigEndian.PutUint16(data[34:36], 0)   // MarkClass=0
	binary.BigEndian.PutUint16(data[36:38], 8)   // MarkAnchorOffset (from MarkArray start at 32, so anchor at 32+8=40)

	// Mark Anchor at 40: format 1
	binary.BigEndian.PutUint16(data[40:42], 1)   // format 1
	binary.BigEndian.PutUint16(data[42:44], 200)  // X
	binary.BigEndian.PutUint16(data[44:46], 400)  // Y

	// BaseArray at 50: BaseCount=1
	binary.BigEndian.PutUint16(data[50:52], 1)   // BaseCount
	binary.BigEndian.PutUint16(data[52:54], 6)   // BaseAnchor[0] offset (from BaseArray start at 50, so anchor at 50+6=56)

	// Base Anchor at 56: format 1
	binary.BigEndian.PutUint16(data[56:58], 1)   // format 1
	binary.BigEndian.PutUint16(data[58:60], 200)  // X
	// Y would be at 60, need 2 more bytes
	data = append(data, 0x02, 0x58) // 600 = 0x0258

	return data, glyphOrder
}

func TestDecodeMarkBasePos(t *testing.T) {
	data, glyphOrder := buildMarkBasePosFixture()

	attachments, err := decodeMarkBasePos(data, glyphOrder, 0, 3)
	if err != nil {
		t.Fatalf("decodeMarkBasePos returned error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("attachment count = %d, want 1", len(attachments))
	}
	got := attachments[0]
	if got.MarkGlyph != "acutecomb" {
		t.Fatalf("mark glyph = %q, want %q", got.MarkGlyph, "acutecomb")
	}
	if got.BaseGlyph != "a" {
		t.Fatalf("base glyph = %q, want %q", got.BaseGlyph, "a")
	}
	if got.MarkClass != 0 {
		t.Fatalf("mark class = %d, want 0", got.MarkClass)
	}
	if got.MarkAnchor.X != 200 || got.MarkAnchor.Y != 400 {
		t.Fatalf("mark anchor = (%d,%d), want (200,400)", got.MarkAnchor.X, got.MarkAnchor.Y)
	}
	if got.BaseAnchor.X != 200 || got.BaseAnchor.Y != 600 {
		t.Fatalf("base anchor = (%d,%d), want (200,600)", got.BaseAnchor.X, got.BaseAnchor.Y)
	}
	if got.LookupType != 4 {
		t.Fatalf("lookup type = %d, want 4", got.LookupType)
	}
	if got.LookupIndex != 3 {
		t.Fatalf("lookup index = %d, want 3", got.LookupIndex)
	}
}

func TestDecodeMarkBasePosSkipsNullAnchor(t *testing.T) {
	// Build a fixture where the base anchor offset is 0 (NULL).
	glyphOrder := []string{".notdef", "a", "acutecomb"}
	data := make([]byte, 58)

	binary.BigEndian.PutUint16(data[0:2], 1)    // format
	binary.BigEndian.PutUint16(data[2:4], 12)   // MarkCoverageOffset
	binary.BigEndian.PutUint16(data[4:6], 22)   // BaseCoverageOffset
	binary.BigEndian.PutUint16(data[6:8], 1)    // ClassCount
	binary.BigEndian.PutUint16(data[8:10], 32)  // MarkArrayOffset
	binary.BigEndian.PutUint16(data[10:12], 50) // BaseArrayOffset

	// MarkCoverage
	binary.BigEndian.PutUint16(data[12:14], 1)
	binary.BigEndian.PutUint16(data[14:16], 1)
	binary.BigEndian.PutUint16(data[16:18], 2)
	// BaseCoverage
	binary.BigEndian.PutUint16(data[22:24], 1)
	binary.BigEndian.PutUint16(data[24:26], 1)
	binary.BigEndian.PutUint16(data[26:28], 1)
	// MarkArray
	binary.BigEndian.PutUint16(data[32:34], 1)
	binary.BigEndian.PutUint16(data[34:36], 0)
	binary.BigEndian.PutUint16(data[36:38], 8)
	// Mark Anchor at 40
	binary.BigEndian.PutUint16(data[40:42], 1)
	binary.BigEndian.PutUint16(data[42:44], 200)
	binary.BigEndian.PutUint16(data[44:46], 400)
	// BaseArray at 50: BaseCount=1, anchor offset=0 (NULL)
	binary.BigEndian.PutUint16(data[50:52], 1)
	binary.BigEndian.PutUint16(data[52:54], 0) // NULL

	attachments, err := decodeMarkBasePos(data, glyphOrder, 0, 0)
	if err != nil {
		t.Fatalf("decodeMarkBasePos returned error: %v", err)
	}
	if len(attachments) != 0 {
		t.Fatalf("attachment count = %d, want 0 (null anchor)", len(attachments))
	}
}

func TestDecodeMarkBasePosRejectsMismatchedCoverage(t *testing.T) {
	glyphOrder := []string{".notdef", "a", "acutecomb", "b"}
	data := make([]byte, 58)

	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 12)
	binary.BigEndian.PutUint16(data[4:6], 22)
	binary.BigEndian.PutUint16(data[6:8], 1)
	binary.BigEndian.PutUint16(data[8:10], 32)
	binary.BigEndian.PutUint16(data[10:12], 50)

	// MarkCoverage: 2 glyphs
	binary.BigEndian.PutUint16(data[12:14], 1)
	binary.BigEndian.PutUint16(data[14:16], 2) // 2 marks
	binary.BigEndian.PutUint16(data[16:18], 2)
	binary.BigEndian.PutUint16(data[18:20], 3)
	// BaseCoverage
	binary.BigEndian.PutUint16(data[22:24], 1)
	binary.BigEndian.PutUint16(data[24:26], 1)
	binary.BigEndian.PutUint16(data[26:28], 1)
	// MarkArray: 1 record (mismatch: 2 coverage, 1 mark record)
	binary.BigEndian.PutUint16(data[32:34], 1)
	binary.BigEndian.PutUint16(data[34:36], 0)
	binary.BigEndian.PutUint16(data[36:38], 8)
	binary.BigEndian.PutUint16(data[40:42], 1)
	binary.BigEndian.PutUint16(data[42:44], 200)
	binary.BigEndian.PutUint16(data[44:46], 400)
	binary.BigEndian.PutUint16(data[50:52], 1)
	binary.BigEndian.PutUint16(data[52:54], 4)
	binary.BigEndian.PutUint16(data[54:56], 1)
	binary.BigEndian.PutUint16(data[56:58], 200)

	_, err := decodeMarkBasePos(data, glyphOrder, 0, 0)
	if err == nil {
		t.Fatal("expected error for mismatched coverage, got nil")
	}
}

func TestDecodeMarkMarkPos(t *testing.T) {
	// .notdef=0, acutecomb=1, gravecomb=2
	glyphOrder := []string{".notdef", "acutecomb", "gravecomb"}

	data := make([]byte, 66)

	// Header
	binary.BigEndian.PutUint16(data[0:2], 1)    // format
	binary.BigEndian.PutUint16(data[2:4], 12)   // Mark1CoverageOffset
	binary.BigEndian.PutUint16(data[4:6], 20)   // Mark2CoverageOffset
	binary.BigEndian.PutUint16(data[6:8], 1)    // ClassCount
	binary.BigEndian.PutUint16(data[8:10], 28)  // Mark1ArrayOffset
	binary.BigEndian.PutUint16(data[10:12], 44) // Mark2ArrayOffset

	// Mark1Coverage at 12: 1 glyph, glyphID=1 (acutecomb)
	binary.BigEndian.PutUint16(data[12:14], 1)
	binary.BigEndian.PutUint16(data[14:16], 1)
	binary.BigEndian.PutUint16(data[16:18], 1)

	// Mark2Coverage at 20: 1 glyph, glyphID=2 (gravecomb)
	binary.BigEndian.PutUint16(data[20:22], 1)
	binary.BigEndian.PutUint16(data[22:24], 1)
	binary.BigEndian.PutUint16(data[24:26], 2)

	// Mark1Array at 28: 1 record
	binary.BigEndian.PutUint16(data[28:30], 1)   // MarkCount
	binary.BigEndian.PutUint16(data[30:32], 0)   // MarkClass=0
	binary.BigEndian.PutUint16(data[32:34], 8)   // MarkAnchorOffset

	// Mark1 Anchor at 28+8=36
	binary.BigEndian.PutUint16(data[36:38], 1)   // format 1
	binary.BigEndian.PutUint16(data[38:40], 100)  // X
	binary.BigEndian.PutUint16(data[40:42], 300)  // Y

	// Mark2Array at 44: 1 mark2, 1 class
	binary.BigEndian.PutUint16(data[44:46], 1)   // Mark2Count
	binary.BigEndian.PutUint16(data[46:48], 6)   // Mark2Anchor[0] offset

	// Mark2 Anchor at 44+6=50
	binary.BigEndian.PutUint16(data[50:52], 1)   // format 1
	binary.BigEndian.PutUint16(data[52:54], 150)  // X
	binary.BigEndian.PutUint16(data[54:56], 450)  // Y

	attachments, err := decodeMarkMarkPos(data, glyphOrder, 0, 7)
	if err != nil {
		t.Fatalf("decodeMarkMarkPos returned error: %v", err)
	}
	if len(attachments) != 1 {
		t.Fatalf("attachment count = %d, want 1", len(attachments))
	}
	got := attachments[0]
	if got.MarkGlyph != "acutecomb" {
		t.Fatalf("mark1 glyph = %q, want %q", got.MarkGlyph, "acutecomb")
	}
	if got.BaseGlyph != "gravecomb" {
		t.Fatalf("mark2 glyph = %q, want %q", got.BaseGlyph, "gravecomb")
	}
	if got.MarkAnchor.X != 100 || got.MarkAnchor.Y != 300 {
		t.Fatalf("mark1 anchor = (%d,%d), want (100,300)", got.MarkAnchor.X, got.MarkAnchor.Y)
	}
	if got.BaseAnchor.X != 150 || got.BaseAnchor.Y != 450 {
		t.Fatalf("mark2 anchor = (%d,%d), want (150,450)", got.BaseAnchor.X, got.BaseAnchor.Y)
	}
	if got.LookupType != 6 {
		t.Fatalf("lookup type = %d, want 6", got.LookupType)
	}
}

func TestDecodeMarkLookupsSkipsNonMarkLookups(t *testing.T) {
	// GPOS with one PairPos lookup (type 2) — should be skipped.
	gpos := make([]byte, 20)
	binary.BigEndian.PutUint16(gpos[0:2], 1)    // major
	binary.BigEndian.PutUint16(gpos[2:4], 0)    // minor
	binary.BigEndian.PutUint16(gpos[8:10], 10)  // LookupList offset
	binary.BigEndian.PutUint16(gpos[10:12], 1)  // LookupCount
	binary.BigEndian.PutUint16(gpos[12:14], 4)  // Lookup offset
	binary.BigEndian.PutUint16(gpos[14:16], 2)  // LookupType PairPos
	binary.BigEndian.PutUint16(gpos[18:20], 1)  // SubTableCount
	// Subtable offset would point somewhere but type 2 is skipped

	attachments, err := DecodeMarkLookups(gpos, []string{".notdef", "A"})
	if err != nil {
		t.Fatalf("DecodeMarkLookups returned error: %v", err)
	}
	if len(attachments) != 0 {
		t.Fatalf("attachment count = %d, want 0", len(attachments))
	}
}

func TestDecodeAnchorFormat1(t *testing.T) {
	data := make([]byte, 6)
	binary.BigEndian.PutUint16(data[0:2], 1)   // format 1
	binary.BigEndian.PutUint16(data[2:4], 100)  // X
	binary.BigEndian.PutUint16(data[4:6], 0xFF9C) // Y = -100

	anchor, err := decodeAnchor(data, 0)
	if err != nil {
		t.Fatalf("decodeAnchor returned error: %v", err)
	}
	if anchor.X != 100 || anchor.Y != -100 {
		t.Fatalf("anchor = (%d,%d), want (100,-100)", anchor.X, anchor.Y)
	}
}

func TestDecodeAnchorRejectsUnsupportedFormat(t *testing.T) {
	data := make([]byte, 6)
	binary.BigEndian.PutUint16(data[0:2], 99) // unsupported format

	_, err := decodeAnchor(data, 0)
	if err == nil {
		t.Fatal("expected error for unsupported anchor format, got nil")
	}
}

func TestDecodeMarkArray(t *testing.T) {
	data := make([]byte, 20)
	binary.BigEndian.PutUint16(data[0:2], 2)    // MarkCount
	binary.BigEndian.PutUint16(data[2:4], 0)    // MarkClass=0
	binary.BigEndian.PutUint16(data[4:6], 10)   // MarkAnchorOffset
	binary.BigEndian.PutUint16(data[6:8], 1)    // MarkClass=1
	binary.BigEndian.PutUint16(data[8:10], 0)   // NULL anchor

	// Anchor at 10
	binary.BigEndian.PutUint16(data[10:12], 1)  // format 1
	binary.BigEndian.PutUint16(data[12:14], 50) // X
	binary.BigEndian.PutUint16(data[14:16], 75) // Y

	records, err := decodeMarkArray(data, 0)
	if err != nil {
		t.Fatalf("decodeMarkArray returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("record count = %d, want 2", len(records))
	}
	if records[0].markClass != 0 || records[0].markAnchor.X != 50 || records[0].markAnchor.Y != 75 {
		t.Fatalf("record[0] = %+v", records[0])
	}
	if records[1].markClass != 1 || records[1].markAnchor.X != 0 || records[1].markAnchor.Y != 0 {
		t.Fatalf("record[1] = %+v (expected null anchor)", records[1])
	}
}
