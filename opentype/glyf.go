// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// GlyphBBox is the bounding box of a single glyph, read from the glyf table
// header. These are the same values reported in the OpenType 'head' table's
// font-wide bbox, but broken down per glyph.
type GlyphBBox struct {
	// GlyphIndex is the zero-based glyph ID.
	GlyphIndex int    `json:"glyphIndex"`
	// XMin is the minimum x coordinate in font units.
	XMin int16 `json:"xMin"`
	// YMin is the minimum y coordinate in font units.
	YMin int16 `json:"yMin"`
	// XMax is the maximum x coordinate in font units.
	XMax int16 `json:"xMax"`
	// YMax is the maximum y coordinate in font units.
	YMax int16 `json:"yMax"`
}

// GlyphBBoxes returns per-glyph bounding boxes from the glyf table.
//
// The bounding box is read from each glyph's header — no contour data is
// decoded. Empty glyphs (no outline data) return a zero bbox with XMin=XMax
// and YMin=YMax. CFF/CFF2 fonts have no glyf table and will return an error
// (use CFFGlyphBBoxes for those).
func (font *Font) GlyphBBoxes() ([]GlyphBBox, error) {
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return nil, fmt.Errorf("glyf bbox: %w", err)
	}
	if numGlyphs == 0 {
		return nil, nil
	}

	// Read loca (glyph location) table.
	locaData, err := font.TableData("loca")
	if err != nil {
		return nil, fmt.Errorf("glyf bbox: loca table: %w", err)
	}

	glyfData, err := font.TableData("glyf")
	if err != nil {
		return nil, fmt.Errorf("glyf bbox: glyf table: %w", err)
	}

	// Determine loca format from head.IndexToLocFormat.
	head, err := font.Head()
	if err != nil {
		return nil, fmt.Errorf("glyf bbox: head table: %w", err)
	}

	entries := numGlyphs + 1
	locaOffsets := make([]int, entries)

	if head.IndexToLocFormat == 0 {
		// Short format: uint16 entries * 2
		if len(locaData) < entries*2 {
			return nil, fmt.Errorf("loca table too short for %d entries (short format)", entries)
		}
		for i := 0; i < entries; i++ {
			locaOffsets[i] = int(binary.BigEndian.Uint16(locaData[i*2:])) * 2
		}
	} else {
		// Long format: uint32 entries
		if len(locaData) < entries*4 {
			return nil, fmt.Errorf("loca table too short for %d entries (long format)", entries)
		}
		for i := 0; i < entries; i++ {
			locaOffsets[i] = int(binary.BigEndian.Uint32(locaData[i*4:]))
		}
	}

	bboxes := make([]GlyphBBox, numGlyphs)
	for i := 0; i < numGlyphs; i++ {
		bboxes[i].GlyphIndex = i

		offset := locaOffsets[i]
		nextOffset := locaOffsets[i+1]

		// Empty glyph: no outline data
		if offset >= nextOffset || offset+10 > len(glyfData) {
			continue
		}

		// Compound glyphs (numberOfContours < 0) have no stored bbox; the
		// 8 bytes at offset+2..offset+9 are component flags/args, not a bbox.
		if int16(binary.BigEndian.Uint16(glyfData[offset:])) < 0 {
			continue
		}

		// Simple glyph: numberOfContours at offset 0, bbox at offset 2-9.
		bboxes[i].XMin = int16(binary.BigEndian.Uint16(glyfData[offset+2:]))
		bboxes[i].YMin = int16(binary.BigEndian.Uint16(glyfData[offset+4:]))
		bboxes[i].XMax = int16(binary.BigEndian.Uint16(glyfData[offset+6:]))
		bboxes[i].YMax = int16(binary.BigEndian.Uint16(glyfData[offset+8:]))
	}

	return bboxes, nil
}
