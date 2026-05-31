// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/gsub"
	"github.com/kamiwanai/fonttools/opentype"
)

func ExtractGSUBReverseSubst(path string) ([]gsub.ReverseSubst, error) {
	font, err := opentype.ParseFile(path)
	if err != nil { return nil, err }
	return ExtractGSUBReverseSubstFromFont(font)
}

func ExtractGSUBReverseSubstFromFont(font *opentype.Font) ([]gsub.ReverseSubst, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil { return nil, err }
	gsubData, err := font.GSUB()
	if err != nil { return nil, nil }
	return gsub.DecodeReverseSubstLookups(gsubData, glyphOrder)
}
