// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import (
	"testing"
)

func TestBuildGPOS_Empty(t *testing.T) {
	_, err := BuildGPOS(nil, nil)
	if err == nil {
		t.Fatal("expected error for empty pairs")
	}
}

func TestBuildGPOS_MissingGlyph(t *testing.T) {
	pairs := []KerningPairInput{
		{Left: "A", Right: "V", Value: -20},
	}
	gm := map[string]uint16{"A": 0}
	_, err := BuildGPOS(pairs, gm)
	if err == nil {
		t.Fatal("expected error for missing right glyph")
	}
}

func TestBuildGPOS_SinglePair(t *testing.T) {
	pairs := []KerningPairInput{
		{Left: "A", Right: "V", Value: -20},
	}
	gm := map[string]uint16{"A": 0, "V": 1}

	data, err := BuildGPOS(pairs, gm)
	if err != nil {
		t.Fatalf("BuildGPOS: %v", err)
	}
	if len(data) < 10 {
		t.Fatalf("GPOS too short: %d bytes", len(data))
	}

	// Check GPOS header
	major := readU16(data, 0)
	minor := readU16(data, 2)
	if major != 1 || minor != 0 {
		t.Fatalf("version = %d.%d, want 1.0", major, minor)
	}

	// Decode back
	glyphOrder := []string{"A", "V"}
	decoded, err := DecodePairPosLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodePairPosLookups: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("got %d pairs, want 1", len(decoded))
	}
	if decoded[0].LeftGlyph != "A" || decoded[0].RightGlyph != "V" || decoded[0].Value != -20 {
		t.Fatalf("pair = (%s,%s,%d), want (A,V,-20)",
			decoded[0].LeftGlyph, decoded[0].RightGlyph, decoded[0].Value)
	}
}

func TestBuildGPOS_MultiplePairs(t *testing.T) {
	pairs := []KerningPairInput{
		{Left: "A", Right: "V", Value: -20},
		{Left: "A", Right: "W", Value: -15},
		{Left: "V", Right: "A", Value: -10},
	}
	gm := map[string]uint16{"A": 0, "V": 1, "W": 2}

	data, err := BuildGPOS(pairs, gm)
	if err != nil {
		t.Fatalf("BuildGPOS: %v", err)
	}

	glyphOrder := []string{"A", "V", "W"}
	decoded, err := DecodePairPosLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodePairPosLookups: %v", err)
	}
	if len(decoded) != 3 {
		t.Fatalf("got %d pairs, want 3", len(decoded))
	}

	// Verify specific pairs exist
	pairMap := make(map[string]int)
	for _, p := range decoded {
		pairMap[p.LeftGlyph+":"+p.RightGlyph] = p.Value
	}
	if pairMap["A:V"] != -20 {
		t.Fatalf("A:V = %d, want -20", pairMap["A:V"])
	}
	if pairMap["A:W"] != -15 {
		t.Fatalf("A:W = %d, want -15", pairMap["A:W"])
	}
	if pairMap["V:A"] != -10 {
		t.Fatalf("V:A = %d, want -10", pairMap["V:A"])
	}
}

func TestBuildGPOS_Deduplication(t *testing.T) {
	// Same pair appears twice — keep LAST value
	pairs := []KerningPairInput{
		{Left: "A", Right: "V", Value: -20},
		{Left: "A", Right: "V", Value: -30},
	}
	gm := map[string]uint16{"A": 0, "V": 1}

	data, err := BuildGPOS(pairs, gm)
	if err != nil {
		t.Fatalf("BuildGPOS: %v", err)
	}

	glyphOrder := []string{"A", "V"}
	decoded, err := DecodePairPosLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodePairPosLookups: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("got %d pairs, want 1 after dedup", len(decoded))
	}
	if decoded[0].Value != -30 {
		t.Fatalf("value = %d, want -30 (last wins)", decoded[0].Value)
	}
}

func TestBuildGPOS_SkipsZeroValues(t *testing.T) {
	pairs := []KerningPairInput{
		{Left: "A", Right: "V", Value: 0},
		{Left: "V", Right: "A", Value: -10},
	}
	gm := map[string]uint16{"A": 0, "V": 1}

	data, err := BuildGPOS(pairs, gm)
	if err != nil {
		t.Fatalf("BuildGPOS: %v", err)
	}

	glyphOrder := []string{"A", "V"}
	decoded, err := DecodePairPosLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodePairPosLookups: %v", err)
	}
	// Only V:A=-10 should survive; A:V=0 is skipped
	if len(decoded) != 1 {
		t.Fatalf("got %d pairs, want 1 (zero-value skipped)", len(decoded))
	}
	if decoded[0].LeftGlyph != "V" || decoded[0].RightGlyph != "A" || decoded[0].Value != -10 {
		t.Fatalf("pair = (%s,%s,%d), want (V,A,-10)",
			decoded[0].LeftGlyph, decoded[0].RightGlyph, decoded[0].Value)
	}
}

func TestBuildGPOS_AllZeroValues(t *testing.T) {
	pairs := []KerningPairInput{
		{Left: "A", Right: "V", Value: 0},
	}
	gm := map[string]uint16{"A": 0, "V": 1}

	_, err := BuildGPOS(pairs, gm)
	if err == nil {
		t.Fatal("expected error when all pairs are zero-valued (no pairs to apply)")
	}
}
