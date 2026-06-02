// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

// KerningReport is a summary of kerning pair statistics for dashboard display.
type KerningReport struct {
	// TotalPairs is the total number of kerning pairs.
	TotalPairs int `json:"totalPairs"`
	// PositivePairs is the count of pairs with value > 0 (positive kerning).
	PositivePairs int `json:"positivePairs"`
	// NegativePairs is the count of pairs with value < 0 (negative kerning).
	NegativePairs int `json:"negativePairs"`
	// ZeroPairs is the count of pairs with value == 0 (neutral kerning).
	ZeroPairs int `json:"zeroPairs"`
	// AvgAbsValue is the average absolute kerning value in font units.
	AvgAbsValue float64 `json:"avgAbsValue"`
	// MaxAbsValue is the maximum absolute kerning value in font units.
	MaxAbsValue int `json:"maxAbsValue"`
}

// AnalyzeKerningPairs computes aggregate statistics from a set of GPOS kerning
// pairs. It is safe to pass an empty slice — all counters will be zero.
func AnalyzeKerningPairs(pairs []PairValue) KerningReport {
	r := KerningReport{TotalPairs: len(pairs)}
	if len(pairs) == 0 {
		return r
	}

	var sumAbs int64
	maxAbs := 0

	for _, p := range pairs {
		abs := p.Value
		if abs < 0 {
			abs = -abs
		}

		if p.Value > 0 {
			r.PositivePairs++
		} else if p.Value < 0 {
			r.NegativePairs++
		} else {
			r.ZeroPairs++
		}

		sumAbs += int64(abs)
		if abs > maxAbs {
			maxAbs = abs
		}
	}

	r.AvgAbsValue = float64(sumAbs) / float64(len(pairs))
	r.MaxAbsValue = maxAbs

	return r
}
