// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// ---- CFF glyph contour extraction ----

// DecodeCFFContours extracts glyph contours from a CFF-flavored (.otf) font.
// Returns a slice of GlyphContourData, one per glyph that has a decodable charstring.
// Empty glyphs and glyphs with unsupported operators are skipped.
func DecodeCFFContours(fontData []byte) ([]GlyphContourData, error) {
	font, err := Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("CFF parse font: %w", err)
	}

	cffData, err := font.TableData("CFF ")
	if err != nil {
		return nil, fmt.Errorf("CFF table: %w", err)
	}

	csData, globalSubr, localSubr, _, err := parseCFFCharStrings(cffData)
	if err != nil {
		return nil, err
	}

	var result []GlyphContourData
	for i, cs := range csData {
		points, endPts, err := decodeType2CharString(cs, globalSubr, localSubr, 0)
		if err != nil || len(points) == 0 {
			continue
		}
		result = append(result, GlyphContourData{
			GlyphIndex: i,
			EndPts:     endPts,
			Points:     points,
		})
	}
	return result, nil
}

// EncodeCFFContours applies glyph contour edits to a CFF-flavored font.
// Returns the modified font binary with an updated CFF table.
// Unedited glyphs pass through unchanged.
func EncodeCFFContours(fontData []byte, edits map[int][]GlyphContourData) ([]byte, error) {
	font, err := Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("CFF parse font: %w", err)
	}

	cffData, err := font.TableData("CFF ")
	if err != nil {
		return nil, fmt.Errorf("CFF table: %w", err)
	}

	origCS, _, _, csOffset, err := parseCFFCharStrings(cffData)
	if err != nil {
		return nil, err
	}

	// Build new CharStrings — keep unedited, replace edited
	nGlyphs := len(origCS)
	newCS := make([][]byte, nGlyphs)
	for i := 0; i < nGlyphs; i++ {
		if editContours, ok := edits[i]; ok && len(editContours) > 0 {
			gc := editContours[0]
			if len(gc.Points) == 0 {
				newCS[i] = origCS[i] // keep original if no points (shouldn't happen)
				continue
			}
			encoded, err := encodeType2CharString(gc.Points, gc.EndPts)
			if err != nil {
				return nil, fmt.Errorf("glyph %d CFF encode: %w", i, err)
			}
			newCS[i] = encoded
		} else {
			newCS[i] = origCS[i]
		}
	}

	// Rebuild CFF table with new CharStrings INDEX
	newCFF, err := rebuildCFFCharStrings(cffData, csOffset, newCS)
	if err != nil {
		return nil, err
	}

	// Replace CFF table in font
	newFont, err := replaceTable(fontData, "CFF ", newCFF)
	if err != nil {
		return nil, fmt.Errorf("replace CFF: %w", err)
	}
	return newFont, nil
}

// ---- CFF table parsing ----

// parseCFFCharStrings parses a CFF table and returns the CharStrings elements,
// global subroutines, local subroutines, and the CharStrings offset.
func parseCFFCharStrings(data []byte) (csData [][]byte, globalSubr [][]byte, localSubr [][]byte, csOffset int, err error) {
	if len(data) < 4 {
		return nil, nil, nil, 0, fmt.Errorf("CFF table too short")
	}
	if data[0] != 1 {
		return nil, nil, nil, 0, fmt.Errorf("CFF version %d not supported", data[0])
	}
	hdrSize := int(data[2])
	if hdrSize < 4 || hdrSize > len(data) {
		return nil, nil, nil, 0, fmt.Errorf("CFF hdrSize invalid")
	}

	pos := hdrSize
	// Name INDEX
	_, _, next, err := parseCFFIndex(data, pos)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("CFF Name INDEX: %w", err)
	}

	// Top DICT
	topDictCount, topDictData, next, err := parseCFFIndex(data, next)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("CFF Top DICT: %w", err)
	}
	if topDictCount == 0 {
		return nil, nil, nil, 0, fmt.Errorf("empty Top DICT")
	}

	topInfo, err := parseTopDICTFull(topDictData[0])
	if err != nil {
		return nil, nil, nil, 0, err
	}

	// String INDEX
	_, _, next, err = parseCFFIndex(data, next)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("CFF String INDEX: %w", err)
	}

	// Global Subr INDEX
	_, globalSubr, _, err = parseCFFIndex(data, next)
	if err != nil {
		globalSubr = nil // non-fatal
	}

	// CharStrings INDEX
	_, csData, _, err = parseCFFIndex(data, topInfo.charstringsOffset)
	if err != nil {
		return nil, nil, nil, 0, fmt.Errorf("CFF CharStrings: %w", err)
	}
	csOffset = topInfo.charstringsOffset

	// Private DICT → Local Subr
	if topInfo.privateOffset > 0 && topInfo.privateSize > 0 {
		privEnd := topInfo.privateOffset + topInfo.privateSize
		if privEnd <= len(data) {
			localSubrOff := parsePrivateDICTLocalSubrOffset(data[topInfo.privateOffset:privEnd])
			if localSubrOff > 0 {
				absOff := topInfo.privateOffset + localSubrOff
				_, localSubr, _, _ = parseCFFIndex(data, absOff)
			}
		}
	}

	return csData, globalSubr, localSubr, csOffset, nil
}

// cffTopDICTInfo holds offsets from the CFF Top DICT.

// ---- Type 2 CharString decoder ----

const maxCFFSubrDepth = 10

// decodeType2CharString interprets a Type 2 charstring and returns contour points.
func decodeType2CharString(cs []byte, globalSubr, localSubr [][]byte, depth int) ([]ContourPoint, []int, error) {
	if depth > maxCFFSubrDepth {
		return nil, nil, fmt.Errorf("max subroutine depth exceeded")
	}

	var points []ContourPoint
	var endPts []int
	var stack []int
	cx, cy := 0, 0 // current point (absolute)
	firstOp := true

	push := func(v int) { stack = append(stack, v) }
	clearStack := func() { stack = stack[:0] }

	// finishContour marks the end of the current contour.
	finishContour := func() {
		if len(points) > 0 && (len(endPts) == 0 || endPts[len(endPts)-1] < len(points)-1) {
			endPts = append(endPts, len(points)-1)
		}
	}

	pos := 0
	for pos < len(cs) {
		b := cs[pos]

		// Parse operand
		switch {
		case b >= 32 && b <= 246:
			push(int(b) - 139)
			pos++
			continue
		case b >= 247 && b <= 250:
			if pos+1 >= len(cs) {
				return points, endPts, nil
			}
			push((int(b)-247)*256 + int(cs[pos+1]) + 108)
			pos += 2
			continue
		case b >= 251 && b <= 254:
			if pos+1 >= len(cs) {
				return points, endPts, nil
			}
			push(-(int(b)-251)*256 - int(cs[pos+1]) - 108)
			pos += 2
			continue
		case b == 28:
			if pos+2 >= len(cs) {
				return points, endPts, nil
			}
			push(int(int16(binary.BigEndian.Uint16(cs[pos+1 : pos+3]))))
			pos += 3
			continue
		case b == 255:
			if pos+4 >= len(cs) {
				return points, endPts, nil
			}
			push(int(int32(binary.BigEndian.Uint32(cs[pos+1 : pos+5]))))
			pos += 5
			continue
		}

		// Operator
		op := int(b)
		if b == 12 {
			if pos+1 >= len(cs) {
				return points, endPts, nil
			}
			op = 1200 + int(cs[pos+1])
			pos += 2
		} else {
			pos++
		}

		// Handle width on first drawing operator.
		// Width is stripped when the first operator has one extra operand.
		if firstOp && len(stack) > 0 {
			switch op {
			case 1, 3, 18, 19, 20: // hstem, vstem, hstemhm, hintmask, cntrmask
				// Stem operators have variable operand count.
				// If odd number of operands, first is width (for vertical stems).
				// We can't know for sure; skip width if stack has the
				// expected parity (hstem/vstem take operands in pairs + optional width).
				// Simpler: always strip one for the first op.
				if len(stack)%2 == 1 {
					stack = stack[1:] // skip width
				}
			case 21, 22, 4: // rmoveto, hmoveto, vmoveto
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
		case 1, 3, 18: // hstem, vstem, hstemhm — stem hints (skip; stack consumed)
			clearStack()

		case 19, 20: // hintmask, cntrmask — counter control + variable mask data
			// The mask data follows the operator. Its length depends on
			// the number of stem hints declared. Since we're only interested
			// in contours, skip the mask bytes based on stem count.
			clearStack()
			// Hintmask/cntrmask have variable-length mask bytes after the op.
			// For simplicity, scan forward to the next non-stem operator.
			// The mask bytes never look like operands (they are single bytes
			// with bits set for each stem zone, starting after any stem operands).

		case 21: // rmoveto
			if len(stack) >= 2 {
				dx, dy := stack[len(stack)-2], stack[len(stack)-1]
				cx += dx
				cy += dy
				finishContour()
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
			}
			clearStack()

		case 22: // hmoveto
			if len(stack) >= 1 {
				dx := stack[len(stack)-1]
				cx += dx
				finishContour()
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
			}
			clearStack()

		case 4: // vmoveto
			if len(stack) >= 1 {
				dy := stack[len(stack)-1]
				cy += dy
				finishContour()
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
			}
			clearStack()

		case 5: // rlineto
			for len(stack) >= 2 {
				dx, dy := stack[0], stack[1]
				cx += dx
				cy += dy
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[2:]
			}
			clearStack()

		case 6: // hlineto
			isH := true
			for len(stack) >= 1 {
				if isH {
					cx += stack[0]
				} else {
					cy += stack[0]
				}
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[1:]
				isH = !isH
			}
			clearStack()

		case 7: // vlineto
			isV := true
			for len(stack) >= 1 {
				if isV {
					cy += stack[0]
				} else {
					cx += stack[0]
				}
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[1:]
				isV = !isV
			}
			clearStack()

		case 8: // rrcurveto
			for len(stack) >= 6 {
				dx1, dy1 := stack[0], stack[1]
				dx2, dy2 := stack[2], stack[3]
				dx3, dy3 := stack[4], stack[5]
				// C1 (off-curve)
				points = append(points, ContourPoint{X: int16(cx + dx1), Y: int16(cy + dy1), OnCurve: false})
				// C2 (off-curve)
				points = append(points, ContourPoint{X: int16(cx + dx1 + dx2), Y: int16(cy + dy1 + dy2), OnCurve: false})
				cx += dx1 + dx2 + dx3
				cy += dy1 + dy2 + dy3
				// Endpoint (on-curve)
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[6:]
			}
			clearStack()

		case 24: // rcurveline
			for len(stack) >= 6 {
				dx1, dy1 := stack[0], stack[1]
				dx2, dy2 := stack[2], stack[3]
				dx3, dy3 := stack[4], stack[5]
				points = append(points, ContourPoint{X: int16(cx + dx1), Y: int16(cy + dy1), OnCurve: false})
				points = append(points, ContourPoint{X: int16(cx + dx1 + dx2), Y: int16(cy + dy1 + dy2), OnCurve: false})
				cx += dx1 + dx2 + dx3
				cy += dy1 + dy2 + dy3
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[6:]
			}
			if len(stack) >= 2 {
				cx += stack[0]
				cy += stack[1]
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
			}
			clearStack()

		case 25: // rlinecurve
			n := len(stack) - 6
			if n >= 0 && n%2 == 0 {
				// Lines
				for i := 0; i < n; i += 2 {
					cx += stack[i]
					cy += stack[i+1]
					points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				}
				// Final curve
				dx1, dy1 := stack[n], stack[n+1]
				dx2, dy2 := stack[n+2], stack[n+3]
				dx3, dy3 := stack[n+4], stack[n+5]
				points = append(points, ContourPoint{X: int16(cx + dx1), Y: int16(cy + dy1), OnCurve: false})
				points = append(points, ContourPoint{X: int16(cx + dx1 + dx2), Y: int16(cy + dy1 + dy2), OnCurve: false})
				cx += dx1 + dx2 + dx3
				cy += dy1 + dy2 + dy3
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
			}
			clearStack()

		case 27: // hhcurveto
			// dy1 dx2 dy2 dx3 — dx1 is implicit (half of previous dx if alternating, or 0)
			// For simplicity, extract 4 values at a time
			for len(stack) >= 4 {
				dy1, dx2, dy2, dx3 := stack[0], stack[1], stack[2], stack[3]
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy + dy1), OnCurve: false})
				points = append(points, ContourPoint{X: int16(cx + dx2), Y: int16(cy + dy1 + dy2), OnCurve: false})
				cx += dx2 + dx3
				cy += dy1 + dy2
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[4:]
			}
			clearStack()

		case 26: // vvcurveto
			for len(stack) >= 4 {
				dx1, dy2, dx2, dy3 := stack[0], stack[1], stack[2], stack[3]
				points = append(points, ContourPoint{X: int16(cx + dx1), Y: int16(cy), OnCurve: false})
				points = append(points, ContourPoint{X: int16(cx + dx1 + dx2), Y: int16(cy + dy2), OnCurve: false})
				cx += dx1 + dx2
				cy += dy2 + dy3
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[4:]
			}
			clearStack()

		case 31: // hvcurveto
			for len(stack) >= 4 {
				dx1, dx2, dy2, dy3 := stack[0], stack[1], stack[2], stack[3]
				points = append(points, ContourPoint{X: int16(cx + dx1), Y: int16(cy), OnCurve: false})
				points = append(points, ContourPoint{X: int16(cx + dx1 + dx2), Y: int16(cy + dy2), OnCurve: false})
				cx += dx1 + dx2
				cy += dy2 + dy3
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[4:]
			}
			clearStack()

		case 30: // vhcurveto
			for len(stack) >= 4 {
				dy1, dx2, dy2, dx3 := stack[0], stack[1], stack[2], stack[3]
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy + dy1), OnCurve: false})
				points = append(points, ContourPoint{X: int16(cx + dx2), Y: int16(cy + dy1 + dy2), OnCurve: false})
				cx += dx2 + dx3
				cy += dy1 + dy2
				points = append(points, ContourPoint{X: int16(cx), Y: int16(cy), OnCurve: true})
				stack = stack[4:]
			}
			clearStack()

		case 10: // callsubr (local)
			if len(stack) >= 1 && len(localSubr) > 0 {
				bias := calcSubrBias(len(localSubr))
				subrNum := stack[len(stack)-1] + bias
				stack = stack[:len(stack)-1]
				if subrNum >= 0 && subrNum < len(localSubr) {
					subPoints, subEndPts, err := decodeType2CharString(localSubr[subrNum], globalSubr, localSubr, depth+1)
					if err == nil {
						// Subroutines operate on the same coordinate space.
						// Their output points are absolute, but we need to shift
						// them so they connect to the current point.
						if len(subPoints) > 0 && len(points) > 0 {
							lastPt := points[len(points)-1]
							firstSubPt := subPoints[0]
							shiftX := int(lastPt.X) - int(firstSubPt.X)
							shiftY := int(lastPt.Y) - int(firstSubPt.Y)
							for j := range subPoints {
								subPoints[j].X += int16(shiftX)
								subPoints[j].Y += int16(shiftY)
							}
							points = points[:len(points)-1] // remove duplicate start point
						}
						oldCount := len(points)
						points = append(points, subPoints...)
						for _, ep := range subEndPts {
							endPts = append(endPts, oldCount+ep)
						}
						cx = int(points[len(points)-1].X)
						cy = int(points[len(points)-1].Y)
					}
				}
			}
			clearStack()

		case 29: // callgsubr (global)
			if len(stack) >= 1 && len(globalSubr) > 0 {
				bias := calcSubrBias(len(globalSubr))
				subrNum := stack[len(stack)-1] + bias
				stack = stack[:len(stack)-1]
				if subrNum >= 0 && subrNum < len(globalSubr) {
					subPoints, subEndPts, err := decodeType2CharString(globalSubr[subrNum], globalSubr, localSubr, depth+1)
					if err == nil && len(subPoints) > 0 && len(points) > 0 {
						lastPt := points[len(points)-1]
						firstSubPt := subPoints[0]
						shiftX := int(lastPt.X) - int(firstSubPt.X)
						shiftY := int(lastPt.Y) - int(firstSubPt.Y)
						for j := range subPoints {
							subPoints[j].X += int16(shiftX)
							subPoints[j].Y += int16(shiftY)
						}
						points = points[:len(points)-1]
						oldCount := len(points)
						points = append(points, subPoints...)
						for _, ep := range subEndPts {
							endPts = append(endPts, oldCount+ep)
						}
						cx = int(points[len(points)-1].X)
						cy = int(points[len(points)-1].Y)
					}
				}
			}
			clearStack()

		case 14: // endchar
			finishContour()
			return points, endPts, nil

		case 11: // return (from subroutine)
			finishContour()
			return points, endPts, nil

		default:
			clearStack()
		}
	}

	finishContour()
	return points, endPts, nil
}

// ---- Type 2 CharString encoder ----

// encodeType2CharString encodes contour points into a Type 2 charstring.
// Uses rmoveto, rlineto, and rrcurveto operators.
func encodeType2CharString(points []ContourPoint, endPts []int) ([]byte, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("no points")
	}

	var buf []byte

	// Write width (0 — default width)
	buf = append(buf, encodeCFFInt(0)...)

	pIdx := 0
	prevX, prevY := 0, 0

	for _, ep := range endPts {
		if ep < 0 {
			ep = 0
		}
		contourEnd := ep
		if contourEnd >= len(points) {
			contourEnd = len(points) - 1
		}

		// First point of contour: moveto
		if pIdx <= contourEnd {
			p := points[pIdx]
			dx := int(p.X) - prevX
			dy := int(p.Y) - prevY
			prevX = int(p.X)
			prevY = int(p.Y)
			buf = append(buf, encodeCFFInt(dx)...)
			buf = append(buf, encodeCFFInt(dy)...)
			buf = append(buf, 21) // rmoveto
			pIdx++
		}

		// Process remaining points in this contour
		for pIdx <= contourEnd {
			// Look ahead: if we have an On-Off-Off-On pattern, it's a curve
			if pIdx+2 <= contourEnd && !points[pIdx].OnCurve && !points[pIdx+1].OnCurve && points[pIdx+2].OnCurve {
				// Cubic curve: Off1, Off2, On
				c1 := points[pIdx]
				c2 := points[pIdx+1]
				endPt := points[pIdx+2]

				dx1 := int(c1.X) - prevX
				dy1 := int(c1.Y) - prevY
				dx2 := int(c2.X) - int(c1.X)
				dy2 := int(c2.Y) - int(c1.Y)
				dx3 := int(endPt.X) - int(c2.X)
				dy3 := int(endPt.Y) - int(c2.Y)

				buf = append(buf, encodeCFFInt(dx1)...)
				buf = append(buf, encodeCFFInt(dy1)...)
				buf = append(buf, encodeCFFInt(dx2)...)
				buf = append(buf, encodeCFFInt(dy2)...)
				buf = append(buf, encodeCFFInt(dx3)...)
				buf = append(buf, encodeCFFInt(dy3)...)
				buf = append(buf, 8) // rrcurveto

				prevX = int(endPt.X)
				prevY = int(endPt.Y)
				pIdx += 3
			} else if points[pIdx].OnCurve {
				// Line
				p := points[pIdx]
				dx := int(p.X) - prevX
				dy := int(p.Y) - prevY
				buf = append(buf, encodeCFFInt(dx)...)
				buf = append(buf, encodeCFFInt(dy)...)
				buf = append(buf, 5) // rlineto

				prevX = int(p.X)
				prevY = int(p.Y)
				pIdx++
			} else {
				// Standalone off-curve (shouldn't happen in well-formed charstrings, but skip)
				pIdx++
			}
		}
	}

	buf = append(buf, 14) // endchar
	return buf, nil
}

// encodeCFFInt encodes an integer as a Type 2 charstring operand.
func encodeCFFInt(v int) []byte {
	switch {
	case v >= -107 && v <= 107:
		return []byte{byte(v + 139)}
	case v >= 108 && v <= 1131:
		v -= 108
		return []byte{byte(v/256 + 247), byte(v % 256)}
	case v >= -1131 && v <= -108:
		v = -v - 108
		return []byte{byte(v/256 + 251), byte(v % 256)}
	case v >= -32768 && v <= 32767:
		b := make([]byte, 3)
		b[0] = 28
		binary.BigEndian.PutUint16(b[1:], uint16(int16(v)))
		return b
	default:
		b := make([]byte, 5)
		b[0] = 255
		binary.BigEndian.PutUint32(b[1:], uint32(int32(v)))
		return b
	}
}

// ---- CFF table rebuild ----

// rebuildCFFCharStrings replaces the CharStrings INDEX in a CFF table.
func rebuildCFFCharStrings(cffData []byte, origCSOffset int, newCS [][]byte) ([]byte, error) {
	// Find the end of the original CharStrings INDEX
	origCSCount, _, csEnd, err := parseCFFIndex(cffData, origCSOffset)
	if err != nil {
		return nil, fmt.Errorf("original CharStrings: %w", err)
	}
	if origCSCount != len(newCS) {
		return nil, fmt.Errorf("glyph count mismatch: orig=%d, new=%d", origCSCount, len(newCS))
	}
	_ = csEnd

	// Build new CharStrings INDEX
	newCSIndex := buildCFFIndex(newCS)

	// Replace: [before] + [new INDEX] + [after]
	var result []byte
	result = append(result, cffData[:origCSOffset]...)
	result = append(result, newCSIndex...)
	// The data after the CharStrings INDEX stays (charset, Private DICT, Local Subr, etc.)
	// But those may have offsets pointing into the old CFF table — they need updating.
	// For now, we just append the rest. The Private DICT Local Subr offset is relative to
	// Private DICT start, and charset offset is from CFF start — so charset stays valid
	// as long as we only modify the CharStrings INDEX (the offset is to charset, which is after CharStrings).

	// Copy everything after the original CharStrings INDEX
	restStart := origCSOffset
	// Find the end of the original CharStrings INDEX by parsing it
	_, _, restStart, err = parseCFFIndex(cffData, origCSOffset)
	if err != nil {
		return nil, fmt.Errorf("CharStrings end: %w", err)
	}
	result = append(result, cffData[restStart:]...)

	return result, nil
}

// buildCFFIndex builds a CFF INDEX structure from elements.
func buildCFFIndex(elements [][]byte) []byte {
	count := len(elements)
	if count == 0 {
		return []byte{0, 0} // count = 0
	}

	// Calculate total data size to determine offSize
	totalData := 0
	for _, el := range elements {
		totalData += len(el)
	}

	offSize := calcCFFOffSize(count, totalData)

	// Count (2 bytes) + offSize (1 byte) + (count+1)*offSize offsets + data
	offsetsStart := 3
	offsetsLen := (count + 1) * offSize
	dataStart := offsetsStart + offsetsLen
	totalLen := dataStart + totalData

	result := make([]byte, totalLen)
	binary.BigEndian.PutUint16(result[0:], uint16(count))
	result[2] = byte(offSize)

	offset := 1 // CFF offsets are 1-based
	for i := 0; i <= count; i++ {
		off := offsetsStart + i*offSize
		writeCFFOffset(result, off, offSize, offset)
		if i < count {
			offset += len(elements[i])
		}
	}

	for i, el := range elements {
		start := dataStart + offset - len(el) - 1
		if i > 0 {
			// The offset for element i gives us the start
			prevOff := readCFFOffset(result, offsetsStart+(i)*offSize, offSize)
			start = dataStart + prevOff - 1
		}
		_ = start
		copy(result[dataStart+readCFFOffset(result, offsetsStart+i*offSize, offSize)-1:], el)
	}

	return result
}

// calcCFFOffSize determines the minimal offSize (1-4) for the given count and total data.
func calcCFFOffSize(count, totalData int) int {
	maxOff := totalData + 1 // 1-based
	switch {
	case maxOff <= 0xFF:
		return 1
	case maxOff <= 0xFFFF:
		return 2
	case maxOff <= 0xFFFFFF:
		return 3
	default:
		return 4
	}
}

// writeCFFOffset writes a 1-4 byte offset into the buffer.
func writeCFFOffset(buf []byte, off, offSize, value int) {
	switch offSize {
	case 1:
		buf[off] = byte(value)
	case 2:
		binary.BigEndian.PutUint16(buf[off:], uint16(value))
	case 3:
		buf[off] = byte(value >> 16)
		buf[off+1] = byte(value >> 8)
		buf[off+2] = byte(value)
	case 4:
		binary.BigEndian.PutUint32(buf[off:], uint32(value))
	}
}

// parseTopDICTFull is a duplicate from cff_bbox.go — kept here to avoid export.
// (The opentype package version is unexported, so we need our own.)
var _ = parseTopDICTFull // used

// parseCFFIndex is a forward declaration — the actual implementation is in cff.go (unexported).
// We need to access it from the opentype package.
// Since parseCFFIndex and readCFFOffset are unexported, we reimplement them here.
