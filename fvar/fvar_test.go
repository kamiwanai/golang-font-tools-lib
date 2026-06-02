// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package fvar

import (
	"encoding/binary"
	"testing"

	"github.com/kamiwanai/golang-font-tools-lib/opentype"
	"golang.org/x/image/font/gofont/goregular"
)

// putU16 writes a big-endian uint16 at offset.
func putU16(buf []byte, offset int, v uint16) {
	binary.BigEndian.PutUint16(buf[offset:offset+2], v)
}

// putFixed writes a Fixed 16.16 value as a big-endian int32 at offset.
func putFixed(buf []byte, offset int, v float64) {
	raw := int32(v * 65536.0)
	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(raw))
}

// --- Unit tests: decode axes ---

func TestDecodeBaseAxes(t *testing.T) {
	// Build a minimal fvar table with 1 axis (wght), 0 instances.
	// Header: 16 bytes
	// Axis: 20 bytes
	data := make([]byte, 36)

	putU16(data, 0, 1)  // majorVersion
	putU16(data, 2, 0)  // minorVersion
	putU16(data, 4, 16) // axesArrayOffset
	putU16(data, 6, 2)  // reserved
	putU16(data, 8, 1)  // axisCount
	putU16(data, 10, 20) // axisSize
	putU16(data, 12, 0)  // instanceCount
	putU16(data, 14, 8)  // instanceSize (min for 1 axis, even though instanceCount=0)

	// Axis 0: wght, 100..400..900
	copy(data[16:20], "wght")
	putFixed(data, 20, 100.0)
	putFixed(data, 24, 400.0)
	putFixed(data, 28, 900.0)
	putU16(data, 32, 0) // flags
	putU16(data, 34, 256) // axisNameID

	fvar, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(fvar.Axes) != 1 {
		t.Fatalf("expected 1 axis, got %d", len(fvar.Axes))
	}
	ax := fvar.Axes[0]
	if ax.Tag != "wght" {
		t.Fatalf("axis tag = %q, want %q", ax.Tag, "wght")
	}
	if ax.MinValue != 100.0 {
		t.Fatalf("axis minValue = %v, want 100", ax.MinValue)
	}
	if ax.DefaultValue != 400.0 {
		t.Fatalf("axis defaultValue = %v, want 400", ax.DefaultValue)
	}
	if ax.MaxValue != 900.0 {
		t.Fatalf("axis maxValue = %v, want 900", ax.MaxValue)
	}
	if ax.AxisNameID != 256 {
		t.Fatalf("axis nameID = %d, want 256", ax.AxisNameID)
	}
	if ax.IsHiddenAxis() {
		t.Fatal("wght axis should not be hidden")
	}
}

func TestDecodeMultipleAxes(t *testing.T) {
	// Build fvar with 2 axes (wght + wdth), 0 instances.
	// Header: 16 bytes + 2 axes * 20 = 56 bytes total
	data := make([]byte, 56)

	putU16(data, 0, 1)  // majorVersion
	putU16(data, 2, 0)  // minorVersion
	putU16(data, 4, 16) // axesArrayOffset
	putU16(data, 6, 2)  // reserved
	putU16(data, 8, 2)  // axisCount
	putU16(data, 10, 20) // axisSize
	putU16(data, 12, 0)  // instanceCount
	putU16(data, 14, 12) // instanceSize (min for 2 axes, even though instanceCount=0)

	// Axis 0: wght
	copy(data[16:20], "wght")
	putFixed(data, 20, 100.0)
	putFixed(data, 24, 400.0)
	putFixed(data, 28, 900.0)
	putU16(data, 32, 0)    // flags
	putU16(data, 34, 256)  // axisNameID

	// Axis 1: wdth
	copy(data[36:40], "wdth")
	putFixed(data, 40, 50.0)
	putFixed(data, 44, 100.0)
	putFixed(data, 48, 200.0)
	putU16(data, 52, 1)    // flags (hidden)
	putU16(data, 54, 257)  // axisNameID

	fvar, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(fvar.Axes) != 2 {
		t.Fatalf("expected 2 axes, got %d", len(fvar.Axes))
	}
	if fvar.Axes[0].Tag != "wght" {
		t.Fatalf("axis 0 tag = %q, want %q", fvar.Axes[0].Tag, "wght")
	}
	if fvar.Axes[1].Tag != "wdth" {
		t.Fatalf("axis 1 tag = %q, want %q", fvar.Axes[1].Tag, "wdth")
	}
	if !fvar.Axes[1].IsHiddenAxis() {
		t.Fatal("wdth axis should be hidden")
	}
}

func TestDecodeHiddenAxis(t *testing.T) {
	ax := Axis{Flags: 0x0001}
	if !ax.IsHiddenAxis() {
		t.Error("Flags 0x0001 should be hidden")
	}
	ax = Axis{Flags: 0x0000}
	if ax.IsHiddenAxis() {
		t.Error("Flags 0x0000 should not be hidden")
	}
}

// --- Unit tests: decode instances ---

func TestDecodeInstances(t *testing.T) {
	// Build fvar with 2 axes, 2 instances (without postScriptNameID).
	// instanceSize = 4 + 2*4 = 12
	// Header: 16 bytes, Axes: 40 bytes, Instances: 2*12 = 24 bytes = 80 total
	data := make([]byte, 80)

	putU16(data, 0, 1)   // majorVersion
	putU16(data, 2, 0)   // minorVersion
	putU16(data, 4, 16)  // axesArrayOffset
	putU16(data, 6, 2)   // reserved
	putU16(data, 8, 2)   // axisCount
	putU16(data, 10, 20) // axisSize
	putU16(data, 12, 2)  // instanceCount
	putU16(data, 14, 12) // instanceSize (4 + 2*4 = 12)

	// Axis 0: wght
	copy(data[16:20], "wght")
	putFixed(data, 20, 100.0)
	putFixed(data, 24, 400.0)
	putFixed(data, 28, 900.0)
	putU16(data, 32, 0)
	putU16(data, 34, 256)

	// Axis 1: wdth
	copy(data[36:40], "wdth")
	putFixed(data, 40, 75.0)
	putFixed(data, 44, 100.0)
	putFixed(data, 48, 125.0)
	putU16(data, 52, 0)
	putU16(data, 54, 257)

	// Instance 0: "Regular" — wght=400, wdth=100
	inst0Off := 56
	putU16(data, inst0Off, 258) // subfamilyNameID
	putU16(data, inst0Off+2, 0)  // flags
	putFixed(data, inst0Off+4, 400.0)
	putFixed(data, inst0Off+8, 100.0)

	// Instance 1: "Bold" — wght=700, wdth=100
	inst1Off := 68
	putU16(data, inst1Off, 259) // subfamilyNameID
	putU16(data, inst1Off+2, 0)  // flags
	putFixed(data, inst1Off+4, 700.0)
	putFixed(data, inst1Off+8, 100.0)

	fvar, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(fvar.Instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(fvar.Instances))
	}
	inst0 := fvar.Instances[0]
	if inst0.SubfamilyNameID != 258 {
		t.Fatalf("instance 0 subfamilyNameID = %d, want 258", inst0.SubfamilyNameID)
	}
	if inst0.Coordinates[0] != 400.0 {
		t.Fatalf("instance 0 wght = %v, want 400", inst0.Coordinates[0])
	}
	if inst0.Coordinates[1] != 100.0 {
		t.Fatalf("instance 0 wdth = %v, want 100", inst0.Coordinates[1])
	}
	inst1 := fvar.Instances[1]
	if inst1.SubfamilyNameID != 259 {
		t.Fatalf("instance 1 subfamilyNameID = %d, want 259", inst1.SubfamilyNameID)
	}
	if inst1.Coordinates[0] != 700.0 {
		t.Fatalf("instance 1 wght = %v, want 700", inst1.Coordinates[0])
	}
	// Without postScriptNameID, should be 0
	if inst0.PostScriptNameID != 0 {
		t.Fatalf("instance 0 postScriptNameID = %d, want 0 (absent)", inst0.PostScriptNameID)
	}
}

func TestDecodeInstancesWithPostScriptNameID(t *testing.T) {
	// Build fvar with 1 axis, 1 instance with postScriptNameID.
	// instanceSize = 4 + 1*4 + 2 = 10
	// Header: 16 bytes, Axis: 20 bytes, Instance: 10 bytes = 46 total
	data := make([]byte, 48)

	putU16(data, 0, 1)   // majorVersion
	putU16(data, 2, 0)   // minorVersion
	putU16(data, 4, 16)  // axesArrayOffset
	putU16(data, 6, 2)   // reserved
	putU16(data, 8, 1)   // axisCount
	putU16(data, 10, 20) // axisSize
	putU16(data, 12, 1)  // instanceCount
	putU16(data, 14, 10) // instanceSize (4 + 1*4 + 2 = 10)

	// Axis 0: wght
	copy(data[16:20], "wght")
	putFixed(data, 20, 100.0)
	putFixed(data, 24, 400.0)
	putFixed(data, 28, 900.0)
	putU16(data, 32, 0)
	putU16(data, 34, 256)

	// Instance 0: "Bold" wght=700, postScriptNameID=300
	inst0Off := 36
	putU16(data, inst0Off, 260)   // subfamilyNameID
	putU16(data, inst0Off+2, 0)    // flags
	putFixed(data, inst0Off+4, 700.0)
	putU16(data, inst0Off+8, 300)  // postScriptNameID

	fvar, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(fvar.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(fvar.Instances))
	}
	inst := fvar.Instances[0]
	if inst.PostScriptNameID != 300 {
		t.Fatalf("instance postScriptNameID = %d, want 300", inst.PostScriptNameID)
	}
}

// --- AxisByTag / DefaultCoordinates ---

func TestAxisByTag(t *testing.T) {
	fvar := FVAR{
		Axes: []Axis{
			{Tag: "wght", DefaultValue: 400.0},
			{Tag: "wdth", DefaultValue: 100.0},
		},
	}
	ax, ok := fvar.AxisByTag("wght")
	if !ok {
		t.Fatal("expected to find wght axis")
	}
	if ax.DefaultValue != 400.0 {
		t.Fatalf("wght defaultValue = %v, want 400", ax.DefaultValue)
	}
	_, ok = fvar.AxisByTag("ital")
	if ok {
		t.Fatal("expected ital axis to not be found")
	}
}

func TestDefaultCoordinates(t *testing.T) {
	fvar := FVAR{
		Axes: []Axis{
			{Tag: "wght", DefaultValue: 400.0},
			{Tag: "wdth", DefaultValue: 100.0},
		},
	}
	coords := fvar.DefaultCoordinates()
	if len(coords) != 2 {
		t.Fatalf("expected 2 default coords, got %d", len(coords))
	}
	if coords[0] != 400.0 {
		t.Fatalf("default wght = %v, want 400", coords[0])
	}
	if coords[1] != 100.0 {
		t.Fatalf("default wdth = %v, want 100", coords[1])
	}
}

// --- Negative values (italics angle) ---

func TestDecodeNegativeAxisValue(t *testing.T) {
	// slnt axis with negative values (-12..0..0)
	data := make([]byte, 36)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 16)
	putU16(data, 6, 2)
	putU16(data, 8, 1)
	putU16(data, 10, 20)
	putU16(data, 12, 0)
	putU16(data, 14, 8)  // instanceSize (min for 1 axis, even though instanceCount=0)

	copy(data[16:20], "slnt")
	putFixed(data, 20, -12.0)
	putFixed(data, 24, 0.0)
	putFixed(data, 28, 0.0)
	putU16(data, 32, 0)
	putU16(data, 34, 256)

	fvar, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if fvar.Axes[0].MinValue != -12.0 {
		t.Fatalf("slnt minValue = %v, want -12", fvar.Axes[0].MinValue)
	}
	if fvar.Axes[0].DefaultValue != 0.0 {
		t.Fatalf("slnt defaultValue = %v, want 0", fvar.Axes[0].DefaultValue)
	}
}

// --- Fractional values (opsz) ---

func TestDecodeFractionalAxisValue(t *testing.T) {
	// opsz axis with fractional values like 8.0..14.0..144.0
	data := make([]byte, 36)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 16)
	putU16(data, 6, 2)
	putU16(data, 8, 1)
	putU16(data, 10, 20)
	putU16(data, 12, 0)
	putU16(data, 14, 0)

	copy(data[16:20], "opsz")
	putFixed(data, 20, 8.0)
	putFixed(data, 24, 14.0)
	putFixed(data, 28, 144.0)
	putU16(data, 32, 0)
	putU16(data, 34, 256)

	fvar, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	ax := fvar.Axes[0]
	if ax.MinValue != 8.0 {
		t.Fatalf("opsz min = %v, want 8", ax.MinValue)
	}
	if ax.DefaultValue != 14.0 {
		t.Fatalf("opsz default = %v, want 14", ax.DefaultValue)
	}
	if ax.MaxValue != 144.0 {
		t.Fatalf("opsz max = %v, want 144", ax.MaxValue)
	}
}

// --- Error cases ---

func TestDecodeTruncatedHeader(t *testing.T) {
	_, err := Decode(make([]byte, 10))
	if err == nil {
		t.Fatal("expected error for truncated fvar header")
	}
}

func TestDecodeBadVersion(t *testing.T) {
	data := make([]byte, 16)
	putU16(data, 0, 2) // bad major version
	putU16(data, 2, 0)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestDecodeBadAxisSize(t *testing.T) {
	data := make([]byte, 16)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 16)
	putU16(data, 6, 2)
	putU16(data, 8, 1)
	putU16(data, 10, 12) // bad axisSize
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for bad axisSize")
	}
}

func TestDecodeAxesExceedTable(t *testing.T) {
	data := make([]byte, 16)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 16)
	putU16(data, 6, 2)
	putU16(data, 8, 5)  // 5 axes
	putU16(data, 10, 20) // but only 0 bytes of axis data
	putU16(data, 12, 0)
	putU16(data, 14, 0)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for axes exceeding table")
	}
}

func TestDecodeInstancesExceedTable(t *testing.T) {
	// Header + 1 axis but instanceCount = 1 but no instance data
	data := make([]byte, 36)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 16)
	putU16(data, 6, 2)
	putU16(data, 8, 1)
	putU16(data, 10, 20)
	putU16(data, 12, 1)  // 1 instance
	putU16(data, 14, 8)  // instanceSize = 8

	copy(data[16:20], "wght")
	putFixed(data, 20, 100.0)
	putFixed(data, 24, 400.0)
	putFixed(data, 28, 900.0)
	putU16(data, 32, 0)
	putU16(data, 34, 256)

	// Instance data would need 36 + 8 = 44 bytes, but we only have 36
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for instances exceeding table")
	}
}

func TestDecodeZeroAxesZeroInstances(t *testing.T) {
	data := make([]byte, 16)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU16(data, 4, 16)
	putU16(data, 6, 2)
	putU16(data, 8, 0)
	putU16(data, 10, 20)
	putU16(data, 12, 0)
	putU16(data, 14, 0)

	fvar, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if len(fvar.Axes) != 0 {
		t.Fatalf("expected 0 axes, got %d", len(fvar.Axes))
	}
	if len(fvar.Instances) != 0 {
		t.Fatalf("expected 0 instances, got %d", len(fvar.Instances))
	}
}

// --- Real font test ---

func TestFVARFromGoRegular(t *testing.T) {
	font, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	fvarData, err := font.TableData("fvar")
	if err != nil {
		t.Skipf("font has no fvar table: %v", err)
	}
	fvar, err := Decode(fvarData)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	t.Logf("Axes: %d, Instances: %d", len(fvar.Axes), len(fvar.Instances))
	for _, ax := range fvar.Axes {
		t.Logf("  Axis: tag=%s min=%v default=%v max=%v flags=%d nameID=%d",
			ax.Tag, ax.MinValue, ax.DefaultValue, ax.MaxValue, ax.Flags, ax.AxisNameID)
	}
}

// --- Fuzz test ---

func FuzzFVARDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 16))

	f.Fuzz(func(t *testing.T, data []byte) {
		fvar, err := Decode(data)
		if err != nil {
			return
		}
		_ = fvar
	})
}