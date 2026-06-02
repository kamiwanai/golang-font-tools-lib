// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import "fmt"

// ContextPosLookup represents a decoded ContextPos (type 7) or ChainContextPos (type 8) lookup.
type ContextPosLookup struct {
	LookupIndex int
	LookupType  uint16
	Format      int
	Rules       []ContextRule
	Backtrack   [][]uint16
	Lookahead   [][]uint16
}

// NestedLookupRef references a GPOS lookup to apply at a specific position.
type NestedLookupRef struct {
	SequenceIndex   uint16
	LookupListIndex uint16
}

// ContextRule represents one expanded context positioning rule.
type ContextRule struct {
	Input         []uint16
	NestedLookups []NestedLookupRef
}

// DecodeContextPosLookups decodes all ContextPos (type 7) lookups.
func DecodeContextPosLookups(data []byte, glyphOrder []string) ([]ContextPosLookup, error) {
	return decodeContextPosLookups(data, glyphOrder, 7)
}

// DecodeChainContextPosLookups decodes all ChainContextPos (type 8) lookups.
func DecodeChainContextPosLookups(data []byte, glyphOrder []string) ([]ContextPosLookup, error) {
	return decodeContextPosLookups(data, glyphOrder, 8)
}

func decodeContextPosLookups(data []byte, glyphOrder []string, wantType uint16) ([]ContextPosLookup, error) {
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
	var result []ContextPosLookup
	for i := 0; i < lookupCount; i++ {
		lo := lookupListOffset + int(readU16(data, lookupListOffset+2+i*2))
		if lo+6 > len(data) {
			return nil, fmt.Errorf("GPOS lookup %d exceeds table", i)
		}
		lt := readU16(data, lo)
		if lt != wantType && lt != 9 {
			continue
		}
		cl, err := decodeOneContextPosLookup(data, lo, i, glyphOrder, wantType)
		if err != nil {
			return nil, err
		}
		if cl != nil {
			result = append(result, *cl)
		}
	}
	return result, nil
}

func decodeOneContextPosLookup(data []byte, lookupOffset int, lookupIndex int, glyphOrder []string, wantType uint16) (*ContextPosLookup, error) {
	lt := readU16(data, lookupOffset)
	sc := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+sc*2 > len(data) {
		return nil, fmt.Errorf("GPOS context lookup %d subtable offsets exceed table", lookupIndex)
	}
	var allRules []ContextRule
	var allBacktrack, allLookahead [][]uint16
	format := 0
	for i := 0; i < sc; i++ {
		so := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		if lt == 9 {
			so = decodeCPExtension(data, so, wantType)
			if so == 0 {
				continue
			}
		}
		if so+2 > len(data) {
			return nil, fmt.Errorf("GPOS context lookup %d sub %d exceeds table", lookupIndex, i)
		}
		sf := int(readU16(data, so))
		format = sf
		var rules []ContextRule
		var bt, la [][]uint16
		var err error
		switch sf {
		case 1:
			rules, bt, la, err = decodeCPFormat1(data, so, glyphOrder, wantType)
		case 2:
			rules, bt, la, err = decodeCPFormat2(data, so, glyphOrder, wantType)
		case 3:
			rules, bt, la, err = decodeCPFormat3(data, so, glyphOrder, wantType)
		default:
			return nil, fmt.Errorf("GPOS context lookup %d: unsupported format %d", lookupIndex, sf)
		}
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, rules...)
		allBacktrack = append(allBacktrack, bt...)
		allLookahead = append(allLookahead, la...)
	}
	if len(allRules) == 0 {
		return nil, nil
	}
	return &ContextPosLookup{LookupIndex: lookupIndex, LookupType: wantType, Format: format, Rules: allRules, Backtrack: allBacktrack, Lookahead: allLookahead}, nil
}

func decodeCPExtension(data []byte, so int, wantType uint16) int {
	if so+8 > len(data) {
		return 0
	}
	if readU16(data, so) != 1 {
		return 0
	}
	if readU16(data, so+2) != wantType {
		return 0
	}
	target := so + int(readU32(data, so+4))
	if target <= so || target > len(data) {
		return 0
	}
	return target
}
