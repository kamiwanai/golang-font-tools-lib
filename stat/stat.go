// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package stat

import (
	"encoding/binary"
	"fmt"
)

type DesignAxis struct {
	Tag          string
	AxisNameID   uint16
	AxisOrdering uint16
}

type AxisValueFormat uint16

const (
	AxisValueFormat1 AxisValueFormat = 1
	AxisValueFormat2 AxisValueFormat = 2
	AxisValueFormat3 AxisValueFormat = 3
	AxisValueFormat4 AxisValueFormat = 4
)

type AxisValue interface {
	Format() AxisValueFormat
	ValueNameID() uint16
	Flags() uint16
}

type AxisValue1 struct {
	Flags_       uint16
	ValueNameID_ uint16
	Value_       float64
}
func (a AxisValue1) Format() AxisValueFormat { return AxisValueFormat1 }
func (a AxisValue1) ValueNameID() uint16     { return a.ValueNameID_ }
func (a AxisValue1) Flags() uint16           { return a.Flags_ }

type AxisValue2 struct {
	Flags_         uint16
	ValueNameID_   uint16
	NominalValue_  float64
	RangeMinValue_ float64
	RangeMaxValue_ float64
}
func (a AxisValue2) Format() AxisValueFormat { return AxisValueFormat2 }
func (a AxisValue2) ValueNameID() uint16     { return a.ValueNameID_ }
func (a AxisValue2) Flags() uint16           { return a.Flags_ }

type AxisValue3 struct {
	Flags_       uint16
	ValueNameID_ uint16
	Value_       float64
	LinkedValue_ float64
}
func (a AxisValue3) Format() AxisValueFormat { return AxisValueFormat3 }
func (a AxisValue3) ValueNameID() uint16     { return a.ValueNameID_ }
func (a AxisValue3) Flags() uint16           { return a.Flags_ }

type AxisValue4 struct {
	Flags_       uint16
	ValueNameID_ uint16
	AxisValues_  []AxisValueRecord
}
func (a AxisValue4) Format() AxisValueFormat { return AxisValueFormat4 }
func (a AxisValue4) ValueNameID() uint16     { return a.ValueNameID_ }
func (a AxisValue4) Flags() uint16           { return a.Flags_ }

type AxisValueRecord struct {
	AxisIndex uint16
	Value     float64
}

type STAT struct {
	MajorVersion          uint16
	MinorVersion          uint16
	DesignAxes            []DesignAxis
	AxisValues            []AxisValue
	ElidedFallbackNameID  uint16
}

func f2dot14ToFloat64(raw uint16) float64 {
	return float64(int16(raw)) / 16384.0
}

func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}


func Decode(data []byte) (STAT, error) {
	if len(data) < 12 {
		return STAT{}, fmt.Errorf("STAT header too short: %d bytes", len(data))
	}
	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)
	designAxisSize := int(readU16(data, 4))
	designAxisCount := int(readU16(data, 6))
	designAxesOffset := int(readU32(data, 8))
	pos := 12
	var axisValueCount int
	var axisValueOffsetsOffset int
	var elidedFallbackNameID uint16
	if minorVersion >= 1 {
		if pos+6 > len(data) {
			return STAT{}, fmt.Errorf("STAT v1.1+ header too short")
		}
		axisValueCount = int(readU16(data, pos))
		axisValueOffsetsOffset = int(readU32(data, pos+2))
		pos += 6
	}
	if minorVersion >= 2 {
		if pos+2 > len(data) {
			return STAT{}, fmt.Errorf("STAT v1.2+ header too short")
		}
		elidedFallbackNameID = readU16(data, pos)
		pos += 2
	}
	if designAxisCount > MaxDesignAxes {
		return STAT{}, fmt.Errorf("STAT designAxisCount %d exceeds limit %d", designAxisCount, MaxDesignAxes)
	}
	axes := make([]DesignAxis, 0, designAxisCount)
	if designAxisCount > 0 {
		if designAxisSize < 8 {
			return STAT{}, fmt.Errorf("STAT designAxisSize %d too small", designAxisSize)
		}
		axesEnd := designAxesOffset + designAxisCount*designAxisSize
		if axesEnd > len(data) {
			return STAT{}, fmt.Errorf("STAT design axes exceed table")
		}
		for i := 0; i < designAxisCount; i++ {
			off := designAxesOffset + i*designAxisSize
			tag := string(data[off : off+4])
			nameID := readU16(data, off+4)
			ordering := readU16(data, off+6)
			axes = append(axes, DesignAxis{Tag: tag, AxisNameID: nameID, AxisOrdering: ordering})
		}
	}
	var axisValues []AxisValue
	if axisValueCount > 0 {
		if axisValueCount > MaxAxisValues {
			return STAT{}, fmt.Errorf("STAT axisValueCount %d exceeds limit %d", axisValueCount, MaxAxisValues)
		}
		if axisValueOffsetsOffset+axisValueCount*2 > len(data) {
			return STAT{}, fmt.Errorf("STAT axis value offsets exceed table")
		}
		for i := 0; i < axisValueCount; i++ {
			avOff := int(readU16(data, axisValueOffsetsOffset+i*2))
			absOff := axisValueOffsetsOffset + avOff
			av, err := decodeAxisValue(data, absOff, axes)
			if err != nil {
				return STAT{}, fmt.Errorf("STAT axis value %d: %w", i, err)
			}
			axisValues = append(axisValues, av)
		}
	}
	return STAT{
		MajorVersion: majorVersion, MinorVersion: minorVersion,
		DesignAxes: axes, AxisValues: axisValues,
		ElidedFallbackNameID: elidedFallbackNameID,
	}, nil
}

func decodeAxisValue(data []byte, offset int, axes []DesignAxis) (AxisValue, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("axis value header too short at offset %d", offset)
	}
	format := readU16(data, offset)
	flags := readU16(data, offset+2)
	valueNameID := readU16(data, offset+4)
	switch format {
	case 1:
		if offset+8 > len(data) { return nil, fmt.Errorf("format 1 axis value too short") }
		value := f2dot14ToFloat64(readU16(data, offset+6))
		return AxisValue1{Flags_: flags, ValueNameID_: valueNameID, Value_: value}, nil
	case 2:
		if offset+14 > len(data) { return nil, fmt.Errorf("format 2 axis value too short") }
		nv := f2dot14ToFloat64(readU16(data, offset+6))
		rmi := f2dot14ToFloat64(readU16(data, offset+8))
		rma := f2dot14ToFloat64(readU16(data, offset+10))
		return AxisValue2{Flags_: flags, ValueNameID_: valueNameID, NominalValue_: nv, RangeMinValue_: rmi, RangeMaxValue_: rma}, nil
	case 3:
		if offset+10 > len(data) { return nil, fmt.Errorf("format 3 axis value too short") }
		value := f2dot14ToFloat64(readU16(data, offset+6))
		linked := f2dot14ToFloat64(readU16(data, offset+8))
		return AxisValue3{Flags_: flags, ValueNameID_: valueNameID, Value_: value, LinkedValue_: linked}, nil
	case 4:
		if offset+8 > len(data) { return nil, fmt.Errorf("format 4 axis value too short") }
		axisCount := int(readU16(data, offset+6))
		recsEnd := offset + 8 + axisCount*4
		if recsEnd > len(data) { return nil, fmt.Errorf("format 4 axis values exceed table") }
		records := make([]AxisValueRecord, axisCount)
		for i := 0; i < axisCount; i++ {
			recOff := offset + 8 + i*4
			records[i] = AxisValueRecord{AxisIndex: readU16(data, recOff), Value: f2dot14ToFloat64(readU16(data, recOff+2))}
		}
		return AxisValue4{Flags_: flags, ValueNameID_: valueNameID, AxisValues_: records}, nil
	default:
		return nil, fmt.Errorf("unsupported axis value format %d", format)
	}
}
