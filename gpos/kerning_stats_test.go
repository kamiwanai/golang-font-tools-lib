// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import (
	"math"
	"testing"
)

func TestAnalyzeKerningPairsEmpty(t *testing.T) {
	r := AnalyzeKerningPairs(nil)
	if r.TotalPairs != 0 {
		t.Fatalf("TotalPairs = %d, want 0", r.TotalPairs)
	}
	if r.PositivePairs != 0 {
		t.Fatalf("PositivePairs = %d, want 0", r.PositivePairs)
	}
	if r.AvgAbsValue != 0 {
		t.Fatalf("AvgAbsValue = %f, want 0", r.AvgAbsValue)
	}
}

func TestAnalyzeKerningPairsMixed(t *testing.T) {
	pairs := []PairValue{
		{LeftGlyph: "A", RightGlyph: "V", Value: -20},
		{LeftGlyph: "V", RightGlyph: "A", Value: -15},
		{LeftGlyph: "T", RightGlyph: "e", Value: -30},
		{LeftGlyph: "f", RightGlyph: "i", Value: 5},
		{LeftGlyph: "A", RightGlyph: "W", Value: 0},
	}
	r := AnalyzeKerningPairs(pairs)

	if r.TotalPairs != 5 {
		t.Fatalf("TotalPairs = %d, want 5", r.TotalPairs)
	}
	if r.PositivePairs != 1 {
		t.Fatalf("PositivePairs = %d, want 1", r.PositivePairs)
	}
	if r.NegativePairs != 3 {
		t.Fatalf("NegativePairs = %d, want 3", r.NegativePairs)
	}
	if r.ZeroPairs != 1 {
		t.Fatalf("ZeroPairs = %d, want 1", r.ZeroPairs)
	}

	// avg abs = (20+15+30+5+0)/5 = 70/5 = 14.0
	expectedAvg := 14.0
	if math.Abs(r.AvgAbsValue-expectedAvg) > 0.001 {
		t.Fatalf("AvgAbsValue = %f, want %f", r.AvgAbsValue, expectedAvg)
	}

	if r.MaxAbsValue != 30 {
		t.Fatalf("MaxAbsValue = %d, want 30", r.MaxAbsValue)
	}
}
