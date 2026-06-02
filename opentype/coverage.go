// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import "fmt"

// MaxCoverageGlyphs limits the number of glyphs expanded from a Coverage
// format 2 range table. Applies to both format 1 (glyphCount) and format 2
// (total expanded glyphs).
var MaxCoverageGlyphs int = 65536

// MaxCoverageRanges limits the number of range records in a Coverage format 2.
var MaxCoverageRanges int = 65536

// DecodeCoverage decodes a Coverage table (format 1 or 2) at the given offset.
// Format 1 is a list of glyph IDs; format 2 is an array of start..end ranges.
// Bounds are enforced via MaxCoverageGlyphs and MaxCoverageRanges to prevent
// denial-of-service from malformed input.
func DecodeCoverage(data []byte, offset int) ([]uint16, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("Coverage offset exceeds table")
	}
	format := readU16(data, offset)
	switch format {
	case 1:
		glyphCount := int(readU16(data, offset+2))
		if glyphCount > MaxCoverageGlyphs {
			return nil, fmt.Errorf("Coverage format 1 glyphCount %d exceeds limit %d", glyphCount, MaxCoverageGlyphs)
		}
		if offset+4+glyphCount*2 > len(data) {
			return nil, fmt.Errorf("Coverage format 1 exceeds table")
		}
		glyphs := make([]uint16, glyphCount)
		for i := 0; i < glyphCount; i++ {
			glyphs[i] = readU16(data, offset+4+i*2)
		}
		return glyphs, nil
	case 2:
		rangeCount := int(readU16(data, offset+2))
		if rangeCount > MaxCoverageRanges {
			return nil, fmt.Errorf("Coverage format 2 rangeCount %d exceeds limit %d", rangeCount, MaxCoverageRanges)
		}
		if offset+4+rangeCount*6 > len(data) {
			return nil, fmt.Errorf("Coverage format 2 exceeds table")
		}
		var glyphs []uint16
		for i := 0; i < rangeCount; i++ {
			rangeOffset := offset + 4 + i*6
			start := readU16(data, rangeOffset)
			end := readU16(data, rangeOffset+2)
			if end < start {
				return nil, fmt.Errorf("Coverage range end before start")
			}
			for glyphID := start; glyphID <= end; glyphID++ {
				glyphs = append(glyphs, glyphID)
				if len(glyphs) > MaxCoverageGlyphs {
					return nil, fmt.Errorf("Coverage format 2 exceeds max glyphs %d", MaxCoverageGlyphs)
				}
				if glyphID == ^uint16(0) {
					break
				}
			}
		}
		return glyphs, nil
	default:
		return nil, fmt.Errorf("unsupported Coverage format %d", format)
	}
}
