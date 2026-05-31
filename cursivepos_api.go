// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/golang-font-tools-lib/gpos"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

func ExtractGPOSCursivePos(path string) ([]gpos.CursiveEntry, error) {
	font, err := opentype.ParseFile(path)
	if err != nil { return nil, err }
	return ExtractGPOSCursivePosFromFont(font)
}

func ExtractGPOSCursivePosFromFont(font *opentype.Font) ([]gpos.CursiveEntry, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil { return nil, err }
	gposData, err := font.GPOS()
	if err != nil { return nil, nil }
	return gpos.DecodeCursivePosLookups(gposData, glyphOrder)
}
