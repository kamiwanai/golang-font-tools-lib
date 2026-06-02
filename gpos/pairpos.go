// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gpos

import (
	"encoding/binary"
	"fmt"
	"math/bits"

	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

const xAdvanceFlag uint16 = 0x0004

// PairValue is one expanded kerning pair decoded from GPOS PairPos data.
type PairValue struct {
	// LeftGlyph is the left glyph name.
	LeftGlyph string
	// RightGlyph is the right glyph name.
	RightGlyph string
	// Value is the aggregated xAdvance adjustment in font units.
	Value int
	// Sources records each lookup contribution to Value.
	Sources []PairSource
}

// PairSource identifies one GPOS lookup contribution to a PairValue.
type PairSource struct {
	// LookupIndex is the zero-based lookup index in the GPOS LookupList.
	LookupIndex int
	// Format is the PairPos subtable format, either 1 or 2.
	Format int
	// Value is the xAdvance contribution from this source.
	Value int
}

// DecodePairPosLookups decodes supported PairPos kerning lookups from a GPOS table.
//
// glyphOrder maps glyph IDs to names. Repeated pairs across lookups are
// aggregated and their individual contributions are retained in Sources.
func DecodePairPosLookups(data []byte, glyphOrder []string) ([]PairValue, error) {
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
	pairs := make([]PairValue, 0)
	pairIndex := make(map[string]int)
	for lookupIndex := 0; lookupIndex < lookupCount; lookupIndex++ {
		lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))
		lookupPairs, err := decodeLookup(data, glyphOrder, lookupOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		for _, pair := range lookupPairs {
			key := pair.LeftGlyph + "\x00" + pair.RightGlyph
			if index, ok := pairIndex[key]; ok {
				pairs[index].Value += pair.Value
				pairs[index].Sources = append(pairs[index].Sources, pair.Sources...)
				continue
			}
			pairIndex[key] = len(pairs)
			pairs = append(pairs, pair)
		}
	}
	return pairs, nil
}

func decodeLookup(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int) ([]PairValue, error) {
	if lookupOffset+6 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d exceeds table", lookupIndex)
	}
	lookupType := readU16(data, lookupOffset)
	if lookupType != 2 && lookupType != 9 {
		return nil, nil
	}
	subtableCount := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable offsets exceed table", lookupIndex)
	}
	pairs := make([]PairValue, 0)
	for subtableIndex := 0; subtableIndex < subtableCount; subtableIndex++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+subtableIndex*2))
		subtablePairs, err := decodeLookupSubtable(data, glyphOrder, lookupType, lookupOffset, lookupIndex, subtableOffset, subtableIndex)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, subtablePairs...)
	}
	return pairs, nil
}

func decodeLookupSubtable(data []byte, glyphOrder []string, lookupType uint16, lookupOffset int, lookupIndex int, subtableOffset int, subtableIndex int) ([]PairValue, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable %d exceeds table", lookupIndex, subtableIndex)
	}
	if lookupType == 9 {
		return decodeExtensionSubtable(data, glyphOrder, lookupOffset, lookupIndex, subtableOffset, subtableIndex)
	}
	format := int(readU16(data, subtableOffset))
	pairs, err := decodePairPosSubtable(data[subtableOffset:], glyphOrder)
	if err != nil {
		return nil, err
	}
	annotatePairSources(pairs, lookupIndex, format)
	return pairs, nil
}

func decodeExtensionSubtable(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int, subtableOffset int, subtableIndex int) ([]PairValue, error) {
	if subtableOffset+8 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable %d extension exceeds table", lookupIndex, subtableIndex)
	}
	format := readU16(data, subtableOffset)
	if format != 1 {
		return nil, nil
	}
	extensionType := readU16(data, subtableOffset+2)
	if extensionType != 2 {
		return nil, nil
	}
	extensionOffset := int(readU32(data, subtableOffset+4))
	targetOffset := subtableOffset + extensionOffset
	if targetOffset <= subtableOffset || targetOffset > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable %d extension target exceeds table", lookupIndex, subtableIndex)
	}
	if targetOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable %d extension target exceeds table", lookupIndex, subtableIndex)
	}
	targetFormat := int(readU16(data, targetOffset))
	pairs, err := decodePairPosSubtable(data[targetOffset:], glyphOrder)
	if err != nil {
		return nil, err
	}
	annotatePairSources(pairs, lookupIndex, targetFormat)
	return pairs, nil
}

func decodePairPosSubtable(subtable []byte, glyphOrder []string) ([]PairValue, error) {
	if len(subtable) < 2 {
		return nil, fmt.Errorf("PairPos subtable too short: %d bytes", len(subtable))
	}
	format := readU16(subtable, 0)
	switch format {
	case 1:
		return DecodePairPosFormat1(subtable, glyphOrder)
	case 2:
		return DecodePairPosFormat2(subtable, glyphOrder)
	default:
		return nil, nil
	}
}

func annotatePairSources(pairs []PairValue, lookupIndex int, format int) {
	for index := range pairs {
		pairs[index].Sources = []PairSource{{
			LookupIndex: lookupIndex,
			Format:      format,
			Value:       pairs[index].Value,
		}}
	}
}

// DecodePairPosFormat2 decodes a PairPos format 2 class kerning subtable.
func DecodePairPosFormat2(data []byte, glyphOrder []string) ([]PairValue, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("PairPos format 2 too short: %d bytes", len(data))
	}
	format := readU16(data, 0)
	if format != 2 {
		return nil, fmt.Errorf("unsupported PairPos format %d", format)
	}
	coverageOffset := int(readU16(data, 2))
	valueFormat1 := readU16(data, 4)
	valueFormat2 := readU16(data, 6)
	classDef1Offset := int(readU16(data, 8))
	classDef2Offset := int(readU16(data, 10))
	class1Count := int(readU16(data, 12))
	class2Count := int(readU16(data, 14))

	coverage, err := decodeCoverage(data, coverageOffset)
	if err != nil {
		return nil, err
	}
	classDef1, err := decodeClassDef(data, classDef1Offset)
	if err != nil {
		return nil, err
	}
	classDef2, err := decodeClassDef(data, classDef2Offset)
	if err != nil {
		return nil, err
	}
	leftMembers, err := classMembersFromCoverage(coverage, classDef1, class1Count, glyphOrder)
	if err != nil {
		return nil, err
	}
	rightMembers := classMembersFromGlyphOrder(classDef2, class2Count, glyphOrder)

	valueSize1 := valueRecordSize(valueFormat1)
	valueSize2 := valueRecordSize(valueFormat2)
	recordSize := valueSize1 + valueSize2
	recordsOffset := 16
	if recordsOffset+class1Count*class2Count*recordSize > len(data) {
		return nil, fmt.Errorf("PairPos format 2 class records exceed subtable")
	}
	pairs := make([]PairValue, 0)
	for class1 := 0; class1 < class1Count; class1++ {
		leftGlyphs := leftMembers[class1]
		if len(leftGlyphs) == 0 {
			continue
		}
		for class2 := 0; class2 < class2Count; class2++ {
			offset := recordsOffset + (class1*class2Count+class2)*recordSize
			value1, err := readXAdvance(data, offset, valueFormat1)
			if err != nil {
				return nil, err
			}
			value2, err := readXAdvance(data, offset+valueSize1, valueFormat2)
			if err != nil {
				return nil, err
			}
			value := value1 + value2
			if value == 0 {
				continue
			}
			rightGlyphs := rightMembers[class2]
			for _, leftGlyph := range leftGlyphs {
				for _, rightGlyph := range rightGlyphs {
					pairs = append(pairs, PairValue{LeftGlyph: leftGlyph, RightGlyph: rightGlyph, Value: value})
					if len(pairs) > MaxClassPairExpansion {
						return nil, fmt.Errorf("PairPos format 2 class expansion exceeds max pairs %d", MaxClassPairExpansion)
					}
				}
			}
		}
	}
	return pairs, nil
}

func decodeClassDef(data []byte, offset int) (map[uint16]int, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("ClassDef offset exceeds subtable")
	}
	format := readU16(data, offset)
	classes := make(map[uint16]int)
	switch format {
	case 1:
		if offset+6 > len(data) {
			return nil, fmt.Errorf("ClassDef format 1 exceeds subtable")
		}
		startGlyphID := readU16(data, offset+2)
		glyphCount := int(readU16(data, offset+4))
		if offset+6+glyphCount*2 > len(data) {
			return nil, fmt.Errorf("ClassDef format 1 exceeds subtable")
		}
		for index := 0; index < glyphCount; index++ {
			classID := int(readU16(data, offset+6+index*2))
			classes[startGlyphID+uint16(index)] = classID
		}
	case 2:
		if offset+4 > len(data) {
			return nil, fmt.Errorf("ClassDef format 2 exceeds subtable")
		}
		classRangeCount := int(readU16(data, offset+2))
		if offset+4+classRangeCount*6 > len(data) {
			return nil, fmt.Errorf("ClassDef format 2 exceeds subtable")
		}
		for index := 0; index < classRangeCount; index++ {
			rangeOffset := offset + 4 + index*6
			startGlyphID := readU16(data, rangeOffset)
			endGlyphID := readU16(data, rangeOffset+2)
			classID := int(readU16(data, rangeOffset+4))
			if endGlyphID < startGlyphID {
				return nil, fmt.Errorf("ClassDef range end before start")
			}
			for glyphID := startGlyphID; glyphID <= endGlyphID; glyphID++ {
				classes[glyphID] = classID
				if len(classes) > MaxClassDefGlyphs {
					return nil, fmt.Errorf("ClassDef exceeds max glyphs %d", MaxClassDefGlyphs)
				}
				if glyphID == ^uint16(0) {
					break
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported ClassDef format %d", format)
	}
	return classes, nil
}

func classMembersFromCoverage(coverage []uint16, classes map[uint16]int, classCount int, glyphOrder []string) (map[int][]string, error) {
	members := make(map[int][]string, classCount)
	for _, glyphID := range coverage {
		glyph, ok := glyphName(glyphOrder, glyphID)
		if !ok {
			return nil, fmt.Errorf("coverage glyph id %d outside glyph order", glyphID)
		}
		classID := classes[glyphID]
		if classID >= 0 && classID < classCount {
			members[classID] = append(members[classID], glyph)
		}
	}
	return members, nil
}

func classMembersFromGlyphOrder(classes map[uint16]int, classCount int, glyphOrder []string) map[int][]string {
	members := make(map[int][]string, classCount)
	for glyphID, glyph := range glyphOrder {
		if glyph == ".notdef" {
			continue
		}
		classID := classes[uint16(glyphID)]
		if classID >= 0 && classID < classCount {
			members[classID] = append(members[classID], glyph)
		}
	}
	return members
}

// DecodePairPosFormat1 decodes a PairPos format 1 glyph pair kerning subtable.
func DecodePairPosFormat1(data []byte, glyphOrder []string) ([]PairValue, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("PairPos format 1 too short: %d bytes", len(data))
	}
	format := readU16(data, 0)
	if format != 1 {
		return nil, fmt.Errorf("unsupported PairPos format %d", format)
	}
	coverageOffset := int(readU16(data, 2))
	valueFormat1 := readU16(data, 4)
	valueFormat2 := readU16(data, 6)
	pairSetCount := int(readU16(data, 8))
	if len(data) < 10+pairSetCount*2 {
		return nil, fmt.Errorf("PairPos pair set offsets exceed subtable")
	}

	coverage, err := decodeCoverage(data, coverageOffset)
	if err != nil {
		return nil, err
	}
	if len(coverage) < pairSetCount {
		return nil, fmt.Errorf("coverage glyph count %d less than PairSetCount %d", len(coverage), pairSetCount)
	}

	valueSize1 := valueRecordSize(valueFormat1)
	valueSize2 := valueRecordSize(valueFormat2)
	recordSize := 2 + valueSize1 + valueSize2
	pairs := make([]PairValue, 0)
	for index := 0; index < pairSetCount; index++ {
		leftGlyphID := coverage[index]
		leftGlyph, ok := glyphName(glyphOrder, leftGlyphID)
		if !ok {
			return nil, fmt.Errorf("left glyph id %d outside glyph order", leftGlyphID)
		}
		pairSetOffset := int(readU16(data, 10+index*2))
		if pairSetOffset+2 > len(data) {
			return nil, fmt.Errorf("PairSet %d offset exceeds subtable", index)
		}
		pairValueCount := int(readU16(data, pairSetOffset))
		recordOffset := pairSetOffset + 2
		if recordOffset+pairValueCount*recordSize > len(data) {
			return nil, fmt.Errorf("PairSet %d records exceed subtable", index)
		}
		for recordIndex := 0; recordIndex < pairValueCount; recordIndex++ {
			offset := recordOffset + recordIndex*recordSize
			rightGlyphID := readU16(data, offset)
			rightGlyph, ok := glyphName(glyphOrder, rightGlyphID)
			if !ok {
				return nil, fmt.Errorf("right glyph id %d outside glyph order", rightGlyphID)
			}
			value1, err := readXAdvance(data, offset+2, valueFormat1)
			if err != nil {
				return nil, err
			}
			value2, err := readXAdvance(data, offset+2+valueSize1, valueFormat2)
			if err != nil {
				return nil, err
			}
			value := value1 + value2
			if value == 0 {
				continue
			}
			pairs = append(pairs, PairValue{LeftGlyph: leftGlyph, RightGlyph: rightGlyph, Value: value})
		}
	}
	return pairs, nil
}

// decodeCoverage decodes a Coverage table at the given offset.
// Delegates to the canonical opentype.DecodeCoverage.
func decodeCoverage(data []byte, offset int) ([]uint16, error) {
	return opentype.DecodeCoverage(data, offset)
}

func readXAdvance(data []byte, offset int, valueFormat uint16) (int, error) {
	if offset+valueRecordSize(valueFormat) > len(data) {
		return 0, fmt.Errorf("ValueRecord exceeds subtable")
	}
	current := offset
	for bit := uint16(1); bit != 0; bit <<= 1 {
		if valueFormat&bit == 0 {
			continue
		}
		if bit == xAdvanceFlag {
			return int(int16(readU16(data, current))), nil
		}
		current += 2
	}
	return 0, nil
}

func valueRecordSize(valueFormat uint16) int {
	return bits.OnesCount16(valueFormat) * 2
}

func glyphName(glyphOrder []string, glyphID uint16) (string, bool) {
	index := int(glyphID)
	if index < 0 || index >= len(glyphOrder) {
		return "", false
	}
	return glyphOrder[index], true
}

func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}
