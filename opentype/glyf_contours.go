// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

// GlyphPoint is a single point in a glyph outline.
type GlyphPoint struct {
	X         int16
	Y         int16
	OnCurve   bool
}

// GlyphContour is a single closed contour (list of points) in a glyph.
type GlyphContour struct {
	Points []GlyphPoint
}

// GlyphOutline holds the full outline data for a glyph.
type GlyphOutline struct {
	GlyphIndex int
	GlyphName  string
	XMin, YMin, XMax, YMax int16
	Contours   []GlyphContour
	// Compound indicates this glyph is compound (composed of other glyphs).
	Compound     bool
	Components   []GlyphComponent
}

// GlyphComponent is one component reference in a compound glyph.
type GlyphComponent struct {
	GlyphIndex   uint16
	WeHaveScale  bool // bit 3: WE_HAVE_A_SCALE
	MoreComponents bool // bit 5: MORE_COMPONENTS
	WeHaveXY     bool // bit 6: WE_HAVE_AN_X_AND_Y_SCALE
	WeHaveTwoByTwo bool // bit 7: WE_HAVE_A_TWO_BY_TWO
	MatchPoints  uint16
	Scale        float64
	ScaleX       float64
	ScaleY       float64
	Scale01      float64
	Scale10      float64
	XOffset      int16
	YOffset      int16
}

// GlyphOutlines parses glyph outline data from the glyf table for all glyphs.
func (font *Font) GlyphOutlines() ([]GlyphOutline, error) {
	numGlyphs, err := font.NumGlyphs()
	if err != nil { return nil, fmt.Errorf("glyf outlines: %w", err) }
	if numGlyphs == 0 { return nil, nil }

	locaData, err := font.TableData("loca")
	if err != nil { return nil, fmt.Errorf("glyf outlines: loca: %w", err) }
	glyfData, err := font.TableData("glyf")
	if err != nil { return nil, fmt.Errorf("glyf outlines: glyf: %w", err) }
	head, err := font.Head()
	if err != nil { return nil, fmt.Errorf("glyf outlines: head: %w", err) }

	entries := numGlyphs + 1
	locaOffsets := make([]int, entries)
	if head.IndexToLocFormat == 0 {
		if len(locaData) < entries*2 { return nil, fmt.Errorf("loca too short") }
		for i := 0; i < entries; i++ { locaOffsets[i] = int(binary.BigEndian.Uint16(locaData[i*2:])) * 2 }
	} else {
		if len(locaData) < entries*4 { return nil, fmt.Errorf("loca too short") }
		for i := 0; i < entries; i++ { locaOffsets[i] = int(binary.BigEndian.Uint32(locaData[i*4:])) }
	}

	glyphOrder, err := font.GlyphOrder()
	if err != nil { glyphOrder = nil }

	var outlines []GlyphOutline
	for i := 0; i < numGlyphs; i++ {
		offset := locaOffsets[i]
		nextOffset := locaOffsets[i+1]
		outline := GlyphOutline{GlyphIndex: i}
		if glyphOrder != nil && i < len(glyphOrder) { outline.GlyphName = glyphOrder[i] }
		if offset >= nextOffset || offset+10 > len(glyfData) {
			outlines = append(outlines, outline)
			continue
		}
		numContours := int(int16(binary.BigEndian.Uint16(glyfData[offset:])))
		outline.XMin = int16(binary.BigEndian.Uint16(glyfData[offset+2:]))
		outline.YMin = int16(binary.BigEndian.Uint16(glyfData[offset+4:]))
		outline.XMax = int16(binary.BigEndian.Uint16(glyfData[offset+6:]))
		outline.YMax = int16(binary.BigEndian.Uint16(glyfData[offset+8:]))

		if numContours >= 0 {
			contours, err := parseSimpleGlyph(glyfData, offset, numContours, nextOffset)
			if err != nil { return nil, fmt.Errorf("glyph %d: %w", i, err) }
			outline.Contours = contours
		} else {
			components, err := parseCompoundGlyph(glyfData, offset)
			if err != nil { return nil, fmt.Errorf("glyph %d (compound): %w", i, err) }
			outline.Compound = true
			outline.Components = components
		}
		outlines = append(outlines, outline)
	}
	return outlines, nil
}
func parseSimpleGlyph(data []byte, offset int, numContours int, nextOffset int) ([]GlyphContour, error) {
	// Header: 10 bytes (numContours at offset+0 already read, bbox at +2..+9)
	pos := offset + 10
	if pos+numContours*2 > len(data) {
		return nil, fmt.Errorf("endpoints exceed data")
	}

	// Read endpoints
	endPts := make([]int, numContours)
	for i := 0; i < numContours; i++ {
		endPts[i] = int(binary.BigEndian.Uint16(data[pos:]))
		pos += 2
	}

	// Instruction length + skip instructions
	if pos+2 > len(data) { return nil, fmt.Errorf("instruction length exceeds data") }
	insLen := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2 + insLen
	if pos > len(data) { return nil, fmt.Errorf("instructions exceed data") }

	// Determine total number of points from the last endpoint
	var numPoints int
	if numContours > 0 { numPoints = endPts[numContours-1] + 1 }
	if numPoints == 0 { return nil, nil }

	// Read flags
	flags := make([]uint8, numPoints)
	for i := 0; i < numPoints; i++ {
		if pos >= len(data) { return nil, fmt.Errorf("flags exceed data") }
		f := data[pos]
		pos++
		flags[i] = f
		if f&0x08 != 0 { // REPEAT flag
			if pos >= len(data) { return nil, fmt.Errorf("repeat count exceeds data") }
			repeat := int(data[pos])
			pos++
			for r := 1; r <= repeat && i+r < numPoints; r++ {
				flags[i+r] = f
			}
			i += repeat
		}
	}

	// Read X coordinates
	xCoords := make([]int16, numPoints)
	var x int16
	for i := 0; i < numPoints; i++ {
		f := flags[i]
		if f&0x02 != 0 { // X_SHORT_VECTOR
			if pos >= len(data) { return nil, fmt.Errorf("x coords exceed data") }
			dx := int16(data[pos])
			pos++
			if f&0x10 != 0 { // X_IS_SAME_OR_POSITIVE_X_SHORT_VECTOR
				x += dx
			} else {
				x -= dx
			}
		} else if f&0x10 == 0 { // Delta X is int16
			if pos+2 > len(data) { return nil, fmt.Errorf("x coords exceed data") }
			dx := int16(binary.BigEndian.Uint16(data[pos:]))
			pos += 2
			x += dx
		}
		// else: same X as previous
		xCoords[i] = x
	}

	// Read Y coordinates
	yCoords := make([]int16, numPoints)
	var y int16
	for i := 0; i < numPoints; i++ {
		f := flags[i]
		if f&0x04 != 0 { // Y_SHORT_VECTOR
			if pos >= len(data) { return nil, fmt.Errorf("y coords exceed data") }
			dy := int16(data[pos])
			pos++
			if f&0x20 != 0 { // Y_IS_SAME_OR_POSITIVE_Y_SHORT_VECTOR
				y += dy
			} else {
				y -= dy
			}
		} else if f&0x20 == 0 { // Delta Y is int16
			if pos+2 > len(data) { return nil, fmt.Errorf("y coords exceed data") }
			dy := int16(binary.BigEndian.Uint16(data[pos:]))
			pos += 2
			y += dy
		}
		// else: same Y as previous
		yCoords[i] = y
	}

	// Build contours from endpoints
	contours := make([]GlyphContour, numContours)
	start := 0
	for c := 0; c < numContours; c++ {
		end := endPts[c] + 1
		pts := make([]GlyphPoint, end-start)
		for p := start; p < end; p++ {
			pts[p-start] = GlyphPoint{
				X:       xCoords[p],
				Y:       yCoords[p],
				OnCurve: flags[p]&0x01 != 0,
			}
		}
		contours[c] = GlyphContour{Points: pts}
		start = end
	}
	return contours, nil
}
func parseCompoundGlyph(data []byte, offset int) ([]GlyphComponent, error) {
	pos := offset + 10 // skip header
	var components []GlyphComponent
	for {
		if pos+4 > len(data) { return nil, fmt.Errorf("compound glyph data exceeds") }
		flags := binary.BigEndian.Uint16(data[pos:])
		pos += 2
		glyphIndex := binary.BigEndian.Uint16(data[pos:])
		pos += 2

		comp := GlyphComponent{
			GlyphIndex:    glyphIndex,
			WeHaveScale:   flags&0x0008 != 0,
			MoreComponents: flags&0x0020 != 0,
			WeHaveXY:      flags&0x0040 != 0,
			WeHaveTwoByTwo: flags&0x0080 != 0,
		}

		// Read arguments
		if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
			if pos+4 > len(data) { return nil, fmt.Errorf("compound args exceed data") }
			if flags&0x0002 != 0 { // ARGS_ARE_XY_VALUES
				comp.XOffset = int16(binary.BigEndian.Uint16(data[pos:]))
				comp.YOffset = int16(binary.BigEndian.Uint16(data[pos+2:]))
			} else {
				comp.MatchPoints = binary.BigEndian.Uint16(data[pos:])
				// second word is also match point
			}
			pos += 4
		} else {
			if pos+2 > len(data) { return nil, fmt.Errorf("compound args exceed data") }
			if flags&0x0002 != 0 {
				comp.XOffset = int16(int8(data[pos]))
				comp.YOffset = int16(int8(data[pos+1]))
			} else {
				comp.MatchPoints = uint16(data[pos])
			}
			pos += 2
		}

		// Read scale/transformation
		if comp.WeHaveScale {
			if pos+2 > len(data) { return nil, fmt.Errorf("compound scale exceeds data") }
			comp.Scale = float64(int16(binary.BigEndian.Uint16(data[pos:]))) / 16384.0
			pos += 2
		} else if comp.WeHaveXY {
			if pos+4 > len(data) { return nil, fmt.Errorf("compound scale xy exceeds data") }
			comp.ScaleX = float64(int16(binary.BigEndian.Uint16(data[pos:]))) / 16384.0
			comp.ScaleY = float64(int16(binary.BigEndian.Uint16(data[pos+2:]))) / 16384.0
			pos += 4
		} else if comp.WeHaveTwoByTwo {
			if pos+8 > len(data) { return nil, fmt.Errorf("compound 2x2 exceeds data") }
			comp.ScaleX = float64(int16(binary.BigEndian.Uint16(data[pos:]))) / 16384.0
			comp.Scale01 = float64(int16(binary.BigEndian.Uint16(data[pos+2:]))) / 16384.0
			comp.Scale10 = float64(int16(binary.BigEndian.Uint16(data[pos+4:]))) / 16384.0
			comp.ScaleY = float64(int16(binary.BigEndian.Uint16(data[pos+6:]))) / 16384.0
			pos += 8
		}

		components = append(components, comp)
		if !comp.MoreComponents { break }
	}
	return components, nil
}