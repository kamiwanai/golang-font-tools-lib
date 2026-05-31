package gpos

import (
	"fmt"
)

// Anchor is a point in the glyph coordinate space used for mark attachment.
type Anchor struct {
	// X is the horizontal coordinate in font units.
	X int16
	// Y is the vertical coordinate in font units.
	Y int16
}

// MarkAttachment describes one resolved mark-to-base/ligature/mark attachment.
type MarkAttachment struct {
	// MarkGlyph is the mark glyph name (e.g. "acutecomb").
	MarkGlyph string
	// BaseGlyph is the base, ligature, or mark2 glyph name.
	BaseGlyph string
	// MarkClass is the mark class identifier (from the MarkArray).
	MarkClass uint16
	// MarkAnchor is the anchor point on the mark glyph.
	MarkAnchor Anchor
	// BaseAnchor is the anchor point on the base/ligature/mark2 glyph.
	BaseAnchor Anchor
	// Component is the ligature component index (0-based). Only meaningful
	// for MarkLigaturePos (lookup type 5). Zero for other types.
	Component int
	// LookupType is the GPOS lookup type: 4 (MarkBase), 5 (MarkLigature),
	// or 6 (MarkMark).
	LookupType int
	// LookupIndex is the zero-based lookup index in the GPOS LookupList.
	LookupIndex int
}

// DecodeMarkLookups decodes GPOS mark attachment lookups (types 4, 5, 6) from
// a GPOS table. It returns a flat list of all resolved mark attachments.
//
// glyphOrder maps glyph IDs to names. The function handles ExtensionPos
// lookups (type 9) targeting mark lookup types.
func DecodeMarkLookups(data []byte, glyphOrder []string) ([]MarkAttachment, error) {
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

	var all []MarkAttachment
	for lookupIndex := 0; lookupIndex < lookupCount; lookupIndex++ {
		lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))
		attachments, err := decodeMarkLookup(data, glyphOrder, lookupOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, attachments...)
	}
	return all, nil
}

func decodeMarkLookup(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int) ([]MarkAttachment, error) {
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

	var all []MarkAttachment
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		attachments, err := decodeMarkSubtable(data, glyphOrder, lookupType, subtableOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, attachments...)
	}
	return all, nil
}

func decodeMarkSubtable(data []byte, glyphOrder []string, lookupType uint16, subtableOffset int, lookupIndex int) ([]MarkAttachment, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS mark subtable exceeds table")
	}
	if lookupType == 9 {
		return decodeMarkExtensionSubtable(data, glyphOrder, subtableOffset, lookupIndex)
	}
	switch lookupType {
	case 4:
		return decodeMarkBasePos(data, glyphOrder, subtableOffset, lookupIndex)
	case 5:
		return decodeMarkLigaturePos(data, glyphOrder, subtableOffset, lookupIndex)
	case 6:
		return decodeMarkMarkPos(data, glyphOrder, subtableOffset, lookupIndex)
	}
	return nil, nil
}

func decodeMarkExtensionSubtable(data []byte, glyphOrder []string, subtableOffset int, lookupIndex int) ([]MarkAttachment, error) {
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
	if targetOffset <= subtableOffset || targetOffset > len(data) {
		return nil, fmt.Errorf("GPOS extension mark target exceeds table")
	}
	if targetOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS extension mark target exceeds table")
	}
	switch extensionType {
	case 4:
		return decodeMarkBasePos(data, glyphOrder, targetOffset, lookupIndex)
	case 5:
		return decodeMarkLigaturePos(data, glyphOrder, targetOffset, lookupIndex)
	case 6:
		return decodeMarkMarkPos(data, glyphOrder, targetOffset, lookupIndex)
	}
	return nil, nil
}

// markRecord holds one parsed MarkRecord from the MarkArray.
type markRecord struct {
	markClass  uint16
	markAnchor Anchor
}

// decodeMarkArray parses a MarkArray table at the given offset.
func decodeMarkArray(data []byte, baseOffset int) ([]markRecord, error) {
	if baseOffset+2 > len(data) {
		return nil, fmt.Errorf("MarkArray too short")
	}
	markCount := int(readU16(data, baseOffset))
	if markCount > MaxMarkCount {
		return nil, fmt.Errorf("MarkArray count %d exceeds limit %d", markCount, MaxMarkCount)
	}
	if baseOffset+2+markCount*4 > len(data) {
		return nil, fmt.Errorf("MarkArray records exceed table")
	}
	records := make([]markRecord, markCount)
	for i := 0; i < markCount; i++ {
		recOffset := baseOffset + 2 + i*4
		records[i].markClass = readU16(data, recOffset)
		markAnchorOffset := int(readU16(data, recOffset+2))
		if markAnchorOffset == 0 {
			continue
		}
		anchor, err := decodeAnchor(data, baseOffset+markAnchorOffset)
		if err != nil {
			return nil, fmt.Errorf("MarkArray record %d anchor: %w", i, err)
		}
		records[i].markAnchor = anchor
	}
	return records, nil
}

// decodeAnchor parses an Anchor table at the given absolute offset.
// Supports formats 1, 2, and 3.
func decodeAnchor(data []byte, offset int) (Anchor, error) {
	if offset+4 > len(data) {
		return Anchor{}, fmt.Errorf("Anchor too short")
	}
	format := readU16(data, offset)
	switch format {
	case 1:
		return Anchor{
			X: int16(readU16(data, offset+2)),
			Y: int16(readU16(data, offset+4)),
		}, nil
	case 2:
		if offset+6 > len(data) {
			return Anchor{}, fmt.Errorf("Anchor format 2 too short")
		}
		return Anchor{
			X: int16(readU16(data, offset+2)),
			Y: int16(readU16(data, offset+4)),
		}, nil
	case 3:
		if offset+8 > len(data) {
			return Anchor{}, fmt.Errorf("Anchor format 3 too short")
		}
		return Anchor{
			X: int16(readU16(data, offset+2)),
			Y: int16(readU16(data, offset+4)),
		}, nil
	default:
		return Anchor{}, fmt.Errorf("unsupported Anchor format %d", format)
	}
}

// decodeMarkBasePos decodes a MarkBasePosFormat1 subtable (lookup type 4).
func decodeMarkBasePos(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]MarkAttachment, error) {
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
		return nil, fmt.Errorf("MarkBasePos mark coverage: %w", err)
	}
	baseCoverage, err := decodeCoverage(data, offset+baseCovOff)
	if err != nil {
		return nil, fmt.Errorf("MarkBasePos base coverage: %w", err)
	}

	markRecords, err := decodeMarkArray(data, offset+markArrOff)
	if err != nil {
		return nil, fmt.Errorf("MarkBasePos mark array: %w", err)
	}
	if len(markRecords) != len(markCoverage) {
		return nil, fmt.Errorf("MarkBasePos mark count %d != coverage count %d", len(markRecords), len(markCoverage))
	}

	baseAbsOff := offset + baseArrOff
	if baseAbsOff+2 > len(data) {
		return nil, fmt.Errorf("MarkBasePos BaseArray too short")
	}
	baseCount := int(readU16(data, baseAbsOff))
	if baseCount != len(baseCoverage) {
		return nil, fmt.Errorf("MarkBasePos base count %d != coverage count %d", baseCount, len(baseCoverage))
	}
	if baseAbsOff+2+baseCount*classCount*2 > len(data) {
		return nil, fmt.Errorf("MarkBasePos BaseArray records exceed table")
	}

	var attachments []MarkAttachment
	for i, markGlyphID := range markCoverage {
		markGlyph, ok := glyphName(glyphOrder, markGlyphID)
		if !ok {
			continue
		}
		rec := markRecords[i]
		classIdx := int(rec.markClass)
		if classIdx >= classCount {
			continue
		}

		for baseIdx, baseGlyphID := range baseCoverage {
			baseGlyph, ok2 := glyphName(glyphOrder, baseGlyphID)
			if !ok2 {
				continue
			}
			baseRecStart := baseAbsOff + 2 + baseIdx*classCount*2
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
			if len(attachments) > MaxMarkAttachments {
				return nil, fmt.Errorf("MarkBasePos exceeds max attachments %d", MaxMarkAttachments)
			}
		}
	}
	return attachments, nil
}

// decodeMarkLigaturePos decodes a MarkLigaturePosFormat1 subtable (lookup type 5).
func decodeMarkLigaturePos(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]MarkAttachment, error) {
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
		return nil, fmt.Errorf("MarkLigaturePos mark coverage: %w", err)
	}
	ligCoverage, err := decodeCoverage(data, offset+ligCovOff)
	if err != nil {
		return nil, fmt.Errorf("MarkLigaturePos ligature coverage: %w", err)
	}

	markRecords, err := decodeMarkArray(data, offset+markArrOff)
	if err != nil {
		return nil, fmt.Errorf("MarkLigaturePos mark array: %w", err)
	}
	if len(markRecords) != len(markCoverage) {
		return nil, fmt.Errorf("MarkLigaturePos mark count %d != coverage %d", len(markRecords), len(markCoverage))
	}

	ligAbsOff := offset + ligArrOff
	if ligAbsOff+2 > len(data) {
		return nil, fmt.Errorf("MarkLigaturePos LigatureArray too short")
	}
	ligCount := int(readU16(data, ligAbsOff))
	if ligCount != len(ligCoverage) {
		return nil, fmt.Errorf("MarkLigaturePos lig count %d != coverage %d", ligCount, len(ligCoverage))
	}
	if ligAbsOff+2+ligCount*2 > len(data) {
		return nil, fmt.Errorf("MarkLigaturePos LigatureArray offsets exceed table")
	}

	var attachments []MarkAttachment
	for i, markGlyphID := range markCoverage {
		markGlyph, ok := glyphName(glyphOrder, markGlyphID)
		if !ok {
			continue
		}
		rec := markRecords[i]
		classIdx := int(rec.markClass)
		if classIdx >= classCount {
			continue
		}

		for ligIdx, ligGlyphID := range ligCoverage {
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
			if componentCount == 0 {
				continue
			}
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
				if len(attachments) > MaxMarkAttachments {
					return nil, fmt.Errorf("MarkLigaturePos exceeds max attachments %d", MaxMarkAttachments)
				}
			}
		}
	}
	return attachments, nil
}

// decodeMarkMarkPos decodes a MarkMarkPosFormat1 subtable (lookup type 6).
func decodeMarkMarkPos(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]MarkAttachment, error) {
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
		return nil, fmt.Errorf("MarkMarkPos mark1 coverage: %w", err)
	}
	mark2Coverage, err := decodeCoverage(data, offset+mark2CovOff)
	if err != nil {
		return nil, fmt.Errorf("MarkMarkPos mark2 coverage: %w", err)
	}

	mark1Records, err := decodeMarkArray(data, offset+mark1ArrOff)
	if err != nil {
		return nil, fmt.Errorf("MarkMarkPos mark1 array: %w", err)
	}
	if len(mark1Records) != len(mark1Coverage) {
		return nil, fmt.Errorf("MarkMarkPos mark1 count %d != coverage %d", len(mark1Records), len(mark1Coverage))
	}

	mark2AbsOff := offset + mark2ArrOff
	if mark2AbsOff+2 > len(data) {
		return nil, fmt.Errorf("MarkMarkPos Mark2Array too short")
	}
	mark2Count := int(readU16(data, mark2AbsOff))
	if mark2Count != len(mark2Coverage) {
		return nil, fmt.Errorf("MarkMarkPos mark2 count %d != coverage %d", mark2Count, len(mark2Coverage))
	}
	if mark2AbsOff+2+mark2Count*classCount*2 > len(data) {
		return nil, fmt.Errorf("MarkMarkPos Mark2Array records exceed table")
	}

	var attachments []MarkAttachment
	for i, mark1GlyphID := range mark1Coverage {
		mark1Glyph, ok := glyphName(glyphOrder, mark1GlyphID)
		if !ok {
			continue
		}
		rec := mark1Records[i]
		classIdx := int(rec.markClass)
		if classIdx >= classCount {
			continue
		}

		for mark2Idx, mark2GlyphID := range mark2Coverage {
			mark2Glyph, ok2 := glyphName(glyphOrder, mark2GlyphID)
			if !ok2 {
				continue
			}
			mark2RecStart := mark2AbsOff + 2 + mark2Idx*classCount*2
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
			if len(attachments) > MaxMarkAttachments {
				return nil, fmt.Errorf("MarkMarkPos exceeds max attachments %d", MaxMarkAttachments)
			}
		}
	}
	return attachments, nil
}
