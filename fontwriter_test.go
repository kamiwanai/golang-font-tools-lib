// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package fonttools

import (
	"os"
	"testing"

	"github.com/kamiwanai/golang-font-tools-lib/gpos"
)

func TestApplyKerningToFont_BuildsValidGPOS(t *testing.T) {
	data, err := os.ReadFile("testdata/Inter.ttf")
	if err != nil {
		t.Skipf("cannot read testdata/Inter.ttf: %v", err)
	}

	pairs := []gpos.KerningPairInput{
		{Left: "A", Right: "V", Value: -50},
	}

	result, err := ApplyKerningToFont(data, pairs)
	if err != nil {
		t.Fatalf("ApplyKerningToFont: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("result is empty")
	}

	// Verify result is parseable
	font, err := Parse(result)
	if err != nil {
		t.Fatalf("Parse result: %v", err)
	}

	// Verify the result font has a GPOS table
	_, err = font.TableData("GPOS")
	if err != nil {
		t.Fatal("result font has no GPOS table")
	}

	t.Logf("ApplyKerningToFont: Inter.ttf + 1 pair = %d bytes", len(result))
}

func TestApplyKerningToFont_EmptyPairs(t *testing.T) {
	data, err := os.ReadFile("testdata/Inter.ttf")
	if err != nil {
		t.Skipf("cannot read testdata/Inter.ttf: %v", err)
	}

	result, err := ApplyKerningToFont(data, nil)
	if err != nil {
		t.Fatalf("ApplyKerningToFont with nil pairs: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("result is empty")
	}
	// Should still be a valid font
	_, err = Parse(result)
	if err != nil {
		t.Fatalf("Parse result: %v", err)
	}
}

func TestBuildGlyphMap(t *testing.T) {
	data, err := os.ReadFile("testdata/Inter.ttf")
	if err != nil {
		t.Skipf("cannot read testdata/Inter.ttf: %v", err)
	}
	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	gm, err := buildGlyphMap(font.Font)
	if err != nil {
		t.Fatalf("buildGlyphMap: %v", err)
	}
	if len(gm) == 0 {
		t.Fatal("empty glyph map")
	}
	t.Logf("glyph map: %d entries", len(gm))
}
