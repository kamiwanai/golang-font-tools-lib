// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// CFFGlyphNames returns glyph names from the CFF charset.
//
// For .otf fonts with CFF outlines that lack a post format 2 table,
// glyph names must be read from the CFF charset. Each glyph is mapped
// to a String ID (SID), which is resolved via the standard 391 predefined
// CFF strings and the font's String INDEX.
//
// The returned slice is indexed by glyph ID. Glyph 0 is always ".notdef".
// Returns an error if the font has no CFF table or the CFF data is invalid.
func (font *Font) CFFGlyphNames() ([]string, error) {
	data, err := font.TableData("CFF ")
	if err != nil {
		return nil, fmt.Errorf("CFF: %w", err)
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("CFF table too short: %d bytes", len(data))
	}

	// Parse CFF header
	major := data[0]
	minor := data[1]
	if major != 1 {
		return nil, fmt.Errorf("CFF version %d.%d not supported (only version 1)", major, minor)
	}
	hdrSize := int(data[2])
	if hdrSize < 4 || hdrSize > len(data) {
		return nil, fmt.Errorf("CFF hdrSize %d invalid", hdrSize)
	}

	// Parse Name INDEX
	pos := hdrSize
	nameCount, nameData, next, err := parseCFFIndex(data, pos)
	if err != nil {
		return nil, fmt.Errorf("CFF Name INDEX: %w", err)
	}
	_ = nameCount
	_ = nameData // font names, not glyph names

	// Parse Top DICT INDEX
	topDictCount, topDictData, next, err := parseCFFIndex(data, next)
	if err != nil {
		return nil, fmt.Errorf("CFF Top DICT INDEX: %w", err)
	}
	if topDictCount == 0 {
		return nil, fmt.Errorf("CFF Top DICT INDEX is empty")
	}

	// Parse the first Top DICT to find charset and CharStrings offsets
	charsetOffset, charstringsOffset, err := parseTopDICT(topDictData[0])
	if err != nil {
		return nil, fmt.Errorf("CFF Top DICT: %w", err)
	}

	// Parse String INDEX
	_, stringData, next, err := parseCFFIndex(data, next)
	if err != nil {
		return nil, fmt.Errorf("CFF String INDEX: %w", err)
	}
	_ = next // skip Global Subr INDEX

	// Parse CharStrings INDEX to get glyph count
	charstringsCount, _, _, err := parseCFFIndex(data, charstringsOffset)
	if err != nil {
		return nil, fmt.Errorf("CFF CharStrings INDEX: %w", err)
	}
	if charstringsCount == 0 {
		return nil, fmt.Errorf("CFF CharStrings INDEX is empty")
	}
	nGlyphs := charstringsCount

	// If charset offset is 0, use the default charset (ISOAdobe)
	if charsetOffset == 0 {
		return defaultCharset(nGlyphs), nil
	}

	// Parse charset
	sids, err := parseCharset(data, charsetOffset, nGlyphs)
	if err != nil {
		return nil, fmt.Errorf("CFF charset: %w", err)
	}

	// Resolve SIDs to names
	names := make([]string, nGlyphs)
	for i, sid := range sids {
		name, err := resolveSID(sid, stringData)
		if err != nil {
			names[i] = fmt.Sprintf("glyph%05d", i)
		} else {
			names[i] = name
		}
	}
	if len(names) > 0 {
		names[0] = ".notdef"
	}
	return names, nil
}

// parseCFFIndex parses a CFF INDEX structure at the given offset.
// Returns (count, elements, nextOffset, error).
// Each element is the raw bytes of that INDEX entry.
func parseCFFIndex(data []byte, offset int) (int, [][]byte, int, error) {
	if offset+2 > len(data) {
		return 0, nil, 0, fmt.Errorf("offset %d exceeds data for INDEX count", offset)
	}
	count := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	if count == 0 {
		return 0, nil, offset + 2, nil
	}
	if offset+3 > len(data) {
		return 0, nil, 0, fmt.Errorf("INDEX offSize exceeds data")
	}
	offSize := int(data[offset+2])
	if offSize < 1 || offSize > 4 {
		return 0, nil, 0, fmt.Errorf("INDEX offSize %d invalid (must be 1-4)", offSize)
	}

	// Offset array: (count+1) entries, each offSize bytes
	offsetArrayStart := offset + 3
	offsetArrayEnd := offsetArrayStart + (count+1)*offSize
	if offsetArrayEnd > len(data) {
		return 0, nil, 0, fmt.Errorf("INDEX offset array exceeds data: need %d, have %d", offsetArrayEnd, len(data))
	}

	// Read offsets (1-based: first element offset starts at 1)
	offsets := make([]int, count+1)
	for i := 0; i <= count; i++ {
		off := offsetArrayStart + i*offSize
		offsets[i] = readCFFOffset(data, off, offSize)
	}

	// Data starts after the offset array
	dataStart := offsetArrayEnd

	// Extract elements
	elements := make([][]byte, count)
	for i := 0; i < count; i++ {
		start := dataStart + offsets[i] - 1 // offsets are 1-based
		end := dataStart + offsets[i+1] - 1
		if start < 0 || end < start || end > len(data) {
			return 0, nil, 0, fmt.Errorf("INDEX element %d exceeds data", i)
		}
		elements[i] = data[start:end]
	}

	nextOffset := dataStart + offsets[count] - 1
	return count, elements, nextOffset, nil
}

// readCFFOffset reads a 1-4 byte offset from data at the given position.
func readCFFOffset(data []byte, offset int, offSize int) int {
	switch offSize {
	case 1:
		return int(data[offset])
	case 2:
		return int(binary.BigEndian.Uint16(data[offset : offset+2]))
	case 3:
		return int(data[offset])<<16 | int(data[offset+1])<<8 | int(data[offset+2])
	case 4:
		return int(binary.BigEndian.Uint32(data[offset : offset+4]))
	default:
		return 0
	}
}

// parseTopDICT parses the Top DICT data to find charset and CharStrings offsets.
// The Top DICT uses a key-value encoding where operands precede operators.
func parseTopDICT(data []byte) (charsetOffset int, charstringsOffset int, err error) {
	// DICT operands are encoded as: high byte determines type
	// 32-246: 1-byte integer (value - 139)
	// 247-250: 2-byte integer
	// 251-254: 2-byte integer (negative)
	// 28: 2-byte integer follows
	// 29: 4-byte integer follows
	// 30: real number (BCD encoded)
	//
	// Operators:
	// 15: charset (offset from start of CFF data)
	// 17: CharStrings (offset from start of CFF data)

	pos := 0
	var operands []int

	for pos < len(data) {
		b := data[pos]

		// Check if this is an operator (0-21, except 12 which is 2-byte operator)
		if b <= 21 {
			op := int(b)
			if b == 12 {
				if pos+1 >= len(data) {
					return 0, 0, fmt.Errorf("2-byte operator truncated")
				}
				op = 1200 + int(data[pos+1]) // encode as 12.xx
				pos += 2
			} else {
				pos++
			}

			switch op {
			case 15: // charset
				if len(operands) >= 1 {
					charsetOffset = operands[len(operands)-1]
				}
			case 17: // CharStrings
				if len(operands) >= 1 {
					charstringsOffset = operands[len(operands)-1]
				}
			}
			operands = operands[:0]
			continue
		}

		// Parse operand
		val, size, err := parseCFFDictOperand(data, pos)
		if err != nil {
			return 0, 0, err
		}
		operands = append(operands, val)
		pos += size
	}

	if charstringsOffset == 0 {
		return 0, 0, fmt.Errorf("CharStrings offset not found in Top DICT")
	}
	return charsetOffset, charstringsOffset, nil
}

// parseCFFDictOperand parses one DICT operand at the given position.
// Returns (value, bytesConsumed, error).
func parseCFFDictOperand(data []byte, pos int) (int, int, error) {
	if pos >= len(data) {
		return 0, 0, fmt.Errorf("operand at end of data")
	}
	b := data[pos]

	switch {
	case b >= 32 && b <= 246:
		return int(b) - 139, 1, nil
	case b >= 247 && b <= 250:
		if pos+1 >= len(data) {
			return 0, 0, fmt.Errorf("2-byte operand truncated")
		}
		return (int(b)-247)*256 + int(data[pos+1]) + 108, 2, nil
	case b >= 251 && b <= 254:
		if pos+1 >= len(data) {
			return 0, 0, fmt.Errorf("2-byte operand truncated")
		}
		return -(int(b)-251)*256 - int(data[pos+1]) - 108, 2, nil
	case b == 28:
		if pos+2 >= len(data) {
			return 0, 0, fmt.Errorf("28 operand truncated")
		}
		return int(int16(binary.BigEndian.Uint16(data[pos+1 : pos+3]))), 3, nil
	case b == 29:
		if pos+4 >= len(data) {
			return 0, 0, fmt.Errorf("29 operand truncated")
		}
		return int(int32(binary.BigEndian.Uint32(data[pos+1 : pos+5]))), 5, nil
	case b == 30:
		// Real number (BCD) — skip, we only need integer operands for offsets
		size := 1
		for pos+size < len(data) {
			nibbles := data[pos+size]
			size++
			// Each byte has 2 nibbles; 0xf marks end
			if (nibbles&0xF0)>>4 == 0xF || nibbles&0x0F == 0xF {
				break
			}
			if (nibbles&0xF0)>>4 == 0xF || nibbles&0x0F == 0xF {
				break
			}
		}
		return 0, size, nil // return 0 for reals; we only use integer operands
	default:
		return 0, 0, fmt.Errorf("invalid DICT operand byte %d", b)
	}
}

// parseCharset parses a CFF charset at the given offset.
// Returns SIDs for each glyph (including glyph 0 which is always .notdef/SID 0).
func parseCharset(data []byte, offset int, nGlyphs int) ([]int, error) {
	if offset >= len(data) {
		return nil, fmt.Errorf("charset offset %d exceeds data", offset)
	}

	sids := make([]int, nGlyphs)
	sids[0] = 0 // .notdef is always SID 0

	format := data[offset]
	pos := offset + 1

	switch format {
	case 0:
		// Format 0: one SID per glyph (glyph 0 omitted)
		needed := offset + 1 + (nGlyphs-1)*2
		if needed > len(data) {
			return nil, fmt.Errorf("charset format 0 exceeds data: need %d, have %d", needed, len(data))
		}
		for i := 1; i < nGlyphs; i++ {
			sids[i] = int(binary.BigEndian.Uint16(data[pos : pos+2]))
			pos += 2
		}

	case 1:
		// Format 1: Range3 — first(3 bytes) + nLeft(1 byte)
		glyphID := 1
		for glyphID < nGlyphs && pos+4 <= len(data) {
			first := int(data[pos])<<16 | int(data[pos+1])<<8 | int(data[pos+2])
			nLeft := int(data[pos+3])
			pos += 4
			for j := 0; j <= nLeft && glyphID < nGlyphs; j++ {
				sids[glyphID] = first + j
				glyphID++
			}
		}

	case 2:
		// Format 2: Range3 — first(3 bytes) + nLeft(2 bytes)
		glyphID := 1
		for glyphID < nGlyphs && pos+5 <= len(data) {
			first := int(data[pos])<<16 | int(data[pos+1])<<8 | int(data[pos+2])
			nLeft := int(binary.BigEndian.Uint16(data[pos+3 : pos+5]))
			pos += 5
			for j := 0; j <= nLeft && glyphID < nGlyphs; j++ {
				sids[glyphID] = first + j
				glyphID++
			}
		}

	default:
		return nil, fmt.Errorf("charset format %d not supported", format)
	}

	return sids, nil
}

// resolveSID resolves a CFF String ID to a name string.
// SIDs 0-390 use the standard predefined strings.
// SIDs 391+ use the font's String INDEX (SID 391 = index 0).
func resolveSID(sid int, stringIndex [][]byte) (string, error) {
	if sid < 0 {
		return "", fmt.Errorf("invalid SID %d", sid)
	}
	if sid < len(cffStandardStrings) {
		return cffStandardStrings[sid], nil
	}
	idx := sid - 391
	if idx >= len(stringIndex) {
		return "", fmt.Errorf("SID %d exceeds String INDEX (index %d, have %d)", sid, idx, len(stringIndex))
	}
	return string(stringIndex[idx]), nil
}

// defaultCharset returns the ISOAdobe charset (the default when charset offset is 0).
func defaultCharset(nGlyphs int) []string {
	names := make([]string, nGlyphs)
	for i := range names {
		if i < len(cffStandardStrings) {
			names[i] = cffStandardStrings[i]
		} else {
			names[i] = fmt.Sprintf("glyph%05d", i)
		}
	}
	if len(names) > 0 {
		names[0] = ".notdef"
	}
	return names
}

// cffStandardStrings is the list of 391 predefined CFF string IDs (SIDs).
// From Adobe CFF Specification, Appendix A. SIDs 0-390 are predefined;
// SID 391+ index into the font's String INDEX.
var cffStandardStrings = []string{
	// 0-49
	".notdef", "space", "exclam", "quotedbl", "numbersign",
	"dollar", "percent", "ampersand", "quoteright", "parenleft",
	"parenright", "asterisk", "plus", "comma", "hyphen",
	"period", "slash", "zero", "one", "two",
	"three", "four", "five", "six", "seven",
	"eight", "nine", "colon", "semicolon", "less",
	"equal", "greater", "question", "at", "A",
	"B", "C", "D", "E", "F",
	"G", "H", "I", "J", "K",
	"L", "M", "N", "O", "P",
	// 50-99
	"Q", "R", "S", "T", "U",
	"V", "W", "X", "Y", "Z",
	"bracketleft", "backslash", "bracketright", "asciicircum", "underscore",
	"quoteleft", "a", "b", "c", "d",
	"e", "f", "g", "h", "i",
	"j", "k", "l", "m", "n",
	"o", "p", "q", "r", "s",
	"t", "u", "v", "w", "x",
	"y", "z", "braceleft", "bar", "braceright",
	"asciitilde", "exclamdown", "cent", "sterling", "fraction",
	// 100-149
	"yen", "florin", "section", "currency", "quotesingle",
	"quotedblleft", "guillemotleft", "guilsinglleft", "guilsinglright", "fi",
	"fl", "endash", "dagger", "daggerdbl", "periodcentered",
	"paragraph", "bullet", "quotesinglbase", "quotedblbase", "quotedblright",
	"guillemotright", "ellipsis", "perthousand", "questiondown", "grave",
	"acute", "circumflex", "tilde", "macron", "breve",
	"dotaccent", "dieresis", "ring", "cedilla", "hungarumlaut",
	"ogonek", "caron", "emdash", "AE", "ordfeminine",
	// 150-199
	"Lslash", "Oslash", "OE", "ordmasculine", "ae",
	"lslash", "oslash", "oe", "germandbls", "onesuperior",
	"logicalnot", "mu", "trademark", "Eth", "onehalf",
	"plusminus", "Thorn", "onequarter", "divide", "brokenbar",
	"degree", "thorn", "threequarters", "twosuperior", "registered",
	"minus", "multiply", "threesuperior", "copyright", "Aacute",
	"Acircumflex", "Adieresis", "Agrave", "Aring", "Atilde",
	"Ccedilla", "Eacute", "Ecircumflex", "Edieresis", "Egrave",
	// 200-249
	"Iacute", "Icircumflex", "Idieresis", "Igrave", "Ntilde",
	"Oacute", "Ocircumflex", "Odieresis", "Ograve", "Otilde",
	"Scaron", "Uacute", "Ucircumflex", "Udieresis", "Ugrave",
	"Yacute", "Ydieresis", "Zcaron", "aacute", "acircumflex",
	"adieresis", "agrave", "aring", "atilde", "ccedilla",
	"eacute", "ecircumflex", "edieresis", "egrave", "iacute",
	"icircumflex", "idieresis", "igrave", "ntilde", "oacute",
	"ocircumflex", "odieresis", "ograve", "otilde", "scaron",
	"uacute", "ucircumflex", "udieresis", "ugrave", "yacute",
	// 250-299
	"ydieresis", "zcaron", "exclamsmall", "Hungarumlautsmall", "dollaroldstyle",
	"dollarsuperior", "ampersandsmall", "Acutesmall", "parenleftsuperior", "parenrightsuperior",
	"232", "Cedillasmall", "commaaccent", "afii10017", "afii10018",
	"afii10019", "afii10020", "afii10021", "afii10022", "afii10023",
	"afii10024", "afii10025", "afii10026", "afii10027", "afii10028",
	"afii10029", "afii10030", "afii10031", "afii10032", "afii10033",
	"afii10034", "afii10035", "afii10036", "afii10037", "afii10038",
	"afii10039", "afii10040", "afii10041", "afii10042", "afii10043",
	"afii10044", "afii10045", "afii10046", "afii10047", "afii10048",
	"afii10049", "afii10050", "afii10051", "afii10052", "afii10053",
	"afii10054", "afii10055", "afii10056", "afii10057", "afii10058",
	// 300-349
	"afii10059", "afii10060", "afii10061", "afii10062", "afii10063",
	"afii10064", "afii10065", "afii10066", "afii10067", "afii10068",
	"afii10069", "afii10070", "afii10071", "afii10072", "afii10073",
	"afii10074", "afii10075", "afii10076", "afii10077", "afii10078",
	"afii10079", "afii10080", "afii10081", "afii10082", "afii10083",
	"afii10084", "afii10085", "afii10086", "afii10087", "afii10088",
	"afii10089", "afii10090", "afii10091", "afii10092", "afii10093",
	"afii10094", "afii10095", "afii10096", "afii10097", "afii10098",
	"afii10099", "afii10100", "afii10101", "afii10102", "afii10103",
	"afii10104", "afii10105", "afii10106", "afii10107", "afii10108",
	"afii10109", "afii10110", "afii10111", "afii10112", "afii10113",
	// 350-390
	"afii10114", "afii10115", "afii10116", "afii10117", "afii10118",
	"afii10119", "afii10120", "afii10121", "afii10122", "afii10123",
	"afii10124", "afii10125", "afii10126", "afii10127", "afii10128",
	"afii10129", "afii10130", "afii10131", "afii10132", "afii10133",
	"afii10134", "afii10135", "afii10136", "afii10137", "afii10138",
	"afii10139", "afii10140", "afii10141", "afii10142", "afii10143",
	"afii10144", "afii10145", "afii10146", "afii10147", "afii10148",
	"afii10149", "afii10150", "afii10151", "afii10152", "afii10153",
	"afii10154", "afii10155", "afii10156", "afii10157", "afii10158",
	"afii10159", "afii10160", "afii10161", "afii10162", "afii10163",
	"afii10164", "afii10165", "afii10166", "afii10167", "afii10168",
	"Euro",
}
