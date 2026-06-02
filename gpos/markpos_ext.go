// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import (
	"fmt"
)

type MarkClassSummary struct {
	// LookupIndex is the zero-based GPOS lookup index.
	LookupIndex int `json:"lookupIndex"`
	// LookupType is 4 (MarkBase), 5 (MarkLigature), or 6 (MarkMark).
	LookupType int `json:"lookupType"`
	// MarkClass is the mark class identifier.
	MarkClass uint16 `json:"markClass"`
	// MarkGlyphCount is the number of mark glyphs in this class.
	MarkGlyphCount int `json:"markGlyphCount"`
	// BaseCount is the number of base/ligature/mark2 glyphs that have
	// an anchor for this class.
	BaseCount int `json:"baseCount"`
}

// MaxMarkClassSummary limits the number of mark class summary entries.
var MaxMarkClassSummary int = 10_000

// DecodeMarkClassSummary returns class-level summary for GPOS mark lookups.
// Instead of expanding all mark*base pairs, it reports per-class counts.
func DecodeMarkClassSummary(data []byte, glyphOrder []string) ([]MarkClassSummary, error) {
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

	var all []MarkClassSummary
	for lookupIndex := 0; lookupIndex < lookupCount; lookupIndex++ {
		lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))
		summaries, err := decodeMarkClassLookup(data, glyphOrder, lookupOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, summaries...)
		if len(all) > MaxMarkClassSummary {
			return all[:MaxMarkClassSummary], nil
		}
	}
	return all, nil
}

func decodeMarkClassLookup(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int) ([]MarkClassSummary, error) {
	if lookupOffset+6 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d exceeds table", lookupIndex)
	}
	lookupType := readU16(data, lookupOffset)
	if lookupType != 4 && lookupType != 5 && lookupType != 6 && lookupType != 9 {
		return nil, nil
	}
	subtableCount := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable offsets exceed table", lookupIndex)
	}

	var all []MarkClassSummary
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		summaries, err := decodeMarkClassSubtable(data, glyphOrder, lookupType, subtableOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, summaries...)
	}
	return all, nil
}

func decodeMarkClassSubtable(data []byte, glyphOrder []string, lookupType uint16, subtableOffset int, lookupIndex int) ([]MarkClassSummary, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS mark subtable exceeds table")
	}
	if lookupType == 9 {
		return decodeMarkClassExtensionSubtable(data, glyphOrder, subtableOffset, lookupIndex)
	}
	switch lookupType {
	case 4:
		return decodeMarkBaseClassSummary(data, glyphOrder, subtableOffset, lookupIndex)
	case 5:
		return decodeMarkLigatureClassSummary(data, glyphOrder, subtableOffset, lookupIndex)
	case 6:
		return decodeMarkMarkClassSummary(data, glyphOrder, subtableOffset, lookupIndex)
	}
	return nil, nil
}

func decodeMarkClassExtensionSubtable(data []byte, glyphOrder []string, subtableOffset int, lookupIndex int) ([]MarkClassSummary, error) {
	if subtableOffset+8 > len(data) {
		return nil, fmt.Errorf("GPOS extension mark subtable exceeds table")
	}
	format := readU16(data, subtableOffset)
	if format != 1 {
		return nil, nil
	}
	extensionType := readU16(data, subtableOffset+2)
	if extensionType != 4 && extensionType != 5 && extensionType != 6 {
		return nil, nil
	}
	extensionOffset := int(readU32(data, subtableOffset+4))
	targetOffset := subtableOffset + extensionOffset
	if targetOffset <= subtableOffset || targetOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS extension mark target exceeds table")
	}
	switch extensionType {
	case 4:
		return decodeMarkBaseClassSummary(data, glyphOrder, targetOffset, lookupIndex)
	case 5:
		return decodeMarkLigatureClassSummary(data, glyphOrder, targetOffset, lookupIndex)
	case 6:
		return decodeMarkMarkClassSummary(data, glyphOrder, targetOffset, lookupIndex)
	}
	return nil, nil
}

func decodeMarkBaseClassSummary(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]MarkClassSummary, error) {
	if offset+12 > len(data) {
		return nil, fmt.Errorf("MarkBasePos too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, nil
	}
	classCount := int(readU16(data, offset+6))
	markArrOff := int(readU16(data, offset+8))
	baseArrOff := int(readU16(data, offset+10))

	markRecords, err := decodeMarkArray(data, offset+markArrOff)
	if err != nil {
		return nil, err
	}

	marksPerClass := make([]int, classCount)
	for _, rec := range markRecords {
		c := int(rec.markClass)
		if c >= 0 && c < classCount {
			marksPerClass[c]++
		}
	}

	baseCovOff := int(readU16(data, offset+4))
	baseCoverage, err := decodeCoverage(data, offset+baseCovOff)
	if err != nil {
		return nil, err
	}
	baseAbsOff := offset + baseArrOff
	if baseAbsOff+2 > len(data) {
		return nil, fmt.Errorf("MarkBasePos BaseArray too short")
	}
	baseCount := int(readU16(data, baseAbsOff))
	if baseAbsOff+2+baseCount*classCount*2 > len(data) {
		return nil, fmt.Errorf("MarkBasePos BaseArray records exceed table")
	}

	basesPerClass := make([]int, classCount)
	for baseIdx := 0; baseIdx < baseCount && baseIdx < len(baseCoverage); baseIdx++ {
		recStart := baseAbsOff + 2 + baseIdx*classCount*2
		for c := 0; c < classCount; c++ {
			anchorOff := int(readU16(data, recStart+c*2))
			if anchorOff != 0 {
				basesPerClass[c]++
			}
		}
	}

	summaries := make([]MarkClassSummary, 0, classCount)
	for c := 0; c < classCount; c++ {
		if marksPerClass[c] == 0 && basesPerClass[c] == 0 {
			continue
		}
		summaries = append(summaries, MarkClassSummary{
			LookupIndex:    lookupIndex,
			LookupType:     4,
			MarkClass:      uint16(c),
			MarkGlyphCount: marksPerClass[c],
			BaseCount:      basesPerClass[c],
		})
	}
	return summaries, nil
}

func decodeMarkLigatureClassSummary(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]MarkClassSummary, error) {
	if offset+12 > len(data) {
		return nil, fmt.Errorf("MarkLigaturePos too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, nil
	}
	classCount := int(readU16(data, offset+6))
	markArrOff := int(readU16(data, offset+8))
	ligArrOff := int(readU16(data, offset+10))

	markRecords, err := decodeMarkArray(data, offset+markArrOff)
	if err != nil {
		return nil, err
	}

	marksPerClass := make([]int, classCount)
	for _, rec := range markRecords {
		c := int(rec.markClass)
		if c >= 0 && c < classCount {
			marksPerClass[c]++
		}
	}

	ligCovOff := int(readU16(data, offset+4))
	ligCoverage, err := decodeCoverage(data, offset+ligCovOff)
	if err != nil {
		return nil, err
	}
	ligAbsOff := offset + ligArrOff
	if ligAbsOff+2 > len(data) {
		return nil, fmt.Errorf("MarkLigaturePos LigatureArray too short")
	}
	ligCount := int(readU16(data, ligAbsOff))
	if ligAbsOff+2+ligCount*2 > len(data) {
		return nil, fmt.Errorf("MarkLigaturePos offsets exceed table")
	}

	basesPerClass := make([]int, classCount)
	for ligIdx := 0; ligIdx < ligCount && ligIdx < len(ligCoverage); ligIdx++ {
		attachOffset := int(readU16(data, ligAbsOff+2+ligIdx*2))
		attachAbs := ligAbsOff + attachOffset
		if attachAbs+2 > len(data) {
			continue
		}
		componentCount := int(readU16(data, attachAbs))
		for comp := 0; comp < componentCount; comp++ {
			compRecStart := attachAbs + 2 + comp*classCount*2
			if compRecStart+classCount*2 > len(data) {
				break
			}
			for c := 0; c < classCount; c++ {
				anchorOff := int(readU16(data, compRecStart+c*2))
				if anchorOff != 0 {
					basesPerClass[c]++
				}
			}
		}
	}

	summaries := make([]MarkClassSummary, 0, classCount)
	for c := 0; c < classCount; c++ {
		if marksPerClass[c] == 0 && basesPerClass[c] == 0 {
			continue
		}
		summaries = append(summaries, MarkClassSummary{
			LookupIndex:    lookupIndex,
			LookupType:     5,
			MarkClass:      uint16(c),
			MarkGlyphCount: marksPerClass[c],
			BaseCount:      basesPerClass[c],
		})
	}
	return summaries, nil
}

func decodeMarkMarkClassSummary(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]MarkClassSummary, error) {
	if offset+12 > len(data) {
		return nil, fmt.Errorf("MarkMarkPos too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, nil
	}
	classCount := int(readU16(data, offset+6))
	mark1ArrOff := int(readU16(data, offset+8))
	mark2ArrOff := int(readU16(data, offset+10))

	mark1Records, err := decodeMarkArray(data, offset+mark1ArrOff)
	if err != nil {
		return nil, err
	}

	marksPerClass := make([]int, classCount)
	for _, rec := range mark1Records {
		c := int(rec.markClass)
		if c >= 0 && c < classCount {
			marksPerClass[c]++
		}
	}

	mark2CovOff := int(readU16(data, offset+4))
	mark2Coverage, err := decodeCoverage(data, offset+mark2CovOff)
	if err != nil {
		return nil, err
	}
	mark2AbsOff := offset + mark2ArrOff
	if mark2AbsOff+2 > len(data) {
		return nil, fmt.Errorf("MarkMarkPos Mark2Array too short")
	}
	mark2Count := int(readU16(data, mark2AbsOff))
	if mark2AbsOff+2+mark2Count*classCount*2 > len(data) {
		return nil, fmt.Errorf("MarkMarkPos records exceed table")
	}

	basesPerClass := make([]int, classCount)
	for m2Idx := 0; m2Idx < mark2Count && m2Idx < len(mark2Coverage); m2Idx++ {
		recStart := mark2AbsOff + 2 + m2Idx*classCount*2
		for c := 0; c < classCount; c++ {
			anchorOff := int(readU16(data, recStart+c*2))
			if anchorOff != 0 {
				basesPerClass[c]++
			}
		}
	}

	summaries := make([]MarkClassSummary, 0, classCount)
	for c := 0; c < classCount; c++ {
		if marksPerClass[c] == 0 && basesPerClass[c] == 0 {
			continue
		}
		summaries = append(summaries, MarkClassSummary{
			LookupIndex:    lookupIndex,
			LookupType:     6,
			MarkClass:      uint16(c),
			MarkGlyphCount: marksPerClass[c],
			BaseCount:      basesPerClass[c],
		})
	}
	return summaries, nil
}

// ExpandMarkClass expands one mark class from a specific lookup into
// individual MarkAttachment pairs. Returns at most maxPairs results.
func ExpandMarkClass(data []byte, glyphOrder []string, lookupIndex int, markClass uint16, maxPairs int) ([]MarkAttachment, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("GPOS header too short")
	}
	lookupListOffset := int(readU16(data, 8))
	if lookupListOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS LookupList offset exceeds table")
	}
	lookupCount := int(readU16(data, lookupListOffset))
	if lookupIndex >= lookupCount {
		return nil, fmt.Errorf("lookup index %d out of range", lookupIndex)
	}
	lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))

	if lookupOffset+6 > len(data) {
		return nil, fmt.Errorf("lookup exceeds table")
	}
	lookupType := readU16(data, lookupOffset)
	if lookupType != 4 && lookupType != 5 && lookupType != 6 && lookupType != 9 {
		return nil, fmt.Errorf("lookup type %d is not a mark lookup", lookupType)
	}
	subtableCount := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("lookup subtable offsets exceed table")
	}

	var all []MarkAttachment
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		attachments, err := expandMarkClassSubtable(data, glyphOrder, lookupType, subtableOffset, lookupIndex, markClass, maxPairs-len(all))
		if err != nil {
			continue
		}
		all = append(all, attachments...)
		if len(all) >= maxPairs {
			return all[:maxPairs], nil
		}
	}
	return all, nil
}

func expandMarkClassSubtable(data []byte, glyphOrder []string, lookupType uint16, subtableOffset int, lookupIndex int, markClass uint16, remaining int) ([]MarkAttachment, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("subtable exceeds table")
	}
	if lookupType == 9 {
		if subtableOffset+8 > len(data) {
			return nil, fmt.Errorf("extension subtable exceeds table")
		}
		format := readU16(data, subtableOffset)
		if format != 1 {
			return nil, nil
		}
		extType := readU16(data, subtableOffset+2)
		if extType != 4 && extType != 5 && extType != 6 {
			return nil, nil
		}
		extOffset := int(readU32(data, subtableOffset+4))
		targetOffset := subtableOffset + extOffset
		if targetOffset <= subtableOffset || targetOffset+2 > len(data) {
			return nil, fmt.Errorf("extension target exceeds table")
		}
		switch extType {
		case 4:
			return expandMarkBaseClass(data, glyphOrder, targetOffset, lookupIndex, markClass, remaining)
		case 5:
			return expandMarkLigatureClass(data, glyphOrder, targetOffset, lookupIndex, markClass, remaining)
		case 6:
			return expandMarkMarkClass(data, glyphOrder, targetOffset, lookupIndex, markClass, remaining)
		}
		return nil, nil
	}
	switch lookupType {
	case 4:
		return expandMarkBaseClass(data, glyphOrder, subtableOffset, lookupIndex, markClass, remaining)
	case 5:
		return expandMarkLigatureClass(data, glyphOrder, subtableOffset, lookupIndex, markClass, remaining)
	case 6:
		return expandMarkMarkClass(data, glyphOrder, subtableOffset, lookupIndex, markClass, remaining)
	}
	return nil, nil
}

func expandMarkBaseClass(data []byte, glyphOrder []string, offset int, lookupIndex int, markClass uint16, maxPairs int) ([]MarkAttachment, error) {
	if offset+12 > len(data) {
		return nil, fmt.Errorf("MarkBasePos too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, nil
	}
	markCovOff := int(readU16(data, offset+2))
	baseCovOff := int(readU16(data, offset+4))
	classCount := int(readU16(data, offset+6))
	markArrOff := int(readU16(data, offset+8))
	baseArrOff := int(readU16(data, offset+10))

	markCoverage, err := decodeCoverage(data, offset+markCovOff)
	if err != nil {
		return nil, err
	}
	baseCoverage, err := decodeCoverage(data, offset+baseCovOff)
	if err != nil {
		return nil, err
	}
	markRecords, err := decodeMarkArray(data, offset+markArrOff)
	if err != nil {
		return nil, err
	}

	baseAbsOff := offset + baseArrOff
	if baseAbsOff+2 > len(data) {
		return nil, fmt.Errorf("BaseArray too short")
	}
	baseCount := int(readU16(data, baseAbsOff))

	var attachments []MarkAttachment
	for i, markGlyphID := range markCoverage {
		if i >= len(markRecords) {
			break
		}
		rec := markRecords[i]
		if rec.markClass != markClass {
			continue
		}
		markGlyph, ok := glyphName(glyphOrder, markGlyphID)
		if !ok {
			continue
		}
		classIdx := int(rec.markClass)
		for baseIdx, baseGlyphID := range baseCoverage {
			if baseIdx >= baseCount {
				break
			}
			baseGlyph, ok2 := glyphName(glyphOrder, baseGlyphID)
			if !ok2 {
				continue
			}
			baseRecStart := baseAbsOff + 2 + baseIdx*classCount*2
			if baseRecStart+classIdx*2+2 > len(data) {
				continue
			}
			anchorOff := int(readU16(data, baseRecStart+classIdx*2))
			if anchorOff == 0 {
				continue
			}
			baseAnchor, err := decodeAnchor(data, baseAbsOff+anchorOff)
			if err != nil {
				continue
			}
			attachments = append(attachments, MarkAttachment{
				MarkGlyph:   markGlyph,
				BaseGlyph:   baseGlyph,
				MarkClass:   rec.markClass,
				MarkAnchor:  rec.markAnchor,
				BaseAnchor:  baseAnchor,
				Component:   0,
				LookupType:  4,
				LookupIndex: lookupIndex,
			})
			if len(attachments) >= maxPairs {
				return attachments, nil
			}
		}
	}
	return attachments, nil
}

func expandMarkLigatureClass(data []byte, glyphOrder []string, offset int, lookupIndex int, markClass uint16, maxPairs int) ([]MarkAttachment, error) {
	if offset+12 > len(data) {
		return nil, fmt.Errorf("MarkLigaturePos too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, nil
	}
	markCovOff := int(readU16(data, offset+2))
	ligCovOff := int(readU16(data, offset+4))
	classCount := int(readU16(data, offset+6))
	markArrOff := int(readU16(data, offset+8))
	ligArrOff := int(readU16(data, offset+10))

	markCoverage, err := decodeCoverage(data, offset+markCovOff)
	if err != nil {
		return nil, err
	}
	ligCoverage, err := decodeCoverage(data, offset+ligCovOff)
	if err != nil {
		return nil, err
	}
	markRecords, err := decodeMarkArray(data, offset+markArrOff)
	if err != nil {
		return nil, err
	}

	ligAbsOff := offset + ligArrOff
	if ligAbsOff+2 > len(data) {
		return nil, fmt.Errorf("LigatureArray too short")
	}
	ligCount := int(readU16(data, ligAbsOff))

	var attachments []MarkAttachment
	for i, markGlyphID := range markCoverage {
		if i >= len(markRecords) {
			break
		}
		rec := markRecords[i]
		if rec.markClass != markClass {
			continue
		}
		markGlyph, ok := glyphName(glyphOrder, markGlyphID)
		if !ok {
			continue
		}
		classIdx := int(rec.markClass)
		for ligIdx, ligGlyphID := range ligCoverage {
			if ligIdx >= ligCount {
				break
			}
			ligGlyph, ok2 := glyphName(glyphOrder, ligGlyphID)
			if !ok2 {
				continue
			}
			ligAttachOffset := int(readU16(data, ligAbsOff+2+ligIdx*2))
			ligAttachAbs := ligAbsOff + ligAttachOffset
			if ligAttachAbs+2 > len(data) {
				continue
			}
			componentCount := int(readU16(data, ligAttachAbs))
			for comp := 0; comp < componentCount; comp++ {
				compRecStart := ligAttachAbs + 2 + comp*classCount*2
				if compRecStart+classCount*2 > len(data) {
					break
				}
				anchorOff := int(readU16(data, compRecStart+classIdx*2))
				if anchorOff == 0 {
					continue
				}
				anchor, err := decodeAnchor(data, ligAttachAbs+anchorOff)
				if err != nil {
					continue
				}
				attachments = append(attachments, MarkAttachment{
					MarkGlyph:   markGlyph,
					BaseGlyph:   ligGlyph,
					MarkClass:   rec.markClass,
					MarkAnchor:  rec.markAnchor,
					BaseAnchor:  anchor,
					Component:   comp,
					LookupType:  5,
					LookupIndex: lookupIndex,
				})
				if len(attachments) >= maxPairs {
					return attachments, nil
				}
			}
		}
	}
	return attachments, nil
}

func expandMarkMarkClass(data []byte, glyphOrder []string, offset int, lookupIndex int, markClass uint16, maxPairs int) ([]MarkAttachment, error) {
	if offset+12 > len(data) {
		return nil, fmt.Errorf("MarkMarkPos too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return nil, nil
	}
	mark1CovOff := int(readU16(data, offset+2))
	mark2CovOff := int(readU16(data, offset+4))
	classCount := int(readU16(data, offset+6))
	mark1ArrOff := int(readU16(data, offset+8))
	mark2ArrOff := int(readU16(data, offset+10))

	mark1Coverage, err := decodeCoverage(data, offset+mark1CovOff)
	if err != nil {
		return nil, err
	}
	mark2Coverage, err := decodeCoverage(data, offset+mark2CovOff)
	if err != nil {
		return nil, err
	}
	mark1Records, err := decodeMarkArray(data, offset+mark1ArrOff)
	if err != nil {
		return nil, err
	}

	mark2AbsOff := offset + mark2ArrOff
	if mark2AbsOff+2 > len(data) {
		return nil, fmt.Errorf("Mark2Array too short")
	}
	mark2Count := int(readU16(data, mark2AbsOff))

	var attachments []MarkAttachment
	for i, mark1GlyphID := range mark1Coverage {
		if i >= len(mark1Records) {
			break
		}
		rec := mark1Records[i]
		if rec.markClass != markClass {
			continue
		}
		mark1Glyph, ok := glyphName(glyphOrder, mark1GlyphID)
		if !ok {
			continue
		}
		classIdx := int(rec.markClass)
		for mark2Idx, mark2GlyphID := range mark2Coverage {
			if mark2Idx >= mark2Count {
				break
			}
			mark2Glyph, ok2 := glyphName(glyphOrder, mark2GlyphID)
			if !ok2 {
				continue
			}
			mark2RecStart := mark2AbsOff + 2 + mark2Idx*classCount*2
			if mark2RecStart+classIdx*2+2 > len(data) {
				continue
			}
			anchorOff := int(readU16(data, mark2RecStart+classIdx*2))
			if anchorOff == 0 {
				continue
			}
			anchor, err := decodeAnchor(data, mark2AbsOff+anchorOff)
			if err != nil {
				continue
			}
			attachments = append(attachments, MarkAttachment{
				MarkGlyph:   mark1Glyph,
				BaseGlyph:   mark2Glyph,
				MarkClass:   rec.markClass,
				MarkAnchor:  rec.markAnchor,
				BaseAnchor:  anchor,
				Component:   0,
				LookupType:  6,
				LookupIndex: lookupIndex,
			})
			if len(attachments) >= maxPairs {
				return attachments, nil
			}
		}
	}
	return attachments, nil
}
