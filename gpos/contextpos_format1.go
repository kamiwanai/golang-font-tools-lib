// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import "fmt"

func decodeCPFormat1(data []byte, offset int, glyphOrder []string, wantType uint16) ([]ContextRule, [][]uint16, [][]uint16, error) {
	pos := offset + 2
	var backtrack, lookahead [][]uint16
	if wantType == 8 {
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt1 too short") }
		bc := int(readU16(data, pos)); pos += 2
		if bc > 0 {
			if pos+bc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt1 bt exceed") }
			backtrack = make([][]uint16, bc)
			for i := 0; i < bc; i++ {
				co := int(readU16(data, pos)); pos += 2
				bt, e := decodeCoverage(data, offset+co)
				if e != nil { return nil, nil, nil, fmt.Errorf("context fmt1 bt cov %d: %w", i, e) }
				backtrack[i] = bt
			}
		}
	}
	if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt1 too short for cov") }
	co := int(readU16(data, pos)); pos += 2
	cov, e := decodeCoverage(data, offset+co)
	if e != nil { return nil, nil, nil, fmt.Errorf("context fmt1 cov: %w", e) }
	if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt1 too short for rsc") }
	rsc := int(readU16(data, pos)); pos += 2
	if pos+rsc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt1 offsets exceed") }
	rsStart := pos
	if wantType == 8 {
		pos += rsc * 2
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt1 too short for la") }
		lc := int(readU16(data, pos)); pos += 2
		if lc > 0 {
			if pos+lc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt1 la exceed") }
			lookahead = make([][]uint16, lc)
			for i := 0; i < lc; i++ {
				laOff := int(readU16(data, pos)); pos += 2
				la, e2 := decodeCoverage(data, offset+laOff)
				if e2 != nil { return nil, nil, nil, fmt.Errorf("context fmt1 la cov %d: %w", i, e2) }
				lookahead[i] = la
			}
		}
	}
	pos = rsStart
	var rules []ContextRule
	for i, fgid := range cov {
		if i >= rsc { break }
		rso := int(readU16(data, pos+i*2))
		rsp := offset + rso
		if rsp+2 > len(data) { return nil, nil, nil, fmt.Errorf("fmt1 SubRuleSet %d exceeds", i) }
		src := int(readU16(data, rsp)); rsp += 2
		if rsp+src*2 > len(data) { return nil, nil, nil, fmt.Errorf("fmt1 SubRuleSet %d offsets exceed", i) }
		for j := 0; j < src; j++ {
			sro := int(readU16(data, rsp+j*2))
			rule, e3 := decodeCPSubRule(data, offset+rso+sro, fgid)
			if e3 != nil { return nil, nil, nil, e3 }
			rules = append(rules, rule)
		}
	}
	return rules, backtrack, lookahead, nil
}

func decodeCPSubRule(data []byte, offset int, firstGlyphID uint16) (ContextRule, error) {
	if offset+4 > len(data) { return ContextRule{}, fmt.Errorf("SubRule too short") }
	gc := int(readU16(data, offset))
	sc := int(readU16(data, offset+2))
	if gc < 1 { return ContextRule{}, fmt.Errorf("SubRule glyphCount %d < 1", gc) }
	il := gc - 1
	ro := offset + 4
	if ro+il*2+sc*4 > len(data) { return ContextRule{}, fmt.Errorf("SubRule exceeds") }
	input := make([]uint16, gc)
	input[0] = firstGlyphID
	for j := 0; j < il; j++ { input[j+1] = readU16(data, ro+j*2) }
	ro += il * 2
	nested := make([]NestedLookupRef, sc)
	for j := 0; j < sc; j++ {
		nested[j] = NestedLookupRef{SequenceIndex: readU16(data, ro+j*4), LookupListIndex: readU16(data, ro+j*4+2)}
	}
	return ContextRule{Input: input, NestedLookups: nested}, nil
}