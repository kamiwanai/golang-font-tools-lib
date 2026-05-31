// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/gpos"
	"github.com/kamiwanai/fonttools/opentype"
)

// ExtractGPOSKerning reads a font file and returns all supported GPOS PairPos
// kerning pairs. It supports lookup type 2, extension lookup type 9 targeting
// PairPos, and PairPos formats 1 and 2.
func ExtractGPOSKerning(path string) ([]gpos.PairValue, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGPOSKerningFromFont(font)
}

// ExtractGPOSKerningFromFont returns supported GPOS PairPos kerning pairs from
// an already parsed font.
func ExtractGPOSKerningFromFont(font *opentype.Font) ([]gpos.PairValue, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		return nil, err
	}
	return gpos.DecodePairPosLookups(gposData, glyphOrder)
}
