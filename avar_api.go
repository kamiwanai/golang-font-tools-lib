package fonttools

import (
	"github.com/kamiwanai/fonttools/avar"
	"github.com/kamiwanai/fonttools/opentype"
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
