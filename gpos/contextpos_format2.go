// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gpos

import "fmt"

func decodeCPFormat2(data []byte, offset int, glyphOrder []string, wantType uint16) ([]ContextRule, [][]uint16, [][]uint16, error) {
	pos := offset + 2
	var backtrack, lookahead [][]uint16
	if wantType == 8 {
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 too short") }
		bc := int(readU16(data, pos)); pos += 2
		if bc > 0 {
			if pos+bc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 bt exceed") }
			backtrack = make([][]uint16, bc)
			for i := 0; i < bc; i++ {
				co := int(readU16(data, pos)); pos += 2
				bt, e := decodeCoverage(data, offset+co)
				if e != nil { return nil, nil, nil, fmt.Errorf("context fmt2 bt cov %d: %w", i, e) }
				backtrack[i] = bt
			}
		}
	}
	if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 too short for cov") }
	co := int(readU16(data, pos)); pos += 2
	cov, e := decodeCoverage(data, offset+co)
	if e != nil { return nil, nil, nil, fmt.Errorf("context fmt2 cov: %w", e) }
	if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 too short for cd") }
	cdOff := int(readU16(data, pos)); pos += 2
	cdm, e2 := decodeCPClassDef(data, offset+cdOff)
	if e2 != nil { return nil, nil, nil, fmt.Errorf("context fmt2 class def: %w", e2) }
	if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 too short for rsc") }
	rsc := int(readU16(data, pos)); pos += 2
	rsStart := pos
	if wantType == 8 {
		pos += rsc * 2
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 chain too short for icd") }
		icdOff := int(readU16(data, pos)); pos += 2
		if icdOff != 0 { if _, e3 := decodeCPClassDef(data, offset+icdOff); e3 != nil { return nil, nil, nil, e3 } }
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 chain too short for lc") }
		lc := int(readU16(data, pos)); pos += 2
		if lc > 0 {
			if pos+lc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 chain la exceed") }
			lookahead = make([][]uint16, lc)
			for i := 0; i < lc; i++ {
				laOff := int(readU16(data, pos+i*2))
				la, e4 := decodeCoverage(data, offset+laOff)
				if e4 != nil { return nil, nil, nil, fmt.Errorf("context fmt2 la cov %d: %w", i, e4) }
				lookahead[i] = la
			}
		}
		pos += lc * 2
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 chain too short for lcd") }
		lcdOff := int(readU16(data, pos)); pos += 2
		if lcdOff != 0 { if _, e5 := decodeCPClassDef(data, offset+lcdOff); e5 != nil { return nil, nil, nil, e5 } }
	}
	pos = rsStart
	if pos+rsc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt2 offsets exceed") }
	var rules []ContextRule
	for i, fgid := range cov {
		if i >= rsc { break }
		rso := int(readU16(data, pos+i*2))
		rsp := offset + rso
		if rsp+2 > len(data) { return nil, nil, nil, fmt.Errorf("fmt2 SubClassRuleSet %d exceeds", i) }
		src := int(readU16(data, rsp)); rsp += 2
		if rsp+src*2 > len(data) { return nil, nil, nil, fmt.Errorf("fmt2 SubClassRuleSet %d offsets exceed", i) }
		for j := 0; j < src; j++ {
			sro := int(readU16(data, rsp+j*2))
			rc, e6 := decodeCPSubClassRule(data, offset+rso+sro, fgid, cdm, wantType)
			if e6 != nil { return nil, nil, nil, e6 }
			rules = append(rules, rc...)
		}
	}
	return rules, backtrack, lookahead, nil
}
func decodeCPSubClassRule(data []byte, offset int, firstGlyphID uint16, classDef map[uint16]int, wantType uint16) ([]ContextRule, error) {
	if offset+4 > len(data) { return nil, fmt.Errorf("SubClassRule too short") }
	gc := int(readU16(data, offset))
	sc := int(readU16(data, offset+2))
	if gc < 1 { return nil, fmt.Errorf("SubClassRule glyphCount %d < 1", gc) }
	il := gc - 1
	ro := offset + 4
	csl := 0
	if wantType == 8 { csl = gc * 2 }
	if ro+il*2+csl*2+sc*4 > len(data) { return nil, fmt.Errorf("SubClassRule exceeds") }
	ic := make([]int, gc)
	ic[0] = classDef[firstGlyphID]
	for j := 0; j < il; j++ { ic[j+1] = int(readU16(data, ro+j*2)) }
	ro += il * 2
	ro += csl * 2
	nested := make([]NestedLookupRef, sc)
	for j := 0; j < sc; j++ {
		nested[j] = NestedLookupRef{SequenceIndex: readU16(data, ro+j*4), LookupListIndex: readU16(data, ro+j*4+2)}
	}
	ctg := make(map[int][]uint16)
	for gid, cls := range classDef { ctg[cls] = append(ctg[cls], gid) }
	var rules []ContextRule
	expandCPClasses(ctg, ic, 0, make([]uint16, len(ic)), nested, &rules)
	return rules, nil
}

func expandCPClasses(ctg map[int][]uint16, classes []int, depth int, cur []uint16, nested []NestedLookupRef, rules *[]ContextRule) {
	if depth >= len(classes) {
		in := make([]uint16, len(cur)); copy(in, cur)
		n := make([]NestedLookupRef, len(nested)); copy(n, nested)
		*rules = append(*rules, ContextRule{Input: in, NestedLookups: n})
		return
	}
	glyphs, ok := ctg[classes[depth]]
	if !ok || len(glyphs) == 0 { return }
	for _, gid := range glyphs {
		cur[depth] = gid
		expandCPClasses(ctg, classes, depth+1, cur, nested, rules)
	}
}

func decodeCPFormat3(data []byte, offset int, glyphOrder []string, wantType uint16) ([]ContextRule, [][]uint16, [][]uint16, error) {
	pos := offset + 2
	var backtrack, lookahead [][]uint16
	if wantType == 8 {
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt3 too short") }
		bc := int(readU16(data, pos)); pos += 2
		if bc > 0 {
			if pos+bc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt3 bt exceed") }
			backtrack = make([][]uint16, bc)
			for i := 0; i < bc; i++ {
				co := int(readU16(data, pos+i*2))
				bt, e := decodeCoverage(data, offset+co)
				if e != nil { return nil, nil, nil, fmt.Errorf("context fmt3 bt cov %d: %w", i, e) }
				backtrack[i] = bt
			}
			pos += bc * 2
		}
	}
	gc := int(readU16(data, pos)); pos += 2
	if gc < 1 { return nil, nil, nil, fmt.Errorf("context fmt3 gc %d < 1", gc) }
	sc := int(readU16(data, pos)); pos += 2
	if pos+gc*2+sc*4 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt3 data exceeds") }
	covs := make([][]uint16, gc)
	for i := 0; i < gc; i++ {
		co := int(readU16(data, pos+i*2))
		cov, e := decodeCoverage(data, offset+co)
		if e != nil { return nil, nil, nil, fmt.Errorf("context fmt3 cov %d: %w", i, e) }
		covs[i] = cov
	}
	pos += gc * 2
	nested := make([]NestedLookupRef, sc)
	for i := 0; i < sc; i++ {
		nested[i] = NestedLookupRef{SequenceIndex: readU16(data, pos+i*4), LookupListIndex: readU16(data, pos+i*4+2)}
	}
	pos += sc * 4
	if wantType == 8 {
		if pos+2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt3 too short for la") }
		lc := int(readU16(data, pos)); pos += 2
		if lc > 0 {
			if pos+lc*2 > len(data) { return nil, nil, nil, fmt.Errorf("context fmt3 la exceed") }
			lookahead = make([][]uint16, lc)
			for i := 0; i < lc; i++ {
				laOff := int(readU16(data, pos+i*2))
				la, e2 := decodeCoverage(data, offset+laOff)
				if e2 != nil { return nil, nil, nil, fmt.Errorf("context fmt3 la cov %d: %w", i, e2) }
				lookahead[i] = la
			}
		}
	}
	var rules []ContextRule
	for _, fgid := range covs[0] {
		in := make([]uint16, gc); in[0] = fgid
		rules = append(rules, ContextRule{Input: in, NestedLookups: nested})
	}
	return rules, backtrack, lookahead, nil
}

func decodeCPClassDef(data []byte, offset int) (map[uint16]int, error) {
	if offset+2 > len(data) { return nil, fmt.Errorf("ClassDef offset exceeds") }
	format := readU16(data, offset)
	switch format {
	case 1:
		if offset+6 > len(data) { return nil, fmt.Errorf("ClassDef fmt1 exceeds") }
		start := readU16(data, offset+2)
		gc := int(readU16(data, offset+4))
		if offset+6+gc*2 > len(data) { return nil, fmt.Errorf("ClassDef fmt1 exceeds") }
		classes := make(map[uint16]int)
		for i := 0; i < gc; i++ { classes[start+uint16(i)] = int(readU16(data, offset+6+i*2)) }
		return classes, nil
	case 2:
		if offset+4 > len(data) { return nil, fmt.Errorf("ClassDef fmt2 exceeds") }
		rc := int(readU16(data, offset+2))
		if offset+4+rc*6 > len(data) { return nil, fmt.Errorf("ClassDef fmt2 exceeds") }
		classes := make(map[uint16]int)
		for i := 0; i < rc; i++ {
			ro := offset + 4 + i*6
			start := readU16(data, ro); end := readU16(data, ro+2); classID := int(readU16(data, ro+4))
			if end < start { return nil, fmt.Errorf("ClassDef range end before start") }
			for gid := start; gid <= end; gid++ {
				classes[gid] = classID
				if len(classes) > MaxClassDefGlyphs { return nil, fmt.Errorf("ClassDef exceeds max glyphs") }
				if gid == ^uint16(0) { break }
			}
		}
		return classes, nil
	default:
		return nil, fmt.Errorf("unsupported ClassDef format %d", format)
	}
}