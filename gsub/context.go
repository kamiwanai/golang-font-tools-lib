// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gsub

import (
	"fmt"
)

// NestedLookupRef references a GSUB lookup to apply at a specific position.
type NestedLookupRef struct {
	SequenceIndex  uint16
	LookupListIndex uint16
}

// ContextRule represents one expanded context substitution rule.
type ContextRule struct {
	Input         []uint16
	NestedLookups []NestedLookupRef
}

// ContextSubstLookup represents a decoded ContextSubst or ChainContextSubst lookup.
type ContextSubstLookup struct {
	LookupIndex int
	LookupType  uint16
	Format      int
	Rules       []ContextRule
	Backtrack   [][]uint16
	Lookahead   [][]uint16
}

// DecodeContextSubstLookups decodes all ContextSubst (type 5) lookups.
func DecodeContextSubstLookups(data []byte, glyphOrder []string) ([]ContextSubstLookup, error) {
	return decodeContextLookups(data, glyphOrder, 5)
}

// DecodeChainContextSubstLookups decodes all ChainContextSubst (type 6) lookups.
func DecodeChainContextSubstLookups(data []byte, glyphOrder []string) ([]ContextSubstLookup, error) {
	return decodeContextLookups(data, glyphOrder, 6)
}

func decodeContextLookups(data []byte, glyphOrder []string, wantType uint16) ([]ContextSubstLookup, error) {
	lookups, err := decodeLookups(data, wantType)
	if err != nil {
		return nil, err
	}
	var result []ContextSubstLookup
	for _, lk := range lookups {
		cl, err := decodeContextLookup(data, lk, glyphOrder, wantType)
		if err != nil {
			return nil, err
		}
		if cl != nil {
			result = append(result, *cl)
		}
	}
	return result, nil
}

// decodeContextLookup decodes one context lookup.
func decodeContextLookup(data []byte, lk lookupInfo, glyphOrder []string, wantType uint16) (*ContextSubstLookup, error) {
	subtableCount := int(readU16(data, lk.offset+4))
	if lk.offset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB context lookup %d subtable offsets exceed table", lk.lookupIndex)
	}
	var allRules []ContextRule
	var allBacktrack [][]uint16
	var allLookahead [][]uint16
	format := 0
	for i := 0; i < subtableCount; i++ {
		subOffset := lk.offset + int(readU16(data, lk.offset+6+i*2))
		if lk.lookupType == 9 {
			subOffset, _ = decodeExtensionSubtable(data, lk, subOffset, wantType)
			if subOffset == 0 {
				continue
			}
		}
		if subOffset+2 > len(data) {
			return nil, fmt.Errorf("GSUB context lookup %d subtable %d exceeds table", lk.lookupIndex, i)
		}
		subFmt := int(readU16(data, subOffset))
		format = subFmt
		var rules []ContextRule
		var backtrack, lookahead [][]uint16
		var err error
		switch subFmt {
		case 1:
			rules, backtrack, lookahead, err = decodeContextFormat1(data, subOffset, glyphOrder, wantType)
		case 2:
			rules, backtrack, lookahead, err = decodeContextFormat2(data, subOffset, glyphOrder, wantType)
		case 3:
			rules, backtrack, lookahead, err = decodeContextFormat3(data, subOffset, glyphOrder, wantType)
		default:
			return nil, fmt.Errorf("GSUB context lookup %d: unsupported format %d", lk.lookupIndex, subFmt)
		}
		if err != nil {
			return nil, err
		}
		allRules = append(allRules, rules...)
		allBacktrack = append(allBacktrack, backtrack...)
		allLookahead = append(allLookahead, lookahead...)
	}
	if len(allRules) == 0 {
		return nil, nil
	}
	return &ContextSubstLookup{
		LookupIndex: lk.lookupIndex,
		LookupType:  wantType,
		Format:      format,
		Rules:       allRules,
		Backtrack:   allBacktrack,
		Lookahead:   allLookahead,
	}, nil
}

// decodeExtensionSubtable handles Extension (type 9) indirection.
func decodeExtensionSubtable(data []byte, lk lookupInfo, subOffset int, wantType uint16) (int, error) {
	if subOffset+8 > len(data) {
		return 0, fmt.Errorf("GSUB extension subtable exceeds table")
	}
	extFormat := readU16(data, subOffset)
	if extFormat != 1 {
		return 0, nil
	}
	extType := readU16(data, subOffset+2)
	if extType != wantType {
		return 0, nil
	}
	extOffset := int(readU32(data, subOffset+4))
	target := subOffset + extOffset
	if target <= subOffset || target > len(data) {
		return 0, fmt.Errorf("GSUB extension target exceeds table")
	}
	return target, nil
}

// ── Format 1: Simple Glyph Context ──────────────────────────────────

func decodeContextFormat1(data []byte, offset int, glyphOrder []string, wantType uint16) (rules []ContextRule, backtrack, lookahead [][]uint16, err error) {
	pos := offset + 2

	if wantType == 6 {
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context format 1 too short")
		}
		backtrackCount := int(readU16(data, pos))
		pos += 2
		if backtrackCount > 0 {
			if pos+backtrackCount*2 > len(data) {
				return nil, nil, nil, fmt.Errorf("context format 1 backtrack exceed table")
			}
			backtrack = make([][]uint16, backtrackCount)
			for i := 0; i < backtrackCount; i++ {
				covOff := int(readU16(data, pos))
				pos += 2
				bt, err := decodeCoverage(data, offset+covOff)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("context fmt1 backtrack cov %d: %w", i, err)
				}
				backtrack[i] = bt
			}
		}
	}
	if pos+2 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt1 too short for coverage")
	}
	covOffset := int(readU16(data, pos))
	pos += 2
	coverage, err := decodeCoverage(data, offset+covOffset)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("context fmt1 coverage: %w", err)
	}
	if pos+2 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt1 too short for rule set count")
	}
	ruleSetCount := int(readU16(data, pos))
	pos += 2
	if pos+ruleSetCount*2 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt1 rule set offsets exceed table")
	}
	ruleSetStart := pos
	if wantType == 6 {
		pos += ruleSetCount * 2
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt1 too short for lookahead")
		}
		lookaheadCount := int(readU16(data, pos))
		pos += 2
		if lookaheadCount > 0 {
			if pos+lookaheadCount*2 > len(data) {
				return nil, nil, nil, fmt.Errorf("context fmt1 lookahead exceed table")
			}
			lookahead = make([][]uint16, lookaheadCount)
			for i := 0; i < lookaheadCount; i++ {
				laOff := int(readU16(data, pos))
				pos += 2
				la, err := decodeCoverage(data, offset+laOff)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("context fmt1 lookahead cov %d: %w", i, err)
				}
				lookahead[i] = la
			}
		}
	}
	pos = ruleSetStart
	ruleSetOffsets := make([]int, ruleSetCount)
	for i := 0; i < ruleSetCount; i++ {
		ruleSetOffsets[i] = int(readU16(data, pos))
		pos += 2
	}
	for i, firstGlyphID := range coverage {
		if i >= ruleSetCount {
			break
		}
		rsOff := ruleSetOffsets[i]
		rsPos := offset + rsOff
		if rsPos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt1 SubRuleSet %d exceeds table", i)
		}
		subRuleCount := int(readU16(data, rsPos))
		rsPos += 2
		if rsPos+subRuleCount*2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt1 SubRuleSet %d offsets exceed table", i)
		}
		var subRuleOffsets []int
		for j := 0; j < subRuleCount; j++ {
			subRuleOffsets = append(subRuleOffsets, int(readU16(data, rsPos)))
			rsPos += 2
		}
		for _, srOff := range subRuleOffsets {
			rule, err := decodeSubRule(data, offset+rsOff+srOff, firstGlyphID)
			if err != nil {
				return nil, nil, nil, err
			}
			rules = append(rules, rule)
		}
	}
	return rules, backtrack, lookahead, nil
}

func decodeSubRule(data []byte, offset int, firstGlyphID uint16) (ContextRule, error) {
	if offset+4 > len(data) {
		return ContextRule{}, fmt.Errorf("SubRule too short")
	}
	glyphCount := int(readU16(data, offset))
	substCount := int(readU16(data, offset+2))
	if glyphCount < 1 {
		return ContextRule{}, fmt.Errorf("SubRule glyphCount %d < 1", glyphCount)
	}
	inputLen := glyphCount - 1
	recOff := offset + 4
	if recOff+inputLen*2+substCount*4 > len(data) {
		return ContextRule{}, fmt.Errorf("SubRule exceeds table")
	}
	input := make([]uint16, glyphCount)
	input[0] = firstGlyphID
	for j := 0; j < inputLen; j++ {
		input[j+1] = readU16(data, recOff+j*2)
	}
	recOff += inputLen * 2
	nested := make([]NestedLookupRef, substCount)
	for j := 0; j < substCount; j++ {
		nested[j] = NestedLookupRef{
			SequenceIndex:  readU16(data, recOff+j*4),
			LookupListIndex: readU16(data, recOff+j*4+2),
		}
	}
	return ContextRule{Input: input, NestedLookups: nested}, nil
}

// ── Format 2: Class-based Context ───────────────────────────────────

func decodeContextFormat2(data []byte, offset int, glyphOrder []string, wantType uint16) (rules []ContextRule, backtrack, lookahead [][]uint16, err error) {
	pos := offset + 2

	if wantType == 6 {
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt2 too short")
		}
		backtrackCount := int(readU16(data, pos))
		pos += 2
		if backtrackCount > 0 {
			if pos+backtrackCount*2 > len(data) {
				return nil, nil, nil, fmt.Errorf("context fmt2 backtrack exceed table")
			}
			backtrack = make([][]uint16, backtrackCount)
			for i := 0; i < backtrackCount; i++ {
				covOff := int(readU16(data, pos))
				pos += 2
				bt, err := decodeCoverage(data, offset+covOff)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("context fmt2 backtrack cov %d: %w", i, err)
				}
				backtrack[i] = bt
			}
		}
	}
	if pos+2 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt2 too short for coverage")
	}
	covOffset := int(readU16(data, pos))
	pos += 2
	coverage, err := decodeCoverage(data, offset+covOffset)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("context fmt2 coverage: %w", err)
	}
	if pos+2 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt2 too short for class def")
	}
	classDefOffset := int(readU16(data, pos))
	pos += 2
	classDefMap, err := decodeClassDefMap(data, offset+classDefOffset)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("context fmt2 class def: %w", err)
	}
	if pos+2 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt2 too short for class rule set count")
	}
	ruleSetCount := int(readU16(data, pos))
	pos += 2

	ruleSetStart := pos
	if wantType == 6 {
		pos += ruleSetCount * 2
		// InputClassDef
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt2 chain too short for input class def")
		}
		inputCDOff := int(readU16(data, pos))
		pos += 2
		if inputCDOff != 0 {
			_, err = decodeClassDefMap(data, offset+inputCDOff)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("context fmt2 chain input class def: %w", err)
			}
		}
		// Lookahead coverages
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt2 chain too short for lookahead count")
		}
		lookaheadCount := int(readU16(data, pos))
		pos += 2
		if lookaheadCount > 0 {
			if pos+lookaheadCount*2 > len(data) {
				return nil, nil, nil, fmt.Errorf("context fmt2 chain lookahead exceed table")
			}
			lookahead = make([][]uint16, lookaheadCount)
			for i := 0; i < lookaheadCount; i++ {
				laOff := int(readU16(data, pos))
				pos += 2
				la, err := decodeCoverage(data, offset+laOff)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("context fmt2 chain lookahead cov %d: %w", i, err)
				}
				lookahead[i] = la
			}
		}
		// LookaheadClassDef
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt2 chain too short for lookahead class def")
		}
		lcdOff := int(readU16(data, pos))
		pos += 2
		if lcdOff != 0 {
			_, err = decodeClassDefMap(data, offset+lcdOff)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("context fmt2 chain lookahead class def: %w", err)
			}
		}
	}
	pos = ruleSetStart

	if pos+ruleSetCount*2 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt2 class rule set offsets exceed table")
	}
	ruleSetOffsets := make([]int, ruleSetCount)
	for i := 0; i < ruleSetCount; i++ {
		ruleSetOffsets[i] = int(readU16(data, pos))
		pos += 2
	}

	for i, firstGlyphID := range coverage {
		if i >= ruleSetCount {
			break
		}
		rsOff := ruleSetOffsets[i]
		rsPos := offset + rsOff
		if rsPos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt2 SubClassRuleSet %d exceeds table", i)
		}
		subRuleCount := int(readU16(data, rsPos))
		rsPos += 2
		if rsPos+subRuleCount*2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt2 SubClassRuleSet %d offsets exceed table", i)
		}
		var subRuleOffsets []int
		for j := 0; j < subRuleCount; j++ {
			subRuleOffsets = append(subRuleOffsets, int(readU16(data, rsPos)))
			rsPos += 2
		}
		for _, srOff := range subRuleOffsets {
			rc, err := decodeSubClassRule(data, offset+rsOff+srOff, firstGlyphID, classDefMap, wantType)
			if err != nil {
				return nil, nil, nil, err
			}
			rules = append(rules, rc...)
		}
	}
	return rules, backtrack, lookahead, nil
}

func decodeSubClassRule(data []byte, offset int, firstGlyphID uint16, classDef map[uint16]int, wantType uint16) ([]ContextRule, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("SubClassRule too short")
	}
	glyphCount := int(readU16(data, offset))
	substCount := int(readU16(data, offset+2))
	if glyphCount < 1 {
		return nil, fmt.Errorf("SubClassRule glyphCount %d < 1", glyphCount)
	}
	inputLen := glyphCount - 1
	recOff := offset + 4
	chainSkipLen := 0
	if wantType == 6 {
		chainSkipLen = glyphCount * 2 // Backtrack[glyphCount] + Lookahead[glyphCount]
	}
	if recOff+inputLen*2+chainSkipLen*2+substCount*4 > len(data) {
		return nil, fmt.Errorf("SubClassRule exceeds table")
	}
	inputClasses := make([]int, glyphCount)
	inputClasses[0] = classDef[firstGlyphID]
	for j := 0; j < inputLen; j++ {
		inputClasses[j+1] = int(readU16(data, recOff+j*2))
	}
	recOff += inputLen * 2
	recOff += chainSkipLen * 2
	nested := make([]NestedLookupRef, substCount)
	for j := 0; j < substCount; j++ {
		nested[j] = NestedLookupRef{
			SequenceIndex:  readU16(data, recOff+j*4),
			LookupListIndex: readU16(data, recOff+j*4+2),
		}
	}
	classToGlyphs := make(map[int][]uint16)
	for gid, class := range classDef {
		classToGlyphs[class] = append(classToGlyphs[class], gid)
	}
	var rules []ContextRule
	expandContextClasses(classToGlyphs, inputClasses, 0, make([]uint16, len(inputClasses)), nested, &rules)
	return rules, nil
}

func expandContextClasses(classToGlyphs map[int][]uint16, classes []int, depth int, current []uint16, nested []NestedLookupRef, rules *[]ContextRule) {
	if depth >= len(classes) {
		input := make([]uint16, len(current))
		copy(input, current)
		n := make([]NestedLookupRef, len(nested))
		copy(n, nested)
		*rules = append(*rules, ContextRule{Input: input, NestedLookups: n})
		return
	}
	glyphs, ok := classToGlyphs[classes[depth]]
	if !ok || len(glyphs) == 0 {
		return
	}
	for _, gid := range glyphs {
		current[depth] = gid
		expandContextClasses(classToGlyphs, classes, depth+1, current, nested, rules)
		if len(*rules) > MaxContextRuleExpansion {
			return
		}
	}
}

// decodeClassDefMap is the gsub-internal ClassDef decoder.
func decodeClassDefMap(data []byte, offset int) (map[uint16]int, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("ClassDef offset exceeds table")
	}
	format := readU16(data, offset)
	switch format {
	case 1:
		if offset+6 > len(data) {
			return nil, fmt.Errorf("ClassDef fmt1 exceeds table")
		}
		startGlyphID := readU16(data, offset+2)
		glyphCount := int(readU16(data, offset+4))
		if offset+6+glyphCount*2 > len(data) {
			return nil, fmt.Errorf("ClassDef fmt1 exceeds table")
		}
		classes := make(map[uint16]int)
		for i := 0; i < glyphCount; i++ {
			classID := int(readU16(data, offset+6+i*2))
			classes[startGlyphID+uint16(i)] = classID
		}
		return classes, nil
	case 2:
		if offset+4 > len(data) {
			return nil, fmt.Errorf("ClassDef fmt2 exceeds table")
		}
		rangeCount := int(readU16(data, offset+2))
		if offset+4+rangeCount*6 > len(data) {
			return nil, fmt.Errorf("ClassDef fmt2 exceeds table")
		}
		classes := make(map[uint16]int)
		for i := 0; i < rangeCount; i++ {
			recOff := offset + 4 + i*6
			start := readU16(data, recOff)
			end := readU16(data, recOff+2)
			classID := int(readU16(data, recOff+4))
			if end < start {
				return nil, fmt.Errorf("ClassDef range end before start")
			}
			for gid := start; gid <= end; gid++ {
				classes[gid] = classID
				if len(classes) > MaxClassDefGlyphs {
					return nil, fmt.Errorf("ClassDef exceeds max glyphs %d", MaxClassDefGlyphs)
				}
				if gid == ^uint16(0) {
					break
				}
			}
		}
		return classes, nil
	default:
		return nil, fmt.Errorf("unsupported ClassDef format %d", format)
	}
}

// ── Format 3: Coverage-based Context ────────────────────────────────

func decodeContextFormat3(data []byte, offset int, glyphOrder []string, wantType uint16) (rules []ContextRule, backtrack, lookahead [][]uint16, err error) {
	pos := offset + 2

	if wantType == 6 {
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt3 too short")
		}
		backtrackCount := int(readU16(data, pos))
		pos += 2
		if backtrackCount > 0 {
			if pos+backtrackCount*2 > len(data) {
				return nil, nil, nil, fmt.Errorf("context fmt3 backtrack exceed table")
			}
			backtrack = make([][]uint16, backtrackCount)
			for i := 0; i < backtrackCount; i++ {
				covOff := int(readU16(data, pos))
				pos += 2
				bt, err := decodeCoverage(data, offset+covOff)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("context fmt3 backtrack cov %d: %w", i, err)
				}
				backtrack[i] = bt
			}
		}
	}

	glyphCount := int(readU16(data, pos))
	pos += 2
	if glyphCount < 1 {
		return nil, nil, nil, fmt.Errorf("context fmt3 glyph count %d < 1", glyphCount)
	}

	substCount := int(readU16(data, pos))
	pos += 2

	if pos+glyphCount*2+substCount*4 > len(data) {
		return nil, nil, nil, fmt.Errorf("context fmt3 data exceeds table")
	}

	coverages := make([][]uint16, glyphCount)
	for i := 0; i < glyphCount; i++ {
		covOff := int(readU16(data, pos))
		pos += 2
		cov, err := decodeCoverage(data, offset+covOff)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("context fmt3 coverage %d: %w", i, err)
		}
		coverages[i] = cov
	}

	nested := make([]NestedLookupRef, substCount)
	for i := 0; i < substCount; i++ {
		nested[i] = NestedLookupRef{
			SequenceIndex:  readU16(data, pos),
			LookupListIndex: readU16(data, pos+2),
		}
		pos += 4
	}

	if wantType == 6 {
		if pos+2 > len(data) {
			return nil, nil, nil, fmt.Errorf("context fmt3 too short for lookahead")
		}
		lookaheadCount := int(readU16(data, pos))
		pos += 2
		if lookaheadCount > 0 {
			if pos+lookaheadCount*2 > len(data) {
				return nil, nil, nil, fmt.Errorf("context fmt3 lookahead exceed table")
			}
			lookahead = make([][]uint16, lookaheadCount)
			for i := 0; i < lookaheadCount; i++ {
				laOff := int(readU16(data, pos))
				pos += 2
				la, err := decodeCoverage(data, offset+laOff)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("context fmt3 lookahead cov %d: %w", i, err)
				}
				lookahead[i] = la
			}
		}
	}

	for _, firstGlyphID := range coverages[0] {
		input := make([]uint16, glyphCount)
		input[0] = firstGlyphID
		rules = append(rules, ContextRule{Input: input, NestedLookups: nested})
	}
	return rules, backtrack, lookahead, nil
}
