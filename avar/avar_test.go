// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package avar

import (
	"encoding/binary"
	"testing"
)

func TestDecodeBasic(t *testing.T) {
	// Build avar table: 1 axis, 3 segments
	// Axis 0: from -1.0→-0.5, 0.0→0.0, 1.0→0.5
	from1 := uint16(0xC000) // -1.0 in F2Dot14
	to1 := uint16(0xE000)   // -0.5 in F2Dot14
	from2 := uint16(0)
	to2 := uint16(0)
	from3 := uint16(int16(1.0 * 16384))  // 16384
	to3 := uint16(int16(0.5 * 16384))    // 8192

	data := make([]byte, 8+2+3*4) // header + count + 3*(from+to)
	binary.BigEndian.PutUint16(data[0:2], 1) // majorVersion
	binary.BigEndian.PutUint16(data[2:4], 0) // minorVersion
	binary.BigEndian.PutUint16(data[4:6], 0) // reserved
	binary.BigEndian.PutUint16(data[6:8], 1) // axisCount=1

	// Axis 0:
	binary.BigEndian.PutUint16(data[8:10], 3) // segmentCount
	binary.BigEndian.PutUint16(data[10:12], from1)
	binary.BigEndian.PutUint16(data[12:14], to1)
	binary.BigEndian.PutUint16(data[14:16], from2)
	binary.BigEndian.PutUint16(data[16:18], to2)
	binary.BigEndian.PutUint16(data[18:20], from3)
	binary.BigEndian.PutUint16(data[20:22], to3)

	av, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if av.MajorVersion != 1 {
		t.Errorf("MajorVersion = %d, want 1", av.MajorVersion)
	}
	if len(av.AxisMaps) != 1 {
		t.Fatalf("expected 1 axis map, got %d", len(av.AxisMaps))
	}
	seg := av.AxisMaps[0].Segments
	if len(seg) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(seg))
	}
	if seg[0].FromCoordinate != -1.0 {
		t.Errorf("seg[0].From = %v, want -1.0", seg[0].FromCoordinate)
	}
	if seg[0].ToCoordinate != -0.5 {
		t.Errorf("seg[0].To = %v, want -0.5", seg[0].ToCoordinate)
	}
	if seg[1].FromCoordinate != 0.0 {
		t.Errorf("seg[1].From = %v, want 0.0", seg[1].FromCoordinate)
	}
	if seg[2].ToCoordinate != 0.5 {
		t.Errorf("seg[2].To = %v, want 0.5", seg[2].ToCoordinate)
	}

	t.Logf("avar: %d axes, %d segments", len(av.AxisMaps), len(seg))
}

func TestMapCoord(t *testing.T) {
	// Same map: -1→-0.5, 0→0, 1→0.5
	from1 := uint16(0xC000) // -1.0 in F2Dot14
	to1 := uint16(0xE000)   // -0.5 in F2Dot14
	from2 := uint16(0)
	to2 := uint16(0)
	from3 := uint16(int16(1.0 * 16384))
	to3 := uint16(int16(0.5 * 16384))

	data := make([]byte, 8+2+3*4)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 0)
	binary.BigEndian.PutUint16(data[4:6], 0)
	binary.BigEndian.PutUint16(data[6:8], 1)
	binary.BigEndian.PutUint16(data[8:10], 3)
	binary.BigEndian.PutUint16(data[10:12], from1)
	binary.BigEndian.PutUint16(data[12:14], to1)
	binary.BigEndian.PutUint16(data[14:16], from2)
	binary.BigEndian.PutUint16(data[16:18], to2)
	binary.BigEndian.PutUint16(data[18:20], from3)
	binary.BigEndian.PutUint16(data[20:22], to3)

	av, _ := Decode(data)

	tests := []struct {
		input, want float64
	}{
		{-2.0, -0.5},  // clamp left
		{-1.0, -0.5},  // first segment
		{-0.5, -0.25}, // midpoint -1..0 → -0.5..0
		{0.0, 0.0},    // middle
		{0.5, 0.25},   // midpoint 0..1 → 0..0.5
		{1.0, 0.5},    // last segment
		{2.0, 0.5},    // clamp right
	}

	for _, tc := range tests {
		got := av.MapCoord(0, tc.input)
		if got != tc.want {
			t.Errorf("MapCoord(0, %v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDecodeEmpty(t *testing.T) {
	data := make([]byte, 8)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 0)
	binary.BigEndian.PutUint16(data[4:6], 0)
	binary.BigEndian.PutUint16(data[6:8], 0) // 0 axes

	av, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(av.AxisMaps) != 0 {
		t.Errorf("expected 0 axis maps, got %d", len(av.AxisMaps))
	}
}

func TestDecodeTruncated(t *testing.T) {
	_, err := Decode([]byte{0x00, 0x01}) // too short
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

func TestDecodeTruncatedAxis(t *testing.T) {
	// header says 1 axis but data truncated
	data := make([]byte, 8)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 0)
	binary.BigEndian.PutUint16(data[4:6], 0)
	binary.BigEndian.PutUint16(data[6:8], 1) // 1 axis but no axis data

	_, err := Decode(data)
	if err == nil {
		t.Error("expected error for truncated axis data")
	}
}
