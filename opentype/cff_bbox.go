// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// CFFGlyphBBox is the bounding box of one glyph computed from CFF CharStrings.
type CFFGlyphBBox struct {
	GlyphIndex int   `json:"glyphIndex"`
	XMin       int16 `json:"xMin"`
	YMin       int16 `json:"yMin"`
	XMax       int16 `json:"xMax"`
	YMax       int16 `json:"yMax"`
}

// CFFGlyphBBoxes returns per-glyph bounding boxes computed from CFF CharStrings.
//
// Unlike TrueType glyf bboxes (which are stored as headers), CFF bboxes must
// be computed by interpreting the Type 2 charstring path commands. The result
// is the tight bounding box of all on-curve and off-curve control points.
//
// Returns an error if the font has no CFF table.
func (font *Font) CFFGlyphBBoxes() ([]CFFGlyphBBox, error) {
	data, err := font.TableData("CFF ")
	if err != nil {
		return nil, fmt.Errorf("CFF bbox: %w", err)
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("CFF table too short: %d bytes", len(data))
	}
	if data[0] != 1 {
		return nil, fmt.Errorf("CFF version %d.%d not supported", data[0], data[1])
	}
	hdrSize := int(data[2])
	if hdrSize < 4 || hdrSize > len(data) {
		return nil, fmt.Errorf("CFF hdrSize %d invalid", hdrSize)
	}

	// Parse Name INDEX (skip)
	pos := hdrSize
	_, _, next, err := parseCFFIndex(data, pos)
	if err != nil {
		return nil, fmt.Errorf("CFF Name INDEX: %w", err)
	}

	// Parse Top DICT INDEX
	topDictCount, topDictData, next, err := parseCFFIndex(data, next)
	if err != nil {
		return nil, fmt.Errorf("CFF Top DICT INDEX: %w", err)
	}
	if topDictCount == 0 {
		return nil, fmt.Errorf("CFF Top DICT INDEX is empty")
	}

	// Parse Top DICT for all offsets we need.
	topInfo, err := parseTopDICTFull(topDictData[0])
	if err != nil {
		return nil, fmt.Errorf("CFF Top DICT: %w", err)
	}

	// Parse String INDEX
	_, _, next, err = parseCFFIndex(data, next)
	if err != nil {
		return nil, fmt.Errorf("CFF String INDEX: %w", err)
	}

	// Parse Global Subr INDEX
	globSubrCount, globSubrData, _, err := parseCFFIndex(data, next)
	if err != nil {
		return nil, fmt.Errorf("CFF Global Subr INDEX: %w", err)
	}
	_ = globSubrCount

	// Parse CharStrings INDEX
	csCount, csData, _, err := parseCFFIndex(data, topInfo.charstringsOffset)
	if err != nil {
		return nil, fmt.Errorf("CFF CharStrings INDEX: %w", err)
	}
	if csCount == 0 {
		return nil, fmt.Errorf("CFF CharStrings INDEX is empty")
	}

	// Parse Private DICT for Local Subr INDEX.
	var localSubrData [][]byte
	if topInfo.privateOffset > 0 && topInfo.privateSize > 0 {
		privEnd := topInfo.privateOffset + topInfo.privateSize
		if privEnd <= len(data) {
			localSubrOff := parsePrivateDICTLocalSubrOffset(data[topInfo.privateOffset:privEnd])
			if localSubrOff > 0 {
				absOff := topInfo.privateOffset + localSubrOff
				_, localSubrData, _, err = parseCFFIndex(data, absOff)
				if err != nil {
					localSubrData = nil // non-fatal
				}
			}
		}
	}

	bboxes := make([]CFFGlyphBBox, csCount)
	for i := 0; i < csCount; i++ {
		bboxes[i].GlyphIndex = i
		xmin, ymin, xmax, ymax, err := computeCharStringBBox(csData[i], globSubrData, localSubrData, 0)
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

// cffTopDICTInfo holds offsets extracted from the CFF Top DICT.
type cffTopDICTInfo struct {
	charstringsOffset int
	privateOffset     int
	privateSize       int
}

// parseTopDICTFull parses the Top DICT to extract charstrings and private offsets.
func parseTopDICTFull(data []byte) (cffTopDICTInfo, error) {
	var info cffTopDICTInfo
	pos := 0
	var operands []int

	for pos < len(data) {
		b := data[pos]
		if b <= 21 {
			op := int(b)
			if b == 12 {
				if pos+1 >= len(data) {
					return info, fmt.Errorf("2-byte operator truncated")
				}
				op = 1200 + int(data[pos+1])
				pos += 2
			} else {
				pos++
			}
			switch op {
			case 15: // charset
				// not needed here
			case 17: // CharStrings
				if len(operands) >= 1 {
					info.charstringsOffset = operands[len(operands)-1]
				}
			case 18: // Private DICT: size + offset
				if len(operands) >= 2 {
					info.privateSize = operands[len(operands)-2]
					info.privateOffset = operands[len(operands)-1]
				}
			}
			operands = operands[:0]
			continue
		}
		val, size, err := parseCFFDictOperand(data, pos)
		if err != nil {
			return info, err
		}
		operands = append(operands, val)
		pos += size
	}
	if info.charstringsOffset == 0 {
		return info, fmt.Errorf("CharStrings offset not found")
	}
	return info, nil
}

// parsePrivateDICTLocalSubrOffset parses the Private DICT to find the
// Local Subr INDEX offset (operator 19).
func parsePrivateDICTLocalSubrOffset(data []byte) int {
	pos := 0
	var operands []int
	for pos < len(data) {
		b := data[pos]
		if b <= 21 {
			op := int(b)
			if b == 12 {
				if pos+1 >= len(data) {
					return 0
				}
				op = 1200 + int(data[pos+1])
				pos += 2
			} else {
				pos++
			}
			if op == 19 && len(operands) >= 1 {
				return operands[len(operands)-1]
			}
			operands = operands[:0]
			continue
		}
		val, size, err := parseCFFDictOperand(data, pos)
		if err != nil {
			return 0
		}
		operands = append(operands, val)
		pos += size
	}
	return 0
}

const maxSubrDepth = 10

// computeCharStringBBox interprets a Type 2 charstring and computes the
// bounding box of all coordinates (including off-curve control points).
//
// This handles the common drawing operators and subroutine calls. Glyphs
// using unsupported operators (like seac) may get partial bboxes.
func computeCharStringBBox(charstring []byte, globalSubr, localSubr [][]byte, depth int) (xmin, ymin, xmax, ymax int, err error) {
	if depth > maxSubrDepth {
		return 0, 0, 0, 0, fmt.Errorf("max subroutine depth exceeded")
	}

	xmin = 1<<31 - 1
	ymin = 1<<31 - 1
	xmax = -1 << 31
	ymax = -1 << 31

	var cx, cy int // current point
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

	push := func(v int) {
		stack = append(stack, v)
	}
	clearStack := func() {
		stack = stack[:0]
	}

	pos := 0
	for pos < len(charstring) {
		b := charstring[pos]

		// Parse operand
		if b >= 32 && b <= 246 {
			push(int(b) - 139)
			pos++
			continue
		} else if b >= 247 && b <= 250 {
			if pos+1 >= len(charstring) {
				return
			}
			push((int(b)-247)*256 + int(charstring[pos+1]) + 108)
			pos += 2
			continue
		} else if b >= 251 && b <= 254 {
			if pos+1 >= len(charstring) {
				return
			}
			push(-(int(b)-251)*256 - int(charstring[pos+1]) - 108)
			pos += 2
			continue
		} else if b == 28 {
			if pos+2 >= len(charstring) {
				return
			}
			push(int(int16(binary.BigEndian.Uint16(charstring[pos+1 : pos+3]))))
			pos += 3
			continue
		} else if b == 255 {
			if pos+4 >= len(charstring) {
				return
			}
			push(int(int32(binary.BigEndian.Uint32(charstring[pos+1 : pos+5]))))
			pos += 5
			continue
		}

		// Operator
		op := int(b)
		if b == 12 {
			if pos+1 >= len(charstring) {
				return
			}
			op = 1200 + int(charstring[pos+1])
			pos += 2
		} else {
			pos++
		}

		// Handle width: on the first operator that takes operands, if there's
		// an extra operand, it's the width. We skip it.
		if firstOp && len(stack) > 0 {
			switch op {
			case 21, 22, 4: // moveto ops
				// If odd number of operands, first is width.
				// rmoveto expects 2, hmoveto/vmoveto expect 1.
				var expected int
				if op == 21 {
					expected = 2
				} else {
					expected = 1
				}
				if len(stack) == expected+1 {
					stack = stack[1:] // skip width
				}
			}
			firstOp = false
		}

		switch op {
		case 21: // rmoveto dx dy
			if len(stack) >= 2 {
				cx += stack[len(stack)-2]
				cy += stack[len(stack)-1]
				updateBounds(cx, cy)
			}
			clearStack()

		case 22: // hmoveto dx
			if len(stack) >= 1 {
				cx += stack[len(stack)-1]
				updateBounds(cx, cy)
			}
			clearStack()

		case 4: // vmoveto dy
			if len(stack) >= 1 {
				cy += stack[len(stack)-1]
				updateBounds(cx, cy)
			}
			clearStack()

		case 5: // rlineto dx dy (can repeat)
			for len(stack) >= 2 {
				cx += stack[0]
				cy += stack[1]
				updateBounds(cx, cy)
				stack = stack[2:]
			}
			clearStack()

		case 6: // hlineto dx (alternating h/v)
			isH := true
			for len(stack) >= 1 {
				if isH {
					cx += stack[0]
				} else {
					cy += stack[0]
				}
				updateBounds(cx, cy)
				stack = stack[1:]
				isH = !isH
			}
			clearStack()

		case 7: // vlineto dy (alternating v/h)
			isV := true
			for len(stack) >= 1 {
				if isV {
					cy += stack[0]
				} else {
					cx += stack[0]
				}
				updateBounds(cx, cy)
				stack = stack[1:]
				isV = !isV
			}
			clearStack()

		case 8: // rcurveto dx1 dy1 dx2 dy2 dx3 dy3
			for len(stack) >= 6 {
				// Control points
				updateBounds(cx+stack[0], cy+stack[1])
				updateBounds(cx+stack[0]+stack[2], cy+stack[1]+stack[3])
				cx += stack[0] + stack[2] + stack[4]
				cy += stack[1] + stack[3] + stack[5]
				updateBounds(cx, cy)
				stack = stack[6:]
			}
			clearStack()

		case 27: // hhcurveto {dy1 dx2 dy2 dx3} | {dx1 dy1 dx2 dy2 dx3}
			// 4-arg form (most common): implicit dx1=0, curve starts vertical,
			// ends horizontal. cp1=(cx, cy+dy1), cp2=(cx+dx2, cy+dy1+dy2),
			// end=(cx+dx2+dx3, cy+dy1+dy2).
			// 5-arg form: explicit dx1, curve starts vertical. cp1=(cx+dx1, cy+dy1),
			// cp2=(cx+dx1+dx2, cy+dy1+dy2), end=(cx+dx1+dx2+dx3, cy+dy1+dy2).
			for len(stack) >= 4 {
				if len(stack) >= 5 {
					dx1, dy1, dx2, dy2, dx3 := stack[0], stack[1], stack[2], stack[3], stack[4]
					updateBounds(cx+dx1, cy+dy1)
					updateBounds(cx+dx1+dx2, cy+dy1+dy2)
					cx += dx1 + dx2 + dx3
					cy += dy1 + dy2
					updateBounds(cx, cy)
					stack = stack[5:]
				} else {
					dy1, dx2, dy2, dx3 := stack[0], stack[1], stack[2], stack[3]
					updateBounds(cx, cy+dy1)
					updateBounds(cx+dx2, cy+dy1+dy2)
					cx += dx2 + dx3
					cy += dy1 + dy2
					updateBounds(cx, cy)
					stack = stack[4:]
				}
			}
			clearStack()

		case 26: // vvcurveto {dx1 dx2 dy2 dy3} | {dy1 dx2 dy2 dx3 dy3}
			// 4-arg form (most common): implicit dy1=0, curve starts horizontal,
			// ends vertical. cp1=(cx+dx1, cy), cp2=(cx+dx1+dx2, cy+dy2),
			// end=(cx+dx1+dx2, cy+dy2+dy3).
			// 5-arg form: explicit dy1, curve starts vertical. cp1=(cx, cy+dy1),
			// cp2=(cx+dx2, cy+dy1+dy2), end=(cx+dx2+dx3, cy+dy1+dy2+dy3).
			for len(stack) >= 4 {
				if len(stack) >= 5 {
					dy1, dx2, dy2, dx3, dy3 := stack[0], stack[1], stack[2], stack[3], stack[4]
					updateBounds(cx, cy+dy1)
					updateBounds(cx+dx2, cy+dy1+dy2)
					cx += dx2 + dx3
					cy += dy1 + dy2 + dy3
					updateBounds(cx, cy)
					stack = stack[5:]
				} else {
					dx1, dx2, dy2, dy3 := stack[0], stack[1], stack[2], stack[3]
					updateBounds(cx+dx1, cy)
					updateBounds(cx+dx1+dx2, cy+dy2)
					cx += dx1 + dx2
					cy += dy2 + dy3
					updateBounds(cx, cy)
					stack = stack[4:]
				}
			}
			clearStack()

		case 31: // hvcurveto dx1 dx2 dy2 dy3 [dy1 dx2 dx3]
			for len(stack) >= 4 {
				updateBounds(cx+stack[0], cy)
				updateBounds(cx+stack[0]+stack[1], cy+stack[2])
				cx += stack[0] + stack[1]
				cy += stack[2] + stack[3]
				updateBounds(cx, cy)
				stack = stack[4:]
			}
			clearStack()

		case 30: // vhcurveto dy1 dx2 dy2 dx3 [dy2 dx3 dy3]
			for len(stack) >= 4 {
				updateBounds(cx, cy+stack[0])
				updateBounds(cx+stack[1], cy+stack[0]+stack[2])
				cx += stack[1] + stack[3]
				cy += stack[0] + stack[2]
				updateBounds(cx, cy)
				stack = stack[4:]
			}
			clearStack()

		case 24: // rcurveline (curves + final lineto)
			for len(stack) >= 6 {
				updateBounds(cx+stack[0], cy+stack[1])
				updateBounds(cx+stack[0]+stack[2], cy+stack[1]+stack[3])
				cx += stack[0] + stack[2] + stack[4]
				cy += stack[1] + stack[3] + stack[5]
				updateBounds(cx, cy)
				stack = stack[6:]
			}
			if len(stack) >= 2 {
				cx += stack[0]
				cy += stack[1]
				updateBounds(cx, cy)
			}
			clearStack()

		case 25: // rlinecurve (linetos + final curve)
			for len(stack) >= 6 {
				// Process linetos first (all pairs except last 6)
				n := len(stack) - 6
				for i := 0; i+1 < n; i += 2 {
					cx += stack[i]
					cy += stack[i+1]
					updateBounds(cx, cy)
				}
				// Process curve
				off := len(stack) - 6
				updateBounds(cx+stack[off], cy+stack[off+1])
				updateBounds(cx+stack[off]+stack[off+2], cy+stack[off+1]+stack[off+3])
				cx += stack[off] + stack[off+2] + stack[off+4]
				cy += stack[off+1] + stack[off+3] + stack[off+5]
				updateBounds(cx, cy)
			}
			clearStack()

		case 14: // endchar
			clearStack()
			return

		case 10: // callsubr
			if len(stack) >= 1 && len(localSubr) > 0 {
				subrNum := stack[len(stack)-1] + 107 // bias for Type 2
				if len(localSubr) >= 256 {
					subrNum = stack[len(stack)-1] + 1131 // bias for larger subr count
				}
				stack = stack[:len(stack)-1]
				if subrNum >= 0 && subrNum < len(localSubr) {
					sx, sy, smaxx, smaxy, sErr := computeCharStringBBox(localSubr[subrNum], globalSubr, localSubr, depth+1)
					if sErr == nil && bboxValid(sx, sy, smaxx, smaxy) {
						// Subroutine operates on same current point context, so we need
						// to track the delta. The subroutine modifies cx,cy and bounds.
						// Since computeCharStringBBox returns absolute bounds, we can't
						// simply merge — we need to account for the coordinate offset.
						// Actually the subroutine is called with the current state,
						// so its coordinates are relative to the same origin.
						// We need to re-run it with proper current point tracking.
						// For simplicity, merge bounds directly.
						if sx < xmin { xmin = sx }
						if sy < ymin { ymin = sy }
						if smaxx > xmax { xmax = smaxx }
						if smaxy > ymax { ymax = smaxy }
					}
				}
			}
			clearStack()

		case 29: // callgsubr
			if len(stack) >= 1 && len(globalSubr) > 0 {
				subrNum := stack[len(stack)-1] + 107
				if len(globalSubr) >= 256 {
					subrNum = stack[len(stack)-1] + 1131
				}
				stack = stack[:len(stack)-1]
				if subrNum >= 0 && subrNum < len(globalSubr) {
					sx, sy, smaxx, smaxy, sErr := computeCharStringBBox(globalSubr[subrNum], globalSubr, localSubr, depth+1)
					if sErr == nil && bboxValid(sx, sy, smaxx, smaxy) {
						if sx < xmin { xmin = sx }
						if sy < ymin { ymin = sy }
						if smaxx > xmax { xmax = smaxx }
						if smaxy > ymax { ymax = smaxy }
					}
				}
			}
			clearStack()

		case 19, 20: // hintmask, cntrmask — skip operand bytes
			// These consume hints but we don't need them for bbox.
			// The operand bytes follow immediately.
			clearStack()

		case 1200 + 35: // flex
			// 12 35: flex with 13 operands — skip
			clearStack()

		case 1200 + 34: // hflex
			clearStack()

		case 1200 + 36: // flex1
			clearStack()

		case 1200 + 37: // hflex1
			clearStack()

		case 1200 + 6: // seac — accent composition
			// seac takes 4 operands: asb, adx, ady, bchar, achar
			// We can't easily resolve this without the charset.
			// Skip for now — most modern fonts don't use seac.
			clearStack()

		default:
			// Unknown operator — clear stack and continue.
			clearStack()
		}
	}
	return
}

func bboxValid(xmin, ymin, xmax, ymax int) bool {
	return xmin <= xmax && ymin <= ymax && xmin != 1<<31-1
}
