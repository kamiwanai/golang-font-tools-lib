// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// CFF2GlyphBBox is the bounding box of one glyph computed from CFF2 CharStrings.
type CFF2GlyphBBox struct {
	GlyphIndex int   `json:"glyphIndex"`
	XMin       int16 `json:"xMin"`
	YMin       int16 `json:"yMin"`
	XMax       int16 `json:"xMax"`
	YMax       int16 `json:"yMax"`
}

// CFF2GlyphBBoxes returns per-glyph bounding boxes computed from CFF2 CharStrings.
func (font *Font) CFF2GlyphBBoxes() ([]CFF2GlyphBBox, error) {
	data, err := font.TableData("CFF2")
	if err != nil {
		return nil, fmt.Errorf("CFF2 bbox: %w", err)
	}
	if len(data) < 5 {
		return nil, fmt.Errorf("CFF2 table too short: %d bytes", len(data))
	}

	major := data[0]
	minor := data[1]
	if major != 2 {
		return nil, fmt.Errorf("CFF2 version %d.%d not supported", major, minor)
	}
	hdrSize := int(data[2])
	topDictLength := int(binary.BigEndian.Uint16(data[3:5]))

	if hdrSize < 5 || hdrSize > len(data) {
		return nil, fmt.Errorf("CFF2 hdrSize %d invalid", hdrSize)
	}
	if hdrSize+topDictLength > len(data) {
		return nil, fmt.Errorf("CFF2 Top DICT exceeds data")
	}

	topDict := data[hdrSize : hdrSize+topDictLength]
	csOffset, _, fdSelectOff, fdArrayOff, _ := parseCFF2TopDICT(topDict)

	if csOffset == 0 {
		return nil, fmt.Errorf("CFF2 CharStrings offset not found")
	}

	csCount, csData, _, err := parseCFFIndex(data, csOffset)
	if err != nil {
		return nil, fmt.Errorf("CFF2 CharStrings INDEX: %w", err)
	}
	if csCount == 0 {
		return nil, fmt.Errorf("CFF2 CharStrings INDEX is empty")
	}

	var fdArray []cff2FontDICT
	if fdArrayOff > 0 {
		fdArray, err = parseCFF2FDArray(data, fdArrayOff)
		if err != nil {
			return nil, fmt.Errorf("CFF2 FDArray: %w", err)
		}
	}

	var fdSelect []uint16
	if fdSelectOff > 0 && len(fdArray) > 0 {
		fdSelect, err = parseCFF2FDSelect(data, fdSelectOff, csCount)
		if err != nil {
			return nil, fmt.Errorf("CFF2 FDSelect: %w", err)
		}
	}

	globSubrCount, globSubrData, _, _ := parseCFFIndex(data, hdrSize+topDictLength)

	var localSubrsByFD [][][]byte
	if len(fdArray) > 0 {
		localSubrsByFD = make([][][]byte, len(fdArray))
		for fdi, fd := range fdArray {
			if fd.privateOffset > 0 && fd.privateSize > 0 {
				privEnd := fd.privateOffset + fd.privateSize
				if privEnd <= len(data) {
					localSubrOff := parsePrivateDICTLocalSubrOffset(data[fd.privateOffset:privEnd])
					if localSubrOff > 0 {
						absOff := fd.privateOffset + localSubrOff
						_, lsData, _, _ := parseCFFIndex(data, absOff)
						localSubrsByFD[fdi] = lsData
					}
				}
			}
		}
	}

	bboxes := make([]CFF2GlyphBBox, csCount)
	var globSubr [][]byte
	if globSubrCount > 0 {
		globSubr = globSubrData
	}

	for i := 0; i < csCount; i++ {
		bboxes[i].GlyphIndex = i
		var locSubr [][]byte
		if len(fdSelect) > i && len(localSubrsByFD) > int(fdSelect[i]) {
			locSubr = localSubrsByFD[fdSelect[i]]
		}
		xmin, ymin, xmax, ymax, err := computeCFF2CharStringBBox(csData[i], globSubr, locSubr, 0)
		if err != nil || !bboxValid(xmin, ymin, xmax, ymax) {
			continue
		}
		bboxes[i].XMin = int16(xmin)
		bboxes[i].YMin = int16(ymin)
		bboxes[i].XMax = int16(xmax)
		bboxes[i].YMax = int16(ymax)
	}
	return bboxes, nil
}

type cff2FontDICT struct {
	privateOffset int
	privateSize   int
}

func parseCFF2FDArray(data []byte, offset int) ([]cff2FontDICT, error) {
	fdCount, fdData, _, err := parseCFFIndex(data, offset)
	if err != nil {
		return nil, err
	}
	result := make([]cff2FontDICT, fdCount)
	for i := 0; i < fdCount; i++ {
		result[i].privateOffset, result[i].privateSize = parseCFF2FontDICT(fdData[i])
	}
	return result, nil
}

func parseCFF2FontDICT(data []byte) (privateOffset, privateSize int) {
	pos := 0
	var operands []int
	for pos < len(data) {
		b := data[pos]
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
			if op == 18 && len(operands) >= 2 {
				privateSize = operands[len(operands)-2]
				privateOffset = operands[len(operands)-1]
			}
			operands = operands[:0]
			continue
		}
		val, size, err := parseCFFDictOperand(data, pos)
		if err != nil {
			return
		}
		operands = append(operands, val)
		pos += size
	}
	return
}

func parseCFF2FDSelect(data []byte, offset int, nGlyphs int) ([]uint16, error) {
	if offset < 0 || offset >= len(data) {
		return nil, fmt.Errorf("FDSelect offset %d out of range", offset)
	}
	format := data[offset]
	switch format {
	case 0:
		if offset+1+nGlyphs > len(data) {
			return nil, fmt.Errorf("FDSelect format 0 data truncated")
		}
		result := make([]uint16, nGlyphs)
		for i := 0; i < nGlyphs; i++ {
			result[i] = uint16(data[offset+1+i])
		}
		return result, nil
	case 3:
		if offset+3 > len(data) {
			return nil, fmt.Errorf("FDSelect format 3 header truncated")
		}
		nRanges := int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
		if nRanges > MaxCFF2FDRanges {
			return nil, fmt.Errorf("FDSelect format 3 range count %d exceeds limit", nRanges)
		}
		result := make([]uint16, nGlyphs)
		pos := offset + 3
		for r := 0; r < nRanges; r++ {
			if pos+5 > len(data) {
				return nil, fmt.Errorf("FDSelect format 3 range %d truncated", r)
			}
			first := int(binary.BigEndian.Uint16(data[pos : pos+2]))
			fd := data[pos+2]
			pos += 5
			if first >= nGlyphs {
				return nil, fmt.Errorf("FDSelect format 3: first glyph %d exceeds count %d", first, nGlyphs)
			}
			result[first] = uint16(fd)
		}
		for i := 1; i < nGlyphs; i++ {
			if result[i] == 0 && result[i-1] != 0 {
				result[i] = result[i-1]
			}
		}
		return result, nil
	case 4:
		if offset+3 > len(data) {
			return nil, fmt.Errorf("FDSelect format 4 header truncated")
		}
		nRanges := int(binary.BigEndian.Uint16(data[offset+1 : offset+3]))
		if nRanges > MaxCFF2FDRanges {
			return nil, fmt.Errorf("FDSelect format 4 range count %d exceeds limit", nRanges)
		}
		result := make([]uint16, nGlyphs)
		pos := offset + 3
		for r := 0; r < nRanges; r++ {
			if pos+4 > len(data) {
				return nil, fmt.Errorf("FDSelect format 4 range %d truncated", r)
			}
			first := int(binary.BigEndian.Uint16(data[pos : pos+2]))
			fd := binary.BigEndian.Uint16(data[pos+2 : pos+4])
			pos += 4
			if first >= nGlyphs {
				return nil, fmt.Errorf("FDSelect format 4: first glyph %d exceeds count %d", first, nGlyphs)
			}
			result[first] = fd
		}
		for i := 1; i < nGlyphs; i++ {
			if result[i] == 0 && result[i-1] != 0 {
				result[i] = result[i-1]
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("CFF2 FDSelect unsupported format %d", format)
	}
}

func computeCFF2CharStringBBox(charstring []byte, globalSubr, localSubr [][]byte, depth int) (xmin, ymin, xmax, ymax int, err error) {
	if depth > maxSubrDepth {
		return 0, 0, 0, 0, fmt.Errorf("max subroutine depth exceeded")
	}
	xmin = 1<<31 - 1
	ymin = 1<<31 - 1
	xmax = -1 << 31
	ymax = -1 << 31

	var cx, cy int
	var stack []int
	started := false
	firstOp := true

	updateBounds := func(px, py int) {
		if !started {
			started = true
			xmin, ymin = px, py
			xmax, ymax = px, py
		}
		if px < xmin { xmin = px }
		if py < ymin { ymin = py }
		if px > xmax { xmax = px }
		if py > ymax { ymax = py }
	}

	push := func(v int) { stack = append(stack, v) }
	clearStack := func() { stack = stack[:0] }

	pos := 0
	for pos < len(charstring) {
		b := charstring[pos]
		if b >= 32 && b <= 246 {
			push(int(b) - 139)
			pos++
			continue
		} else if b >= 247 && b <= 250 {
			if pos+1 >= len(charstring) { return }
			push((int(b)-247)*256 + int(charstring[pos+1]) + 108)
			pos += 2
			continue
		} else if b >= 251 && b <= 254 {
			if pos+1 >= len(charstring) { return }
			push(-(int(b)-251)*256 - int(charstring[pos+1]) - 108)
			pos += 2
			continue
		} else if b == 28 {
			if pos+2 >= len(charstring) { return }
			push(int(int16(binary.BigEndian.Uint16(charstring[pos+1 : pos+3]))))
			pos += 3
			continue
		} else if b == 255 {
			if pos+4 >= len(charstring) { return }
			push(int(int32(binary.BigEndian.Uint32(charstring[pos+1 : pos+5]))))
			pos += 5
			continue
		}

		op := int(b)
		if b == 12 {
			if pos+1 >= len(charstring) { return }
			op = 1200 + int(charstring[pos+1])
			pos += 2
		} else {
			pos++
		}

		if firstOp && len(stack) > 0 {
			switch op {
			case 21, 22, 4:
				var expected int
				if op == 21 { expected = 2 } else { expected = 1 }
				if len(stack) == expected+1 {
					stack = stack[1:]
				}
			}
			firstOp = false
		}

		switch op {
		case 21:
			if len(stack) >= 2 {
				cx += stack[len(stack)-2]
				cy += stack[len(stack)-1]
				updateBounds(cx, cy)
			}
			clearStack()
		case 22:
			if len(stack) >= 1 {
				cx += stack[len(stack)-1]
				updateBounds(cx, cy)
			}
			clearStack()
		case 4:
			if len(stack) >= 1 {
				cy += stack[len(stack)-1]
				updateBounds(cx, cy)
			}
			clearStack()
		case 5:
			for len(stack) >= 2 {
				cx += stack[0]; cy += stack[1]
				updateBounds(cx, cy)
				stack = stack[2:]
			}
			clearStack()
		case 6:
			isH := true
			for len(stack) >= 1 {
				if isH { cx += stack[0] } else { cy += stack[0] }
				updateBounds(cx, cy)
				stack = stack[1:]
				isH = !isH
			}
			clearStack()
		case 7:
			isV := true
			for len(stack) >= 1 {
				if isV { cy += stack[0] } else { cx += stack[0] }
				updateBounds(cx, cy)
				stack = stack[1:]
				isV = !isV
			}
			clearStack()
		case 8:
			for len(stack) >= 6 {
				updateBounds(cx+stack[0], cy+stack[1])
				updateBounds(cx+stack[0]+stack[2], cy+stack[1]+stack[3])
				cx += stack[0] + stack[2] + stack[4]
				cy += stack[1] + stack[3] + stack[5]
				updateBounds(cx, cy)
				stack = stack[6:]
			}
			clearStack()
		case 27:
			for len(stack) >= 4 {
				updateBounds(cx+stack[0], cy+stack[1])
				updateBounds(cx+stack[0]+stack[2], cy+stack[1]+stack[3])
				cx += stack[0] + stack[2]
				cy += stack[1] + stack[3]
				updateBounds(cx, cy)
				stack = stack[4:]
			}
			clearStack()
		case 26:
			for len(stack) >= 4 {
				updateBounds(cx+stack[0], cy+stack[1])
				updateBounds(cx+stack[0]+stack[2], cy+stack[1]+stack[3])
				cx += stack[0] + stack[2]
				cy += stack[1] + stack[3]
				updateBounds(cx, cy)
				stack = stack[4:]
			}
			clearStack()
		case 31:
			for len(stack) >= 4 {
				updateBounds(cx+stack[0], cy)
				updateBounds(cx+stack[0]+stack[1], cy+stack[2])
				cx += stack[0] + stack[1]
				cy += stack[2] + stack[3]
				updateBounds(cx, cy)
				stack = stack[4:]
			}
			clearStack()
		case 30:
			for len(stack) >= 4 {
				updateBounds(cx, cy+stack[0])
				updateBounds(cx+stack[1], cy+stack[0]+stack[2])
				cx += stack[1] + stack[3]
				cy += stack[0] + stack[2]
				updateBounds(cx, cy)
				stack = stack[4:]
			}
			clearStack()
		case 24:
			for len(stack) >= 6 {
				updateBounds(cx+stack[0], cy+stack[1])
				updateBounds(cx+stack[0]+stack[2], cy+stack[1]+stack[3])
				cx += stack[0] + stack[2] + stack[4]
				cy += stack[1] + stack[3] + stack[5]
				updateBounds(cx, cy)
				stack = stack[6:]
			}
			if len(stack) >= 2 {
				cx += stack[0]; cy += stack[1]
				updateBounds(cx, cy)
			}
			clearStack()
		case 25:
			n := len(stack) - 6
			for i := 0; i+1 < n; i += 2 {
				cx += stack[i]; cy += stack[i+1]
				updateBounds(cx, cy)
			}
			off := len(stack) - 6
			if off >= 0 && len(stack) >= 6 {
				updateBounds(cx+stack[off], cy+stack[off+1])
				updateBounds(cx+stack[off]+stack[off+2], cy+stack[off+1]+stack[off+3])
				cx += stack[off] + stack[off+2] + stack[off+4]
				cy += stack[off+1] + stack[off+3] + stack[off+5]
				updateBounds(cx, cy)
			}
			clearStack()
		case 14:
			clearStack()
			return
		case 10:
			if len(stack) >= 1 && len(localSubr) > 0 {
				subrNum := stack[len(stack)-1] + calcSubrBias(len(localSubr))
				stack = stack[:len(stack)-1]
				if subrNum >= 0 && subrNum < len(localSubr) {
					sx, sy, smaxx, smaxy, sErr := computeCFF2CharStringBBox(localSubr[subrNum], globalSubr, localSubr, depth+1)
					if sErr == nil && bboxValid(sx, sy, smaxx, smaxy) {
						if sx < xmin { xmin = sx }
						if sy < ymin { ymin = sy }
						if smaxx > xmax { xmax = smaxx }
						if smaxy > ymax { ymax = smaxy }
					}
				}
			}
			clearStack()
		case 29:
			if len(stack) >= 1 && len(globalSubr) > 0 {
				subrNum := stack[len(stack)-1] + calcSubrBias(len(globalSubr))
				stack = stack[:len(stack)-1]
				if subrNum >= 0 && subrNum < len(globalSubr) {
					sx, sy, smaxx, smaxy, sErr := computeCFF2CharStringBBox(globalSubr[subrNum], globalSubr, localSubr, depth+1)
					if sErr == nil && bboxValid(sx, sy, smaxx, smaxy) {
						if sx < xmin { xmin = sx }
						if sy < ymin { ymin = sy }
						if smaxx > xmax { xmax = smaxx }
						if smaxy > ymax { ymax = smaxy }
					}
				}
			}
			clearStack()
		case 19, 20, 1200 + 34, 1200 + 35, 1200 + 36, 1200 + 37, 1200 + 6:
			clearStack()
		default:
			clearStack()
		}
	}
	return
}

func calcSubrBias(count int) int {
	if count < 1240 { return 107 }
	if count < 33900 { return 1131 }
	return 32768
}
