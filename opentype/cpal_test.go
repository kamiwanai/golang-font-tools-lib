// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"testing"
)

func buildCPALTableV0(numEntries uint16, paletteStarts []uint16, colors []paletteColorFixture) []byte {
	// Header: version(2) + numPaletteEntries(2) + numPalettes(2) + numColorRecords(2) + colorRecordsOffset(4)
	// Palette indices: uint16[numPalettes]
	// Color records: 4 bytes each (B,G,R,A)
	numPalettes := len(paletteStarts)
	numColors := len(colors)
	headerSize := 12
	indicesSize := numPalettes * 2
	colorsSize := numColors * 4
	colorRecOff := headerSize + indicesSize

	data := make([]byte, headerSize+indicesSize+colorsSize)
	binary.BigEndian.PutUint16(data[0:2], 0)           // version
	binary.BigEndian.PutUint16(data[2:4], numEntries)   // numPaletteEntries
	binary.BigEndian.PutUint16(data[4:6], uint16(numPalettes))
	binary.BigEndian.PutUint16(data[6:8], uint16(numColors))
	binary.BigEndian.PutUint32(data[8:12], uint32(colorRecOff))

	for i, start := range paletteStarts {
		binary.BigEndian.PutUint16(data[12+i*2:14+i*2], start)
	}
	for i, c := range colors {
		off := colorRecOff + i*4
		data[off] = c.B
		data[off+1] = c.G
		data[off+2] = c.R
		data[off+3] = c.A
	}
	return data
}

type paletteColorFixture struct {
	B, G, R, A uint8
}

func TestCPALParsesSinglePalette(t *testing.T) {
	cpal := buildCPALTableV0(2,
		[]uint16{0},
		[]paletteColorFixture{
			{B: 255, G: 0, R: 0, A: 255},    // red
			{B: 0, G: 255, R: 0, A: 128},    // green, half-transparent
		},
	)
	data := buildFont(map[string][]byte{"CPAL": cpal})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	table, err := font.CPAL()
	if err != nil {
		t.Fatalf("CPAL: %v", err)
	}
	if table == nil {
		t.Fatal("CPAL returned nil")
	}
	if table.Version != 0 {
		t.Fatalf("Version = %d, want 0", table.Version)
	}
	if table.NumPaletteEntries != 2 {
		t.Fatalf("NumPaletteEntries = %d, want 2", table.NumPaletteEntries)
	}
	if len(table.Palettes) != 1 {
		t.Fatalf("Palettes count = %d, want 1", len(table.Palettes))
	}
	p := table.Palettes[0]
	if p.Index != 0 {
		t.Fatalf("Index = %d, want 0", p.Index)
	}
	if len(p.Colors) != 2 {
		t.Fatalf("Colors count = %d, want 2", len(p.Colors))
	}
	if p.Colors[0].Red != 0 || p.Colors[0].Green != 0 || p.Colors[0].Blue != 255 || p.Colors[0].Alpha != 255 {
		t.Fatalf("Color[0] = %+v, want {255 0 0 255}", p.Colors[0])
	}
	if p.Colors[1].Red != 0 || p.Colors[1].Green != 255 || p.Colors[1].Blue != 0 || p.Colors[1].Alpha != 128 {
		t.Fatalf("Color[1] = %+v, want {0 255 0 128}", p.Colors[1])
	}
}

func TestCPALMultiplePalettes(t *testing.T) {
	cpal := buildCPALTableV0(2,
		[]uint16{0, 2}, // palette 0: colors[0:2], palette 1: colors[2:4]
		[]paletteColorFixture{
			{B: 255, G: 0, R: 0, A: 255},    // p0: red
			{B: 0, G: 255, R: 0, A: 255},    // p0: green
			{B: 0, G: 0, R: 255, A: 255},    // p1: blue
			{B: 128, G: 128, R: 128, A: 255}, // p1: gray
		},
	)
	data := buildFont(map[string][]byte{"CPAL": cpal})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	table, err := font.CPAL()
	if err != nil {
		t.Fatalf("CPAL: %v", err)
	}
	if len(table.Palettes) != 2 {
		t.Fatalf("Palettes count = %d, want 2", len(table.Palettes))
	}
	if table.Palettes[0].Index != 0 || table.Palettes[1].Index != 1 {
		t.Fatal("Palette indices wrong")
	}
	if table.Palettes[0].Colors[0].Blue != 255 {
		t.Fatal("Palette 0 color 0 wrong")
	}
	if table.Palettes[1].Colors[0].Red != 255 {
		t.Fatal("Palette 1 color 0 wrong")
	}
}

func TestCPALMissingTable(t *testing.T) {
	data := buildFont(map[string][]byte{})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	table, err := font.CPAL()
	if err != nil {
		t.Fatalf("CPAL error: %v", err)
	}
	if table != nil {
		t.Fatal("CPAL should be nil for missing table")
	}
}

func TestCPALTruncated(t *testing.T) {
	data := buildFont(map[string][]byte{"CPAL": {0, 0}})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.CPAL()
	if err == nil {
		t.Fatal("CPAL should fail for truncated table")
	}
}

func TestCPALUnsupportedVersion(t *testing.T) {
	cpal := make([]byte, 14)
	binary.BigEndian.PutUint16(cpal[0:2], 99) // unsupported
	data := buildFont(map[string][]byte{"CPAL": cpal})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.CPAL()
	if err == nil {
		t.Fatal("CPAL should fail for unsupported version")
	}
}

func TestCPALEmptyPalette(t *testing.T) {
	cpal := buildCPALTableV0(3,
		[]uint16{0},
		[]paletteColorFixture{}, // no colors, but palette expects 3
	)
	data := buildFont(map[string][]byte{"CPAL": cpal})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Should still parse — empty colors slice for palette
	table, err := font.CPAL()
	if err != nil {
		t.Fatalf("CPAL: %v", err)
	}
	if table == nil {
		t.Fatal("CPAL returned nil")
	}
	if len(table.Palettes[0].Colors) != 0 {
		t.Fatalf("Expected empty colors, got %d", len(table.Palettes[0].Colors))
	}
}

func TestCPALBGRAOrder(t *testing.T) {
	// CPAL stores in BGRA order (Blue first)
	cpal := buildCPALTableV0(1,
		[]uint16{0},
		[]paletteColorFixture{
			{B: 0x12, G: 0x34, R: 0x56, A: 0x78},
		},
	)
	data := buildFont(map[string][]byte{"CPAL": cpal})
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	table, err := font.CPAL()
	if err != nil {
		t.Fatalf("CPAL: %v", err)
	}
	c := table.Palettes[0].Colors[0]
	if c.Blue != 0x12 || c.Green != 0x34 || c.Red != 0x56 || c.Alpha != 0x78 {
		t.Fatalf("Color = %+v, want {Blue:0x12 Green:0x34 Red:0x56 Alpha:0x78}", c)
	}
}
