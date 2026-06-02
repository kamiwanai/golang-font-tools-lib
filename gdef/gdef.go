// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gdef

import (
	"encoding/binary"
	"fmt"

	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

// Glyph class constants as defined by the OpenType GDEF specification.
const (
	ClassBase      = 1 // Base glyph
	ClassLigature  = 2 // Ligature glyph
	ClassMark      = 3 // Mark glyph
	ClassComponent = 4 // Component glyph
)

// ClassRangeRecord maps a range of glyph IDs to a class value.
type ClassRangeRecord struct {
	StartGlyphID uint16
	EndGlyphID   uint16
	Class        uint16
}

// AttachPoint lists contour attachment points for one glyph.
type AttachPoint struct {
	// GlyphID is the glyph identifier.
	GlyphID uint16
	// PointIndices are contour point indices.
	PointIndices []uint16
}

// LigGlyph holds caret positions for one ligature glyph.
type LigGlyph struct {
	// GlyphID is the glyph identifier.
	GlyphID uint16
	// CaretOffsets are x-coordinates of caret positions (in font units).
	CaretOffsets []int16
}

// GDEF holds the parsed fields from an OpenType GDEF table.
type GDEF struct {
	// MajorVersion is the major version number (1 or 2).
	MajorVersion uint16
	// MinorVersion is the minor version number.
	MinorVersion uint16
	// GlyphClassMap maps each glyph ID to its class (1=Base, 2=Ligature,
	// 3=Mark, 4=Component). Glyphs not in the map have class 0 (no class).
	GlyphClassMap map[uint16]uint16
	// AttachList contains attachment points per glyph.
	AttachList []AttachPoint
	// LigCaretList contains caret positions for ligature glyphs.
	LigCaretList []LigGlyph
	// MarkAttachClassMap maps each glyph ID to its mark attachment class.
	// Only present in GDEF version 1.2+.
	MarkAttachClassMap map[uint16]uint16
}

// Decode parses a GDEF table from raw bytes and returns a GDEF struct.
// It supports GDEF version 1.0 through 1.3 (and version 2.0 which adds
// ItemVariationStore, not parsed here).
//
// glyphOrder maps glyph IDs to names; it is used to resolve glyph names
// for ClassRangeRecords in GlyphClassDef.
func Decode(data []byte, glyphOrder []string) (GDEF, error) {
	if len(data) < 12 {
		return GDEF{}, fmt.Errorf("GDEF header too short: %d bytes", len(data))
	}
	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)

	gdef := GDEF{
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
	}

	glyphClassDefOffset := int(readU16(data, 4))
	attachListOffset := int(readU16(data, 6))
	ligCaretListOffset := int(readU16(data, 8))
	markAttachClassDefOffset := int(readU16(data, 10))

	// Version 1.2+: markAttachClassDef at offset 10
	// Version 1.3+: markGlyphSetsDefOffset at offset 12
	// Version 2.0+: itemVariationStoreOffset at offset 14

	if glyphClassDefOffset != 0 {
		classMap, err := decodeClassDef(data, glyphClassDefOffset)
		if err != nil {
			return GDEF{}, fmt.Errorf("GDEF GlyphClassDef: %w", err)
		}
		gdef.GlyphClassMap = classMap
	}

	if attachListOffset != 0 {
		attachList, err := decodeAttachList(data, attachListOffset, glyphOrder)
		if err != nil {
			return GDEF{}, fmt.Errorf("GDEF AttachList: %w", err)
		}
		gdef.AttachList = attachList
	}

	if ligCaretListOffset != 0 {
		ligCaretList, err := decodeLigCaretList(data, ligCaretListOffset, glyphOrder)
		if err != nil {
			return GDEF{}, fmt.Errorf("GDEF LigCaretList: %w", err)
		}
		gdef.LigCaretList = ligCaretList
	}

	if markAttachClassDefOffset != 0 {
		classMap, err := decodeClassDef(data, markAttachClassDefOffset)
		if err != nil {
			return GDEF{}, fmt.Errorf("GDEF MarkAttachClassDef: %w", err)
		}
		gdef.MarkAttachClassMap = classMap
	}

	return gdef, nil
}

// GlyphClass returns the glyph class for the given glyph ID.
// Returns 0 (no class) if the glyph is not in the GlyphClassDef.
func (g GDEF) GlyphClass(glyphID uint16) uint16 {
	if g.GlyphClassMap == nil {
		return 0
	}
	return g.GlyphClassMap[glyphID]
}

// IsBase returns true if the glyph is classified as a base glyph.
func (g GDEF) IsBase(glyphID uint16) bool {
	return g.GlyphClass(glyphID) == ClassBase
}

// IsLigature returns true if the glyph is classified as a ligature glyph.
func (g GDEF) IsLigature(glyphID uint16) bool {
	return g.GlyphClass(glyphID) == ClassLigature
}

// IsMark returns true if the glyph is classified as a mark glyph.
func (g GDEF) IsMark(glyphID uint16) bool {
	return g.GlyphClass(glyphID) == ClassMark
}

// IsComponent returns true if the glyph is classified as a component glyph.
func (g GDEF) IsComponent(glyphID uint16) bool {
	return g.GlyphClass(glyphID) == ClassComponent
}

// MarkAttachClass returns the mark attachment class for the given glyph ID.
// Returns 0 if the glyph has no mark attachment class.
func (g GDEF) MarkAttachClass(glyphID uint16) uint16 {
	if g.MarkAttachClassMap == nil {
		return 0
	}
	return g.MarkAttachClassMap[glyphID]
}

// ClassName maps a GDEF class value to a human-readable name.
func ClassName(class uint16) string {
	switch class {
	case 0:
		return "NoClass"
	case ClassBase:
		return "Base"
	case ClassLigature:
		return "Ligature"
	case ClassMark:
		return "Mark"
	case ClassComponent:
		return "Component"
	default:
		return fmt.Sprintf("Unknown(%d)", class)
	}
}

// ClassDefByNames returns a map from glyph name to class value.
// Glyphs not in the class definition are omitted from the result.
func (g GDEF) ClassDefByNames(glyphOrder []string) map[string]uint16 {
	result := make(map[string]uint16)
	for glyphID, class := range g.GlyphClassMap {
		if int(glyphID) < len(glyphOrder) {
			result[glyphOrder[glyphID]] = class
		}
	}
	return result
}

// decodeClassDef decodes a ClassDef table (format 1 or 2) at the given offset.
func decodeClassDef(data []byte, offset int) (map[uint16]uint16, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("offset exceeds table")
	}
	format := readU16(data, offset)
	switch format {
	case 1:
		return decodeClassDefFormat1(data, offset)
	case 2:
		return decodeClassDefFormat2(data, offset)
	default:
		return nil, fmt.Errorf("unsupported ClassDef format %d", format)
	}
}

// ClassDef Format 1: startGlyphID + glyphCount + classValueArray.
func decodeClassDefFormat1(data []byte, offset int) (map[uint16]uint16, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("ClassDef format 1 too short")
	}
	startGlyphID := readU16(data, offset+2)
	glyphCount := int(readU16(data, offset+4))
	if glyphCount > MaxClassDefGlyphs {
		return nil, fmt.Errorf("ClassDef format 1 glyphCount %d exceeds limit %d", glyphCount, MaxClassDefGlyphs)
	}
	if offset+6+glyphCount*2 > len(data) {
		return nil, fmt.Errorf("ClassDef format 1 class values exceed table")
	}
	result := make(map[uint16]uint16, glyphCount)
	for i := 0; i < glyphCount; i++ {
		glyphID := startGlyphID + uint16(i)
		class := readU16(data, offset+6+i*2)
		if class != 0 {
			result[glyphID] = class
		}
	}
	return result, nil
}

// ClassDef Format 2: classRangeRecords.
func decodeClassDefFormat2(data []byte, offset int) (map[uint16]uint16, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("ClassDef format 2 too short")
	}
	rangeCount := int(readU16(data, offset+2))
	if rangeCount > MaxClassDefRanges {
		return nil, fmt.Errorf("ClassDef format 2 rangeCount %d exceeds limit %d", rangeCount, MaxClassDefRanges)
	}
	if offset+4+rangeCount*6 > len(data) {
		return nil, fmt.Errorf("ClassDef format 2 range records exceed table")
	}
	result := make(map[uint16]uint16)
	for i := 0; i < rangeCount; i++ {
		recOffset := offset + 4 + i*6
		startGlyphID := readU16(data, recOffset)
		endGlyphID := readU16(data, recOffset+2)
		class := readU16(data, recOffset+4)
		if endGlyphID < startGlyphID {
			return nil, fmt.Errorf("ClassDef format 2 range end %d before start %d", endGlyphID, startGlyphID)
		}
		if class != 0 {
			for glyphID := startGlyphID; ; glyphID++ {
				result[glyphID] = class
				if len(result) > MaxClassDefGlyphs {
					return nil, fmt.Errorf("ClassDef format 2 exceeds max glyphs %d", MaxClassDefGlyphs)
				}
				if glyphID == endGlyphID {
					break
				}
				// Guard against overflow when endGlyphID == 0xFFFF
				if glyphID == ^uint16(0) {
					break
				}
			}
		}
	}
	return result, nil
}

// decodeClassDefRecords decodes a ClassDef table and returns the raw records
// (format 2) or synthetic records (format 1).
func decodeClassDefRecords(data []byte, offset int) ([]ClassRangeRecord, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("offset exceeds table")
	}
	format := readU16(data, offset)
	switch format {
	case 1:
		return decodeClassDefFormat1Records(data, offset)
	case 2:
		return decodeClassDefFormat2Records(data, offset)
	default:
		return nil, fmt.Errorf("unsupported ClassDef format %d", format)
	}
}

func decodeClassDefFormat1Records(data []byte, offset int) ([]ClassRangeRecord, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("ClassDef format 1 too short")
	}
	startGlyphID := readU16(data, offset+2)
	glyphCount := int(readU16(data, offset+4))
	if glyphCount > MaxClassDefGlyphs {
		return nil, fmt.Errorf("ClassDef format 1 glyphCount %d exceeds limit", glyphCount)
	}
	if offset+6+glyphCount*2 > len(data) {
		return nil, fmt.Errorf("ClassDef format 1 class values exceed table")
	}
	records := make([]ClassRangeRecord, 0, glyphCount)
	for i := 0; i < glyphCount; i++ {
		class := readU16(data, offset+6+i*2)
		if class != 0 {
			records = append(records, ClassRangeRecord{
				StartGlyphID: startGlyphID + uint16(i),
				EndGlyphID:   startGlyphID + uint16(i),
				Class:        class,
			})
		}
	}
	return records, nil
}

func decodeClassDefFormat2Records(data []byte, offset int) ([]ClassRangeRecord, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("ClassDef format 2 too short")
	}
	rangeCount := int(readU16(data, offset+2))
	if rangeCount > MaxClassDefRanges {
		return nil, fmt.Errorf("ClassDef format 2 rangeCount %d exceeds limit", rangeCount)
	}
	if offset+4+rangeCount*6 > len(data) {
		return nil, fmt.Errorf("ClassDef format 2 range records exceed table")
	}
	records := make([]ClassRangeRecord, 0, rangeCount)
	for i := 0; i < rangeCount; i++ {
		recOffset := offset + 4 + i*6
		records = append(records, ClassRangeRecord{
			StartGlyphID: readU16(data, recOffset),
			EndGlyphID:   readU16(data, recOffset+2),
			Class:        readU16(data, recOffset+4),
		})
	}
	return records, nil
}

// decodeAttachList decodes an AttachList table at the given offset.
// AttachList layout:
//   +0: Coverage offset (uint16, from start of AttachList)
//   +2: GlyphCount (uint16)
//   +4: AttachPoint offsets[GlyphCount] (uint16, from start of AttachList)
func decodeAttachList(data []byte, offset int, glyphOrder []string) ([]AttachPoint, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("AttachList too short")
	}
	coverageOffset := int(readU16(data, offset))
	glyphCount := int(readU16(data, offset + 2))
	if glyphCount > MaxAttachListGlyphs {
		return nil, fmt.Errorf("AttachList glyphCount %d exceeds limit %d", glyphCount, MaxAttachListGlyphs)
	}
	if offset+4+glyphCount*2 > len(data) {
		return nil, fmt.Errorf("AttachList offsets exceed table")
	}

	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, fmt.Errorf("AttachList coverage: %w", err)
	}
	if len(coverage) != glyphCount {
		return nil, fmt.Errorf("AttachList coverage count %d != glyph count %d", len(coverage), glyphCount)
	}

	result := make([]AttachPoint, 0, glyphCount)
	for i := 0; i < glyphCount; i++ {
		attachPointOffset := int(readU16(data, offset+4+i*2))
		ap, err := decodeAttachPoint(data, offset+attachPointOffset)
		if err != nil {
			return nil, err
		}
		ap.GlyphID = coverage[i]
		result = append(result, ap)
	}
	return result, nil
}

func decodeAttachPoint(data []byte, offset int) (AttachPoint, error) {
	if offset+2 > len(data) {
		return AttachPoint{}, fmt.Errorf("AttachPoint too short")
	}
	pointCount := int(readU16(data, offset))
	if pointCount > MaxAttachPointCount {
		return AttachPoint{}, fmt.Errorf("AttachPoint count %d exceeds limit %d", pointCount, MaxAttachPointCount)
	}
	if offset+2+pointCount*2 > len(data) {
		return AttachPoint{}, fmt.Errorf("AttachPoint indices exceed table")
	}
	indices := make([]uint16, pointCount)
	for i := 0; i < pointCount; i++ {
		indices[i] = readU16(data, offset+2+i*2)
	}
	return AttachPoint{PointIndices: indices}, nil
}

// decodeLigCaretList decodes a LigCaretList table at the given offset.
// LigCaretList layout:
//   +0: Coverage offset (uint16, from start of LigCaretList)
//   +2: LigGlyphCount (uint16)
//   +4: LigGlyph offsets[LigGlyphCount] (uint16, from start of LigCaretList)
func decodeLigCaretList(data []byte, offset int, glyphOrder []string) ([]LigGlyph, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("LigCaretList too short")
	}
	coverageOffset := int(readU16(data, offset))
	ligGlyphCount := int(readU16(data, offset+2))
	if ligGlyphCount > MaxLigCaretCount {
		return nil, fmt.Errorf("LigCaretList count %d exceeds limit %d", ligGlyphCount, MaxLigCaretCount)
	}
	if offset+4+ligGlyphCount*2 > len(data) {
		return nil, fmt.Errorf("LigCaretList offsets exceed table")
	}

	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, fmt.Errorf("LigCaretList coverage: %w", err)
	}
	if len(coverage) != ligGlyphCount {
		return nil, fmt.Errorf("LigCaretList coverage count %d != glyph count %d", len(coverage), ligGlyphCount)
	}

	result := make([]LigGlyph, 0, ligGlyphCount)
	for i := 0; i < ligGlyphCount; i++ {
		ligGlyphOffset := int(readU16(data, offset+4+i*2))
		lg, err := decodeLigGlyph(data, offset+ligGlyphOffset)
		if err != nil {
			return nil, err
		}
		lg.GlyphID = coverage[i]
		result = append(result, lg)
	}
	return result, nil
}

func decodeLigGlyph(data []byte, offset int) (LigGlyph, error) {
	if offset+2 > len(data) {
		return LigGlyph{}, fmt.Errorf("LigGlyph too short")
	}
	caretCount := int(readU16(data, offset))
	if caretCount > MaxLigCaretValues {
		return LigGlyph{}, fmt.Errorf("LigGlyph caretCount %d exceeds limit %d", caretCount, MaxLigCaretValues)
	}
	// CaretValue table: offset array starts at offset+2, each entry is 2 bytes
	// pointing to a CaretValue table.
	if offset+2+caretCount*2 > len(data) {
		return LigGlyph{}, fmt.Errorf("LigGlyph caret offsets exceed table")
	}
	carets := make([]int16, 0, caretCount)
	for i := 0; i < caretCount; i++ {
		caretValueOffset := int(readU16(data, offset+2+i*2))
		caretValue, err := decodeCaretValue(data, offset+caretValueOffset)
		if err != nil {
			return LigGlyph{}, err
		}
		carets = append(carets, caretValue)
	}
	return LigGlyph{CaretOffsets: carets}, nil
}

// decodeCaretValue decodes a CaretValue table (format 1 or 2; format 3
// decodes the Coordinate only).
func decodeCaretValue(data []byte, offset int) (int16, error) {
	if offset+4 > len(data) {
		return 0, fmt.Errorf("CaretValue too short")
	}
	format := readU16(data, offset)
	switch format {
	case 1:
		// CaretValueFormat1: Coordinate (int16)
		return int16(readU16(data, offset+2)), nil
	case 2:
		// CaretValueFormat2: CaretValuePoint (uint16)
		// We return it as an int16; the caller can interpret as point index.
		return int16(readU16(data, offset+2)), nil
	case 3:
		// CaretValueFormat3: Coordinate + DeviceTable offset
		if offset+6 > len(data) {
			return 0, fmt.Errorf("CaretValue format 3 too short")
		}
		return int16(readU16(data, offset+2)), nil
	default:
		return 0, fmt.Errorf("unsupported CaretValue format %d", format)
	}
}

// ClassDefRecords decodes and returns the GlyphClassDef as raw ClassRangeRecords.
// This is useful when you need the range structure rather than the expanded map.
func (g GDEF) ClassDefRecords(data []byte) ([]ClassRangeRecord, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("GDEF header too short")
	}
	glyphClassDefOffset := int(readU16(data, 4))
	if glyphClassDefOffset == 0 {
		return nil, nil
	}
	return decodeClassDefRecords(data, glyphClassDefOffset)
}

// MarkAttachClassDefRecords decodes and returns MarkAttachClassDef as raw ClassRangeRecords.
func (g GDEF) MarkAttachClassDefRecords(data []byte) ([]ClassRangeRecord, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("GDEF header too short")
	}
	markAttachClassDefOffset := int(readU16(data, 10))
	if markAttachClassDefOffset == 0 {
		return nil, nil
	}
	return decodeClassDefRecords(data, markAttachClassDefOffset)
}

// decodeCoverage decodes a Coverage table at the given offset.
// Delegates to the canonical opentype.DecodeCoverage.
func decodeCoverage(data []byte, offset int) ([]uint16, error) {
	return opentype.DecodeCoverage(data, offset)
}

// readU16 reads a big-endian uint16 from data at offset.
func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}