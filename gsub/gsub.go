// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gsub

import (
	"encoding/binary"
	"fmt"
)

// SingleSubst is one glyph substitution from a SingleSubst lookup.
type SingleSubst struct {
	// From is the original glyph name.
	From string
	// To is the substitution glyph name.
	To string
}

// LigatureSubst is one ligature substitution rule from a LigatureSubst lookup.
type LigatureSubst struct {
	// Components are the input glyph names that form the ligature,
	// in order: first glyph, then each subsequent component.
	Components []string
	// Ligature is the output glyph name.
	Ligature string
}

// MultipleSubst is one multiple substitution from a MultipleSubst lookup.
type MultipleSubst struct {
	// From is the original glyph name.
	From string
	// To is the list of replacement glyph names.
	To []string
}

// AlternateSubst is one alternate substitution from an AlternateSubst lookup.
// One input glyph maps to a list of alternative glyphs (e.g. for salt/ss01-ss20).
type AlternateSubst struct {
	// From is the original glyph name.
	From string
	// Alternates lists the alternative glyph names for this input.
	Alternates []string
}

// readU16 reads a big-endian uint16 from data at offset.
func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

// readU32 reads a big-endian uint32 from data at offset.
func readU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}

// DecodeSingleSubstLookups decodes all SingleSubst (type 1) lookups from a
// GSUB table. glyphOrder maps glyph IDs to names.
func DecodeSingleSubstLookups(data []byte, glyphOrder []string) ([]SingleSubst, error) {
	lookups, err := decodeLookups(data, 1)
	if err != nil {
		return nil, err
	}
	var result []SingleSubst
	for _, lk := range lookups {
		subs, err := decodeSingleSubstLookup(data, lk, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

// DecodeLigatureSubstLookups decodes all LigatureSubst (type 4) lookups from
// a GSUB table. glyphOrder maps glyph IDs to names.
func DecodeLigatureSubstLookups(data []byte, glyphOrder []string) ([]LigatureSubst, error) {
	lookups, err := decodeLookups(data, 4)
	if err != nil {
		return nil, err
	}
	var result []LigatureSubst
	for _, lk := range lookups {
		subs, err := decodeLigatureSubstLookup(data, lk, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

// DecodeAlternateSubstLookups decodes all AlternateSubst (type 3) lookups from
// a GSUB table. glyphOrder maps glyph IDs to names.
func DecodeAlternateSubstLookups(data []byte, glyphOrder []string) ([]AlternateSubst, error) {
	lookups, err := decodeLookups(data, 3)
	if err != nil {
		return nil, err
	}
	var result []AlternateSubst
	for _, lk := range lookups {
		subs, err := decodeAlternateSubstLookup(data, lk, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

// DecodeMultipleSubstLookups decodes all MultipleSubst (type 2) lookups from a
// GSUB table. glyphOrder maps glyph IDs to names.
func DecodeMultipleSubstLookups(data []byte, glyphOrder []string) ([]MultipleSubst, error) {
	lookups, err := decodeLookups(data, 2)
	if err != nil {
		return nil, err
	}
	var result []MultipleSubst
	for _, lk := range lookups {
		subs, err := decodeMultipleSubstLookup(data, lk, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

// lookupInfo holds the parsed offset and type for one GSUB lookup.
type lookupInfo struct {
	offset      int
	lookupType  uint16
	lookupIndex int
}

func decodeLookups(data []byte, wantType uint16) ([]lookupInfo, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("GSUB header too short: %d bytes", len(data))
	}
	lookupListOffset := int(readU16(data, 8))
	if lookupListOffset+2 > len(data) {
		return nil, fmt.Errorf("GSUB LookupList offset exceeds table")
	}
	lookupCount := int(readU16(data, lookupListOffset))
	if lookupCount > MaxLookupCount {
		return nil, fmt.Errorf("GSUB LookupCount %d exceeds limit", lookupCount)
	}
	if lookupListOffset+2+lookupCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB LookupList records exceed table")
	}
	var result []lookupInfo
	for i := 0; i < lookupCount; i++ {
		lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+i*2))
		if lookupOffset+6 > len(data) {
			return nil, fmt.Errorf("GSUB lookup %d exceeds table", i)
		}
		lookupType := readU16(data, lookupOffset)
		if lookupType == wantType || lookupType == 9 {
			result = append(result, lookupInfo{
				offset:      lookupOffset,
				lookupType:  lookupType,
				lookupIndex: i,
			})
		}
	}
	return result, nil
}

func decodeSingleSubstLookup(data []byte, lk lookupInfo, glyphOrder []string) ([]SingleSubst, error) {
	subtableCount := int(readU16(data, lk.offset+4))
	if lk.offset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB SingleSubst lookup %d subtable offsets exceed table", lk.lookupIndex)
	}
	var result []SingleSubst
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lk.offset + int(readU16(data, lk.offset+6+i*2))
		subs, err := decodeSingleSubstSubtable(data, lk, subtableOffset, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

func decodeSingleSubstSubtable(data []byte, lk lookupInfo, subtableOffset int, glyphOrder []string) ([]SingleSubst, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GSUB SingleSubst subtable exceeds table")
	}
	// Handle extension lookup type 9.
	if lk.lookupType == 9 {
		if subtableOffset+8 > len(data) {
			return nil, fmt.Errorf("GSUB SingleSubst extension subtable exceeds table")
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
			return nil, fmt.Errorf("GSUB SingleSubst extension target exceeds table")
		}
		subtableOffset = targetOffset
	}

	format := readU16(data, subtableOffset)
	switch format {
	case 1:
		return decodeSingleSubstFormat1(data, subtableOffset, glyphOrder)
	case 2:
		return decodeSingleSubstFormat2(data, subtableOffset, glyphOrder)
	default:
		return nil, nil
	}
}

// SingleSubst Format 1: DeltaGlyphID
func decodeSingleSubstFormat1(data []byte, offset int, glyphOrder []string) ([]SingleSubst, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("GSUB SingleSubst format 1 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	deltaGlyphID := int16(readU16(data, offset+4))
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	var result []SingleSubst
	for _, glyphID := range coverage {
		from, ok := glyphName(glyphOrder, glyphID)
		if !ok {
			continue
		}
		toID := uint16(int16(glyphID) + deltaGlyphID)
		to, ok := glyphName(glyphOrder, toID)
		if !ok {
			continue
		}
		result = append(result, SingleSubst{From: from, To: to})
	}
	return result, nil
}

// SingleSubst Format 2: GlyphID array
func decodeSingleSubstFormat2(data []byte, offset int, glyphOrder []string) ([]SingleSubst, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("GSUB SingleSubst format 2 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	glyphCount := int(readU16(data, offset+4))
	if offset+6+glyphCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB SingleSubst format 2 substitute glyphs exceed table")
	}
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	if len(coverage) != glyphCount {
		return nil, fmt.Errorf("GSUB SingleSubst format 2 coverage count %d != glyph count %d", len(coverage), glyphCount)
	}
	var result []SingleSubst
	for i, fromID := range coverage {
		from, ok := glyphName(glyphOrder, fromID)
		if !ok {
			continue
		}
		toID := readU16(data, offset+6+i*2)
		to, ok := glyphName(glyphOrder, toID)
		if !ok {
			continue
		}
		result = append(result, SingleSubst{From: from, To: to})
	}
	return result, nil
}

func decodeLigatureSubstLookup(data []byte, lk lookupInfo, glyphOrder []string) ([]LigatureSubst, error) {
	subtableCount := int(readU16(data, lk.offset+4))
	if lk.offset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB LigatureSubst lookup %d subtable offsets exceed table", lk.lookupIndex)
	}
	var result []LigatureSubst
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lk.offset + int(readU16(data, lk.offset+6+i*2))
		subs, err := decodeLigatureSubstSubtable(data, lk, subtableOffset, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

func decodeLigatureSubstSubtable(data []byte, lk lookupInfo, subtableOffset int, glyphOrder []string) ([]LigatureSubst, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GSUB LigatureSubst subtable exceeds table")
	}
	if lk.lookupType == 9 {
		if subtableOffset+8 > len(data) {
			return nil, fmt.Errorf("GSUB LigatureSubst extension subtable exceeds table")
		}
		format := readU16(data, subtableOffset)
		if format != 1 {
			return nil, nil
		}
		extensionType := readU16(data, subtableOffset+2)
		if extensionType != 4 {
			return nil, nil
		}
		extensionOffset := int(readU32(data, subtableOffset+4))
		targetOffset := subtableOffset + extensionOffset
		if targetOffset <= subtableOffset || targetOffset > len(data) {
			return nil, fmt.Errorf("GSUB LigatureSubst extension target exceeds table")
		}
		subtableOffset = targetOffset
	}

	return decodeLigatureSubstFormat1(data, subtableOffset, glyphOrder)
}

// LigatureSubst Format 1
func decodeLigatureSubstFormat1(data []byte, offset int, glyphOrder []string) ([]LigatureSubst, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("GSUB LigatureSubst format 1 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	ligSetCount := int(readU16(data, offset+4))
	if offset+6+ligSetCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB LigatureSubst ligature set offsets exceed table")
	}
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	if len(coverage) != ligSetCount {
		return nil, fmt.Errorf("GSUB LigatureSubst coverage count %d != ligature set count %d", len(coverage), ligSetCount)
	}
	var result []LigatureSubst
	for i, firstGlyphID := range coverage {
		firstGlyph, ok := glyphName(glyphOrder, firstGlyphID)
		if !ok {
			continue
		}
		ligSetOffset := int(readU16(data, offset+6+i*2))
		ligatures, err := decodeLigatureSet(data, offset+ligSetOffset, firstGlyph, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, ligatures...)
	}
	return result, nil
}

func decodeLigatureSet(data []byte, offset int, firstGlyph string, glyphOrder []string) ([]LigatureSubst, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("GSUB LigatureSet too short")
	}
	ligCount := int(readU16(data, offset))
	if ligCount > MaxLigatureSetEntries {
		return nil, fmt.Errorf("GSUB LigatureSet count %d exceeds limit %d", ligCount, MaxLigatureSetEntries)
	}
	if offset+2+ligCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB LigatureSet offsets exceed table")
	}
	var result []LigatureSubst
	for i := 0; i < ligCount; i++ {
		ligOffset := int(readU16(data, offset+2+i*2))
		lig, err := decodeLigature(data, offset+ligOffset, firstGlyph, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, lig)
	}
	return result, nil
}

func decodeLigature(data []byte, offset int, firstGlyph string, glyphOrder []string) (LigatureSubst, error) {
	if offset+4 > len(data) {
		return LigatureSubst{}, fmt.Errorf("GSUB Ligature table too short")
	}
	ligGlyphID := readU16(data, offset)
	ligGlyph, ok := glyphName(glyphOrder, ligGlyphID)
	if !ok {
		return LigatureSubst{}, fmt.Errorf("GSUB Ligature glyph id %d outside glyph order", ligGlyphID)
	}
	compCount := int(readU16(data, offset+2))
	if compCount < 2 {
		return LigatureSubst{}, fmt.Errorf("GSUB Ligature compCount %d < 2", compCount)
	}
	if compCount > MaxLigatureComponents {
		return LigatureSubst{}, fmt.Errorf("GSUB Ligature compCount %d exceeds limit %d", compCount, MaxLigatureComponents)
	}
	if offset+4+(compCount-1)*2 > len(data) {
		return LigatureSubst{}, fmt.Errorf("GSUB Ligature component glyphs exceed table")
	}
	components := make([]string, compCount)
	components[0] = firstGlyph
	for j := 1; j < compCount; j++ {
		compID := readU16(data, offset+4+(j-1)*2)
		name, ok := glyphName(glyphOrder, compID)
		if !ok {
			return LigatureSubst{}, fmt.Errorf("GSUB Ligature component glyph id %d outside glyph order", compID)
		}
		components[j] = name
	}
	return LigatureSubst{Components: components, Ligature: ligGlyph}, nil
}

func decodeMultipleSubstLookup(data []byte, lk lookupInfo, glyphOrder []string) ([]MultipleSubst, error) {
	subtableCount := int(readU16(data, lk.offset+4))
	if lk.offset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB MultipleSubst lookup %d subtable offsets exceed table", lk.lookupIndex)
	}
	var result []MultipleSubst
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lk.offset + int(readU16(data, lk.offset+6+i*2))
		subs, err := decodeMultipleSubstSubtable(data, lk, subtableOffset, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

func decodeMultipleSubstSubtable(data []byte, lk lookupInfo, subtableOffset int, glyphOrder []string) ([]MultipleSubst, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GSUB MultipleSubst subtable exceeds table")
	}
	if lk.lookupType == 9 {
		if subtableOffset+8 > len(data) {
			return nil, fmt.Errorf("GSUB MultipleSubst extension subtable exceeds table")
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
			return nil, fmt.Errorf("GSUB MultipleSubst extension target exceeds table")
		}
		subtableOffset = targetOffset
	}

	return decodeMultipleSubstFormat1(data, subtableOffset, glyphOrder)
}

// MultipleSubst Format 1
func decodeMultipleSubstFormat1(data []byte, offset int, glyphOrder []string) ([]MultipleSubst, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("GSUB MultipleSubst format 1 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	sequenceCount := int(readU16(data, offset+4))
	if offset+6+sequenceCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB MultipleSubst sequence offsets exceed table")
	}
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	if len(coverage) != sequenceCount {
		return nil, fmt.Errorf("GSUB MultipleSubst coverage count %d != sequence count %d", len(coverage), sequenceCount)
	}
	totalSeqs := 0
	var result []MultipleSubst
	for i, fromID := range coverage {
		from, ok := glyphName(glyphOrder, fromID)
		if !ok {
			continue
		}
		seqOffset := int(readU16(data, offset+6+i*2))
		to, err := decodeSequence(data, offset+seqOffset, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, MultipleSubst{From: from, To: to})
		totalSeqs++
		if totalSeqs > MaxMultipleSubstSequences {
			return nil, fmt.Errorf("GSUB MultipleSubst exceeds max sequences %d", MaxMultipleSubstSequences)
		}
	}
	return result, nil
}

func decodeSequence(data []byte, offset int, glyphOrder []string) ([]string, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("GSUB Sequence too short")
	}
	glyphCount := int(readU16(data, offset))
	if offset+2+glyphCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB Sequence glyph IDs exceed table")
	}
	names := make([]string, glyphCount)
	for i := 0; i < glyphCount; i++ {
		glyphID := readU16(data, offset+2+i*2)
		name, ok := glyphName(glyphOrder, glyphID)
		if !ok {
			return nil, fmt.Errorf("GSUB Sequence glyph id %d outside glyph order", glyphID)
		}
		names[i] = name
	}
	return names, nil
}

func decodeAlternateSubstLookup(data []byte, lk lookupInfo, glyphOrder []string) ([]AlternateSubst, error) {
	subtableCount := int(readU16(data, lk.offset+4))
	if lk.offset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB AlternateSubst lookup %d subtable offsets exceed table", lk.lookupIndex)
	}
	var result []AlternateSubst
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lk.offset + int(readU16(data, lk.offset+6+i*2))
		subs, err := decodeAlternateSubstSubtable(data, lk, subtableOffset, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, subs...)
	}
	return result, nil
}

func decodeAlternateSubstSubtable(data []byte, lk lookupInfo, subtableOffset int, glyphOrder []string) ([]AlternateSubst, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GSUB AlternateSubst subtable exceeds table")
	}
	// Handle extension lookup type 9.
	if lk.lookupType == 9 {
		if subtableOffset+8 > len(data) {
			return nil, fmt.Errorf("GSUB AlternateSubst extension subtable exceeds table")
		}
		format := readU16(data, subtableOffset)
		if format != 1 {
			return nil, nil
		}
		extensionType := readU16(data, subtableOffset+2)
		if extensionType != 3 {
			return nil, nil
		}
		extensionOffset := int(readU32(data, subtableOffset+4))
		targetOffset := subtableOffset + extensionOffset
		if targetOffset <= subtableOffset || targetOffset > len(data) {
			return nil, fmt.Errorf("GSUB AlternateSubst extension target exceeds table")
		}
		subtableOffset = targetOffset
	}

	return decodeAlternateSubstFormat1(data, subtableOffset, glyphOrder)
}

// AlternateSubst Format 1: Coverage + AlternateSet array.
func decodeAlternateSubstFormat1(data []byte, offset int, glyphOrder []string) ([]AlternateSubst, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("GSUB AlternateSubst format 1 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	altSetCount := int(readU16(data, offset+4))
	if offset+6+altSetCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB AlternateSubst alternate set offsets exceed table")
	}
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	if len(coverage) != altSetCount {
		return nil, fmt.Errorf("GSUB AlternateSubst coverage count %d != alternate set count %d", len(coverage), altSetCount)
	}
	var result []AlternateSubst
	for i, fromID := range coverage {
		from, ok := glyphName(glyphOrder, fromID)
		if !ok {
			continue
		}
		altSetOffset := int(readU16(data, offset+6+i*2))
		alts, err := decodeAlternateSet(data, offset+altSetOffset, glyphOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, AlternateSubst{From: from, Alternates: alts})
	}
	return result, nil
}

func decodeAlternateSet(data []byte, offset int, glyphOrder []string) ([]string, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("GSUB AlternateSet too short")
	}
	glyphCount := int(readU16(data, offset))
	if offset+2+glyphCount*2 > len(data) {
		return nil, fmt.Errorf("GSUB AlternateSet glyph IDs exceed table")
	}
	names := make([]string, glyphCount)
	for i := 0; i < glyphCount; i++ {
		glyphID := readU16(data, offset+2+i*2)
		name, ok := glyphName(glyphOrder, glyphID)
		if !ok {
			return nil, fmt.Errorf("GSUB AlternateSet glyph id %d outside glyph order", glyphID)
		}
		names[i] = name
	}
	return names, nil
}

// decodeCoverage decodes a Coverage table at the given offset within data.
// Supports format 1 (glyph list) and format 2 (range array).
func decodeCoverage(data []byte, offset int) ([]uint16, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("GSUB Coverage offset exceeds table")
	}
	format := readU16(data, offset)
	switch format {
	case 1:
		glyphCount := int(readU16(data, offset+2))
		if offset+4+glyphCount*2 > len(data) {
			return nil, fmt.Errorf("GSUB Coverage format 1 exceeds table")
		}
		glyphs := make([]uint16, glyphCount)
		for i := 0; i < glyphCount; i++ {
			glyphs[i] = readU16(data, offset+4+i*2)
		}
		return glyphs, nil
	case 2:
		rangeCount := int(readU16(data, offset+2))
		if offset+4+rangeCount*6 > len(data) {
			return nil, fmt.Errorf("GSUB Coverage format 2 exceeds table")
		}
		var glyphs []uint16
		for i := 0; i < rangeCount; i++ {
			rangeOffset := offset + 4 + i*6
			start := readU16(data, rangeOffset)
			end := readU16(data, rangeOffset+2)
			if end < start {
				return nil, fmt.Errorf("GSUB Coverage range end before start")
			}
			for glyphID := start; glyphID <= end; glyphID++ {
				glyphs = append(glyphs, glyphID)
				if len(glyphs) > MaxCoverageGlyphs {
					return nil, fmt.Errorf("GSUB Coverage format 2 exceeds max glyphs %d", MaxCoverageGlyphs)
				}
				if glyphID == ^uint16(0) {
					break
				}
			}
		}
		return glyphs, nil
	default:
		return nil, fmt.Errorf("unsupported GSUB Coverage format %d", format)
	}
}

func glyphName(glyphOrder []string, glyphID uint16) (string, bool) {
	index := int(glyphID)
	if index < 0 || index >= len(glyphOrder) {
		return "", false
	}
	return glyphOrder[index], true
}