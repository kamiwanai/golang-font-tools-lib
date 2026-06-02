// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package fvar

import (
	"encoding/binary"
	"fmt"
)

// Axis describes one variation axis in an fvar table.
type Axis struct {
	// Tag is the 4-byte axis identifier (e.g. "wght", "wdth", "ital", "slnt", "opsz").
	Tag string
	// MinValue is the minimum value for this axis (in Fixed 16.16 format, converted to float64).
	MinValue float64
	// DefaultValue is the default value for this axis.
	DefaultValue float64
	// MaxValue is the maximum value for this axis.
	MaxValue float64
	// Flags is the axis flags. Bit 0: HIDDEN_AXIS — the axis should not be
	// exposed directly in user interfaces.
	Flags uint16
	// AxisNameID is the name table entry ID for this axis.
	AxisNameID uint16
}

// Instance describes one named instance in an fvar table.
type Instance struct {
	// SubfamilyNameID is the name table entry ID for this instance's subfamily name.
	SubfamilyNameID uint16
	// Flags is the instance flags. Bit 0: MSID (bit used for instances with
	// names that don't follow the RIBBI model).
	Flags uint16
	// Coordinates holds the axis coordinate values, one per axis, in the same
	// order as the axes array.
	Coordinates []float64
	// PostScriptNameID is the name table entry ID for the PostScript name
	// of this instance. Value 0xFFFF or 0 means not present.
	PostScriptNameID uint16
}

// FVAR holds the parsed fields from an OpenType fvar (Font Variations) table.
type FVAR struct {
	// MajorVersion is the major version number (expected 1).
	MajorVersion uint16
	// MinorVersion is the minor version number (expected 0).
	MinorVersion uint16
	// Axes is the list of variation axes.
	Axes []Axis
	// Instances is the list of named instances.
	Instances []Instance
}

// fixed1616toInt32 reads a Fixed 16.16 value from data at offset.
// The value is stored as a signed 32-bit integer in big-endian.
func fixed1616toFloat64(data []byte, offset int) float64 {
	raw := int32(binary.BigEndian.Uint32(data[offset : offset+4]))
	return float64(raw) / 65536.0
}

// readU16 reads a big-endian uint16 from data at offset.
func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

// Decode parses an fvar table from raw bytes and returns an FVAR struct.
func Decode(data []byte) (FVAR, error) {
	// fvar header is 16 bytes minimum.
	if len(data) < 16 {
		return FVAR{}, fmt.Errorf("fvar header too short: %d bytes", len(data))
	}

	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)
	axesArrayOffset := int(readU16(data, 4))
	// reserved := readU16(data, 6) — should be 2, but we don't enforce
	axisCount := int(readU16(data, 8))
	axisSize := int(readU16(data, 10))
	instanceCount := int(readU16(data, 12))
	instanceSize := int(readU16(data, 14))

	if majorVersion != 1 {
		return FVAR{}, fmt.Errorf("fvar unsupported major version %d", majorVersion)
	}
	if axisSize != 20 {
		return FVAR{}, fmt.Errorf("fvar unsupported axisSize %d (expected 20)", axisSize)
	}
	if axisCount > MaxAxes {
		return FVAR{}, fmt.Errorf("fvar axisCount %d exceeds limit %d", axisCount, MaxAxes)
	}
	if instanceCount > MaxInstances {
		return FVAR{}, fmt.Errorf("fvar instanceCount %d exceeds limit %d", instanceCount, MaxInstances)
	}

	// Each axis is 20 bytes.
	axesEnd := axesArrayOffset + axisCount*axisSize
	if axesEnd > len(data) {
		return FVAR{}, fmt.Errorf("fvar axes array exceeds table: need offset %d, table length %d", axesEnd, len(data))
	}

	axes := make([]Axis, axisCount)
	for i := 0; i < axisCount; i++ {
		off := axesArrayOffset + i*axisSize
		tag := string(data[off : off+4])
		minVal := fixed1616toFloat64(data, off+4)
		defaultVal := fixed1616toFloat64(data, off+8)
		maxVal := fixed1616toFloat64(data, off+12)
		flags := readU16(data, off+16)
		axisNameID := readU16(data, off+18)

		axes[i] = Axis{
			Tag:         tag,
			MinValue:    minVal,
			DefaultValue: defaultVal,
			MaxValue:    maxVal,
			Flags:       flags,
			AxisNameID:  axisNameID,
		}
	}

	// Determine instance array start.
	// Instances start right after axes array.
	instanceArrayOffset := axesArrayOffset + axisCount*axisSize

	// Minimum instance size: subfamilyNameID(2) + flags(2) + axisCount*4 = 4 + axisCount*4
	// When instanceCount is 0, instanceSize is irrelevant (may be 0 or a placeholder).
	minInstanceSize := 4 + axisCount*4
	if instanceCount > 0 && instanceSize < minInstanceSize {
		return FVAR{}, fmt.Errorf("fvar instanceSize %d too small for %d axes (minimum %d)", instanceSize, axisCount, minInstanceSize)
	}

	// Check that the instance array fits.
	instanceEnd := instanceArrayOffset + instanceCount*instanceSize
	if instanceEnd > len(data) {
		return FVAR{}, fmt.Errorf("fvar instance array exceeds table: need offset %d, table length %d", instanceEnd, len(data))
	}

	// Check for postScriptNameID field.
	// If instanceSize > minInstanceSize, there is a postScriptNameID (uint16) at the end.
	hasPostScriptNameID := instanceSize >= minInstanceSize+2

	instances := make([]Instance, instanceCount)
	for i := 0; i < instanceCount; i++ {
		off := instanceArrayOffset + i*instanceSize
		subfamilyNameID := readU16(data, off)
		flags := readU16(data, off+2)

		coordinates := make([]float64, axisCount)
		for j := 0; j < axisCount; j++ {
			coordinates[j] = fixed1616toFloat64(data, off+4+j*4)
		}

		var postScriptNameID uint16
		if hasPostScriptNameID {
			postScriptNameID = readU16(data, off+4+axisCount*4)
		}

		instances[i] = Instance{
			SubfamilyNameID:  subfamilyNameID,
			Flags:            flags,
			Coordinates:      coordinates,
			PostScriptNameID: postScriptNameID,
		}
	}

	return FVAR{
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
		Axes:         axes,
		Instances:    instances,
	}, nil
}

// AxisByTag returns the axis with the given tag, or false if not found.
func (f FVAR) AxisByTag(tag string) (Axis, bool) {
	for _, a := range f.Axes {
		if a.Tag == tag {
			return a, true
		}
	}
	return Axis{}, false
}

// IsHiddenAxis returns true if the axis has the HIDDEN_AXIS flag set.
func (a Axis) IsHiddenAxis() bool {
	return a.Flags&1 != 0
}

// DefaultCoordinates returns the default coordinates (one per axis) as a slice.
func (f FVAR) DefaultCoordinates() []float64 {
	coords := make([]float64, len(f.Axes))
	for i, a := range f.Axes {
		coords[i] = a.DefaultValue
	}
	return coords
}