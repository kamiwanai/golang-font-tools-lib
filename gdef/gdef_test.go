// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gdef

import (
	"encoding/binary"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

func putU16(buf []byte, offset int, v uint16) {
	binary.BigEndian.PutUint16(buf[offset:offset+2], v)
}

func putInt16(buf []byte, offset int, v int16) {
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(v))
}

// --- GlyphClassDef tests ---

func TestDecodeClassDefFormat1(t *testing.T) {
	data := make([]byte, 30)
	putU16(data, 0, 1)  // majorVersion
	putU16(data, 2, 0)  // minorVersion
	putU16(data, 4, 12) // GlyphClassDef offset
	putU16(data, 6, 0)  // AttachList offset (none)
	putU16(data, 8, 0)  // LigCaretList offset (none)
	putU16(data, 10, 0) // MarkAttachClassDef offset (none)

	// ClassDef format 1 at offset 12
	putU16(data, 12, 1) // format
	putU16(data, 14, 1) // startGlyphID
	putU16(data, 16, 3) // glyphCount
	putU16(data, 18, 1) // class[0] = Base (glyph 1)
	putU16(data, 20, 2) // class[1] = Ligature (glyph 2)
	putU16(data, 22, 3) // class[2] = Mark (glyph 3)

	glyphOrder := []string{".notdef", "A", "B", "C"}
	gdef, err := Decode(data, glyphOrder)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(gdef.GlyphClassMap) != 3 {
		t.Fatalf("expected 3 class entries, got %d", len(gdef.GlyphClassMap))
	}
	if gdef.GlyphClass(1) != ClassBase {
		t.Fatalf("glyph 1 class = %d, want Base(1)", gdef.GlyphClass(1))
	}
	if gdef.GlyphClass(2) != ClassLigature {
		t.Fatalf("glyph 2 class = %d, want Ligature(2)", gdef.GlyphClass(2))
	}
	if gdef.GlyphClass(3) != ClassMark {
		t.Fatalf("glyph 3 class = %d, want Mark(3)", gdef.GlyphClass(3))
	}
	if gdef.GlyphClass(0) != 0 {
		t.Fatalf("glyph 0 (not in def) class = %d, want 0", gdef.GlyphClass(0))
	}
	if !gdef.IsBase(1) {
		t.Fatal("IsBase(1) should be true")
	}
	if !gdef.IsLigature(2) {
		t.Fatal("IsLigature(2) should be true")
	}
	if !gdef.IsMark(3) {
		t.Fatal("IsMark(3) should be true")
	}

	if ClassName(ClassBase) != "Base" {
		t.Fatalf("ClassName(Base) = %q, want %q", ClassName(ClassBase), "Base")
	}
	if ClassName(0) != "NoClass" {
		t.Fatalf("ClassName(0) = %q, want %q", ClassName(0), "NoClass")
	}

	byName := gdef.ClassDefByNames(glyphOrder)
	if byName["A"] != ClassBase {
		t.Fatalf("class by name A = %d, want Base(1)", byName["A"])
	}
}

func TestDecodeClassDefFormat2(t *testing.T) {
	data := make([]byte, 30)
	putU16(data, 0, 1)  // majorVersion
	putU16(data, 2, 0)  // minorVersion
	putU16(data, 4, 12) // GlyphClassDef offset
	putU16(data, 6, 0)  // AttachList offset (none)
	putU16(data, 8, 0)  // LigCaretList offset (none)
	putU16(data, 10, 0) // MarkAttachClassDef offset (none)

	putU16(data, 12, 2) // format
	putU16(data, 14, 1) // rangeCount
	putU16(data, 16, 1) // startGlyphID
	putU16(data, 18, 3) // endGlyphID
	putU16(data, 20, 1) // class = Base

	gdef, err := Decode(data, nil)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(gdef.GlyphClassMap) != 3 {
		t.Fatalf("expected 3 class entries, got %d", len(gdef.GlyphClassMap))
	}
	if gdef.GlyphClass(1) != ClassBase {
		t.Fatalf("glyph 1 class = %d, want Base(1)", gdef.GlyphClass(1))
	}
	if gdef.GlyphClass(2) != ClassBase {
		t.Fatalf("glyph 2 class = %d, want Base(1)", gdef.GlyphClass(2))
	}
	if gdef.GlyphClass(3) != ClassBase {
		t.Fatalf("glyph 3 class = %d, want Base(1)", gdef.GlyphClass(3))
	}
}

func TestDecodeClassDefSkipsClass0(t *testing.T) {
	data := make([]byte, 30)
	putU16(data, 0, 1)  // majorVersion
	putU16(data, 2, 0)  // minorVersion
	putU16(data, 4, 12) // GlyphClassDef offset
	putU16(data, 6, 0)
	putU16(data, 8, 0)
	putU16(data, 10, 0)

	putU16(data, 12, 1) // format
	putU16(data, 14, 1) // startGlyphID
	putU16(data, 16, 4) // glyphCount
	putU16(data, 18, 0) // class glyph 1 = 0
	putU16(data, 20, 1) // class glyph 2 = Base
	putU16(data, 22, 0) // class glyph 3 = 0
	putU16(data, 24, 3) // class glyph 4 = Mark

	gdef, err := Decode(data, nil)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(gdef.GlyphClassMap) != 2 {
		t.Fatalf("expected 2 class entries, got %d", len(gdef.GlyphClassMap))
	}
	if gdef.GlyphClass(2) != ClassBase {
		t.Fatalf("glyph 2 class = %d, want Base(1)", gdef.GlyphClass(2))
	}
	if gdef.GlyphClass(4) != ClassMark {
		t.Fatalf("glyph 4 class = %d, want Mark(3)", gdef.GlyphClass(4))
	}
}

// --- AttachList tests ---

func TestDecodeAttachList(t *testing.T) {
	// GDEF header: 0-11
	// AttachList at 12:
	//   +0: coverageOffset = 10 → coverage at 22
	//   +2: glyphCount = 1
	//   +4: attachPointOffset[0] = 16 → attachPoint at 28
	// Coverage at 22: format 1, count=1, glyph=1
	// AttachPoint at 28: pointCount=1, point[0]=5
	data := make([]byte, 40)
	putU16(data, 0, 1)  // majorVersion
	putU16(data, 2, 0)  // minorVersion
	putU16(data, 4, 0)  // no GlyphClassDef
	putU16(data, 6, 12) // AttachList offset
	putU16(data, 8, 0)  // no LigCaretList
	putU16(data, 10, 0) // no MarkAttachClassDef

	// AttachList at 12:
	putU16(data, 12, 10) // coverageOffset (from AttachList start → 12+10=22)
	putU16(data, 14, 1)  // glyphCount
	putU16(data, 16, 16) // attachPointOffset[0] (from AttachList start → 12+16=28)

	// Coverage at 22:
	putU16(data, 22, 1) // format 1
	putU16(data, 24, 1) // glyphCount = 1
	putU16(data, 26, 1) // glyph 1

	// AttachPoint at 28:
	putU16(data, 28, 1) // pointCount = 1
	putU16(data, 30, 5) // point[0] = 5

	gdef, err := Decode(data, nil)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(gdef.AttachList) != 1 {
		t.Fatalf("expected 1 AttachPoint, got %d", len(gdef.AttachList))
	}
	if gdef.AttachList[0].GlyphID != 1 {
		t.Fatalf("AttachPoint GlyphID = %d, want 1", gdef.AttachList[0].GlyphID)
	}
	if len(gdef.AttachList[0].PointIndices) != 1 || gdef.AttachList[0].PointIndices[0] != 5 {
		t.Fatalf("AttachPoint PointIndices = %v, want [5]", gdef.AttachList[0].PointIndices)
	}
}

// --- LigCaretList tests ---

func TestDecodeLigCaretList(t *testing.T) {
	// GDEF header: 0-11
	// LigCaretList at 12:
	//   +0: coverageOffset = 8   → coverage at 20
	//   +2: ligGlyphCount = 1
	//   +4: ligGlyphOffset[0] = 14 → ligGlyph at 26
	// Coverage at 20: format 1, count=1, glyph=3
	// LigGlyph at 26: caretCount=1, caretOffset[0]=4 → caretValue at 30
	// CaretValue at 30: format 1, coordinate=250
	data := make([]byte, 40)
	putU16(data, 0, 1)  // majorVersion
	putU16(data, 2, 0)  // minorVersion
	putU16(data, 4, 0)  // no GlyphClassDef
	putU16(data, 6, 0)  // no AttachList
	putU16(data, 8, 12) // LigCaretList offset
	putU16(data, 10, 0) // no MarkAttachClassDef

	// LigCaretList at 12:
	putU16(data, 12, 8)  // coverageOffset (12+8=20)
	putU16(data, 14, 1)  // ligGlyphCount
	putU16(data, 16, 14) // ligGlyphOffset[0] (12+14=26)

	// Coverage at 20:
	putU16(data, 20, 1) // format 1
	putU16(data, 22, 1) // glyphCount = 1
	putU16(data, 24, 3) // glyph 3 (ligature)

	// LigGlyph at 26:
	putU16(data, 26, 1) // caretCount = 1
	putU16(data, 28, 4) // caretValueOffset[0] (26+4=30)

	// CaretValue at 30 (format 1):
	putU16(data, 30, 1)    // format 1
	putInt16(data, 32, 250) // coordinate = 250

	gdef, err := Decode(data, nil)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(gdef.LigCaretList) != 1 {
		t.Fatalf("expected 1 LigGlyph, got %d", len(gdef.LigCaretList))
	}
	if gdef.LigCaretList[0].GlyphID != 3 {
		t.Fatalf("LigGlyph GlyphID = %d, want 3", gdef.LigCaretList[0].GlyphID)
	}
	if len(gdef.LigCaretList[0].CaretOffsets) != 1 {
		t.Fatalf("LigGlyph CaretOffsets length = %d, want 1", len(gdef.LigCaretList[0].CaretOffsets))
	}
	if gdef.LigCaretList[0].CaretOffsets[0] != 250 {
		t.Fatalf("LigGlyph caret = %d, want 250", gdef.LigCaretList[0].CaretOffsets[0])
	}
}

// --- MarkAttachClassDef tests ---

func TestDecodeMarkAttachClassDef(t *testing.T) {
	data := make([]byte, 30)
	putU16(data, 0, 1)   // majorVersion
	putU16(data, 2, 2)   // minorVersion (1.2)
	putU16(data, 4, 0)   // no GlyphClassDef
	putU16(data, 6, 0)   // no AttachList
	putU16(data, 8, 0)   // no LigCaretList
	putU16(data, 10, 12) // MarkAttachClassDef offset

	putU16(data, 12, 1) // format
	putU16(data, 14, 5) // startGlyphID
	putU16(data, 16, 2) // glyphCount
	putU16(data, 18, 1) // class[0] = 1
	putU16(data, 20, 2) // class[1] = 2

	gdef, err := Decode(data, nil)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if gdef.MarkAttachClassMap == nil {
		t.Fatal("expected MarkAttachClassMap to be non-nil")
	}
	if gdef.MarkAttachClass(5) != 1 {
		t.Fatalf("MarkAttachClass(5) = %d, want 1", gdef.MarkAttachClass(5))
	}
	if gdef.MarkAttachClass(6) != 2 {
		t.Fatalf("MarkAttachClass(6) = %d, want 2", gdef.MarkAttachClass(6))
	}
	if gdef.MarkAttachClass(0) != 0 {
		t.Fatalf("MarkAttachClass(0) = %d, want 0", gdef.MarkAttachClass(0))
	}
}

// --- No subtable tests ---

func TestDecodeEmptyGDEF(t *testing.T) {
	data := make([]byte, 12)
	putU16(data, 0, 1) // majorVersion
	putU16(data, 2, 0) // minorVersion

	gdef, err := Decode(data, nil)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(gdef.GlyphClassMap) != 0 {
		t.Fatalf("expected empty GlyphClassMap, got %d entries", len(gdef.GlyphClassMap))
	}
	if len(gdef.AttachList) != 0 {
		t.Fatalf("expected empty AttachList, got %d entries", len(gdef.AttachList))
	}
	if len(gdef.LigCaretList) != 0 {
		t.Fatalf("expected empty LigCaretList, got %d entries", len(gdef.LigCaretList))
	}
	if gdef.MarkAttachClassMap != nil {
		t.Fatal("expected nil MarkAttachClassMap")
	}
}

func TestDecodeTruncatedHeader(t *testing.T) {
	_, err := Decode(make([]byte, 5), nil)
	if err == nil {
		t.Fatal("expected error for truncated GDEF header")
	}
}

func TestDecodeClassDefFormat1Truncated(t *testing.T) {
	data := make([]byte, 12)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 12) // ClassDef offset = 12 (past data end)

	_, err := Decode(data, nil)
	if err == nil {
		t.Fatal("expected error for truncated ClassDef")
	}
}

func TestDecodeClassDefFormat2RangeEndBeforeStart(t *testing.T) {
	data := make([]byte, 30)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 12) // ClassDef offset

	putU16(data, 12, 2) // format
	putU16(data, 14, 1) // rangeCount
	putU16(data, 16, 5) // start=5
	putU16(data, 18, 3) // end=3 (end < start → error)
	putU16(data, 20, 1) // class

	_, err := Decode(data, nil)
	if err == nil {
		t.Fatal("expected error for range end before start")
	}
}

func TestIsMethods(t *testing.T) {
	gdef := GDEF{
		GlyphClassMap: map[uint16]uint16{
			1: ClassBase,
			2: ClassLigature,
			3: ClassMark,
			4: ClassComponent,
		},
	}
	if !gdef.IsBase(1) {
		t.Error("IsBase(1) should be true")
	}
	if !gdef.IsLigature(2) {
		t.Error("IsLigature(2) should be true")
	}
	if !gdef.IsMark(3) {
		t.Error("IsMark(3) should be true")
	}
	if !gdef.IsComponent(4) {
		t.Error("IsComponent(4) should be true")
	}
	if gdef.IsBase(99) {
		t.Error("IsBase(99) should be false")
	}
}

func TestGDEFFromGoRegular(t *testing.T) {
	font, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	gdefData, err := font.TableData("GDEF")
	if err != nil {
		t.Skipf("font has no GDEF table: %v", err)
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder error: %v", err)
	}
	gdef, err := Decode(gdefData, glyphOrder)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	t.Logf("GlyphClassMap: %d entries", len(gdef.GlyphClassMap))
	t.Logf("AttachList: %d entries", len(gdef.AttachList))
	t.Logf("LigCaretList: %d entries", len(gdef.LigCaretList))
	t.Logf("MarkAttachClassMap: %d entries", len(gdef.MarkAttachClassMap))

	if gdef.MajorVersion != 1 {
		t.Fatalf("GDEF major version = %d, want 1", gdef.MajorVersion)
	}
}

func FuzzGDEFDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 12))

	f.Fuzz(func(t *testing.T, data []byte) {
		gdef, err := Decode(data, nil)
		if err != nil {
			return
		}
		_ = gdef
	})
}

func FuzzGDEFClassDef(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		classMap, err := decodeClassDef(data, 0)
		if err != nil {
			return
		}
		_ = classMap
	})
}