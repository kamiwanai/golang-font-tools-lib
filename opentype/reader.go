package opentype

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode/utf16"
)

// Table describes one OpenType table directory entry.
type Table struct {
	// Tag is the four-byte OpenType table tag, such as "head" or "GPOS".
	Tag string
	// Checksum is the checksum stored in the table directory.
	Checksum uint32
	// Offset is the byte offset from the start of the font data.
	Offset uint32
	// Length is the table length in bytes.
	Length uint32
}

// Font is a parsed OpenType font with table directory metadata.
type Font struct {
	// ScalerType is the font scaler type from the OpenType header.
	ScalerType uint32
	// Tables maps table tags to directory entries.
	Tables map[string]Table
	data   []byte
}

// NameRecord is one decoded record from the OpenType name table.
type NameRecord struct {
	// PlatformID is the name record platform identifier.
	PlatformID uint16
	// EncodingID is the name record encoding identifier.
	EncodingID uint16
	// LanguageID is the name record language identifier.
	LanguageID uint16
	// NameID is the OpenType name identifier.
	NameID uint16
	// Text is the decoded name string.
	Text string
}

// ParseFile reads and parses an OpenType font from path.
func ParseFile(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Parse parses OpenType font bytes and validates table directory bounds.
func Parse(data []byte) (*Font, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("font header too short: %d bytes", len(data))
	}
	scalerType := binary.BigEndian.Uint32(data[0:4])
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	recordsStart := 12
	recordsEnd := recordsStart + numTables*16
	if len(data) < recordsEnd {
		return nil, fmt.Errorf("table directory too short: need %d bytes, got %d", recordsEnd, len(data))
	}

	tables := make(map[string]Table, numTables)
	for index := 0; index < numTables; index++ {
		offset := recordsStart + index*16
		record := data[offset : offset+16]
		tag := string(record[0:4])
		table := Table{
			Tag:      tag,
			Checksum: binary.BigEndian.Uint32(record[4:8]),
			Offset:   binary.BigEndian.Uint32(record[8:12]),
			Length:   binary.BigEndian.Uint32(record[12:16]),
		}
		if uint64(table.Offset)+uint64(table.Length) > uint64(len(data)) {
			return nil, fmt.Errorf("table %q exceeds font bounds", tag)
		}
		tables[tag] = table
	}

	return &Font{ScalerType: scalerType, Tables: tables, data: data}, nil
}

// TableData returns the raw bytes for the table with tag.
func (font *Font) TableData(tag string) ([]byte, error) {
	table, ok := font.Tables[tag]
	if !ok {
		return nil, fmt.Errorf("table %q not found", tag)
	}
	start := int(table.Offset)
	end := start + int(table.Length)
	if start < 0 || end < start || end > len(font.data) {
		return nil, fmt.Errorf("table %q exceeds font bounds", tag)
	}
	return font.data[start:end], nil
}

// Name returns the first non-empty decoded name matching one of nameIDs.
func (font *Font) Name(nameIDs ...uint16) string {
	records, err := font.NameRecords()
	if err != nil {
		return ""
	}
	for _, nameID := range nameIDs {
		for _, record := range records {
			if record.NameID != nameID {
				continue
			}
			text := strings.TrimSpace(record.Text)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

// NameRecords returns all decoded records from the OpenType name table.
func (font *Font) NameRecords() ([]NameRecord, error) {
	name, err := font.TableData("name")
	if err != nil {
		return nil, err
	}
	if len(name) < 6 {
		return nil, fmt.Errorf("name table too short: %d bytes", len(name))
	}
	count := int(readU16(name, 2))
	if count > MaxNameRecords {
		return nil, fmt.Errorf("name table has %d records, exceeds limit %d", count, MaxNameRecords)
	}
	stringOffset := int(readU16(name, 4))
	if len(name) < 6+count*12 {
		return nil, fmt.Errorf("name records exceed table")
	}
	records := make([]NameRecord, 0, count)
	for index := 0; index < count; index++ {
		offset := 6 + index*12
		platformID := readU16(name, offset)
		encodingID := readU16(name, offset+2)
		languageID := readU16(name, offset+4)
		nameID := readU16(name, offset+6)
		length := int(readU16(name, offset+8))
		stringStart := stringOffset + int(readU16(name, offset+10))
		if stringStart < 0 || stringStart+length > len(name) {
			return nil, fmt.Errorf("name string exceeds table")
		}
		records = append(records, NameRecord{
			PlatformID: platformID,
			EncodingID: encodingID,
			LanguageID: languageID,
			NameID:     nameID,
			Text:       decodeNameText(platformID, name[stringStart:stringStart+length]),
		})
	}
	return records, nil
}

// NumGlyphs returns the glyph count from the maxp table.
func (font *Font) NumGlyphs() (int, error) {
	maxp, err := font.TableData("maxp")
	if err != nil {
		return 0, err
	}
	if len(maxp) < 6 {
		return 0, fmt.Errorf("maxp table too short: %d bytes", len(maxp))
	}
	return int(readU16(maxp, 4)), nil
}

// GlyphOrder returns glyph names by glyph ID.
//
// It uses post format 2 names when available. Other post formats fall back to
// stable synthetic names, with glyph 0 named ".notdef".
func (font *Font) GlyphOrder() ([]string, error) {
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return nil, err
	}
	fallback := func() []string {
		glyphs := make([]string, numGlyphs)
		for index := range glyphs {
			glyphs[index] = fmt.Sprintf("glyph%05d", index)
		}
		if len(glyphs) > 0 {
			glyphs[0] = ".notdef"
		}
		return glyphs
	}

	post, err := font.TableData("post")
	if err != nil {
		return fallback(), nil
	}
	if len(post) < 32 {
		return nil, fmt.Errorf("post table too short: %d bytes", len(post))
	}
	format := readU32(post, 0)
	if format == 0x00030000 {
		return fallback(), nil
	}
	if format != 0x00020000 {
		return fallback(), nil
	}
	if len(post) < 34 {
		return nil, fmt.Errorf("post format 2 table too short: %d bytes", len(post))
	}
	postGlyphs := int(readU16(post, 32))
	if len(post) < 34+postGlyphs*2 {
		return nil, fmt.Errorf("post glyph name indexes exceed table")
	}
	indexes := make([]uint16, postGlyphs)
	for index := 0; index < postGlyphs; index++ {
		indexes[index] = readU16(post, 34+index*2)
	}
	customNames, err := readPostCustomNames(post[34+postGlyphs*2:])
	if err != nil {
		return nil, err
	}
	glyphs := fallback()
	limit := postGlyphs
	if limit > numGlyphs {
		limit = numGlyphs
	}
	for index := 0; index < limit; index++ {
		nameIndex := indexes[index]
		switch {
		case int(nameIndex) < len(macGlyphNames):
			glyphs[index] = macGlyphNames[nameIndex]
		case nameIndex >= 258:
			customIndex := int(nameIndex - 258)
			if customIndex >= len(customNames) {
				return nil, fmt.Errorf("post custom glyph name index %d out of range", customIndex)
			}
			glyphs[index] = customNames[customIndex]
		}
	}
	return glyphs, nil
}

// BestCmap returns the best supported Unicode cmap as codepoint to glyph ID.
//
// Supported subtables are cmap format 12 and format 4. Unicode format 12 is
// preferred when present.
func (font *Font) BestCmap() (map[uint32]uint16, error) {
	cmap, err := font.TableData("cmap")
	if err != nil {
		return nil, err
	}
	if len(cmap) < 4 {
		return nil, fmt.Errorf("cmap table too short: %d bytes", len(cmap))
	}
	numTables := int(readU16(cmap, 2))
	if len(cmap) < 4+numTables*8 {
		return nil, fmt.Errorf("cmap encoding records exceed table")
	}
	records := make([]cmapRecord, 0, numTables)
	for index := 0; index < numTables; index++ {
		offset := 4 + index*8
		record := cmapRecord{
			platformID: readU16(cmap, offset),
			encodingID: readU16(cmap, offset+2),
			offset:     readU32(cmap, offset+4),
		}
		if int(record.offset)+2 > len(cmap) {
			continue
		}
		record.format = readU16(cmap, int(record.offset))
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		return cmapPriority(records[i]) > cmapPriority(records[j])
	})
	for _, record := range records {
		subtable := cmap[record.offset:]
		switch record.format {
		case 12:
			return decodeCmapFormat12(subtable)
		case 4:
			return decodeCmapFormat4(subtable)
		}
	}
	return nil, fmt.Errorf("no supported cmap subtable found")
}

type cmapRecord struct {
	platformID uint16
	encodingID uint16
	format     uint16
	offset     uint32
}

func cmapPriority(record cmapRecord) int {
	if record.platformID == 3 && record.encodingID == 10 && record.format == 12 {
		return 100
	}
	if record.platformID == 0 && record.format == 12 {
		return 90
	}
	if record.platformID == 3 && record.encodingID == 1 && record.format == 4 {
		return 80
	}
	if record.platformID == 0 && record.format == 4 {
		return 70
	}
	return 1
}

func decodeCmapFormat12(data []byte) (map[uint32]uint16, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("cmap format 12 too short: %d bytes", len(data))
	}
	length := int(readU32(data, 4))
	if length > len(data) {
		return nil, fmt.Errorf("cmap format 12 length exceeds table")
	}
	nGroups := int(readU32(data, 12))
	if nGroups > MaxCmapGroupCodepoints {
		nGroups = MaxCmapGroupCodepoints
	}
	if 16+nGroups*12 > length {
		return nil, fmt.Errorf("cmap format 12 groups exceed subtable")
	}
	mapping := make(map[uint32]uint16)
	for index := 0; index < nGroups; index++ {
		offset := 16 + index*12
		startChar := readU32(data, offset)
		endChar := readU32(data, offset+4)
		startGlyph := readU32(data, offset+8)
		if endChar < startChar {
			return nil, fmt.Errorf("cmap format 12 group end before start")
		}
		numCodepoints := endChar - startChar + 1
		if int(numCodepoints) > MaxCmapGroupCodepoints {
			return nil, fmt.Errorf("cmap format 12 group has %d codepoints, exceeds limit %d", numCodepoints, MaxCmapGroupCodepoints)
		}
		for codepoint := startChar; codepoint <= endChar; codepoint++ {
			mapping[codepoint] = uint16(startGlyph + codepoint - startChar)
			if codepoint == ^uint32(0) {
				break
			}
		}
	}
	return mapping, nil
}

func decodeCmapFormat4(data []byte) (map[uint32]uint16, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("cmap format 4 too short: %d bytes", len(data))
	}
	length := int(readU16(data, 2))
	if length > len(data) {
		return nil, fmt.Errorf("cmap format 4 length exceeds table")
	}
	segCount := int(readU16(data, 6) / 2)
	if segCount > MaxCmapFormat4Segments {
		return nil, fmt.Errorf("cmap format 4 has %d segments, exceeds limit %d", segCount, MaxCmapFormat4Segments)
	}
	endCodeOffset := 14
	startCodeOffset := endCodeOffset + segCount*2 + 2
	idDeltaOffset := startCodeOffset + segCount*2
	idRangeOffsetOffset := idDeltaOffset + segCount*2
	if idRangeOffsetOffset+segCount*2 > length {
		return nil, fmt.Errorf("cmap format 4 arrays exceed subtable")
	}
	mapping := make(map[uint32]uint16)
	for segment := 0; segment < segCount; segment++ {
		endCode := readU16(data, endCodeOffset+segment*2)
		startCode := readU16(data, startCodeOffset+segment*2)
		idDelta := int16(readU16(data, idDeltaOffset+segment*2))
		idRangeOffsetPos := idRangeOffsetOffset + segment*2
		idRangeOffset := readU16(data, idRangeOffsetPos)
		if endCode < startCode {
			return nil, fmt.Errorf("cmap format 4 segment end before start")
		}
		for codepoint := startCode; codepoint <= endCode; codepoint++ {
			if codepoint == 0xffff {
				break
			}
			var glyphID uint16
			if idRangeOffset == 0 {
				glyphID = uint16(int(codepoint) + int(idDelta))
			} else {
				glyphOffset := idRangeOffsetPos + int(idRangeOffset) + 2*int(codepoint-startCode)
				if glyphOffset+2 > length {
					continue
				}
				glyphID = readU16(data, glyphOffset)
				if glyphID != 0 {
					glyphID = uint16(int(glyphID) + int(idDelta))
				}
			}
			if glyphID != 0 {
				mapping[uint32(codepoint)] = glyphID
			}
		}
	}
	return mapping, nil
}

func readPostCustomNames(data []byte) ([]string, error) {
	names := make([]string, 0)
	for offset := 0; offset < len(data); {
		length := int(data[offset])
		offset++
		if offset+length > len(data) {
			return nil, fmt.Errorf("post custom glyph name exceeds table")
		}
		names = append(names, string(data[offset:offset+length]))
		offset += length
		if len(names) > MaxPostCustomNames {
			return nil, fmt.Errorf("post custom names exceed limit %d", MaxPostCustomNames)
		}
	}
	return names, nil
}

func decodeNameText(platformID uint16, data []byte) string {
	if platformID == 0 || platformID == 3 {
		if len(data)%2 != 0 {
			return ""
		}
		codes := make([]uint16, len(data)/2)
		for index := range codes {
			codes[index] = readU16(data, index*2)
		}
		return string(utf16.Decode(codes))
	}
	return string(data)
}

func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}

var macGlyphNames = []string{
	".notdef", ".null", "nonmarkingreturn", "space", "exclam", "quotedbl", "numbersign", "dollar", "percent", "ampersand",
	"quotesingle", "parenleft", "parenright", "asterisk", "plus", "comma", "hyphen", "period", "slash", "zero",
	"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "colon", "semicolon", "less", "equal",
	"greater", "question", "at", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N",
	"O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "bracketleft", "backslash", "bracketright",
	"asciicircum", "underscore", "grave", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o",
	"p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z", "braceleft", "bar", "braceright", "asciitilde",
	"Adieresis", "Aring", "Ccedilla", "Eacute", "Ntilde", "Odieresis", "Udieresis", "aacute", "agrave", "acircumflex",
	"adieresis", "atilde", "aring", "ccedilla", "eacute", "egrave", "ecircumflex", "edieresis", "iacute", "igrave",
	"icircumflex", "idieresis", "ntilde", "oacute", "ograve", "ocircumflex", "odieresis", "otilde", "uacute", "ugrave",
	"ucircumflex", "udieresis", "dagger", "degree", "cent", "sterling", "section", "bullet", "paragraph", "germandbls",
	"registered", "copyright", "trademark", "acute", "dieresis", "notequal", "AE", "Oslash", "infinity", "plusminus",
	"lessequal", "greaterequal", "yen", "mu", "partialdiff", "summation", "product", "pi", "integral", "ordfeminine",
	"ordmasculine", "Omega", "ae", "oslash", "questiondown", "exclamdown", "logicalnot", "radical", "florin", "approxequal",
	"Delta", "guillemotleft", "guillemotright", "ellipsis", "nonbreakingspace", "Agrave", "Atilde", "Otilde", "OE", "oe",
	"endash", "emdash", "quotedblleft", "quotedblright", "quoteleft", "quoteright", "divide", "lozenge", "ydieresis", "Ydieresis",
	"fraction", "currency", "guilsinglleft", "guilsinglright", "fi", "fl", "daggerdbl", "periodcentered", "quotesinglbase", "quotedblbase",
	"perthousand", "Acircumflex", "Ecircumflex", "Aacute", "Edieresis", "Egrave", "Iacute", "Icircumflex", "Idieresis", "Igrave",
	"Oacute", "Ocircumflex", "apple", "Ograve", "Uacute", "Ucircumflex", "Ugrave", "dotlessi", "circumflex", "tilde",
	"macron", "breve", "dotaccent", "ring", "cedilla", "hungarumlaut", "ogonek", "caron", "Lslash", "lslash",
	"Scaron", "scaron", "Zcaron", "zcaron", "brokenbar", "Eth", "eth", "Yacute", "yacute", "Thorn", "thorn", "minus",
	"multiply", "onesuperior", "twosuperior", "threesuperior", "onehalf", "onequarter", "threequarters", "franc", "Gbreve", "gbreve",
	"Idotaccent", "Scedilla", "scedilla", "Cacute", "cacute", "Ccaron", "ccaron", "dcroat",
}
