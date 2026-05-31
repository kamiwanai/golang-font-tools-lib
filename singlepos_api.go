// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/golang-font-tools-lib/gpos"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

// ExtractGPOSSinglePos reads a font file and returns all GPOS SinglePos
// (lookup type 1) positioning adjustments. It supports formats 1 and 2,
// plus extension type 9 targeting SinglePos.
func ExtractGPOSSinglePos(path string) ([]gpos.SinglePosValue, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGPOSSinglePosFromFont(font)
}

// ExtractGPOSSinglePosFromFont returns GPOS SinglePos positioning adjustments
// from an already parsed font.
func ExtractGPOSSinglePosFromFont(font *opentype.Font) ([]gpos.SinglePosValue, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		return nil, err
	}
	return gpos.DecodeSinglePosLookups(gposData, glyphOrder)
}
