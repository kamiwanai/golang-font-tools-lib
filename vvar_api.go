// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/vvar"
	"github.com/kamiwanai/fonttools/opentype"
)

// ExtractVVAR reads a font file and returns the parsed VVAR (Vertical Metrics Variations) table.
// Returns an error if the font has no VVAR table or if decoding fails.
func ExtractVVAR(path string) (vvar.VVAR, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return vvar.VVAR{}, err
	}
	return ExtractVVARFromFont(font)
}

// ExtractVVARFromFont returns the parsed VVAR table from an already parsed font.
// Returns an error if the font has no VVAR table or if decoding fails.
func ExtractVVARFromFont(font *opentype.Font) (vvar.VVAR, error) {
	vvarData, err := font.TableData("VVAR")
	if err != nil {
		return vvar.VVAR{}, err
	}
	return vvar.Decode(vvarData)
}