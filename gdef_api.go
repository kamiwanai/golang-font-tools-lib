// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/golang-font-tools-lib/gdef"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

// ExtractGDEF reads a font file and returns the parsed GDEF table.
// Returns an error wrapping gdef.ErrNoGDEF if the font has no GDEF table.
func ExtractGDEF(path string) (gdef.GDEF, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return gdef.GDEF{}, err
	}
	return ExtractGDEFFromFont(font)
}

// ExtractGDEFFromFont returns the parsed GDEF table from an already parsed font.
// Returns an error if the font has no GDEF table or if decoding fails.
func ExtractGDEFFromFont(font *opentype.Font) (gdef.GDEF, error) {
	gdefData, err := font.GDEF()
	if err != nil {
		return gdef.GDEF{}, err
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return gdef.GDEF{}, err
	}
	return gdef.Decode(gdefData, glyphOrder)
}

// ExtractGDEFGlyphClasses reads a font file and returns a map from glyph ID
// to its GDEF class (1=Base, 2=Ligature, 3=Mark, 4=Component).
// Returns nil if the font has no GDEF table.
func ExtractGDEFGlyphClasses(path string) (map[uint16]uint16, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGDEFGlyphClassesFromFont(font)
}

// ExtractGDEFGlyphClassesFromFont returns a map from glyph ID to its GDEF class
// from an already parsed font. Returns nil if the font has no GDEF table.
func ExtractGDEFGlyphClassesFromFont(font *opentype.Font) (map[uint16]uint16, error) {
	gdefData, err := font.GDEF()
	if err != nil {
		return nil, nil
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gd, err := gdef.Decode(gdefData, glyphOrder)
	if err != nil {
		return nil, err
	}
	return gd.GlyphClassMap, nil
}

// ExtractGDEFGlyphClassesByName reads a font file and returns a map from glyph
// name to its GDEF class. Returns nil if the font has no GDEF table.
func ExtractGDEFGlyphClassesByName(path string) (map[string]uint16, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGDEFGlyphClassesByNameFromFont(font)
}

// ExtractGDEFGlyphClassesByNameFromFont returns a map from glyph name to its
// GDEF class from an already parsed font. Returns nil if the font has no GDEF table.
func ExtractGDEFGlyphClassesByNameFromFont(font *opentype.Font) (map[string]uint16, error) {
	gdefData, err := font.GDEF()
	if err != nil {
		return nil, nil
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gd, err := gdef.Decode(gdefData, glyphOrder)
	if err != nil {
		return nil, err
	}
	return gd.ClassDefByNames(glyphOrder), nil
}