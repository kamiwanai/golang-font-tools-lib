// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/gvar"
	"github.com/kamiwanai/fonttools/opentype"
)

// ExtractGVAR reads a font file and returns the parsed gvar (Glyph Variations) table.
// Returns an error if the font has no gvar table or if decoding fails.
func ExtractGVAR(path string) (gvar.GVAR, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return gvar.GVAR{}, err
	}
	return ExtractGVARFromFont(font)
}

// ExtractGVARFromFont returns the parsed gvar table from an already parsed font.
// Returns an error if the font has no gvar table or if decoding fails.
func ExtractGVARFromFont(font *opentype.Font) (gvar.GVAR, error) {
	gvarData, err := font.TableData("gvar")
	if err != nil {
		return gvar.GVAR{}, err
	}
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return gvar.GVAR{}, err
	}
	return gvar.Decode(gvarData, numGlyphs)
}
