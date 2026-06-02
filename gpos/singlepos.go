// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import "fmt"

type SinglePosValue struct {
	Glyph       string
	XPlacement  int16
	YPlacement  int16
	XAdvance    int16
	YAdvance    int16
	LookupIndex int
	Format      int
}

// DecodeSinglePosLookups decodes GPOS SinglePos lookups (type 1) from a GPOS
// table. It returns a flat list of all resolved single positioning adjustments.
func DecodeSinglePosLookups(data []byte, glyphOrder []string) ([]SinglePosValue, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("GPOS header too short: %d bytes", len(data))
	}
	lookupListOffset := int(readU16(data, 8))
	if lookupListOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS LookupList offset exceeds table")
	}
	lookupCount := int(readU16(data, lookupListOffset))
	if lookupCount > MaxLookupCount {
		return nil, fmt.Errorf("GPOS LookupCount %d exceeds limit", lookupCount)
	}
	if lookupListOffset+2+lookupCount*2 > len(data) {
		return nil, fmt.Errorf("GPOS LookupList records exceed table")
	}

	var all []SinglePosValue
	for lookupIndex := 0; lookupIndex < lookupCount; lookupIndex++ {
		lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))
		vals, err := decodeSingleLookup(data, glyphOrder, lookupOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, vals...)
	}
	return all, nil
}

func decodeSingleLookup(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int) ([]SinglePosValue, error) {
	if lookupOffset+6 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d exceeds table", lookupIndex)
	}
	lookupType := readU16(data, lookupOffset)
	if lookupType != 1 && lookupType != 9 {
		return nil, nil
	}
	subtableCount := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable offsets exceed table", lookupIndex)
	}

	var all []SinglePosValue
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		vals, err := decodeSingleSubtable(data, glyphOrder, lookupType, subtableOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, vals...)
	}
	return all, nil
}

func decodeSingleSubtable(data []byte, glyphOrder []string, lookupType uint16, subtableOffset int, lookupIndex int) ([]SinglePosValue, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS single subtable exceeds table")
	}
	if lookupType == 9 {
		return decodeSingleExtensionSubtable(data, glyphOrder, subtableOffset, lookupIndex)
	}
	format := int(readU16(data, subtableOffset))
	switch format {
	case 1:
		return decodeSinglePosFormat1(data, glyphOrder, subtableOffset, lookupIndex)
	case 2:
		return decodeSinglePosFormat2(data, glyphOrder, subtableOffset, lookupIndex)
	}
	return nil, nil
}

func decodeSingleExtensionSubtable(data []byte, glyphOrder []string, subtableOffset int, lookupIndex int) ([]SinglePosValue, error) {
	if subtableOffset+8 > len(data) {
		return nil, fmt.Errorf("GPOS single extension subtable exceeds table")
	}
	format := readU16(data, subtableOffset)
	if format != 1 {
		return nil, nil
	}
	extensionType := readU16(data, subtableOffset+2)
	if extensionType != 1 {
		return nil, nil
	}
	extensionOffset := int(readU32(data, subtableOffset+4))
	targetOffset := subtableOffset + extensionOffset
	if targetOffset <= subtableOffset || targetOffset > len(data) {
		return nil, fmt.Errorf("GPOS single extension target exceeds table")
	}
	if targetOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS single extension target exceeds table")
	}
	targetFormat := int(readU16(data, targetOffset))
	switch targetFormat {
	case 1:
		return decodeSinglePosFormat1(data, glyphOrder, targetOffset, lookupIndex)
	case 2:
		return decodeSinglePosFormat2(data, glyphOrder, targetOffset, lookupIndex)
	}
	return nil, nil
}

const (
	vfXPlacement = 0x0001
	vfYPlacement = 0x0002
	vfXAdvance   = 0x0004
	vfYAdvance   = 0x0008
)

type valueRecord struct {
	XPlacement int16
	YPlacement int16
	XAdvance   int16
	YAdvance   int16
}

func decodeValueRecord(data []byte, offset int, valueFormat uint16) (valueRecord, int, error) {
	var vr valueRecord
	consumed := 0
	for bit := uint16(1); bit != 0; bit <<= 1 {
		if valueFormat&bit == 0 {
			continue
		}
		if offset+consumed+2 > len(data) {
			return vr, consumed, fmt.Errorf("ValueRecord exceeds table")
		}
		v := int16(readU16(data, offset+consumed))
		consumed += 2
		switch bit {
		case vfXPlacement:
			vr.XPlacement = v
		case vfYPlacement:
			vr.YPlacement = v
		case vfXAdvance:
			vr.XAdvance = v
		case vfYAdvance:
			vr.YAdvance = v
		}
	}
	return vr, consumed, nil
}

func decodeSinglePosFormat1(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]SinglePosValue, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("SinglePosFormat1 too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, fmt.Errorf("expected SinglePosFormat1, got format %d", format)
	}
	coverageOffset := int(readU16(data, offset+2))
	valueFormat := readU16(data, offset+4)

	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, fmt.Errorf("SinglePosFormat1 coverage: %w", err)
	}

	vr, _, err := decodeValueRecord(data, offset+6, valueFormat)
	if err != nil {
		return nil, fmt.Errorf("SinglePosFormat1 value record: %w", err)
	}

	vals := make([]SinglePosValue, 0, len(coverage))
	for _, glyphID := range coverage {
		glyph, ok := glyphName(glyphOrder, glyphID)
		if !ok {
			continue
		}
		vals = append(vals, SinglePosValue{
			Glyph:       glyph,
			XPlacement:  vr.XPlacement,
			YPlacement:  vr.YPlacement,
			XAdvance:    vr.XAdvance,
			YAdvance:    vr.YAdvance,
			LookupIndex: lookupIndex,
			Format:      1,
		})
	}
	return vals, nil
}

func decodeSinglePosFormat2(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]SinglePosValue, error) {
	if offset+8 > len(data) {
		return nil, fmt.Errorf("SinglePosFormat2 too short")
	}
	format := readU16(data, offset)
	if format != 2 {
		return nil, fmt.Errorf("expected SinglePosFormat2, got format %d", format)
	}
	coverageOffset := int(readU16(data, offset+2))
	valueFormat := readU16(data, offset+4)
	valueCount := int(readU16(data, offset+6))

	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, fmt.Errorf("SinglePosFormat2 coverage: %w", err)
	}
	if len(coverage) < valueCount {
		return nil, fmt.Errorf("SinglePosFormat2 coverage count %d < valueCount %d", len(coverage), valueCount)
	}

	pos := offset + 8
	vals := make([]SinglePosValue, 0, valueCount)
	for i := 0; i < valueCount; i++ {
		glyphID := coverage[i]
		glyph, ok := glyphName(glyphOrder, glyphID)
		if !ok {
			_, consumed, err := decodeValueRecord(data, pos, valueFormat)
			if err != nil {
				return nil, fmt.Errorf("SinglePosFormat2 value %d: %w", i, err)
			}
			pos += consumed
			continue
		}
		vr, consumed, err := decodeValueRecord(data, pos, valueFormat)
		if err != nil {
			return nil, fmt.Errorf("SinglePosFormat2 value %d: %w", i, err)
		}
		pos += consumed
		vals = append(vals, SinglePosValue{
			Glyph:       glyph,
			XPlacement:  vr.XPlacement,
			YPlacement:  vr.YPlacement,
			XAdvance:    vr.XAdvance,
			YAdvance:    vr.YAdvance,
			LookupIndex: lookupIndex,
			Format:      2,
		})
	}
	return vals, nil
}
