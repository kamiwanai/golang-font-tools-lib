// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// ColorLayer is one layer in a COLRv0 color glyph.
type ColorLayer struct {
	// LayerGlyphID is the glyph ID that provides the shape for this layer.
	LayerGlyphID uint16
	// PaletteIndex is the index into the CPAL palette for this layer's color.
	PaletteIndex uint16
}

// ColorGlyph is one COLR base glyph with its color layers.
type ColorGlyph struct {
	// BaseGlyphID is the base glyph that carries the COLR color definition.
	BaseGlyphID uint16
	// BaseGlyph is the glyph name.
	BaseGlyph string
	// Layers is the ordered list of color layers for this base glyph.
	Layers []ColorLayer
}

// COLRTable holds the parsed COLR table data.
type COLRTable struct {
	// Version is the COLR table version (0 or 1).
	Version uint16
	// BaseGlyphs lists all base glyphs with their color layers.
	BaseGlyphs []ColorGlyph
}

// COLR parses the COLR table from the font. Returns nil, nil if the font
// has no COLR table.
func (font *Font) COLR() (*COLRTable, error) {
	data, err := font.TableData("COLR")
	if err != nil {
		return nil, nil
	}
	if len(data) < 14 {
		return nil, fmt.Errorf("COLR table too short: %d bytes", len(data))
	}

	version := binary.BigEndian.Uint16(data[0:])
	if version > 1 {
		return nil, fmt.Errorf("COLR version %d not supported", version)
	}

	numBaseGlyphRecords := int(binary.BigEndian.Uint16(data[2:]))
	baseGlyphRecordsOffset := int(binary.BigEndian.Uint32(data[4:]))
	layerRecordsOffset := int(binary.BigEndian.Uint32(data[8:]))
	numLayerRecords := int(binary.BigEndian.Uint16(data[12:]))

	return parseCOLRv0(data, numBaseGlyphRecords, baseGlyphRecordsOffset,
		layerRecordsOffset, numLayerRecords, version)
}

func parseCOLRv0(data []byte, numBaseGlyphs, baseOff, layerOff, numLayers int, version uint16) (*COLRTable, error) {
	// Read layer records first
	layers := make([]ColorLayer, numLayers)
	for i := 0; i < numLayers; i++ {
		off := layerOff + i*4
		if off+4 > len(data) {
			return nil, fmt.Errorf("COLR layer record %d exceeds table", i)
		}
		layers[i] = ColorLayer{
			LayerGlyphID: binary.BigEndian.Uint16(data[off:]),
			PaletteIndex: binary.BigEndian.Uint16(data[off+2:]),
		}
	}

	// Read base glyph records
	baseGlyphs := make([]ColorGlyph, 0, numBaseGlyphs)
	for i := 0; i < numBaseGlyphs; i++ {
		off := baseOff + i*6
		if off+6 > len(data) {
			return nil, fmt.Errorf("COLR base glyph record %d exceeds table", i)
		}
		glyphID := binary.BigEndian.Uint16(data[off:])
		firstLayer := int(binary.BigEndian.Uint16(data[off+2:]))
		nLayers := int(binary.BigEndian.Uint16(data[off+4:]))

		if firstLayer+nLayers > len(layers) {
			return nil, fmt.Errorf("COLR base glyph %d layers exceed layer count", i)
		}

		baseGlyphs = append(baseGlyphs, ColorGlyph{
			BaseGlyphID: glyphID,
			BaseGlyph:   fmt.Sprintf("glyph%05d", glyphID),
			Layers:      layers[firstLayer : firstLayer+nLayers],
		})
	}

	return &COLRTable{
		Version:    version,
		BaseGlyphs: baseGlyphs,
	}, nil
}

// COLRWithNames returns COLR data with resolved glyph names.
func (font *Font) COLRWithNames() (*COLRTable, error) {
	colr, err := font.COLR()
	if err != nil || colr == nil {
		return colr, err
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return colr, nil
	}
	for i := range colr.BaseGlyphs {
		gid := colr.BaseGlyphs[i].BaseGlyphID
		if int(gid) < len(glyphOrder) {
			colr.BaseGlyphs[i].BaseGlyph = glyphOrder[gid]
		}
	}
	return colr, nil
}
