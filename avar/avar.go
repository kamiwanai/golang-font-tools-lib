// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package avar

import (
	"encoding/binary"
	"fmt"
)

// SegmentMap is one segment of the axis value map. Each segment defines
// a linear mapping from one coordinate range to another.
type SegmentMap struct {
	// FromCoordinate is the normalized input coordinate (F2Dot14).
	FromCoordinate float64
	// ToCoordinate is the mapped output coordinate (F2Dot14).
	ToCoordinate float64
}

// AxisValueMap holds the piecewise linear map for one variation axis.
type AxisValueMap struct {
	// Segments is the ordered list of mapping points. Coordinates between
	// consecutive entries are linearly interpolated.
	Segments []SegmentMap
}

// AVAR holds the parsed fields from an OpenType avar table.
type AVAR struct {
	// MajorVersion is the major version number (expected 1).
	MajorVersion uint16
	// MinorVersion is the minor version number (expected 0).
	MinorVersion uint16
	// AxisMaps is the list of axis value maps, one per variation axis,
	// in the same order as in the fvar table.
	AxisMaps []AxisValueMap
}

// f2dot14ToFloat64 converts a 16-bit F2Dot14 to float64.
func f2dot14ToFloat64(raw uint16) float64 {
	// F2Dot14: signed 16-bit with 14 fraction bits.
	return float64(int16(raw)) / 16384.0
}

// readU16 reads a big-endian uint16 from data at offset.
func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

// Decode parses an avar table from raw bytes and returns an AVAR struct.
func Decode(data []byte) (AVAR, error) {
	if len(data) < 8 {
		return AVAR{}, fmt.Errorf("avar header too short: %d bytes", len(data))
	}

	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)
	_ = readU16(data, 4) // reserved, must be 0
	axisCount := int(readU16(data, 6))

	if majorVersion != 1 {
		return AVAR{}, fmt.Errorf("avar unsupported major version %d", majorVersion)
	}
	if axisCount > MaxAxes {
		return AVAR{}, fmt.Errorf("avar axisCount %d exceeds limit %d", axisCount, MaxAxes)
	}

	axisMaps := make([]AxisValueMap, axisCount)
	pos := 8

	for i := 0; i < axisCount; i++ {
		if pos+2 > len(data) {
			return AVAR{}, fmt.Errorf("avar axis %d: too short for segment count", i)
		}
		segmentCount := int(readU16(data, pos))
		pos += 2

		if segmentCount > MaxSegmentMaps {
			return AVAR{}, fmt.Errorf("avar axis %d: segment count %d exceeds limit %d", i, segmentCount, MaxSegmentMaps)
		}

		// Each segment: fromCoord(2) + toCoord(2) = 4 bytes
		segEnd := pos + segmentCount*4
		if segEnd > len(data) {
			return AVAR{}, fmt.Errorf("avar axis %d: segments exceed table", i)
		}

		segments := make([]SegmentMap, segmentCount)
		for j := 0; j < segmentCount; j++ {
			fromCoord := f2dot14ToFloat64(readU16(data, pos))
			pos += 2
			toCoord := f2dot14ToFloat64(readU16(data, pos))
			pos += 2
			segments[j] = SegmentMap{
				FromCoordinate: fromCoord,
				ToCoordinate:   toCoord,
			}
		}
		axisMaps[i] = AxisValueMap{Segments: segments}
	}

	return AVAR{
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
		AxisMaps:     axisMaps,
	}, nil
}

// MapCoord applies the axis value map for the given axis index, mapping
// a normalized coordinate from the input space to the output space.
// If no map exists for this axis, the coordinate is returned unchanged.
func (a AVAR) MapCoord(axisIndex int, coord float64) float64 {
	if axisIndex < 0 || axisIndex >= len(a.AxisMaps) {
		return coord
	}
	seg := a.AxisMaps[axisIndex].Segments
	if len(seg) == 0 {
		return coord
	}

	// Clamp to segments range.
	if coord <= seg[0].FromCoordinate {
		return seg[0].ToCoordinate
	}
	lastIdx := len(seg) - 1
	if coord >= seg[lastIdx].FromCoordinate {
		return seg[lastIdx].ToCoordinate
	}

	// Linear interpolation between the two surrounding segments.
	for j := 0; j < len(seg)-1; j++ {
		if coord >= seg[j].FromCoordinate && coord <= seg[j+1].FromCoordinate {
			fromDelta := seg[j+1].FromCoordinate - seg[j].FromCoordinate
			if fromDelta == 0 {
				return seg[j].ToCoordinate
			}
			t := (coord - seg[j].FromCoordinate) / fromDelta
			return seg[j].ToCoordinate + t*(seg[j+1].ToCoordinate-seg[j].ToCoordinate)
		}
	}
	return coord
}
