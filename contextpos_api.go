// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/gpos"
	"github.com/kamiwanai/fonttools/opentype"
)

func ExtractGPOSContextPos(path string) ([]gpos.ContextPosLookup, error) {
	font, err := opentype.ParseFile(path)
	if err != nil { return nil, err }
	return ExtractGPOSContextPosFromFont(font)
}

func ExtractGPOSContextPosFromFont(font *opentype.Font) ([]gpos.ContextPosLookup, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil { return nil, err }
	gposData, err := font.GPOS()
	if err != nil { return nil, nil }
	return gpos.DecodeContextPosLookups(gposData, glyphOrder)
}

func ExtractGPOSChainContextPos(path string) ([]gpos.ContextPosLookup, error) {
	font, err := opentype.ParseFile(path)
	if err != nil { return nil, err }
	return ExtractGPOSChainContextPosFromFont(font)
}

func ExtractGPOSChainContextPosFromFont(font *opentype.Font) ([]gpos.ContextPosLookup, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil { return nil, err }
	gposData, err := font.GPOS()
	if err != nil { return nil, nil }
	return gpos.DecodeChainContextPosLookups(gposData, glyphOrder)
}
// ExtractContextValueRecords extracts X/Y placement and advance values from
// nested lookups referenced by ContextPos (type 7) rules.
func ExtractContextValueRecords(path string) ([]gpos.ContextValueRecord, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractContextValueRecordsFromFont(font, 7)
}

// ExtractContextValueRecordsFromFont extracts X/Y values from ContextPos nested lookups.
func ExtractContextValueRecordsFromFont(font *opentype.Font, wantType uint16) ([]gpos.ContextValueRecord, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := font.GPOS()
	if err != nil {
		return nil, nil
	}
	return gpos.DecodeContextValueRecords(gposData, glyphOrder, wantType)
}

// ExtractChainContextValueRecords extracts X/Y placement and advance values from
// nested lookups referenced by ChainContextPos (type 8) rules.
func ExtractChainContextValueRecords(path string) ([]gpos.ContextValueRecord, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractContextValueRecordsFromFont(font, 8)
}
