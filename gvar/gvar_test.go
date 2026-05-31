// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gvar

import (
	"encoding/binary"
	"testing"

	"github.com/kamiwanai/fonttools/opentype"
)

func putU16(buf []byte, offset int, v uint16) {
	binary.BigEndian.PutUint16(buf[offset:offset+2], v)
}

func putU32(buf []byte, offset int, v uint32) {
	binary.BigEndian.PutUint32(buf[offset:offset+4], v)
}

func putI16(buf []byte, offset int, v int16) {
	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(v))
}

func buildGvarHeader(axisCount, sharedTupleCount, glyphCount int, sharedTuplesData []byte, glyphVarData [][]byte, useLongOffsets bool) []byte {
	offsetSize := 2
	if useLongOffsets {
		offsetSize = 4
	}
	flags := uint16(0)
	if useLongOffsets {
		flags = 1
	}

	offsetsLen := (glyphCount + 1) * offsetSize
	headerLen := 20 + offsetsLen
	sharedTuplesOff := uint32(headerLen)
	sharedTuplesLen := len(sharedTuplesData)
	gvarDataOff := headerLen + sharedTuplesLen

	totalLen := gvarDataOff
	for _, gd := range glyphVarData {
		if gd != nil {
			totalLen += len(gd)
		}
	}

	buf := make([]byte, totalLen)
	putU16(buf, 0, 1)
	putU16(buf, 2, 0)
	putU16(buf, 4, uint16(axisCount))
	putU16(buf, 6, uint16(sharedTupleCount))
	putU32(buf, 8, sharedTuplesOff)
	putU16(buf, 12, uint16(glyphCount))
	putU16(buf, 14, flags)
	putU32(buf, 16, uint32(gvarDataOff))

	offTablePos := 20
	currentOff := 0
	for i := 0; i <= glyphCount; i++ {
		val := currentOff
		if !useLongOffsets {
			val = currentOff / 2
			putU16(buf, offTablePos, uint16(val))
			offTablePos += 2
		} else {
			putU32(buf, offTablePos, uint32(val))
			offTablePos += 4
		}
		if i < glyphCount && i < len(glyphVarData) && glyphVarData[i] != nil {
			currentOff += len(glyphVarData[i])
		}
	}

	if sharedTuplesData != nil {
		copy(buf[sharedTuplesOff:], sharedTuplesData)
	}

	dataPos := gvarDataOff
	for _, gd := range glyphVarData {
		if gd != nil {
			copy(buf[dataPos:], gd)
			dataPos += len(gd)
		}
	}

	return buf
}

func buildSimpleGlyphVarData(axisCount int, peak []int16, deltasX, deltasY []int16) []byte {
	headerSize := 4 + axisCount*2
	pointsData := []byte{2, 0x81, 0, 0, 0, 1}
	xDeltaData := buildDeltaData(deltasX)
	yDeltaData := buildDeltaData(deltasY)
	serializedData := append(append(pointsData, xDeltaData...), yDeltaData...)
	varDataSize := len(serializedData)
	dataOffset := 4 + headerSize
	totalSize := dataOffset + varDataSize
	buf := make([]byte, totalSize)

	flags := uint16(embeddedPeakTuple | privatePointNumbers)
	putU16(buf, 0, 1)
	putU16(buf, 2, uint16(dataOffset))
	putU16(buf, 4, uint16(varDataSize))
	putU16(buf, 6, flags)
	for j := 0; j < axisCount; j++ {
		putI16(buf, 8+j*2, peak[j])
	}
	copy(buf[dataOffset:], serializedData)
	return buf
}

func buildDeltaData(deltas []int16) []byte {
	if len(deltas) == 0 {
		return nil
	}
	buf := make([]byte, 1+len(deltas)*2)
	buf[0] = byte(len(deltas)-1) | deltasAreWords
	for i, d := range deltas {
		putI16(buf, 1+i*2, d)
	}
	return buf
}

func TestDecodeHeader(t *testing.T) {
	gvarData := buildGvarHeader(1, 0, 3, nil, nil, false)
	gv, err := Decode(gvarData, 3)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if gv.MajorVersion != 1 || gv.MinorVersion != 0 {
		t.Fatalf("version = %d.%d, want 1.0", gv.MajorVersion, gv.MinorVersion)
	}
	if gv.AxisCount != 1 {
		t.Fatalf("axisCount = %d, want 1", gv.AxisCount)
	}
}

func TestDecodeSimpleTuple(t *testing.T) {
	peak := []int16{16384}
	dX := []int16{10, -5}
	dY := []int16{20, 0}
	glyphData := buildSimpleGlyphVarData(1, peak, dX, dY)
	gvarData := buildGvarHeader(1, 0, 2, nil, [][]byte{glyphData, nil}, false)

	gv, err := Decode(gvarData, 2)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	gvd := gv.Glyphs[0]
	if gvd == nil || len(gvd.Tuples) != 1 {
		t.Fatal("expected 1 tuple")
	}
	tv := gvd.Tuples[0]
	if tv.Peak[0].Float() != 1.0 {
		t.Fatalf("peak=%v want 1.0", tv.Peak[0].Float())
	}
	if len(tv.PointNumbers) != 2 {
		t.Fatalf("points=%v want [0,1]", tv.PointNumbers)
	}
	if len(tv.DeltasX) != 2 || tv.DeltasX[0] != 10 || tv.DeltasX[1] != -5 {
		t.Fatalf("dX=%v want [10,-5]", tv.DeltasX)
	}
	if len(tv.DeltasY) != 2 || tv.DeltasY[0] != 20 || tv.DeltasY[1] != 0 {
		t.Fatalf("dY=%v want [20,0]", tv.DeltasY)
	}
}

func TestLongOffsets(t *testing.T) {
	gvarData := buildGvarHeader(1, 0, 3, nil, nil, true)
	gv, err := Decode(gvarData, 3)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if gv.Flags != 1 {
		t.Fatalf("flags=%d want 1", gv.Flags)
	}
}

func TestSharedTupleRef(t *testing.T) {
	sharedData := make([]byte, 2)
	putI16(sharedData, 0, 16384)

	// 2 points: {0, 2} using byte encoding
	// count=2, control=0 (1-byte, runCount=1), values {0, 2}... but that gives {0, 2} absolute
	// Actually delta encoding: 0, then +2 = 2
	pointsData := []byte{2, 1, 0, 2}
	xData := buildDeltaData([]int16{7, 0})
	yData := buildDeltaData([]int16{3, 0})
	serData := append(append(pointsData, xData...), yData...)

	headerSize := 4
	dataOff := 4 + headerSize
	buf := make([]byte, dataOff+len(serData))
	putU16(buf, 0, 1)
	putU16(buf, 2, uint16(dataOff))
	putU16(buf, 4, uint16(len(serData)))
	putU16(buf, 6, privatePointNumbers)
	copy(buf[dataOff:], serData)

	gvarData := buildGvarHeader(1, 1, 1, sharedData, [][]byte{buf}, false)
	gv, err := Decode(gvarData, 1)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	tv := gv.Glyphs[0].Tuples[0]
	if tv.Peak[0].Float() != 1.0 {
		t.Fatalf("peak=%v want 1.0", tv.Peak[0].Float())
	}
}

func TestIntermediateRegion(t *testing.T) {
	headerSize := 4 + 1*2*3
	// 2 points: {0, 2}
	pointsData := []byte{2, 1, 0, 2}
	serData := append(append(pointsData, buildDeltaData([]int16{10, 0})...), buildDeltaData([]int16{20, 0})...)
	dataOff := 4 + headerSize
	buf := make([]byte, dataOff+len(serData))
	flags := uint16(embeddedPeakTuple | intermediateRegion | privatePointNumbers)
	putU16(buf, 0, 1)
	putU16(buf, 2, uint16(dataOff))
	putU16(buf, 4, uint16(len(serData)))
	putU16(buf, 6, flags)
	putI16(buf, 8, 16384)
	putI16(buf, 10, -8192)
	putI16(buf, 12, 24576)
	copy(buf[dataOff:], serData)

	gvarData := buildGvarHeader(1, 0, 1, nil, [][]byte{buf}, false)
	gv, err := Decode(gvarData, 1)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	tv := gv.Glyphs[0].Tuples[0]
	if tv.Start[0].Float() != -0.5 {
		t.Fatalf("start=%v", tv.Start[0].Float())
	}
	if tv.End[0].Float() != 1.5 {
		t.Fatalf("end=%v", tv.End[0].Float())
	}
}

func TestSharedPoints(t *testing.T) {
	// Build a GVD with shared points.
	// Serialized data: shared points (6 bytes) + tuple data (deltas only)
	sharedPts := []byte{2, 0x81, 0, 0, 0, 2}  // points {0, 2}
	xData := buildDeltaData([]int16{10, -5})    // 2 deltas for 2 points
	yData := buildDeltaData([]int16{0, 0})
	tupleData := append(xData, yData...)  // no private points, just deltas

	// The GVD serialized data has shared points first, then tuple data
	fullSerData := append(sharedPts, tupleData...)

	// One tuple, embedded peak, no private points (uses shared)
	headerSize := 4 + 1*2 // 4 + peak
	dataOff := 4 + headerSize

	// varDataSize should only be the tuple's own serialized data (after shared points)
	varDataSize := len(tupleData)

	buf := make([]byte, dataOff+len(fullSerData))
	tvc := uint16(1 | tuplesSharePointNumbers) // 1 tuple, shared points flag
	putU16(buf, 0, tvc)
	putU16(buf, 2, uint16(dataOff))
	putU16(buf, 4, uint16(varDataSize))
	putU16(buf, 6, embeddedPeakTuple) // no private, embedded peak
	putI16(buf, 8, 16384)
	copy(buf[dataOff:], fullSerData) // all serialized data

	gvarData := buildGvarHeader(1, 0, 1, nil, [][]byte{buf}, false)
	gv, err := Decode(gvarData, 1)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	gvd := gv.Glyphs[0]
	if gvd.SharedPointNumbers == nil {
		t.Fatal("expected shared points")
	}
	if len(gvd.SharedPointNumbers) != 2 || gvd.SharedPointNumbers[0] != 0 || gvd.SharedPointNumbers[1] != 2 {
		t.Fatalf("shared points = %v, want [0,2]", gvd.SharedPointNumbers)
	}
	tv := gvd.Tuples[0]
	if tv.PointNumbers == nil || len(tv.PointNumbers) != 2 {
		t.Fatalf("pointNumbers = %v, want [0,2]", tv.PointNumbers)
	}
	if len(tv.DeltasX) != 2 || tv.DeltasX[0] != 10 || tv.DeltasX[1] != -5 {
		t.Fatalf("dX = %v, want [10,-5]", tv.DeltasX)
	}
}




func TestRealVariableFont(t *testing.T) {
	// Test with SegUIVar.ttf — a Windows variable font.
	font, err := opentype.ParseFile("C:/Windows/Fonts/SegUIVar.ttf")
	if err != nil {
		t.Skipf("cannot open SegUIVar.ttf: %v", err)
	}
	gvarData, err := font.TableData("gvar")
	if err != nil {
		t.Skipf("no gvar table: %v", err)
	}
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		t.Fatal(err)
	}
	gv, err := Decode(gvarData, numGlyphs)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	t.Logf("GVAR: version=%d.%d axisCount=%d sharedTuples=%d glyphs=%d flags=0x%04X",
		gv.MajorVersion, gv.MinorVersion, gv.AxisCount, len(gv.SharedTuples), len(gv.Glyphs), gv.Flags)

	// Count glyphs with variation data.
	varCount := 0
	tupleCount := 0
	for gid, gvd := range gv.Glyphs {
		if gvd != nil {
			varCount++
			tupleCount += len(gvd.Tuples)
			if gid < 5 {
				t.Logf("  glyph %d: %d tuples, sharedPoints=%v",
					gid, len(gvd.Tuples), gvd.SharedPointNumbers != nil)
			}
		}
	}
	t.Logf("Glyphs with variations: %d/%d, total tuples: %d", varCount, numGlyphs, tupleCount)

	// Check shared tuples.
	if len(gv.SharedTuples) > 0 {
		t.Logf("First shared tuple: %v", gv.SharedTuples[0])
	}
}

func TestTruncatedHeader(t *testing.T) {
	_, err := Decode(make([]byte, 10), 10)
	if err == nil {
		t.Fatal("expected error")
	}
}
