// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gsub

import "fmt"

// ReverseSubst represents one reverse chaining contextual single
// substitution rule (GSUB lookup type 8).
type ReverseSubst struct {
	From      string
	To        string
	Backtrack [][]uint16
	Lookahead [][]uint16
}

// DecodeReverseSubstLookups decodes all ReverseChainSingleSubst (type 8)
// lookups from a GSUB table. glyphOrder maps glyph IDs to names.
func DecodeReverseSubstLookups(data []byte, glyphOrder []string) ([]ReverseSubst, error) {
	lookups, err := decodeLookups(data, 8)
	if err != nil {
		return nil, err
	}
	var result []ReverseSubst
	for _, lk := range lookups {
		subs, err := decodeReverseSubstLookup(data, lk, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

func decodeReverseSubstLookup(data []byte, lk lookupInfo, glyphOrder []string) ([]ReverseSubst, error) {
	subtableCount := int(readU16(data, lk.offset+4))
	if lk.offset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB ReverseSubst lookup %d subtable offsets exceed table", lk.lookupIndex)
	}
	var result []ReverseSubst
	for i := 0; i < subtableCount; i++ {
		subOffset := lk.offset + int(readU16(data, lk.offset+6+i*2))
		subs, err := decodeReverseSubstFormat1(data, subOffset, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

// decodeReverseSubstFormat1 decodes a ReverseChainSingleSubstFormat1 subtable.
func decodeReverseSubstFormat1(data []byte, offset int, glyphOrder []string) ([]ReverseSubst, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("ReverseSubst format 1 too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, nil
	}
	pos := offset + 2
	if pos+2 > len(data) {
		return nil, fmt.Errorf("ReverseSubst too short for coverage")
	}
	covOffset := int(readU16(data, pos))
	pos += 2

	coverage, err := decodeCoverage(data, offset+covOffset)
	if err != nil {
		return nil, fmt.Errorf("ReverseSubst coverage: %w", err)
	}

	// Backtrack
	if pos+2 > len(data) {
		return nil, fmt.Errorf("ReverseSubst too short for backtrack count")
	}
	backtrackCount := int(readU16(data, pos))
	pos += 2
	var backtrack [][]uint16
	if backtrackCount > 0 {
		if pos+backtrackCount*2 > len(data) {
			return nil, fmt.Errorf("ReverseSubst backtrack offsets exceed table")
		}
		backtrack = make([][]uint16, backtrackCount)
		for i := 0; i < backtrackCount; i++ {
			btOff := int(readU16(data, pos))
			pos += 2
			bt, err := decodeCoverage(data, offset+btOff)
			if err != nil {
				return nil, fmt.Errorf("ReverseSubst backtrack cov %d: %w", i, err)
			}
			backtrack[i] = bt
		}
	}

	// Lookahead
	if pos+2 > len(data) {
		return nil, fmt.Errorf("ReverseSubst too short for lookahead count")
	}
	lookaheadCount := int(readU16(data, pos))
	pos += 2
	var lookahead [][]uint16
	if lookaheadCount > 0 {
		if pos+lookaheadCount*2 > len(data) {
			return nil, fmt.Errorf("ReverseSubst lookahead offsets exceed table")
		}
		lookahead = make([][]uint16, lookaheadCount)
		for i := 0; i < lookaheadCount; i++ {
			laOff := int(readU16(data, pos))
			pos += 2
			la, err := decodeCoverage(data, offset+laOff)
			if err != nil {
				return nil, fmt.Errorf("ReverseSubst lookahead cov %d: %w", i, err)
			}
			lookahead[i] = la
		}
	}

	// Substitute glyphs
	if pos+2 > len(data) {
		return nil, fmt.Errorf("ReverseSubst too short for glyph count")
	}
	glyphCount := int(readU16(data, pos))
	pos += 2
	if pos+glyphCount*2 > len(data) {
		return nil, fmt.Errorf("ReverseSubst substitute glyphs exceed table")
	}
	if glyphCount != len(coverage) {
		return nil, fmt.Errorf("ReverseSubst glyph count %d != coverage count %d", glyphCount, len(coverage))
	}

	subs := make([]ReverseSubst, 0, glyphCount)
	for i := 0; i < glyphCount; i++ {
		fromID := coverage[i]
		from, ok := glyphName(glyphOrder, fromID)
		if !ok {
			continue
		}
		toID := readU16(data, pos+i*2)
		to, ok := glyphName(glyphOrder, toID)
		if !ok {
			continue
		}
		subs = append(subs, ReverseSubst{
			From:      from,
			To:        to,
			Backtrack: backtrack,
			Lookahead: lookahead,
		})
	}
	return subs, nil
}
