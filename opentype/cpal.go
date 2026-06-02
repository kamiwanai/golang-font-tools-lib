// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// PaletteColor is one BGRA color entry in a CPAL palette.
type PaletteColor struct {
	Blue  uint8
	Green uint8
	Red   uint8
	Alpha uint8
}

// Palette is one color palette from the CPAL table.
type Palette struct {
	// Index is the zero-based palette index.
	Index int
	// Colors is the list of colors in this palette.
	Colors []PaletteColor
}

// CPALTable holds the parsed CPAL table data.
type CPALTable struct {
	// Version is the CPAL table version (0 or 1).
	Version uint16
	// NumPaletteEntries is the number of entries per palette.
	NumPaletteEntries uint16
	// Palettes is the list of palettes.
	Palettes []Palette
}

// CPAL parses the CPAL table from the font. Returns nil, nil if the font
// has no CPAL table.
func (font *Font) CPAL() (*CPALTable, error) {
	data, err := font.TableData("CPAL")
	if err != nil {
		return nil, nil
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("CPAL table too short: %d bytes", len(data))
	}

	version := binary.BigEndian.Uint16(data[0:])
	if version > 1 {
		return nil, fmt.Errorf("CPAL version %d not supported", version)
	}

	numPaletteEntries := binary.BigEndian.Uint16(data[2:])
	numPalettes := int(binary.BigEndian.Uint16(data[4:]))
	numColorRecords := int(binary.BigEndian.Uint16(data[6:]))
	colorRecordsOffset := int(binary.BigEndian.Uint32(data[8:]))

	if numPalettes > 256 {
		return nil, fmt.Errorf("CPAL numPalettes %d exceeds limit", numPalettes)
	}

	// Read palette type array (uint32 per palette, for v1+)
	// For v0, there is no palette type array (offset is at data[12:] which is colorRecordIndices)
	// format: version 0 has colorRecordIndices at offset 12 (uint16[numPalettes])
	// version 1 has: paletteLabelsOffset(uint32), paletteEntryLabelsOffset(uint32), paletteTypesOffset(uint32)
	// We parse palette indices the same way for v0; v1 has extra metadata.

	var paletteFirstColorIndex []int
	if version == 0 {
		// Format 0: colorRecordIndices at offset 12
		if 12+numPalettes*2 > len(data) {
			return nil, fmt.Errorf("CPAL colorRecordIndices exceed table")
		}
		paletteFirstColorIndex = make([]int, numPalettes)
		for i := 0; i < numPalettes; i++ {
			paletteFirstColorIndex[i] = int(binary.BigEndian.Uint16(data[12+i*2:]))
		}
	} else {
		// Version 1: paletteLabelsOffset(12), paletteEntryLabelsOffset(16),
		// paletteTypesOffset(20), then colorRecordIndices[uint16] at offset 24.
		paletteTypesOffset := uint32(0)
		if len(data) >= 24 {
			paletteTypesOffset = binary.BigEndian.Uint32(data[20:])
		}
		if paletteTypesOffset != 0 {
			return nil, fmt.Errorf("CPAL v1 palette types not supported")
		}
		if 24+numPalettes*2 > len(data) {
			return nil, fmt.Errorf("CPAL v1 colorRecordIndices exceed table")
		}
		paletteFirstColorIndex = make([]int, numPalettes)
		for i := 0; i < numPalettes; i++ {
			paletteFirstColorIndex[i] = int(binary.BigEndian.Uint16(data[24+i*2:]))
		}
	}

	// Read all color records from the table
	allColors := make([]PaletteColor, numColorRecords)
	for i := 0; i < numColorRecords; i++ {
		off := colorRecordsOffset + i*4
		if off+4 > len(data) {
			return nil, fmt.Errorf("CPAL color record %d exceeds table", i)
		}
		allColors[i] = PaletteColor{
			Blue:  data[off],
			Green: data[off+1],
			Red:   data[off+2],
			Alpha: data[off+3],
		}
	}

	// Build per-palette color slices
	palettes := make([]Palette, numPalettes)
	for i := 0; i < numPalettes; i++ {
		start := paletteFirstColorIndex[i]
		end := start + int(numPaletteEntries)
		if end > len(allColors) {
			end = len(allColors)
		}
		colors := make([]PaletteColor, 0, int(numPaletteEntries))
		if start < len(allColors) {
			colors = append(colors, allColors[start:end]...)
		}
		palettes[i] = Palette{
			Index:  i,
			Colors: colors,
		}
	}

	return &CPALTable{
		Version:           version,
		NumPaletteEntries: numPaletteEntries,
		Palettes:          palettes,
	}, nil
}
