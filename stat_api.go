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
