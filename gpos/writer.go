// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gpos

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// KerningPairInput is a kerning pair to apply, using glyph names.
type KerningPairInput struct {
	Left  string
	Right string
	Value int16
}

type pairRec struct {
	right uint16
	value int16
}

// BuildGPOS builds a minimal GPOS v1.0 table with PairPos Format 1 kerning.
// glyphMap maps glyph names to glyph IDs (both left and right glyphs).
func BuildGPOS(pairs []KerningPairInput, glyphMap map[string]uint16) ([]byte, error) {
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no kerning pairs to apply")
	}

	// Group by left glyph, deduplicate (right glyph, value).
	// When the same pair appears multiple times, keep the LAST value.
	leftGroupsRaw := make(map[uint16]map[uint16]int16)
	for _, p := range pairs {
		lid, ok := glyphMap[p.Left]
		if !ok {
			return nil, fmt.Errorf("glyph %q not found", p.Left)
		}
		rid, ok := glyphMap[p.Right]
		if !ok {
			return nil, fmt.Errorf("glyph %q not found", p.Right)
		}
		if leftGroupsRaw[lid] == nil {
			leftGroupsRaw[lid] = make(map[uint16]int16)
		}
		leftGroupsRaw[lid][rid] = p.Value
	}
	leftGroups := make(map[uint16][]pairRec)
	for lid, rightMap := range leftGroupsRaw {
		recs := make([]pairRec, 0, len(rightMap))
		for rid, val := range rightMap {
			if val == 0 {
				continue // skip zero-value pairs (dead weight)
			}
			recs = append(recs, pairRec{right: rid, value: val})
		}
		if len(recs) > 0 {
			leftGroups[lid] = recs
		}
	}

	if len(leftGroups) == 0 {
		return nil, fmt.Errorf("no kerning pairs to apply (all zero-value)")
	}

	// Sorted left glyphs → Coverage.
	coverage := make([]uint16, 0, len(leftGroups))
	for lid := range leftGroups {
		coverage = append(coverage, lid)
	}
	sort.Slice(coverage, func(i, j int) bool { return coverage[i] < coverage[j] })

	// Build Coverage Format 1.
	covBytes := makeCovFmt1(coverage)

	// Build PairSets (one per left glyph).
	pairSets := make([][]byte, len(coverage))
	for i, lid := range coverage {
		recs := leftGroups[lid]
		sort.Slice(recs, func(a, b int) bool { return recs[a].right < recs[b].right })
		pairSets[i] = makePairSet(recs)
	}

	// ── PairPos Format 1 subtable ──
	// Header: posFormat(2) + coverageOff(2) + valFmt1(2) + valFmt2(2) + pairSetCount(2) + offsets
	// valFmt1 = 0x0004 (xAdvance), valFmt2 = 0
	const valFmt1 uint16 = 0x0004
	ppHdr := 10 + len(coverage)*2
	covOff := ppHdr
	curOff := covOff + len(covBytes)
	pairSetOffs := make([]uint16, len(coverage))
	for i := range coverage {
		pairSetOffs[i] = uint16(curOff)
		curOff += len(pairSets[i])
	}
	pp := make([]byte, curOff)
	w := 0
	wu16(pp, &w, 1) // posFormat
	wu16(pp, &w, uint16(covOff))
	wu16(pp, &w, valFmt1)
	wu16(pp, &w, 0) // valFmt2
	wu16(pp, &w, uint16(len(coverage)))
	for _, o := range pairSetOffs {
		wu16(pp, &w, o)
	}
	copy(pp[w:], covBytes)
	w += len(covBytes)
	for _, ps := range pairSets {
		copy(pp[w:], ps)
		w += len(ps)
	}

	// ── Lookup (type 2) ──
	// lookupType(2)+lookupFlag(2)+subtableCount(2)+subtableOffsets[...]+pp
	lkHdr := []byte{0, 2, 0, 0, 0, 1} // lookupType=2, flag=0, subtableCount=1
	ppRel := uint16(6 + 2) // after header + 1 offset
	lkOff := []byte{byte(ppRel >> 8), byte(ppRel)}
	lookup := append(append([]byte{}, lkHdr...), append(lkOff, pp...)...)

	// ── LookupList ──
	llHdr := []byte{0, 1} // lookupCount=1
	lkRel := uint16(4)
	llOff := []byte{byte(lkRel >> 8), byte(lkRel)}
	lookupList := append(append([]byte{}, llHdr...), append(llOff, lookup...)...)

	// ── LangSys ──
	langSys := []byte{0, 0, 0xFF, 0xFF, 0, 1, 0, 0}

	// ── ScriptTable ──
	scriptTable := append([]byte{0, 2, 0, 0}, langSys...)

	// ── ScriptList ──
	scriptList := []byte{0, 1, 'D', 'F', 'L', 'T', 0, 8}
	scriptList = append(scriptList, scriptTable...)

	// ── FeatureTable ──
	featureTable := []byte{0, 0, 0, 0, 0, 1, 0, 0}

	// ── FeatureList ──
	featureList := []byte{0, 1, 'k', 'e', 'r', 'n', 0, 8}
	featureList = append(featureList, featureTable...)

	// ── GPOS v1.0 header (10 bytes) ──
	scrOff := uint16(10)
	featOff := scrOff + uint16(len(scriptList))
	lklOff := featOff + uint16(len(featureList))

	gpos := make([]byte, 10)
	w = 0
	wu16(gpos, &w, 1)       // majorVersion
	wu16(gpos, &w, 0)       // minorVersion
	wu16(gpos, &w, scrOff)  // scriptListOffset
	wu16(gpos, &w, featOff) // featureListOffset
	wu16(gpos, &w, lklOff)  // lookupListOffset
	gpos = append(gpos, scriptList...)
	gpos = append(gpos, featureList...)
	gpos = append(gpos, lookupList...)

	return gpos, nil
}

func makeCovFmt1(glyphs []uint16) []byte {
	b := make([]byte, 4+len(glyphs)*2)
	p := 0
	wu16(b, &p, 1)
	wu16(b, &p, uint16(len(glyphs)))
	for _, g := range glyphs {
		wu16(b, &p, g)
	}
	return b
}

func makePairSet(recs []pairRec) []byte {
	b := make([]byte, 2+len(recs)*4)
	p := 0
	wu16(b, &p, uint16(len(recs)))
	for _, r := range recs {
		wu16(b, &p, r.right)
		wu16(b, &p, uint16(r.value))
	}
	return b
}

func wu16(b []byte, p *int, v uint16) {
	binary.BigEndian.PutUint16(b[*p:], v)
	*p += 2
}
