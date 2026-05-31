package fonttools

import (
	"github.com/kamiwanai/fonttools/gpos"
	"github.com/kamiwanai/fonttools/opentype"
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
