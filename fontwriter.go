// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package fonttools

import (
	"encoding/binary"
	"fmt"

	"github.com/kamiwanai/golang-font-tools-lib/gpos"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

func ApplyKerningToFont(fontData []byte, pairs []gpos.KerningPairInput) ([]byte, error) {
	font, err := opentype.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parse font: %w", err)
	}
	glyphMap, err := buildGlyphMap(font)
	if err != nil {
		return nil, fmt.Errorf("build glyph map: %w", err)
	}
	if result, ok := tryInlineFormat1(fontData, font, pairs, glyphMap); ok {
		return result, nil
	}
	return rebuildFont(fontData, font, pairs, glyphMap)
}

func buildGlyphMap(font *opentype.Font) (map[string]uint16, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	m := make(map[string]uint16, len(glyphOrder)*2)
	// Direct glyph name lookup.
	for i, name := range glyphOrder {
		m[name] = uint16(i)
	}
	// Also map Unicode characters to glyph IDs via cmap.
	// This allows the frontend to send single characters (like "e", "A", "ж")
	// instead of requiring exact glyph names.
	cmap, err := font.BestCmap()
	if err == nil {
		for codepoint, glyphID := range cmap {
			char := string(rune(codepoint))
			// Only add if not already mapped (glyph name takes priority).
			if _, exists := m[char]; !exists {
				m[char] = glyphID
			}
		}
	}
	return m, nil
}

type subInfo struct {
	coverage      []uint16
	pairSetAbsOff []int
	valFmt1       uint16
}

func tryInlineFormat1(fontData []byte, font *opentype.Font, pairs []gpos.KerningPairInput, glyphMap map[string]uint16) ([]byte, bool) {
	gposData, err := font.TableData("GPOS")
	if err != nil {
		return nil, false
	}
	var subtables []subInfo
	opentype.WalkGPOSLookups(gposData, func(lookupIndex int, lookupData []byte) bool {
		if len(lookupData) < 6 || opentype.ReadU16(lookupData, 0) != 2 {
			return true
		}
		subCount := int(opentype.ReadU16(lookupData, 4))
		for si := 0; si < subCount; si++ {
			if 6+si*2+2 > len(lookupData) {
				continue
			}
			subOff := int(opentype.ReadU16(lookupData, 6+si*2))
			if subOff+10 > len(lookupData) || opentype.ReadU16(lookupData, subOff) != 1 {
				continue
			}
			covOff := int(opentype.ReadU16(lookupData, subOff+2))
			valFmt1 := opentype.ReadU16(lookupData, subOff+4)
			pairSetCnt := int(opentype.ReadU16(lookupData, subOff+8))
			if subOff+10+pairSetCnt*2 > len(lookupData) {
				continue
			}
			cov, err := opentype.DecodeCov(lookupData, subOff+covOff)
			if err != nil || len(cov) < pairSetCnt {
				continue
			}
			gposTable := font.Tables["GPOS"]
			absSub := int(gposTable.Offset) + subOff
			psAbs := make([]int, pairSetCnt)
			for pi := 0; pi < pairSetCnt; pi++ {
				psAbs[pi] = absSub + int(opentype.ReadU16(lookupData, subOff+10+pi*2))
			}
			subtables = append(subtables, subInfo{cov, psAbs, valFmt1})
		}
		return true
	})
	if len(subtables) == 0 {
		return nil, false
	}

	// Build a lookup of all existing pairs: (leftGID, rightGID) -> exists.
	existingPairs := make(map[uint32]bool)
	for _, st := range subtables {
		for i, lid := range st.coverage {
			ps := fontData[st.pairSetAbsOff[i]:]
			if len(ps) < 2 {
				continue
			}
			pvCount := int(opentype.ReadU16(ps, 0))
			recSize := 2
			if st.valFmt1&0x0004 != 0 {
				recSize += 2
			}
			for ri := 0; ri < pvCount; ri++ {
				recOff := 2 + ri*recSize
				if recOff+2 > len(ps) {
					continue
				}
				rid := opentype.ReadU16(ps, recOff)
				existingPairs[(uint32(lid)<<16)|uint32(rid)] = true
			}
		}
	}

	// Verify ALL input pairs exist in the font. If any don't, fall through to rebuildFont.
	for _, p := range pairs {
		lid := glyphMap[p.Left]
		rid := glyphMap[p.Right]
		key := (uint32(lid) << 16) | uint32(rid)
		if !existingPairs[key] {
			return nil, false // new pair not in font — need full rebuild
		}
	}

	// All pairs exist — safe to modify in-place.
	result := make([]byte, len(fontData))
	copy(result, fontData)
	modified := false
	for _, st := range subtables {
		leftIdx := make(map[uint16]int)
		for i, gid := range st.coverage {
			leftIdx[gid] = i
		}
		for _, p := range pairs {
			lid := glyphMap[p.Left]
			rid := glyphMap[p.Right]
			psi, ok := leftIdx[lid]
			if !ok {
				continue
			}
			ps := result[st.pairSetAbsOff[psi]:]
			if len(ps) < 2 {
				continue
			}
			pvCount := int(opentype.ReadU16(ps, 0))
			recSize := 2
			if st.valFmt1&0x0004 != 0 {
				recSize += 2
			}
			if len(ps) < 2+pvCount*recSize {
				continue
			}
			for ri := 0; ri < pvCount; ri++ {
				recOff := 2 + ri*recSize
				if opentype.ReadU16(ps, recOff) == rid {
					binary.BigEndian.PutUint16(result[st.pairSetAbsOff[psi]+recOff+2:], uint16(p.Value))
					modified = true
					break
				}
			}
		}
	}
	if !modified {
		return nil, false
	}
	opentype.RecalcTableChecksum(result, "GPOS")
	opentype.RecalcHeadChecksum(result)
	return result, true
}

func rebuildFont(fontData []byte, font *opentype.Font, pairs []gpos.KerningPairInput, glyphMap map[string]uint16) ([]byte, error) {
	existing := collectExistingPairs(font, fontData)
	merged := mergePairs(existing, pairs, glyphMap)
	newGPOS, err := gpos.BuildGPOS(merged, glyphMap)
	if err != nil {
		return nil, fmt.Errorf("build GPOS: %w", err)
	}
	allTags := opentype.SortedTableTags(font)
	tables := make(map[string][]byte)
	haveGPOS := false
	for _, tag := range allTags {
		if tag == "GPOS" {
			haveGPOS = true
			tables[tag] = newGPOS
		} else {
			data, err := font.TableData(tag)
			if err != nil {
				return nil, fmt.Errorf("read table %s: %w", tag, err)
			}
			tables[tag] = data
		}
	}
	if !haveGPOS {
		tables["GPOS"] = newGPOS
		newOrder := make([]string, 0, len(allTags)+1)
		inserted := false
		for _, tag := range allTags {
			if !inserted && (tag == "hmtx" || tag == "kern") {
				newOrder = append(newOrder, "GPOS")
				inserted = true
			}
			newOrder = append(newOrder, tag)
		}
		if !inserted {
			newOrder = append(newOrder, "GPOS")
		}
		allTags = newOrder
	}
	return opentype.SerializeFont(allTags, tables)
}

func collectExistingPairs(font *opentype.Font, fontData []byte) []gpos.KerningPairInput {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		return nil
	}
	existing, err := gpos.DecodePairPosLookups(gposData, glyphOrder)
	if err != nil {
		return nil
	}
	var pairs []gpos.KerningPairInput
	for _, p := range existing {
		pairs = append(pairs, gpos.KerningPairInput{
			Left:  p.LeftGlyph,
			Right: p.RightGlyph,
			Value: int16(p.Value),
		})
	}
	return pairs
}

func mergePairs(existing, new []gpos.KerningPairInput, glyphMap map[string]uint16) []gpos.KerningPairInput {
	// Phase 1: deduplicate new pairs — last wins (most recent source).
	// Also drop zero-value pairs (dead weight in GPOS).
	deduped := make(map[uint32]int16)
	order := make([]uint32, 0, len(new))
	for _, p := range new {
		key := (uint32(glyphMap[p.Left]) << 16) | uint32(glyphMap[p.Right])
		if _, seen := deduped[key]; !seen {
			order = append(order, key)
		}
		deduped[key] = p.Value
	}
	var result []gpos.KerningPairInput
	for _, key := range order {
		val := deduped[key]
		if val == 0 {
			continue // skip zero-value pairs
		}
		result = append(result, gpos.KerningPairInput{
			Left:  decodeGlyphName(glyphMap, uint16(key>>16)),
			Right: decodeGlyphName(glyphMap, uint16(key&0xFFFF)),
			Value: val,
		})
	}

	// Phase 2: add existing pairs not already covered by new pairs.
	for _, p := range existing {
		if p.Value == 0 {
			continue
		}
		key := (uint32(glyphMap[p.Left]) << 16) | uint32(glyphMap[p.Right])
		if _, seen := deduped[key]; !seen {
			result = append(result, p)
		}
	}
	return result
}

// decodeGlyphName looks up a glyph name by its glyph ID.
func decodeGlyphName(glyphMap map[string]uint16, gid uint16) string {
	for name, id := range glyphMap {
		if id == gid {
			return name
		}
	}
	return ""
}
