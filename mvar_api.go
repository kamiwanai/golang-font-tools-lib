// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/mvar"
	"github.com/kamiwanai/fonttools/opentype"
)

// ExtractMVAR reads a font file and returns the parsed MVAR (Metrics Variations) table.
// Returns an error if the font has no MVAR table or if decoding fails.
func ExtractMVAR(path string) (mvar.MVAR, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return mvar.MVAR{}, err
	}
	return ExtractMVARFromFont(font)
}

// ExtractMVARFromFont returns the parsed MVAR table from an already parsed font.
func ExtractMVARFromFont(font *opentype.Font) (mvar.MVAR, error) {
	mvarData, err := font.MVAR()
	if err != nil {
		return mvar.MVAR{}, err
	}
	return mvar.Decode(mvarData)
}
