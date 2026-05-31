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