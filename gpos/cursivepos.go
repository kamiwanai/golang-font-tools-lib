// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gpos

import "fmt"

// CursiveEntry describes one resolved cursive attachment entry for a glyph.
type CursiveEntry struct {
	// Glyph is the glyph name.
	Glyph string
	// EntryAnchor is the entry anchor point on this glyph (where the
	// previous glyph's exit connects). Zero if no entry anchor.
	EntryAnchor Anchor
	// ExitAnchor is the exit anchor point on this glyph (where the next
	// glyph's entry connects). Zero if no exit anchor.
	ExitAnchor Anchor
	// LookupIndex is the zero-based lookup index in the GPOS LookupList.
	LookupIndex int
}

// DecodeCursivePosLookups decodes GPOS CursivePos lookups (type 3) from a
// GPOS table. It returns a flat list of all resolved cursive attachment entries.
//
// glyphOrder maps glyph IDs to names. The function handles ExtensionPos
// lookups (type 9) targeting CursivePos.
func DecodeCursivePosLookups(data []byte, glyphOrder []string) ([]CursiveEntry, error) {
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

	var all []CursiveEntry
	for lookupIndex := 0; lookupIndex < lookupCount; lookupIndex++ {
		lookupOffset := lookupListOffset + int(readU16(data, lookupListOffset+2+lookupIndex*2))
		entries, err := decodeCursiveLookup(data, glyphOrder, lookupOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
	}
	return all, nil
}

func decodeCursiveLookup(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int) ([]CursiveEntry, error) {
	if lookupOffset+6 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d exceeds table", lookupIndex)
	}
	lookupType := readU16(data, lookupOffset)
	if lookupType != 3 && lookupType != 9 {
		return nil, nil
	}
	subtableCount := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("GPOS lookup %d subtable offsets exceed table", lookupIndex)
	}

	var all []CursiveEntry
	for i := 0; i < subtableCount; i++ {
		subtableOffset := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		entries, err := decodeCursiveSubtable(data, glyphOrder, lookupType, subtableOffset, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
	}
	return all, nil
}

func decodeCursiveSubtable(data []byte, glyphOrder []string, lookupType uint16, subtableOffset int, lookupIndex int) ([]CursiveEntry, error) {
	if subtableOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS cursive subtable exceeds table")
	}
	if lookupType == 9 {
		return decodeCursiveExtensionSubtable(data, glyphOrder, subtableOffset, lookupIndex)
	}
	format := readU16(data, subtableOffset)
	if format != 1 {
		return nil, nil
	}
	return decodeCursivePosFormat1(data, glyphOrder, subtableOffset, lookupIndex)
}

func decodeCursiveExtensionSubtable(data []byte, glyphOrder []string, subtableOffset int, lookupIndex int) ([]CursiveEntry, error) {
	if subtableOffset+8 > len(data) {
		return nil, fmt.Errorf("GPOS cursive extension subtable exceeds table")
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
		return nil, fmt.Errorf("GPOS cursive extension target exceeds table")
	}
	if targetOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS cursive extension target exceeds table")
	}
	targetFormat := readU16(data, targetOffset)
	if targetFormat != 1 {
		return nil, nil
	}
	return decodeCursivePosFormat1(data, glyphOrder, targetOffset, lookupIndex)
}

// decodeCursivePosFormat1 decodes a CursivePosFormat1 subtable.
//
// Layout:
//
//	PosFormat      uint16  (2 bytes) = 1
//	CoverageOffset uint16  (2 bytes) offset from start of subtable
//	EntryExitCount uint16  (2 bytes)
//	EntryExitRecord[EntryExitCount]:
//	  EntryAnchorOffset  uint16  (2 bytes) offset from start of subtable, 0 if NULL
//	  ExitAnchorOffset   uint16  (2 bytes) offset from start of subtable, 0 if NULL
func decodeCursivePosFormat1(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]CursiveEntry, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("CursivePosFormat1 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	entryExitCount := int(readU16(data, offset+4))

	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, fmt.Errorf("CursivePosFormat1 coverage: %w", err)
	}
	if len(coverage) != entryExitCount {
		return nil, fmt.Errorf("CursivePosFormat1 coverage count %d != entryExitCount %d",
			len(coverage), entryExitCount)
	}

	recordsOffset := offset + 6
	if recordsOffset+entryExitCount*4 > len(data) {
		return nil, fmt.Errorf("CursivePosFormat1 EntryExit records exceed table")
	}

	entries := make([]CursiveEntry, 0, entryExitCount)
	for i := 0; i < entryExitCount; i++ {
		glyphID := coverage[i]
		glyph, ok := glyphName(glyphOrder, glyphID)
		if !ok {
			continue
		}
		recOff := recordsOffset + i*4
		entryAnchorOff := int(readU16(data, recOff))
		exitAnchorOff := int(readU16(data, recOff+2))

		var entryAnchor Anchor
		if entryAnchorOff != 0 {
			entryAnchor, err = decodeAnchor(data, offset+entryAnchorOff)
			if err != nil {
				return nil, fmt.Errorf("CursivePosFormat1 glyph %s entry anchor: %w", glyph, err)
			}
		}

		var exitAnchor Anchor
		if exitAnchorOff != 0 {
			exitAnchor, err = decodeAnchor(data, offset+exitAnchorOff)
			if err != nil {
				return nil, fmt.Errorf("CursivePosFormat1 glyph %s exit anchor: %w", glyph, err)
			}
		}

		entries = append(entries, CursiveEntry{
			Glyph:       glyph,
			EntryAnchor: entryAnchor,
			ExitAnchor:  exitAnchor,
			LookupIndex: lookupIndex,
		})
	}
	return entries, nil
}
