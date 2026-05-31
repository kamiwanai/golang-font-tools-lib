package fonttools

import (
	"github.com/kamiwanai/fonttools/hvar"
	"github.com/kamiwanai/fonttools/opentype"
)

// ExtractHVAR reads a font file and returns the parsed HVAR (Horizontal Metrics Variations) table.
// Returns an error if the font has no HVAR table or if decoding fails.
func ExtractHVAR(path string) (hvar.HVAR, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return hvar.HVAR{}, err
	}
	return ExtractHVARFromFont(font)
}

// ExtractHVARFromFont returns the parsed HVAR table from an already parsed font.
// Returns an error if the font has no HVAR table or if decoding fails.
func ExtractHVARFromFont(font *opentype.Font) (hvar.HVAR, error) {
	hvarData, err := font.TableData("HVAR")
	if err != nil {
		return hvar.HVAR{}, err
	}
	return hvar.Decode(hvarData)
}