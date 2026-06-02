// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import (
	"fmt"
)

// ClassPair is a class-level kerning rule from PairPos format 2.
// It represents one class×class cell without expanding to individual glyphs.
type ClassPair struct {
	// LookupIndex is the zero-based GPOS lookup index.
	LookupIndex int
	// Class1 is the left class ID.
	Class1 int
	// Class2 is the right class ID.
	Class2 int
	// Value is the xAdvance adjustment in font units.
	Value int
	// LeftGlyphCount is the number of glyphs in class1.
	LeftGlyphCount int
	// RightGlyphCount is the number of glyphs in class2.
	RightGlyphCount int
}

// MaxClassSummaryPairs limits the number of class×class cells stored in
// the class-level summary. This is intentionally small — each cell is just
// a few ints, so 100k cells is ~few MB.
var MaxClassSummaryPairs int = 100_000

// MaxExpandPairs limits the number of individual glyph pairs returned
// by ExpandClassPair. Prevents combinatorial explosion on large classes.
var MaxExpandPairs int = 50_000

// DecodeClassPairLookups decodes PairPos format 2 lookups from a GPOS table
// at the class level — one ClassPair per class1×class2 cell that has a
// non-zero value. No individual glyph expansion happens here.
func DecodeClassPairLookups(data []byte, glyphOrder []string) ([]ClassPair, error) {
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

	var pairs []ClassPair
	for lookupIndex := 0; lookupIndex < lookupCount; lookupIndex++ {
		lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))
		lp, err := decodeClassLookup(data, glyphOrder, lookupOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, lp...)
		if len(pairs) > MaxClassSummaryPairs {
			return pairs[:MaxClassSummaryPairs], nil
		}
	}
	return pairs, nil
}

func decodeClassLookup(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int) ([]ClassPair, error) {
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

	var pairs []ClassPair
	for subtableIndex := 0; subtableIndex < subtableCount; subtableIndex++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+subtableIndex*2))
		sp, err := decodeClassSubtable(data, glyphOrder, lookupType, lookupOffset, lookupIndex, subtableOffset, subtableIndex)
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, sp...)
	}
	return pairs, nil
}

func decodeClassSubtable(data []byte, glyphOrder []string, lookupType uint16, lookupOffset int, lookupIndex int, subtableOffset int, subtableIndex int) ([]ClassPair, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable %d exceeds table", lookupIndex, subtableIndex)
	}
	if lookupType == 9 {
		return decodeClassExtensionSubtable(data, glyphOrder, lookupOffset, lookupIndex, subtableOffset, subtableIndex)
	}
	format := int(readU16(data, subtableOffset))
	if format != 2 {
		return nil, nil
	}
	pairs, err := decodeClassPairPosFormat2(data[subtableOffset:], glyphOrder, lookupIndex)
	if err != nil {
		return nil, err
	}
	return pairs, nil
}

func decodeClassExtensionSubtable(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int, subtableOffset int, subtableIndex int) ([]ClassPair, error) {
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
	if targetFormat != 2 {
		return nil, nil
	}
	pairs, err := decodeClassPairPosFormat2(data[targetOffset:], glyphOrder, lookupIndex)
	if err != nil {
		return nil, err
	}
	return pairs, nil
}

// decodeClassPairPosFormat2 decodes a PairPos format 2 subtable and returns
// class-level pairs without expanding to individual glyphs.
func decodeClassPairPosFormat2(subtable []byte, glyphOrder []string, lookupIndex int) ([]ClassPair, error) {
	if len(subtable) < 16 {
		return nil, fmt.Errorf("PairPos format 2 too short: %d bytes", len(subtable))
	}
	format := readU16(subtable, 0)
	if format != 2 {
		return nil, fmt.Errorf("unsupported PairPos format %d", format)
	}
	coverageOffset := int(readU16(subtable, 2))
	valueFormat1 := readU16(subtable, 4)
	valueFormat2 := readU16(subtable, 6)
	classDef1Offset := int(readU16(subtable, 8))
	classDef2Offset := int(readU16(subtable, 10))
	class1Count := int(readU16(subtable, 12))
	class2Count := int(readU16(subtable, 14))

	// We need coverage and class defs to compute per-class glyph counts.
	coverage, err := decodeCoverage(subtable, coverageOffset)
	if err != nil {
		return nil, err
	}
	classDef1, err := decodeClassDef(subtable, classDef1Offset)
	if err != nil {
		return nil, err
	}
	classDef2, err := decodeClassDef(subtable, classDef2Offset)
	if err != nil {
		return nil, err
	}

	// Count glyphs per class for class1 (from coverage) and class2 (from all glyphs).
	leftCounts := countClassMembersFromCoverage(coverage, classDef1, class1Count)
	rightCounts := countClassMembersFromGlyphOrder(classDef2, class2Count, len(glyphOrder))

	valueSize1 := valueRecordSize(valueFormat1)
	valueSize2 := valueRecordSize(valueFormat2)
	recordSize := valueSize1 + valueSize2
	recordsOffset := 16
	if recordsOffset+class1Count*class2Count*recordSize > len(subtable) {
		return nil, fmt.Errorf("PairPos format 2 class records exceed subtable")
	}

	pairs := make([]ClassPair, 0)
	for class1 := 0; class1 < class1Count; class1++ {
		leftCount := leftCounts[class1]
		if leftCount == 0 {
			continue
		}
		for class2 := 0; class2 < class2Count; class2++ {
			offset := recordsOffset + (class1*class2Count+class2)*recordSize
			value1, err := readXAdvance(subtable, offset, valueFormat1)
			if err != nil {
				return nil, err
			}
			value2, err := readXAdvance(subtable, offset+valueSize1, valueFormat2)
			if err != nil {
				return nil, err
			}
			value := value1 + value2
			if value == 0 {
				continue
			}
			pairs = append(pairs, ClassPair{
				LookupIndex:     lookupIndex,
				Class1:          class1,
				Class2:          class2,
				Value:           value,
				LeftGlyphCount:  leftCount,
				RightGlyphCount: rightCounts[class2],
			})
			if len(pairs) > MaxClassSummaryPairs {
				return pairs[:MaxClassSummaryPairs], nil
			}
		}
	}
	return pairs, nil
}

// ExpandClassPair expands a single class×class cell into individual glyph pairs.
// Returns at most MaxExpandPairs pairs.
func ExpandClassPair(data []byte, glyphOrder []string, lookupIndex int, class1 int, class2 int) ([]PairValue, error) {
	// Walk to the correct lookup and subtable to find the PairPos format 2 data.
	if len(data) < 10 {
		return nil, fmt.Errorf("GPOS header too short")
	}
	lookupListOffset := int(readU16(data, 8))
	if lookupListOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS LookupList offset exceeds table")
	}
	lookupCount := int(readU16(data, lookupListOffset))
	if lookupIndex >= lookupCount {
		return nil, fmt.Errorf("lookup index %d out of range (count %d)", lookupIndex, lookupCount)
	}
	lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))

	// Find PairPos format 2 subtable(s) in this lookup.
	subtables, err := findPairPosFormat2Subtables(data, lookupOffset)
	if err != nil {
		return nil, err
	}

	var allPairs []PairValue
	for _, subtableData := range subtables {
		pairs, err := expandClassPairFromSubtable(subtableData, glyphOrder, class1, class2)
		if err != nil {
			continue // skip broken subtable, try next
		}
		allPairs = append(allPairs, pairs...)
		if len(allPairs) > MaxExpandPairs {
			return allPairs[:MaxExpandPairs], nil
		}
	}
	return allPairs, nil
}

// findPairPosFormat2Subtables returns raw subtable bytes for all PairPos
// format 2 subtables in the given lookup.
func findPairPosFormat2Subtables(data []byte, lookupOffset int) ([][]byte, error) {
	if lookupOffset+6 > len(data) {
		return nil, fmt.Errorf("lookup exceeds table")
	}
	lookupType := readU16(data, lookupOffset)
	if lookupType != 2 && lookupType != 9 {
		return nil, nil
	}
	subtableCount := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("lookup subtable offsets exceed table")
	}

	var result [][]byte
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		if subtableOffset+2 > len(data) {
			continue
		}
		if lookupType == 9 {
			// Extension lookup
			if subtableOffset+8 > len(data) {
				continue
			}
			extFormat := readU16(data, subtableOffset)
			if extFormat != 1 {
				continue
			}
			extType := readU16(data, subtableOffset+2)
			if extType != 2 {
				continue
			}
			extOffset := int(readU32(data, subtableOffset+4))
			targetOffset := subtableOffset + extOffset
			if targetOffset <= subtableOffset || targetOffset+2 > len(data) {
				continue
			}
			result = append(result, data[targetOffset:])
		} else {
			format := int(readU16(data, subtableOffset))
			if format == 2 {
				result = append(result, data[subtableOffset:])
			}
		}
	}
	return result, nil
}

// expandClassPairFromSubtable expands one class1×class2 cell from a PairPos
// format 2 subtable into individual glyph pairs.
func expandClassPairFromSubtable(subtable []byte, glyphOrder []string, targetClass1 int, targetClass2 int) ([]PairValue, error) {
	if len(subtable) < 16 {
		return nil, fmt.Errorf("PairPos format 2 too short")
	}
	format := readU16(subtable, 0)
	if format != 2 {
		return nil, fmt.Errorf("not PairPos format 2")
	}
	coverageOffset := int(readU16(subtable, 2))
	valueFormat1 := readU16(subtable, 4)
	valueFormat2 := readU16(subtable, 6)
	classDef1Offset := int(readU16(subtable, 8))
	classDef2Offset := int(readU16(subtable, 10))
	class1Count := int(readU16(subtable, 12))
	class2Count := int(readU16(subtable, 14))

	if targetClass1 >= class1Count || targetClass2 >= class2Count {
		return nil, fmt.Errorf("class index out of range")
	}

	coverage, err := decodeCoverage(subtable, coverageOffset)
	if err != nil {
		return nil, err
	}
	classDef1, err := decodeClassDef(subtable, classDef1Offset)
	if err != nil {
		return nil, err
	}
	classDef2, err := decodeClassDef(subtable, classDef2Offset)
	if err != nil {
		return nil, err
	}

	leftMembers := classMembersFromCoverageForExpand(coverage, classDef1, targetClass1, glyphOrder)
	rightMembers := classMembersFromGlyphOrderForExpand(classDef2, targetClass2, glyphOrder)

	valueSize1 := valueRecordSize(valueFormat1)
	valueSize2 := valueRecordSize(valueFormat2)
	recordSize := valueSize1 + valueSize2
	recordsOffset := 16
	offset := recordsOffset + (targetClass1*class2Count+targetClass2)*recordSize
	if offset+recordSize > len(subtable) {
		return nil, fmt.Errorf("class record exceeds subtable")
	}
	value1, err := readXAdvance(subtable, offset, valueFormat1)
	if err != nil {
		return nil, err
	}
	value2, err := readXAdvance(subtable, offset+valueSize1, valueFormat2)
	if err != nil {
		return nil, err
	}
	value := value1 + value2

	pairs := make([]PairValue, 0, len(leftMembers)*len(rightMembers))
	for _, left := range leftMembers {
		for _, right := range rightMembers {
			pairs = append(pairs, PairValue{LeftGlyph: left, RightGlyph: right, Value: value})
			if len(pairs) > MaxExpandPairs {
				return pairs[:MaxExpandPairs], nil
			}
		}
	}
	return pairs, nil
}

// classMembersFromCoverageForExpand returns glyph names from coverage that
// belong to the target class.
func classMembersFromCoverageForExpand(coverage []uint16, classes map[uint16]int, targetClass int, glyphOrder []string) []string {
	var members []string
	for _, glyphID := range coverage {
		if classes[glyphID] != targetClass {
			continue
		}
		if name, ok := glyphName(glyphOrder, glyphID); ok {
			members = append(members, name)
		}
	}
	return members
}

// classMembersFromGlyphOrderForExpand returns glyph names from the full
// glyph order that belong to the target class (class2 — not limited to
// coverage).
func classMembersFromGlyphOrderForExpand(classes map[uint16]int, targetClass int, glyphOrder []string) []string {
	var members []string
	for glyphID, glyph := range glyphOrder {
		if glyph == ".notdef" {
			continue
		}
		if classes[uint16(glyphID)] == targetClass {
			members = append(members, glyph)
		}
	}
	return members
}

// countClassMembersFromCoverage counts how many coverage glyphs fall into
// each class (for class1).
func countClassMembersFromCoverage(coverage []uint16, classes map[uint16]int, classCount int) []int {
	counts := make([]int, classCount)
	for _, glyphID := range coverage {
		classID := classes[glyphID]
		if classID >= 0 && classID < classCount {
			counts[classID]++
		}
	}
	return counts
}

// countClassMembersFromGlyphOrder counts how many glyphs in the full order
// fall into each class (for class2).
func countClassMembersFromGlyphOrder(classes map[uint16]int, classCount int, numGlyphs int) []int {
	counts := make([]int, classCount)
	for glyphID := 0; glyphID < numGlyphs; glyphID++ {
		if glyphID == 0 {
			continue // skip .notdef
		}
		classID := classes[uint16(glyphID)]
		if classID >= 0 && classID < classCount {
			counts[classID]++
		}
	}
	return counts
}
