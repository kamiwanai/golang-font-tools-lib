// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/golang-font-tools-lib/gsub"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

// ExtractGSUBSingleSubst reads a font file and returns all SingleSubst rules
// from its GSUB table. Returns nil if the font has no GSUB table.
func ExtractGSUBSingleSubst(path string) ([]gsub.SingleSubst, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBSingleSubstFromFont(font)
}

// ExtractGSUBSingleSubstFromFont returns SingleSubst rules from an already
// parsed font. Returns nil if the font has no GSUB table.
func ExtractGSUBSingleSubstFromFont(font *opentype.Font) ([]gsub.SingleSubst, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	return gsub.DecodeSingleSubstLookups(gsubData, glyphOrder)
}

// ExtractGSUBLigatureSubst reads a font file and returns all LigatureSubst
// rules from its GSUB table. Returns nil if the font has no GSUB table.
func ExtractGSUBLigatureSubst(path string) ([]gsub.LigatureSubst, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBLigatureSubstFromFont(font)
}

// ExtractGSUBLigatureSubstFromFont returns LigatureSubst rules from an already
// parsed font. Returns nil if the font has no GSUB table.
func ExtractGSUBLigatureSubstFromFont(font *opentype.Font) ([]gsub.LigatureSubst, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	return gsub.DecodeLigatureSubstLookups(gsubData, glyphOrder)
}

// ExtractGSUBMultipleSubst reads a font file and returns all MultipleSubst
// rules from its GSUB table. Returns nil if the font has no GSUB table.
func ExtractGSUBMultipleSubst(path string) ([]gsub.MultipleSubst, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBMultipleSubstFromFont(font)
}

// ExtractGSUBMultipleSubstFromFont returns MultipleSubst rules from an already
// parsed font. Returns nil if the font has no GSUB table.
func ExtractGSUBMultipleSubstFromFont(font *opentype.Font) ([]gsub.MultipleSubst, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	return gsub.DecodeMultipleSubstLookups(gsubData, glyphOrder)
}

// ExtractGSUBAlternateSubst reads a font file and returns all AlternateSubst
// rules from its GSUB table. Returns nil if the font has no GSUB table.
func ExtractGSUBAlternateSubst(path string) ([]gsub.AlternateSubst, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBAlternateSubstFromFont(font)
}

// ExtractGSUBAlternateSubstFromFont returns AlternateSubst rules from an already
// parsed font. Returns nil if the font has no GSUB table.
func ExtractGSUBAlternateSubstFromFont(font *opentype.Font) ([]gsub.AlternateSubst, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	return gsub.DecodeAlternateSubstLookups(gsubData, glyphOrder)
}

// ExtractGSUBScripts returns the script list from a font's GSUB table.
// Returns nil if the font has no GSUB table.
func ExtractGSUBScripts(path string) ([]gsub.Script, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBScriptsFromFont(font)
}

// ExtractGSUBScriptsFromFont returns the script list from an already parsed
// font's GSUB table. Returns nil if the font has no GSUB table.
func ExtractGSUBScriptsFromFont(font *opentype.Font) ([]gsub.Script, error) {
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	return gsub.Scripts(gsubData)
}

// ExtractGSUBFeatures returns the feature list from a font's GSUB table.
// Returns nil if the font has no GSUB table.
func ExtractGSUBFeatures(path string) ([]gsub.Feature, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBFeaturesFromFont(font)
}

// ExtractGSUBFeaturesFromFont returns the feature list from an already parsed
// font's GSUB table. Returns nil if the font has no GSUB table.
func ExtractGSUBFeaturesFromFont(font *opentype.Font) ([]gsub.Feature, error) {
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	return gsub.Features(gsubData)
}

// ExtractGSUBActiveLookups returns lookup indices active for a given
// script/language/feature from a font's GSUB table.
// Returns nil if the font has no GSUB table.
func ExtractGSUBActiveLookups(path string, scriptTag, langTag, featureTag string) ([]uint16, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBActiveLookupsFromFont(font, scriptTag, langTag, featureTag)
}

// ExtractGSUBActiveLookupsFromFont returns lookup indices active for a given
// script/language/feature from an already parsed font's GSUB table.
// Returns nil if the font has no GSUB table.
func ExtractGSUBActiveLookupsFromFont(font *opentype.Font, scriptTag, langTag, featureTag string) ([]uint16, error) {
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	return gsub.ActiveLookups(gsubData, scriptTag, langTag, featureTag)
}

// ExtractGSUBContextSubst reads a font file and returns all ContextSubst (type 5)
// lookups from its GSUB table. Returns nil if the font has no GSUB table.
func ExtractGSUBContextSubst(path string) ([]gsub.ContextSubstLookup, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBContextSubstFromFont(font)
}

// ExtractGSUBContextSubstFromFont returns ContextSubst lookups from an already
// parsed font.
func ExtractGSUBContextSubstFromFont(font *opentype.Font) ([]gsub.ContextSubstLookup, error) {
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	return gsub.DecodeContextSubstLookups(gsubData, glyphOrder)
}

// ExtractGSUBChainContextSubst reads a font file and returns all
// ChainContextSubst (type 6) lookups from its GSUB table.
func ExtractGSUBChainContextSubst(path string) ([]gsub.ContextSubstLookup, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGSUBChainContextSubstFromFont(font)
}

// ExtractGSUBChainContextSubstFromFont returns ChainContextSubst lookups from
// an already parsed font.
func ExtractGSUBChainContextSubstFromFont(font *opentype.Font) ([]gsub.ContextSubstLookup, error) {
	gsubData, err := font.GSUB()
	if err != nil {
		return nil, nil
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	return gsub.DecodeChainContextSubstLookups(gsubData, glyphOrder)
}