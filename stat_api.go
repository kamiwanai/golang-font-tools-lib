// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/opentype"
	"github.com/kamiwanai/fonttools/stat"
)

// ExtractSTAT reads a font file and returns the parsed STAT (Style Attributes) table.
func ExtractSTAT(path string) (stat.STAT, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return stat.STAT{}, err
	}
	return ExtractSTATFromFont(font)
}

// ExtractSTATFromFont returns the parsed STAT table from an already parsed font.
func ExtractSTATFromFont(font *opentype.Font) (stat.STAT, error) {
	statData, err := font.TableData("STAT")
	if err != nil {
		return stat.STAT{}, err
	}
	return stat.Decode(statData)
}
