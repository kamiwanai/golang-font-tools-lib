// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
	"log"
)

// ---- Contour data types ----

// ContourPoint is one point on a glyph outline.
type ContourPoint struct {
	X       int16 `json:"x"`
	Y       int16 `json:"y"`
	OnCurve bool  `json:"onCurve"`
}

// GlyphContourData is the decoded outline for one glyph.
// For simple glyphs: EndPts + Points are populated.
// For compound glyphs: Components are populated, EndPts/Points are nil.
// Instructions (TrueType hinting bytecode) are preserved for both types.
type GlyphContourData struct {
	GlyphIndex   int              `json:"glyphIndex"`
	EndPts       []int            `json:"endPts,omitempty"`   // indices into Points marking contour ends (simple)
	Points       []ContourPoint   `json:"points,omitempty"`   // outline points (simple)
	Instructions []byte           `json:"instructions,omitempty"` // TrueType hinting bytecode; preserved roundtrip
	Components   []EditGlyphComponent `json:"components,omitempty"`   // compound glyph components
}

// GlyphComponent is one component in a compound (composite) glyph.
// Describes a reference to another glyph with a 2×3 affine transform:
//   x' = ScaleX * x + Scale01 * y + Arg1
//   y' = Scale10 * x + ScaleY * y + Arg2
type EditGlyphComponent struct {
	GlyphIndex      uint16  `json:"glyphIndex"`
	Arg1            int16   `json:"arg1"`            // X offset (or point number if !ArgsAreXYValues)
	Arg2            int16   `json:"arg2"`            // Y offset (or point number if !ArgsAreXYValues)
	ScaleX          float64 `json:"scaleX,omitempty"`   // 1.0 default
	Scale01         float64 `json:"scale01,omitempty"`  // 0.0 default (shear/cross term)
	Scale10         float64 `json:"scale10,omitempty"`  // 0.0 default (shear/cross term)
	ScaleY          float64 `json:"scaleY,omitempty"`   // 1.0 default
	ArgsAreXYValues bool    `json:"argsAreXYValues"`    // true: args are XY offsets; false: point numbers for matching
	RoundXYToGrid   bool    `json:"roundXYToGrid,omitempty"`
	UseMyMetrics    bool    `json:"useMyMetrics,omitempty"`
	OverlapCompound bool    `json:"overlapCompound,omitempty"`
	ScaledOffset    bool    `json:"scaledOffset,omitempty"`
	UnscaledOffset  bool    `json:"unscaledOffset,omitempty"`
}

// TrueType compound glyph component flags (OpenType spec, glyf table).
const (
	glyfArg1And2AreWords  = 0x0001
	glyfArgsAreXYValues   = 0x0002
	glyfRoundXYToGrid     = 0x0004
	glyfWeHaveAScale      = 0x0008
	glyfMoreComponents    = 0x0020
	glyfWeHaveAnXYScale   = 0x0040
	glyfWeHaveATwoByTwo   = 0x0080
	glyfWeHaveInstructions = 0x0100
	glyfUseMyMetrics      = 0x0200
	glyfOverlapCompound   = 0x0400
	glyfScaledOffset      = 0x0800
	glyfUnscaledOffset    = 0x1000
)

// GlyphContoursJSON is the JSON representation for the parse response.

// ---- hmtx edit types ----

// HmtxEdit is one hmtx change.
type HmtxEdit struct {
	GlyphIndex   int    `json:"glyphIndex"`
	AdvanceWidth uint16 `json:"advanceWidth"`
	LSB          int16  `json:"lsb"`
}

// ---- ApplyEdits request ----


// ---- Glyf decoder: binary → structured ----

const (
	glyfOnCurve     = 0x01
	glyfXShort      = 0x02
	glyfYShort      = 0x04
	glyfRepeat      = 0x08
	glyfXSame       = 0x10
	glyfXPos        = 0x10 // alias for clarity: if set, x is positive/same
	glyfYSame       = 0x20
	glyfYPos        = 0x20 // alias
)

// DecodeGlyfContours decodes all glyphs from the glyf+loca tables.
// Returns contours for glyphs that have simple outlines (no compounds, no CFF).
func DecodeGlyfContours(fontData []byte) ([]GlyphContourData, error) {
	font, err := Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	glyfData, err := font.TableData("glyf")
	if err != nil {
		return nil, fmt.Errorf("no glyf table: %w", err)
	}

	locaData, err := font.TableData("loca")
	if err != nil {
		return nil, fmt.Errorf("no loca table: %w", err)
	}

	head, err := font.Head()
	if err != nil {
		return nil, fmt.Errorf("no head table: %w", err)
	}

	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return nil, fmt.Errorf("numGlyphs: %w", err)
	}

	// Build loca offsets
	entries := numGlyphs + 1
	locaOffsets := make([]int, entries)
	if head.IndexToLocFormat == 0 {
		for i := 0; i < entries; i++ {
			locaOffsets[i] = int(binary.BigEndian.Uint16(locaData[i*2:])) * 2
		}
	} else {
		for i := 0; i < entries; i++ {
			locaOffsets[i] = int(binary.BigEndian.Uint32(locaData[i*4:]))
		}
	}

	var result []GlyphContourData
	for i := 0; i < numGlyphs; i++ {
		off := locaOffsets[i]
		nextOff := locaOffsets[i+1]
		if off >= nextOff || off+10 > len(glyfData) {
			continue
		}

		numContours := int(int16(binary.BigEndian.Uint16(glyfData[off:])))
		if numContours < 0 {
			// Compound glyph: decode components + optional instructions
			compData := glyfData[off:nextOff]
			components, inst, err := decodeCompoundGlyph(compData)
			if err != nil {
				return nil, fmt.Errorf("glyph %d: %w", i, err)
			}
			result = append(result, GlyphContourData{
				GlyphIndex:   i,
				Components:   components,
				Instructions: inst,
			})
			continue
		}
		if numContours == 0 {
			continue // empty glyph
		}

		// endPtsOfContours
		endPts := make([]int, numContours)
		for c := 0; c < numContours; c++ {
			eptOff := off + 10 + c*2
			if eptOff+2 > len(glyfData) {
				return nil, fmt.Errorf("glyph %d: endPts overflow", i)
			}
			endPts[c] = int(binary.BigEndian.Uint16(glyfData[eptOff:]))
		}

		// instruction length + bytes
		instLenOff := off + 10 + numContours*2
		if instLenOff+2 > len(glyfData) {
			return nil, fmt.Errorf("glyph %d: instLen overflow", i)
		}
		instLen := int(binary.BigEndian.Uint16(glyfData[instLenOff:]))

		// Preserve instruction bytes for hinting roundtrip
		var instructions []byte
		if instLen > 0 {
			instBytesOff := instLenOff + 2
			instBytesEnd := instBytesOff + instLen
			if instBytesEnd <= len(glyfData) {
				instructions = make([]byte, instLen)
				copy(instructions, glyfData[instBytesOff:instBytesEnd])
			}
		}

		// flags start
		flagsOff := instLenOff + 2 + instLen
		if flagsOff > len(glyfData) {
			return nil, fmt.Errorf("glyph %d: flags overflow", i)
		}

		// Decode flags + coordinates
		points, err := decodeSimpleGlyphPoints(glyfData[flagsOff:], numContours, endPts)
		if err != nil {
			return nil, fmt.Errorf("glyph %d: %w", i, err)
		}

		result = append(result, GlyphContourData{
			GlyphIndex:   i,
			EndPts:       endPts,
			Points:       points,
			Instructions: instructions,
		})
	}

	return result, nil
}

// decodeCompoundGlyph parses a compound glyph from its raw glyf table data.
// The input must be the full glyph slice (including the 10-byte header).
// Returns components, optional TrueType instructions (after the last component),
// and an error.
func decodeCompoundGlyph(data []byte) ([]EditGlyphComponent, []byte, error) {
	if len(data) < 10 {
		return nil, nil, fmt.Errorf("compound glyph too short (%d bytes)", len(data))
	}

	pos := 10 // skip header (numContours + bbox)
	var components []EditGlyphComponent
	var instructions []byte

	for {
		if pos+4 > len(data) {
			return nil, nil, fmt.Errorf("compound component overflow at pos %d", pos)
		}
		flags := binary.BigEndian.Uint16(data[pos:])
		glyphIndex := binary.BigEndian.Uint16(data[pos+2:])
		pos += 4

		comp := EditGlyphComponent{
			GlyphIndex:      glyphIndex,
			ArgsAreXYValues: flags&glyfArgsAreXYValues != 0,
			RoundXYToGrid:   flags&glyfRoundXYToGrid != 0,
			UseMyMetrics:    flags&glyfUseMyMetrics != 0,
			OverlapCompound: flags&glyfOverlapCompound != 0,
			ScaledOffset:    flags&glyfScaledOffset != 0,
			UnscaledOffset:  flags&glyfUnscaledOffset != 0,
		}

		// Read arguments (offsets or point numbers)
		if flags&glyfArg1And2AreWords != 0 {
			if pos+4 > len(data) {
				return nil, nil, fmt.Errorf("compound args overflow at pos %d", pos)
			}
			comp.Arg1 = int16(binary.BigEndian.Uint16(data[pos:]))
			comp.Arg2 = int16(binary.BigEndian.Uint16(data[pos+2:]))
			pos += 4
		} else {
			if pos+2 > len(data) {
				return nil, nil, fmt.Errorf("compound args overflow at pos %d", pos)
			}
			comp.Arg1 = int16(int8(data[pos]))
			comp.Arg2 = int16(int8(data[pos+1]))
			pos += 2
		}

		// Read transformation
		comp.ScaleX, comp.ScaleY = 1.0, 1.0
		if flags&glyfWeHaveAScale != 0 {
			if pos+2 > len(data) {
				return nil, nil, fmt.Errorf("compound scale overflow at pos %d", pos)
			}
			scale := f2Dot14ToFloat(int16(binary.BigEndian.Uint16(data[pos:])))
			comp.ScaleX = scale
			comp.ScaleY = scale
			pos += 2
		} else if flags&glyfWeHaveAnXYScale != 0 {
			if pos+4 > len(data) {
				return nil, nil, fmt.Errorf("compound xy-scale overflow at pos %d", pos)
			}
			comp.ScaleX = f2Dot14ToFloat(int16(binary.BigEndian.Uint16(data[pos:])))
			comp.ScaleY = f2Dot14ToFloat(int16(binary.BigEndian.Uint16(data[pos+2:])))
			pos += 4
		} else if flags&glyfWeHaveATwoByTwo != 0 {
			if pos+8 > len(data) {
				return nil, nil, fmt.Errorf("compound 2x2 overflow at pos %d", pos)
			}
			comp.ScaleX = f2Dot14ToFloat(int16(binary.BigEndian.Uint16(data[pos:])))
			comp.Scale01 = f2Dot14ToFloat(int16(binary.BigEndian.Uint16(data[pos+2:])))
			comp.Scale10 = f2Dot14ToFloat(int16(binary.BigEndian.Uint16(data[pos+4:])))
			comp.ScaleY = f2Dot14ToFloat(int16(binary.BigEndian.Uint16(data[pos+6:])))
			pos += 8
		}

		components = append(components, comp)

		// Check for instructions after the last component
		if flags&glyfWeHaveInstructions != 0 {
			if pos+2 > len(data) {
				return nil, nil, fmt.Errorf("compound instLen overflow at pos %d", pos)
			}
			instLen := int(binary.BigEndian.Uint16(data[pos:]))
			pos += 2
			if instLen > 0 {
				if pos+instLen > len(data) {
					return nil, nil, fmt.Errorf("compound instructions overflow (%d bytes at pos %d)", instLen, pos)
				}
				instructions = make([]byte, instLen)
				copy(instructions, data[pos:pos+instLen])
				pos += instLen
			}
			break // instructions always come after the last component
		}

		// No more components?
		if flags&glyfMoreComponents == 0 {
			break
		}
	}

	return components, instructions, nil
}

// f2Dot14ToFloat converts a signed 2.14 fixed-point value to float64.
func f2Dot14ToFloat(v int16) float64 {
	return float64(v) / 16384.0
}

// floatToF2Dot14 converts a float64 to signed 2.14 fixed-point.
func floatToF2Dot14(v float64) int16 {
	return int16(v * 16384.0)
}

func decodeSimpleGlyphPoints(data []byte, numContours int, endPts []int) ([]ContourPoint, error) {
	if numContours == 0 || len(endPts) == 0 {
		return nil, nil
	}
	totalPts := endPts[len(endPts)-1] + 1

	// Read flags
	flags := make([]uint8, totalPts)
	pos := 0
	for i := 0; i < totalPts; i++ {
		if pos >= len(data) {
			return nil, fmt.Errorf("flags truncated at point %d/%d", i, totalPts)
		}
		f := data[pos]
		pos++
		flags[i] = f
		if f&glyfRepeat != 0 {
			if pos >= len(data) {
				return nil, fmt.Errorf("repeat count truncated at point %d", i)
			}
			repeat := int(data[pos])
			pos++
			for r := 0; r < repeat && i+1 < totalPts; r++ {
				i++
				flags[i] = f
			}
		}
	}

	// Read x coordinates
	xs := make([]int16, totalPts)
	var prevX int16
	for i := 0; i < totalPts; i++ {
		f := flags[i]
		if f&glyfXShort != 0 {
			// Short (1 byte)
			if pos >= len(data) {
				return nil, fmt.Errorf("x short truncated at point %d", i)
			}
			dx := int16(data[pos])
			pos++
			if f&glyfXSame == 0 {
				dx = -dx
			}
			prevX += dx
		} else if f&glyfXSame == 0 {
			// Long delta
			if pos+2 > len(data) {
				return nil, fmt.Errorf("x long truncated at point %d", i)
			}
			dx := int16(binary.BigEndian.Uint16(data[pos:]))
			pos += 2
			prevX += dx
		}
		// else: XSame set and not short → repeat same value (dx=0)
		xs[i] = prevX
	}

	// Read y coordinates
	ys := make([]int16, totalPts)
	var prevY int16
	for i := 0; i < totalPts; i++ {
		f := flags[i]
		if f&glyfYShort != 0 {
			if pos >= len(data) {
				return nil, fmt.Errorf("y short truncated at point %d", i)
			}
			dy := int16(data[pos])
			pos++
			if f&glyfYSame == 0 {
				dy = -dy
			}
			prevY += dy
		} else if f&glyfYSame == 0 {
			if pos+2 > len(data) {
				return nil, fmt.Errorf("y long truncated at point %d", i)
			}
			dy := int16(binary.BigEndian.Uint16(data[pos:]))
			pos += 2
			prevY += dy
		}
		ys[i] = prevY
	}

	points := make([]ContourPoint, totalPts)
	for i := 0; i < totalPts; i++ {
		points[i] = ContourPoint{
			X:       xs[i],
			Y:       ys[i],
			OnCurve: flags[i]&glyfOnCurve != 0,
		}
	}

	return points, nil
}

// ---- Glyf encoder: structured → binary ----

// EncodeGlyfContours encodes contour data back to binary glyf table data.
// Returns the complete glyf table byte slice, preserving original empty/compound glyphs.
func EncodeGlyfContours(fontData []byte, edits map[int][]GlyphContourData) ([]byte, []uint32, error) {
	font, err := Parse(fontData)
	if err != nil {
		return nil, nil, fmt.Errorf("parse font: %w", err)
	}

	origGlyf, err := font.TableData("glyf")
	if err != nil {
		return nil, nil, fmt.Errorf("glyf: %w", err)
	}
	locaData, err := font.TableData("loca")
	if err != nil {
		return nil, nil, fmt.Errorf("loca: %w", err)
	}
	head, err := font.Head()
	if err != nil {
		return nil, nil, fmt.Errorf("head: %w", err)
	}
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return nil, nil, fmt.Errorf("numGlyphs: %w", err)
	}

	// Build original loca offsets
	entries := numGlyphs + 1
	origOffsets := make([]int, entries)
	if head.IndexToLocFormat == 0 {
		for i := 0; i < entries; i++ {
			origOffsets[i] = int(binary.BigEndian.Uint16(locaData[i*2:])) * 2
		}
	} else {
		for i := 0; i < entries; i++ {
			origOffsets[i] = int(binary.BigEndian.Uint32(locaData[i*4:]))
		}
	}

	// Build new glyf entries
	type glyphBuf struct {
		data   []byte
		bboxXMin, bboxYMin, bboxXMax, bboxYMax int16
		empty   bool
	}
	glyphs := make([]glyphBuf, numGlyphs)

	for i := 0; i < numGlyphs; i++ {
		// Check if this glyph has edits
		if editContours, ok := edits[i]; ok && len(editContours) > 0 {
			gc := editContours[0]

			if len(gc.Components) > 0 {
				// Compound glyph: encode components + instructions
				// Preserve original bbox from the font
				off := origOffsets[i]
				nextOff := origOffsets[i+1]
				var bbox glyphBBox
				if off < nextOff && off+10 <= len(origGlyf) {
					bbox.xMin = int16(binary.BigEndian.Uint16(origGlyf[off+2:]))
					bbox.yMin = int16(binary.BigEndian.Uint16(origGlyf[off+4:]))
					bbox.xMax = int16(binary.BigEndian.Uint16(origGlyf[off+6:]))
					bbox.yMax = int16(binary.BigEndian.Uint16(origGlyf[off+8:]))
				}
				encoded, err := encodeCompoundGlyph(gc.Components, gc.Instructions, bbox)
				if err != nil {
					return nil, nil, fmt.Errorf("glyph %d compound encode: %w", i, err)
				}
				glyphs[i] = glyphBuf{
					data:     encoded,
					bboxXMin: bbox.xMin,
					bboxYMin: bbox.yMin,
					bboxXMax: bbox.xMax,
					bboxYMax: bbox.yMax,
				}
				continue
			}

			// Simple glyph: encode contours + instructions
			encoded, bbox, err := encodeSimpleGlyph(gc.EndPts, gc.Points, gc.Instructions)
			if err != nil {
				return nil, nil, fmt.Errorf("glyph %d encode: %w", i, err)
			}
			glyphs[i] = glyphBuf{
				data:     encoded,
				bboxXMin: bbox.xMin,
				bboxYMin: bbox.yMin,
				bboxXMax: bbox.xMax,
				bboxYMax: bbox.yMax,
			}
			continue
		}

		// Copy original glyph data
		off := origOffsets[i]
		nextOff := origOffsets[i+1]
		if off >= nextOff {
			glyphs[i].empty = true
			continue
		}
		copied := make([]byte, nextOff-off)
		copy(copied, origGlyf[off:nextOff])
		glyphs[i].data = copied
		glyphs[i].empty = false

		// Extract original bbox
		if len(copied) >= 10 {
			glyphs[i].bboxXMin = int16(binary.BigEndian.Uint16(copied[2:]))
			glyphs[i].bboxYMin = int16(binary.BigEndian.Uint16(copied[4:]))
			glyphs[i].bboxXMax = int16(binary.BigEndian.Uint16(copied[6:]))
			glyphs[i].bboxYMax = int16(binary.BigEndian.Uint16(copied[8:]))
		}
	}

	// Recalculate bboxes for edited glyphs from their point data
	for i, editContours := range edits {
		if len(editContours) == 0 {
			continue
		}
		gc := editContours[0]
		var xMin, yMin, xMax, yMax int16
		first := true
		for _, p := range gc.Points {
			if first {
				xMin, xMax = p.X, p.X
				yMin, yMax = p.Y, p.Y
				first = false
				continue
			}
			if p.X < xMin {
				xMin = p.X
			}
			if p.X > xMax {
				xMax = p.X
			}
			if p.Y < yMin {
				yMin = p.Y
			}
			if p.Y > yMax {
				yMax = p.Y
			}
		}
		if !first {
			glyphs[i].bboxXMin = xMin
			glyphs[i].bboxYMin = yMin
			glyphs[i].bboxXMax = xMax
			glyphs[i].bboxYMax = yMax
		}
	}

	// Serialize: write each glyph, build new loca offsets
	var buf []byte
	newOffsets := make([]uint32, numGlyphs+1)
	for i := 0; i < numGlyphs; i++ {
		newOffsets[i] = uint32(len(buf))
		if glyphs[i].empty || len(glyphs[i].data) == 0 {
			continue
		}
		// Glyph header: numContours(2) + bbox(8) = 10 bytes
		header := make([]byte, 10)
		if i < len(glyphs) && len(glyphs[i].data) >= 10 {
			copy(header, glyphs[i].data[:10])
		}
		// Update bbox for edited glyphs
		if _, edited := edits[i]; edited {
			binary.BigEndian.PutUint16(header[2:], uint16(glyphs[i].bboxXMin))
			binary.BigEndian.PutUint16(header[4:], uint16(glyphs[i].bboxYMin))
			binary.BigEndian.PutUint16(header[6:], uint16(glyphs[i].bboxXMax))
			binary.BigEndian.PutUint16(header[8:], uint16(glyphs[i].bboxYMax))
		}
		buf = append(buf, header...)
		buf = append(buf, glyphs[i].data[10:]...)
	}
	newOffsets[numGlyphs] = uint32(len(buf))

	// Pad to 4-byte boundary
	for len(buf)%4 != 0 {
		buf = append(buf, 0)
	}

	return buf, newOffsets, nil
}

type glyphBBox struct {
	xMin, yMin, xMax, yMax int16
}

func encodeSimpleGlyph(endPts []int, points []ContourPoint, instructions []byte) ([]byte, glyphBBox, error) {
	if len(points) == 0 || len(endPts) == 0 {
		return nil, glyphBBox{}, fmt.Errorf("empty glyph")
	}

	numContours := len(endPts)

	// Compute bbox
	var bbox glyphBBox
	bbox.xMin = points[0].X
	bbox.xMax = points[0].X
	bbox.yMin = points[0].Y
	bbox.yMax = points[0].Y
	for _, p := range points {
		if p.X < bbox.xMin {
			bbox.xMin = p.X
		}
		if p.X > bbox.xMax {
			bbox.xMax = p.X
		}
		if p.Y < bbox.yMin {
			bbox.yMin = p.Y
		}
		if p.Y > bbox.yMax {
			bbox.yMax = p.Y
		}
	}

	// Build flags
	flags := make([]uint8, len(points))
	for i, p := range points {
		if p.OnCurve {
			flags[i] |= glyfOnCurve
		}
	}

	// Delta-encode coordinates
	xDeltas := make([]int16, len(points))
	yDeltas := make([]int16, len(points))
	var prevX, prevY int16
	for i := range points {
		xDeltas[i] = points[i].X - prevX
		yDeltas[i] = points[i].Y - prevY
		prevX = points[i].X
		prevY = points[i].Y
	}

	// Determine short/long encoding
	// For short vectors: XSame/YSame flag means the delta is POSITIVE (per TrueType spec),
	// NOT "same as previous". The absence of XSame/YSame means negative.
	for i, dx := range xDeltas {
		if dx == 0 {
			flags[i] |= glyfXSame // same as previous (dx=0)
		} else if dx >= -128 && dx <= 127 {
			flags[i] |= glyfXShort
			if dx > 0 {
				flags[i] |= glyfXSame // for short: XSame=1 means positive
			}
		} else {
			// long (needs 2 bytes)
		}
	}

	for i, dy := range yDeltas {
		if dy == 0 {
			flags[i] |= glyfYSame
		} else if dy >= -128 && dy <= 127 {
			flags[i] |= glyfYShort
			if dy > 0 {
				flags[i] |= glyfYSame // for short: YSame=1 means positive
			}
		} else {
			// long
		}
	}

	// Build binary
	var buf []byte

	// Header: numContours + bbox
	header := make([]byte, 10)
	binary.BigEndian.PutUint16(header[0:], uint16(numContours))
	binary.BigEndian.PutUint16(header[2:], uint16(bbox.xMin))
	binary.BigEndian.PutUint16(header[4:], uint16(bbox.yMin))
	binary.BigEndian.PutUint16(header[6:], uint16(bbox.xMax))
	binary.BigEndian.PutUint16(header[8:], uint16(bbox.yMax))
	buf = append(buf, header...)

	// endPtsOfContours
	for _, ep := range endPts {
		ept := make([]byte, 2)
		binary.BigEndian.PutUint16(ept, uint16(ep))
		buf = append(buf, ept...)
	}

	// instructionLength + TrueType instructions (hinting bytecode)
	instLen := make([]byte, 2)
	binary.BigEndian.PutUint16(instLen, uint16(len(instructions)))
	buf = append(buf, instLen...)
	if len(instructions) > 0 {
		buf = append(buf, instructions...)
	}

	// Flags (with repeat)
	for i := 0; i < len(flags); {
		f := flags[i]
		i++
		// Count repeats (how many more identical flags follow)
		repeat := 0
		for i < len(flags) && flags[i] == f && repeat < 255 {
			repeat++
			i++
		}
		if repeat > 0 {
			buf = append(buf, f|glyfRepeat)
			buf = append(buf, uint8(repeat))
		} else {
			buf = append(buf, f)
		}
	}

	// X coordinates
	for i, dx := range xDeltas {
		if flags[i]&glyfXSame != 0 && flags[i]&glyfXShort == 0 {
			continue // dx == 0, no data (XSame only matters for zero when not short)
		}
		if flags[i]&glyfXShort != 0 {
			// Short: write absolute value (sign is in XSame flag)
			absDx := dx
			if absDx < 0 {
				absDx = -absDx
			}
			buf = append(buf, uint8(absDx))
		} else {
			// Long
			coord := make([]byte, 2)
			binary.BigEndian.PutUint16(coord, uint16(dx))
			buf = append(buf, coord...)
		}
	}

	// Y coordinates
	for i, dy := range yDeltas {
		if flags[i]&glyfYSame != 0 && flags[i]&glyfYShort == 0 {
			continue // dy == 0, no data
		}
		if flags[i]&glyfYShort != 0 {
			// Short: write absolute value
			absDy := dy
			if absDy < 0 {
				absDy = -absDy
			}
			buf = append(buf, uint8(absDy))
		} else {
			// Long
			coord := make([]byte, 2)
			binary.BigEndian.PutUint16(coord, uint16(dy))
			buf = append(buf, coord...)
		}
	}

	return buf, bbox, nil
}

// encodeCompoundGlyph encodes a compound (composite) glyph into binary glyf format.
// Includes the 10-byte glyph header. Returns the complete glyph byte slice.
func encodeCompoundGlyph(components []EditGlyphComponent, instructions []byte, bbox glyphBBox) ([]byte, error) {
	if len(components) == 0 {
		return nil, fmt.Errorf("compound glyph with no components")
	}

	var buf []byte

	// Header: numContours (0xFFFF = compound) + bbox
	header := make([]byte, 10)
	binary.BigEndian.PutUint16(header[0:], 0xFFFF) // -1 as uint16
	binary.BigEndian.PutUint16(header[2:], uint16(bbox.xMin))
	binary.BigEndian.PutUint16(header[4:], uint16(bbox.yMin))
	binary.BigEndian.PutUint16(header[6:], uint16(bbox.xMax))
	binary.BigEndian.PutUint16(header[8:], uint16(bbox.yMax))
	buf = append(buf, header...)

	for i, comp := range components {
		flags := uint16(0)

		// MORE_COMPONENTS for all but the last
		if i < len(components)-1 {
			flags |= glyfMoreComponents
		}

		if comp.ArgsAreXYValues {
			flags |= glyfArgsAreXYValues
		}
		if comp.RoundXYToGrid {
			flags |= glyfRoundXYToGrid
		}
		if comp.UseMyMetrics {
			flags |= glyfUseMyMetrics
		}
		if comp.OverlapCompound {
			flags |= glyfOverlapCompound
		}
		if comp.ScaledOffset {
			flags |= glyfScaledOffset
		}
		if comp.UnscaledOffset {
			flags |= glyfUnscaledOffset
		}

		// Determine argument encoding
		if comp.Arg1 < -128 || comp.Arg1 > 127 || comp.Arg2 < -128 || comp.Arg2 > 127 {
			flags |= glyfArg1And2AreWords
		}

		// Determine scale transform encoding
		hasScale := comp.ScaleX != 1.0 || comp.ScaleY != 1.0 || comp.Scale01 != 0.0 || comp.Scale10 != 0.0
		if hasScale {
			if comp.Scale01 == 0.0 && comp.Scale10 == 0.0 {
				if comp.ScaleX == comp.ScaleY {
					flags |= glyfWeHaveAScale
				} else {
					flags |= glyfWeHaveAnXYScale
				}
			} else {
				flags |= glyfWeHaveATwoByTwo
			}
		}

		// WE_HAVE_INSTRUCTIONS on the last component if instructions are present
		if i == len(components)-1 && len(instructions) > 0 {
			flags |= glyfWeHaveInstructions
		}

		// Write flags (2 bytes) + glyphIndex (2 bytes)
		fl := make([]byte, 2)
		binary.BigEndian.PutUint16(fl, flags)
		buf = append(buf, fl...)

		gi := make([]byte, 2)
		binary.BigEndian.PutUint16(gi, comp.GlyphIndex)
		buf = append(buf, gi...)

		// Arguments
		if flags&glyfArg1And2AreWords != 0 {
			ab := make([]byte, 4)
			binary.BigEndian.PutUint16(ab[0:], uint16(comp.Arg1))
			binary.BigEndian.PutUint16(ab[2:], uint16(comp.Arg2))
			buf = append(buf, ab...)
		} else {
			buf = append(buf, uint8(comp.Arg1), uint8(comp.Arg2))
		}

		// Transformation
		if flags&glyfWeHaveAScale != 0 {
			sb := make([]byte, 2)
			binary.BigEndian.PutUint16(sb, uint16(floatToF2Dot14(comp.ScaleX)))
			buf = append(buf, sb...)
		} else if flags&glyfWeHaveAnXYScale != 0 {
			sb := make([]byte, 4)
			binary.BigEndian.PutUint16(sb[0:], uint16(floatToF2Dot14(comp.ScaleX)))
			binary.BigEndian.PutUint16(sb[2:], uint16(floatToF2Dot14(comp.ScaleY)))
			buf = append(buf, sb...)
		} else if flags&glyfWeHaveATwoByTwo != 0 {
			sb := make([]byte, 8)
			binary.BigEndian.PutUint16(sb[0:], uint16(floatToF2Dot14(comp.ScaleX)))
			binary.BigEndian.PutUint16(sb[2:], uint16(floatToF2Dot14(comp.Scale01)))
			binary.BigEndian.PutUint16(sb[4:], uint16(floatToF2Dot14(comp.Scale10)))
			binary.BigEndian.PutUint16(sb[6:], uint16(floatToF2Dot14(comp.ScaleY)))
			buf = append(buf, sb...)
		}
	}

	// Instructions after the last component
	if len(instructions) > 0 {
		il := make([]byte, 2)
		binary.BigEndian.PutUint16(il, uint16(len(instructions)))
		buf = append(buf, il...)
		buf = append(buf, instructions...)
	}

	return buf, nil
}

// ---- hmtx patcher ----

// PatchHmtx applies hmtx edits to the font binary.
func PatchHmtx(fontData []byte, edits []HmtxEdit) ([]byte, error) {
	font, err := Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}

	hheaData, err := font.TableData("hhea")
	if err != nil {
		return nil, fmt.Errorf("hhea: %w", err)
	}

	if len(hheaData) < 36 {
		return nil, fmt.Errorf("hhea too short")
	}
	numHMetrics := int(binary.BigEndian.Uint16(hheaData[34:]))
	numGlyphs, _ := font.NumGlyphs()

	result := make([]byte, len(fontData))
	copy(result, fontData)

	// Locate hmtx in result
	table, ok := font.Tables["hmtx"]
	if !ok {
		return nil, fmt.Errorf("hmtx table not found")
	}
	hmtxStart := int(table.Offset)

	for _, e := range edits {
		if e.GlyphIndex < 0 {
			continue
		}
		if e.GlyphIndex < numHMetrics {
			// Full entry: 4 bytes (advance + lsb)
			off := hmtxStart + e.GlyphIndex*4
			if off+4 <= len(result) {
				binary.BigEndian.PutUint16(result[off:], e.AdvanceWidth)
				binary.BigEndian.PutUint16(result[off+2:], uint16(e.LSB))
			}
		} else if e.GlyphIndex < numGlyphs {
			// LSB-only entry: 2 bytes
			off := hmtxStart + numHMetrics*4 + (e.GlyphIndex-numHMetrics)*2
			if off+2 <= len(result) {
				binary.BigEndian.PutUint16(result[off:], uint16(e.LSB))
			}
		}
	}

	// Recalc checksums
	RecalcTableChecksum(result, "hmtx")
	RecalcTableChecksum(result, "hhea")
	RecalcHeadChecksum(result)

	return result, nil
}

// ---- apply-edits handler ----

// replaceGlyfAndLoca rebuilds both glyf and loca tables with edited contours.
func replaceGlyfAndLoca(fontData []byte, edits map[int][]GlyphContourData) ([]byte, error) {
	font, err := Parse(fontData)
	if err != nil {
		return nil, err
	}

	// Get original loca for boundary info
	origLoca, err := font.TableData("loca")
	if err != nil {
		return nil, fmt.Errorf("loca: %w", err)
	}
	head, err := font.Head()
	if err != nil {
		return nil, fmt.Errorf("head: %w", err)
	}
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return nil, fmt.Errorf("numGlyphs: %w", err)
	}

	// Build original offsets
	entries := numGlyphs + 1
	origOffsets := make([]int, entries)
	if head.IndexToLocFormat == 0 {
		for i := 0; i < entries; i++ {
			origOffsets[i] = int(binary.BigEndian.Uint16(origLoca[i*2:])) * 2
		}
	} else {
		for i := 0; i < entries; i++ {
			origOffsets[i] = int(binary.BigEndian.Uint32(origLoca[i*4:]))
		}
	}

	// Encode new glyf with edits — returns offsets computed during serialization
	newGlyf, newOffsets, err := EncodeGlyfContours(fontData, edits)
	if err != nil {
		return nil, fmt.Errorf("encode glyf: %w", err)
	}

	// Build loca directly from the offsets returned by EncodeGlyfContours
	locaEntries := entries
	actualFormat := head.IndexToLocFormat

	// Check if short format is possible
	if actualFormat == 0 {
		for _, off := range newOffsets {
			if off%2 != 0 {
				log.Printf("[glyf-edit] short loca not possible (offset %d unaligned), switching to long format", off)
				actualFormat = 1
				break
			}
		}
	}

	var newLoca []byte
	if actualFormat == 0 {
		newLoca = make([]byte, locaEntries*2)
		for i, off := range newOffsets {
			if i >= locaEntries {
				break
			}
			binary.BigEndian.PutUint16(newLoca[i*2:], uint16(off/2))
		}
	} else {
		newLoca = make([]byte, locaEntries*4)
		for i, off := range newOffsets {
			if i >= locaEntries {
				break
			}
			binary.BigEndian.PutUint32(newLoca[i*4:], off)
		}
	}

	// Serialize font with replaced glyf + loca
	tables := make(map[string][]byte)

	// If format changed (short→long), update head table before serialization
	if actualFormat != head.IndexToLocFormat {
		headData, err := font.TableData("head")
		if err != nil {
			return nil, fmt.Errorf("read head: %w", err)
		}
		// IndexToLocFormat is at byte offset 50 in the head table
		if len(headData) > 52 {
			headData[50] = 0
			headData[51] = 1 // indexToLocFormat = 1 (long)
			log.Printf("[glyf-edit] loca format switched short→long (offset misalignment)")
		}
		tables["head"] = headData
	}

	order := SortedTableTags(font)
	for _, t := range order {
		// If head was already set (format switch), skip it here
		if t == "head" && tables["head"] != nil {
			continue
		}
		switch t {
		case "glyf":
			tables[t] = newGlyf
		case "loca":
			tables[t] = newLoca
		default:
			td, err := font.TableData(t)
			if err != nil {
				return nil, fmt.Errorf("read table %s: %w", t, err)
			}
			tables[t] = td
		}
	}
	return SerializeFont(order, tables)
}

// buildLocaFromGlyfWithContext builds loca using original offsets for unedited
// glyphs and scanning for edited ones. Returns loca data and the actual format used
// (may differ from indexToLocFormat when short format can't align offsets to 2 bytes).
func buildLocaFromGlyfWithContext(glyfData []byte, origOffsets []int, numGlyphs int, indexToLocFormat int16) ([]byte, int16, error) {
	entries := numGlyphs + 1
	offsets := make([]uint32, entries)
	pos := uint32(0)

	// Track which glyphs have been re-encoded as simple glyphs
	// (their length is computed by scanSimpleGlyphLen)
	for i := 0; i < numGlyphs; i++ {
		offsets[i] = pos

		if int(pos) >= len(glyfData) {
			// Remainder are empty
			for j := i; j <= numGlyphs; j++ {
				offsets[j] = uint32(len(glyfData))
			}
			break
		}

		// Check if glyph header indicates simple glyph
		numContours := int(int16(binary.BigEndian.Uint16(glyfData[pos:])))
		if numContours > 0 {
			glyphLen, err := scanSimpleGlyphLen(glyfData[pos:], numContours)
			if err != nil {
				return nil, 0, fmt.Errorf("glyph %d: %w", i, err)
			}
			pos += glyphLen
		} else {
			// Compound or empty: use original length
			origLen := origOffsets[i+1] - origOffsets[i]
			if origLen < 0 {
				origLen = 0
			}
			pos += uint32(origLen)
		}
	}
	offsets[numGlyphs] = pos

	// Build loca bytes
	// If short format requested, check alignment; fall back to long if any offset isn't 2-byte aligned
	actualFormat := indexToLocFormat
	if indexToLocFormat == 0 {
		actualFormat = 0
		for _, off := range offsets {
			if off%2 != 0 {
				log.Printf("[glyf-edit] short loca not possible (offset %d unaligned), switching to long format", off)
				actualFormat = 1
				break
			}
		}
	}
	if actualFormat == 0 {
		loca := make([]byte, entries*2)
		for i, off := range offsets {
			binary.BigEndian.PutUint16(loca[i*2:], uint16(off/2))
		}
		return loca, 0, nil
	}
	loca := make([]byte, entries*4)
	for i, off := range offsets {
		binary.BigEndian.PutUint32(loca[i*4:], off)
	}
	return loca, 1, nil
}

// scanSimpleGlyphLen returns the byte length of a simple glyph in glyf data.
func scanSimpleGlyphLen(data []byte, numContours int) (uint32, error) {
	if len(data) < 10+numContours*2+2 {
		return 0, fmt.Errorf("too short for header+endPts+instLen")
	}
	instLenOff := 10 + numContours*2
	instLen := int(binary.BigEndian.Uint16(data[instLenOff:]))
	flagsOff := instLenOff + 2 + instLen
	if flagsOff > len(data) {
		return 0, fmt.Errorf("flags overflow")
	}

	// Count total points from endPts
	lastEndPt := 0
	if numContours > 0 {
		lastEndPt = int(binary.BigEndian.Uint16(data[10+(numContours-1)*2:]))
	}
	totalPts := lastEndPt + 1

	// Scan flags (variable length with repeats) — store for later
	flags := make([]uint8, totalPts)
	pos := flagsOff
	ptsRead := 0
	for ptsRead < totalPts {
		if pos >= len(data) {
			return 0, fmt.Errorf("truncated flags")
		}
		f := data[pos]
		pos++
		flags[ptsRead] = f
		ptsRead++
		if f&glyfRepeat != 0 {
			if pos >= len(data) {
				return 0, fmt.Errorf("truncated repeat count")
			}
			repeat := int(data[pos])
			pos++
			for r := 0; r < repeat && ptsRead < totalPts; r++ {
				flags[ptsRead] = f
				ptsRead++
			}
		}
	}

	// Skip x coordinates
	for i := 0; i < totalPts; i++ {
		if flags[i]&glyfXShort != 0 {
			pos++
		} else if flags[i]&glyfXSame == 0 {
			pos += 2
		}
	}

	// Skip y coordinates
	for i := 0; i < totalPts; i++ {
		if flags[i]&glyfYShort != 0 {
			pos++
		} else if flags[i]&glyfYSame == 0 {
			pos += 2
		}
	}

	if pos > len(data) {
		return 0, fmt.Errorf("glyph data truncated")
	}
	return uint32(pos), nil
}

// validateFont checks that the font binary is parseable and has
// the critical outline tables (glyf+loca or CFF).
func validateFont(fontData []byte) error {
	font, err := Parse(fontData)
	if err != nil {
		return fmt.Errorf("re-parse failed: %w", err)
	}

	// Verify critical tables are present
	_, glyfErr := font.TableData("glyf")
	_, locaErr := font.TableData("loca")
	_, cffErr := font.TableData("CFF ")
	_, cff2Err := font.TableData("CFF2")

	hasTrueType := glyfErr == nil && locaErr == nil
	hasCFF := cffErr == nil || cff2Err == nil

	if !hasTrueType && !hasCFF {
		return fmt.Errorf("no outline tables found (glyf+loca or CFF/CFF2)")
	}

	// Verify glyph count makes sense
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return fmt.Errorf("numGlyphs: %w", err)
	}
	if numGlyphs == 0 {
		return fmt.Errorf("font has 0 glyphs")
	}
	if numGlyphs > 65535 {
		return fmt.Errorf("glyph count %d exceeds maximum", numGlyphs)
	}

	// For TrueType: verify loca bounds
	if hasTrueType {
		loca, _ := font.TableData("loca")
		glyf, _ := font.TableData("glyf")
		headTable, err := font.Head()
		if err != nil {
			return fmt.Errorf("head: %w", err)
		}
		entries := numGlyphs + 1
		var lastOff uint32
		if headTable.IndexToLocFormat == 0 {
			if len(loca) < entries*2 {
				return fmt.Errorf("loca too short (short format): %d < %d", len(loca), entries*2)
			}
			lastOff = uint32(binary.BigEndian.Uint16(loca[(entries-1)*2:])) * 2
		} else {
			if len(loca) < entries*4 {
				return fmt.Errorf("loca too short (long format): %d < %d", len(loca), entries*4)
			}
			lastOff = binary.BigEndian.Uint32(loca[(entries-1)*4:])
		}
		if lastOff > uint32(len(glyf)) {
			return fmt.Errorf("loca exceeds glyf: last offset %d > glyf length %d", lastOff, len(glyf))
		}
	}

	// For CFF: verify CharStrings INDEX is accessible
	if hasCFF {
		cffData, cffErr := font.TableData("CFF ")
		if cffErr == nil && len(cffData) > 4 {
			// Basic sanity: check header major version
			major := cffData[0]
			if major != 1 && major != 2 && major != 3 {
				return fmt.Errorf("CFF table has invalid major version %d", major)
			}
		}
	}

	return nil
}


// replaceTable replaces one table in the font binary using full font serialization.
// This rebuilds the entire font to guarantee correct offsets and checksums.
func replaceTable(fontData []byte, tag string, newData []byte) ([]byte, error) {
	font, err := Parse(fontData)
	if err != nil {
		return nil, err
	}

	// Collect all existing tables
	tables := make(map[string][]byte)
	order := SortedTableTags(font)

	for _, t := range order {
		if t == tag {
			tables[t] = newData
		} else {
			td, err := font.TableData(t)
			if err != nil {
				return nil, fmt.Errorf("read table %s: %w", t, err)
			}
			tables[t] = td
		}
	}
	if _, exists := tables[tag]; !exists {
		tables[tag] = newData
		order = append(order, tag)
	}

	return SerializeFont(order, tables)
}

func recalcAllChecksums(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	numTables := int(binary.BigEndian.Uint16(result[4:6]))
	var sum uint32
	for i := 0; i+3 < len(result); i += 4 {
		sum += binary.BigEndian.Uint32(result[i:])
	}

	// Update each table's checksum
	for i := 0; i < numTables; i++ {
		dirOff := 12 + i*16
		off := binary.BigEndian.Uint32(result[dirOff+8 : dirOff+12])
		length := binary.BigEndian.Uint32(result[dirOff+12 : dirOff+16])
		if int(off)+int(length) <= len(result) {
			chk := calcChecksum(result[off : off+length])
			binary.BigEndian.PutUint32(result[dirOff+4:dirOff+8], chk)
		}
	}

	// head checksum adjustment
	for i := 0; i < numTables; i++ {
		dirOff := 12 + i*16
		if string(result[dirOff:dirOff+4]) == "head" {
			headOff := binary.BigEndian.Uint32(result[dirOff+8 : dirOff+12])
			if int(headOff)+12 <= len(result) {
				binary.BigEndian.PutUint32(result[headOff+8:headOff+12], 0xB1B0AFBA-sum)
			}
			break
		}
	}

	return result
}

// ---- Contour extraction for parse response ----

// ExtractGlyphContours extracts glyph contours from the font, capped at limit.

// Helper for GPOS rebuild (used by fontwriter.go)
