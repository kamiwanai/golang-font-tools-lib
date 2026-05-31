// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package stat

import (
	"encoding/binary"
	"testing"
)

func TestDecodeV1_0(t *testing.T) {
	// STAT v1.0: header only, no axis values
	// format: major(2) + minor(2) + axisSize(2) + axisCount(2) + axesOffset(4) = 12 bytes
	data := make([]byte, 12+8) // 12 header + 8 for one axis
	binary.BigEndian.PutUint16(data[0:2], 1)  // major=1
	binary.BigEndian.PutUint16(data[2:4], 0)  // minor=0
	binary.BigEndian.PutUint16(data[4:6], 8)  // axisSize=8
	binary.BigEndian.PutUint16(data[6:8], 1)  // axisCount=1
	binary.BigEndian.PutUint32(data[8:12], 12) // axesOffset=12
	// DesignAxis: tag=wght, nameID=256, ordering=0
	copy(data[12:16], []byte("wght"))
	binary.BigEndian.PutUint16(data[16:18], 256)
	binary.BigEndian.PutUint16(data[18:20], 0)

	s, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if s.MajorVersion != 1 {
		t.Errorf("MajorVersion = %d, want 1", s.MajorVersion)
	}
	if len(s.DesignAxes) != 1 {
		t.Fatalf("expected 1 axis, got %d", len(s.DesignAxes))
	}
	if s.DesignAxes[0].Tag != "wght" {
		t.Errorf("axis tag = %q, want wght", s.DesignAxes[0].Tag)
	}
	if s.DesignAxes[0].AxisNameID != 256 {
		t.Errorf("nameID = %d, want 256", s.DesignAxes[0].AxisNameID)
	}
	if len(s.AxisValues) != 0 {
		t.Errorf("expected 0 axis values, got %d", len(s.AxisValues))
	}
}

func TestDecodeV1_1(t *testing.T) {
	// STAT v1.1: header + axis values
	data := make([]byte, 18+8+2+8) // 18 header + 8 axis + 4 axisValue
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 1) // minor=1
	binary.BigEndian.PutUint16(data[4:6], 8)
	binary.BigEndian.PutUint16(data[6:8], 1)
	binary.BigEndian.PutUint32(data[8:12], 18) // axesOffset
	// v1.1 fields:
	binary.BigEndian.PutUint16(data[12:14], 1) // axisValueCount=1
	binary.BigEndian.PutUint32(data[14:18], 26) // axisValueOffsetsOffset
	// DesignAxis at 18:
	copy(data[18:22], []byte("wght"))
	binary.BigEndian.PutUint16(data[22:24], 256)
	binary.BigEndian.PutUint16(data[24:26], 0)
	// AxisValue offset at 26: points to 28 (=26+2)
	binary.BigEndian.PutUint16(data[26:28], 2)
	// AxisValue Format 1 at 28:
	binary.BigEndian.PutUint16(data[28:30], 1) // format=1
	binary.BigEndian.PutUint16(data[30:32], 0) // flags=0
	binary.BigEndian.PutUint16(data[32:34], 257) // valueNameID
	binary.BigEndian.PutUint16(data[34:36], 0x4000) // F2Dot14 value=1.0

	s, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(s.AxisValues) != 1 {
		t.Fatalf("expected 1 axis value, got %d", len(s.AxisValues))
	}
	av := s.AxisValues[0]
	if av.Format() != AxisValueFormat1 {
		t.Errorf("format = %d, want 1", av.Format())
	}
	av1 := av.(AxisValue1)
	if av1.Value_ != 1.0 {
		t.Errorf("value = %v, want 1.0", av1.Value_)
	}
	if av1.ValueNameID_ != 257 {
		t.Errorf("nameID = %d, want 257", av1.ValueNameID_)
	}
}

func TestDecodeTruncated(t *testing.T) {
	_, err := Decode([]byte{0x00, 0x01})
	if err == nil {
		t.Error("expected error for truncated data")
	}
}
