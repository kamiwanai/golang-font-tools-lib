// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/golang-font-tools-lib/avar"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

// ExtractAVAR reads a font file and returns the parsed avar (Axis Variations) table.
// Returns an error if the font has no avar table or if decoding fails.
func ExtractAVAR(path string) (avar.AVAR, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return avar.AVAR{}, err
	}
	return ExtractAVARFromFont(font)
}

// ExtractAVARFromFont returns the parsed avar table from an already parsed font.
// Returns an error if the font has no avar table or if decoding fails.
func ExtractAVARFromFont(font *opentype.Font) (avar.AVAR, error) {
	avarData, err := font.TableData("avar")
	if err != nil {
		return avar.AVAR{}, err
	}
	return avar.Decode(avarData)
}
