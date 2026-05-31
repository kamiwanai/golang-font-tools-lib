// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// CFF2GlyphNames returns glyph names from a CFF2 table's charset.
//
// For .otf variable fonts with CFF2 outlines, glyph names are read from
// the CFF2 charset. The CFF2 header differs from CFF1: there is no
// Name INDEX, and the Top DICT data starts immediately after the header.
//
// Glyph 0 is always ".notdef". Returns nil, nil if the font has no
// CFF2 table.
func (font *Font) CFF2GlyphNames() ([]string, error) {
	data, err := font.TableData("CFF2")
	if err != nil {
		return nil, nil
	}
	if len(data) < 5 {
		return nil, fmt.Errorf("CFF2 table too short: %d bytes", len(data))
	}

	// CFF2 header
	major := data[0]
	minor := data[1]
	if major != 2 {
		return nil, fmt.Errorf("CFF2 version %d.%d not supported (expected 2.0)", major, minor)
	}
	hdrSize := int(data[2])
	topDictLength := int(binary.BigEndian.Uint16(data[3:5]))

	if hdrSize < 5 || hdrSize > len(data) {
		return nil, fmt.Errorf("CFF2 hdrSize %d invalid", hdrSize)
	}
	if hdrSize+topDictLength > len(data) {
		return nil, fmt.Errorf("CFF2 Top DICT exceeds data")
	}

	// Parse Top DICT
	topDict := data[hdrSize : hdrSize+topDictLength]
	charstringsOffset, charsetOffset, fdSelectOffset, fdArrayOffset, _ := parseCFF2TopDICT(topDict)

	if charstringsOffset == 0 {
		return nil, fmt.Errorf("CFF2 CharStrings offset not found")
	}

	// Parse CharStrings INDEX
	nGlyphs, _, _, err := parseCFFIndex(data, charstringsOffset)
	if err != nil {
		return nil, fmt.Errorf("CFF2 CharStrings INDEX: %w", err)
	}
	if nGlyphs == 0 {
		return nil, fmt.Errorf("CFF2 CharStrings INDEX is empty")
	}

	_ = fdSelectOffset
	_ = fdArrayOffset

	// If charset offset is 0 or 1, use default charsets
	if charsetOffset <= 1 {
		return defaultCFF2Charset(nGlyphs, charsetOffset), nil
	}

	// Parse charset
	sids, err := parseCharset(data, charsetOffset, nGlyphs)
	if err != nil {
		return nil, fmt.Errorf("CFF2 charset: %w", err)
	}

	// For CFF2, there's no String INDEX — SIDs are resolved via CFF
	// standard strings only. SID 391+ use glyphNNNNN names.
	names := make([]string, nGlyphs)
	for i, sid := range sids {
		if sid >= 0 && sid < len(cffStandardStrings) {
			names[i] = cffStandardStrings[sid]
		} else {
			names[i] = fmt.Sprintf("glyph%05d", i)
		}
	}
	if len(names) > 0 {
		names[0] = ".notdef"
	}
	return names, nil
}

// defaultCFF2Charset returns the charset for a CFF2 font when the charset
// offset is 0 (ISOAdobe) or 1 (Expert).
func defaultCFF2Charset(nGlyphs int, charsetID int) []string {
	names := make([]string, nGlyphs)
	if nGlyphs > 0 {
		names[0] = ".notdef"
	}
	// CFF2 charset 0 = ISOAdobe, 1 = Expert, 2 = ExpertSubset
	// For simplicity, we use glyphNNNNN for charset 1/2
	if charsetID == 0 {
		for i := 1; i < nGlyphs && i < len(cffStandardStrings); i++ {
			names[i] = cffStandardStrings[i]
		}
	}
	for i := 0; i < nGlyphs; i++ {
		if names[i] == "" {
			names[i] = fmt.Sprintf("glyph%05d", i)
		}
	}
	return names
}

// parseCFF2TopDICT parses a CFF2 Top DICT and returns key offsets.
// CFF2 Top DICT operators are the same operand encoding as CFF1 but
// with CFF2-specific operator codes.
func parseCFF2TopDICT(data []byte) (charstringsOffset, charsetOffset, fdSelectOffset, fdArrayOffset, varStoreOffset int) {
	pos := 0
	var operands []int

	for pos < len(data) {
		b := data[pos]

		// Operator: 0-31 (CFF2 extends the range to 31)
		if b <= 31 {
			op := int(b)
			if b == 12 {
				if pos+1 >= len(data) {
					return
				}
				op = 1200 + int(data[pos+1])
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
			case 18: // FDArray
				if len(operands) >= 1 {
					fdArrayOffset = operands[len(operands)-1]
				}
			case 19: // FDSelect
				if len(operands) >= 1 {
					fdSelectOffset = operands[len(operands)-1]
				}
			case 24: // VariationStore (CFF2-specific)
				if len(operands) >= 1 {
					varStoreOffset = operands[len(operands)-1]
				}
			}
			operands = operands[:0]
			continue
		}

		// Parse operand (same encoding as CFF1)
		val, size, err := parseCFFDictOperand(data, pos)
		if err != nil {
			return
		}
		operands = append(operands, val)
		pos += size
	}
	return
}

// HasCFF2 reports whether the font has a CFF2 table.
func (font *Font) HasCFF2() bool {
	_, err := font.TableData("CFF2")
	return err == nil
}
