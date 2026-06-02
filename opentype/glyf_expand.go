// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ExpandCompoundGlyph recursively resolves a compound glyph into flat contours.
//
// It loads each component glyph's contours (recursively expanding nested
// compounds), applies the component's transformation (scale and offset),
// and returns the concatenated flat contour list.
func (font *Font) ExpandCompoundGlyph(glyphIndex int) (*GlyphOutline, error) {
	numGlyphs, err := font.NumGlyphs()
	if err != nil {
		return nil, fmt.Errorf("expand compound: %w", err)
	}
	if glyphIndex < 0 || glyphIndex >= numGlyphs {
		return nil, fmt.Errorf("expand compound: glyph %d out of range", glyphIndex)
	}

	glyfData, err := font.TableData("glyf")
	if err != nil {
		return nil, fmt.Errorf("expand compound: glyf: %w", err)
	}
	locaData, err := font.TableData("loca")
	if err != nil {
		return nil, fmt.Errorf("expand compound: loca: %w", err)
	}
	head, err := font.Head()
	if err != nil {
		return nil, fmt.Errorf("expand compound: head: %w", err)
	}

	entries := numGlyphs + 1
	locaOffsets := make([]int, entries)
	if head.IndexToLocFormat == 0 {
		if len(locaData) < entries*2 {
			return nil, fmt.Errorf("loca too short")
		}
		for i := 0; i < entries; i++ {
			locaOffsets[i] = int(binary.BigEndian.Uint16(locaData[i*2:])) * 2
		}
	} else {
		if len(locaData) < entries*4 {
			return nil, fmt.Errorf("loca too short")
		}
		for i := 0; i < entries; i++ {
			locaOffsets[i] = int(binary.BigEndian.Uint32(locaData[i*4:]))
		}
	}

	glyphOrder, _ := font.GlyphOrder()
	return expandCompound(glyfData, locaOffsets, glyphIndex, glyphOrder)
}

// expandCompound recursively expands a single glyph (simple or compound) into contours.
func expandCompound(glyfData []byte, locaOffsets []int, glyphIndex int, glyphOrder []string) (*GlyphOutline, error) {
	if glyphIndex < 0 || glyphIndex >= len(locaOffsets)-1 {
		return nil, fmt.Errorf("glyph %d out of range", glyphIndex)
	}

	offset := locaOffsets[glyphIndex]
	nextOffset := locaOffsets[glyphIndex+1]

	outline := &GlyphOutline{GlyphIndex: glyphIndex}
	if glyphOrder != nil && glyphIndex < len(glyphOrder) {
		outline.GlyphName = glyphOrder[glyphIndex]
	}

	if offset >= nextOffset || offset+10 > len(glyfData) {
		return outline, nil
	}

	numContours := int(int16(binary.BigEndian.Uint16(glyfData[offset:])))

	if numContours >= 0 {
		// Simple glyph: bbox at offset+2..offset+9.
		outline.XMin = int16(binary.BigEndian.Uint16(glyfData[offset+2:]))
		outline.YMin = int16(binary.BigEndian.Uint16(glyfData[offset+4:]))
		outline.XMax = int16(binary.BigEndian.Uint16(glyfData[offset+6:]))
		outline.YMax = int16(binary.BigEndian.Uint16(glyfData[offset+8:]))
		contours, err := parseSimpleGlyph(glyfData, offset, numContours, nextOffset)
		if err != nil {
			return nil, fmt.Errorf("glyph %d: %w", glyphIndex, err)
		}
		outline.Contours = contours
		return outline, nil
	}

	// Compound glyph: no stored bbox; the 8 bytes at offset+2..offset+9
	// are component flags/args. Bbox is left at zero.

	outline.Compound = true
	components, err := parseCompoundGlyph(glyfData, offset)
	if err != nil {
		return nil, fmt.Errorf("glyph %d (compound): %w", glyphIndex, err)
	}
	outline.Components = components

	for _, comp := range components {
		child, err := expandCompound(glyfData, locaOffsets, int(comp.GlyphIndex), glyphOrder)
		if err != nil {
			return nil, fmt.Errorf("glyph %d -> component %d: %w", glyphIndex, comp.GlyphIndex, err)
		}
		transformed := transformContours(child.Contours, comp)
		outline.Contours = append(outline.Contours, transformed...)
	}
	return outline, nil
}

// transformContours applies a component's transformation to a set of contours.
func transformContours(contours []GlyphContour, comp GlyphComponent) []GlyphContour {
	result := make([]GlyphContour, len(contours))
	for ci, contour := range contours {
		pts := make([]GlyphPoint, len(contour.Points))
		for pi, pt := range contour.Points {
			x, y := float64(pt.X), float64(pt.Y)

			if comp.WeHaveTwoByTwo {
				nx := comp.ScaleX*x + comp.Scale01*y
				ny := comp.Scale10*x + comp.ScaleY*y
				x, y = nx, ny
			} else if comp.WeHaveXY {
				x *= comp.ScaleX
				y *= comp.ScaleY
			} else if comp.WeHaveScale {
				x *= comp.Scale
				y *= comp.Scale
			}

			x += float64(comp.XOffset)
			y += float64(comp.YOffset)

			pts[pi] = GlyphPoint{
				X:       int16(math.Round(x)),
				Y:       int16(math.Round(y)),
				OnCurve: pt.OnCurve,
			}
		}
		result[ci] = GlyphContour{Points: pts}
	}
	return result
}

// ExpandAllCompounds expands all compound glyphs in the font into flat contours.
func (font *Font) ExpandAllCompounds() ([]GlyphOutline, error) {
	outlines, err := font.GlyphOutlines()
	if err != nil {
		return nil, err
	}
	for i := range outlines {
		if outlines[i].Compound {
			expanded, err := font.ExpandCompoundGlyph(i)
			if err != nil {
				return nil, err
			}
			outlines[i] = *expanded
		}
	}
	return outlines, nil
}
