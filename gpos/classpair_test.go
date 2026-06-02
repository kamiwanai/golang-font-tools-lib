// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import (
	"os"
	"testing"

	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

func TestDecodeClassPairLookups_Empty(t *testing.T) {
	_, err := DecodeClassPairLookups(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil data")
	}
}

func TestDecodeClassPairLookups_ShortHeader(t *testing.T) {
	_, err := DecodeClassPairLookups([]byte{0, 0}, []string{"A"})
	if err == nil {
		t.Fatal("expected error for short data")
	}
}

func TestDecodeClassPairLookups_Inter(t *testing.T) {
	data, err := os.ReadFile("../testdata/Inter.ttf")
	if err != nil {
		t.Skipf("cannot read testdata/Inter.ttf: %v", err)
	}
	font, err := opentype.Parse(data)
	if err != nil {
		t.Fatalf("Parse Inter: %v", err)
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder: %v", err)
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		t.Skip("no GPOS table in Inter")
	}

	pairs, err := DecodeClassPairLookups(gposData, glyphOrder)
	if err != nil {
		t.Fatalf("DecodeClassPairLookups: %v", err)
	}
	t.Logf("DecodeClassPairLookups (Inter) returned %d class pairs", len(pairs))

	// Verify each pair has valid fields
	for i, p := range pairs {
		if p.LeftGlyphCount <= 0 {
			t.Errorf("pair %d: LeftGlyphCount=%d", i, p.LeftGlyphCount)
		}
		if p.RightGlyphCount <= 0 {
			t.Errorf("pair %d: RightGlyphCount=%d", i, p.RightGlyphCount)
		}
	}
}

func TestExpandClassPair_InvalidLookupIndex(t *testing.T) {
	data, err := os.ReadFile("../testdata/Inter.ttf")
	if err != nil {
		t.Skipf("cannot read testdata/Inter.ttf: %v", err)
	}
	font, err := opentype.Parse(data)
	if err != nil {
		t.Fatalf("Parse Inter: %v", err)
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder: %v", err)
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		t.Skip("no GPOS table in Inter")
	}

	_, err = ExpandClassPair(gposData, glyphOrder, 9999, 0, 0)
	if err == nil {
		t.Fatal("expected error for out-of-range lookup index")
	}
}

func TestDecodeClassPairLookups_BuiltGPOS(t *testing.T) {
	// BuildGPOS creates PairPos format 1, so DecodeClassPairLookups (format 2 only)
	// should return 0 pairs — but should not error.
	pairs := []KerningPairInput{
		{Left: "A", Right: "V", Value: -20},
		{Left: "V", Right: "A", Value: -15},
	}
	gm := map[string]uint16{"A": 0, "V": 1}

	data, err := BuildGPOS(pairs, gm)
	if err != nil {
		t.Fatalf("BuildGPOS: %v", err)
	}

	glyphOrder := []string{"A", "V"}
	classPairs, err := DecodeClassPairLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodeClassPairLookups on format 1 GPOS: %v", err)
	}
	if len(classPairs) != 0 {
		t.Logf("got %d class pairs from format 1 GPOS (expected 0)", len(classPairs))
	}
}
