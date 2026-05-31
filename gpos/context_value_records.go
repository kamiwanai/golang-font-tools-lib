package gpos

import "fmt"

// ContextValueRecord holds X/Y placement and advance extracted from a nested
// lookup referenced by a context/chain-context rule.
type ContextValueRecord struct {
	Glyph       string
	XPlacement  int16
	YPlacement  int16
	XAdvance    int16
	YAdvance    int16
	LookupIndex int
	LookupType  uint16
	Format      int
}

// DecodeContextValueRecords extracts ValueRecord data from nested lookups
// referenced by ContextPos/ChainContextPos rules.
func DecodeContextValueRecords(data []byte, glyphOrder []string, wantType uint16) ([]ContextValueRecord, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("GPOS header too short: %d bytes", len(data))
	}
	lookupListOffset := int(readU16(data, 8))
	if lookupListOffset+2 > len(data) {
		return nil, fmt.Errorf("GPOS LookupList offset exceeds table")
	}
	lookupCount := int(readU16(data, lookupListOffset))
	if lookupCount > MaxLookupCount {
		return nil, fmt.Errorf("GPOS LookupCount %d exceeds limit", lookupCount)
	}
	lookups, err := decodeContextPosLookups(data, glyphOrder, wantType)
	if err != nil {
		return nil, err
	}
	var all []ContextValueRecord
	for _, cl := range lookups {
		for _, rule := range cl.Rules {
			for _, nested := range rule.NestedLookups {
				li := int(nested.LookupListIndex)
				if li >= lookupCount {
					continue
				}
				lo := lookupListOffset + int(readU16(data, lookupListOffset+2+li*2))
				vals, err := decodeNestedLookupValues(data, glyphOrder, lo, li)
				if err != nil {
					return nil, fmt.Errorf("nested lookup %d: %w", li, err)
				}
				all = append(all, vals...)
			}
		}
	}
	return all, nil
}

func decodeNestedLookupValues(data []byte, glyphOrder []string, lookupOffset int, lookupIndex int) ([]ContextValueRecord, error) {
	if lookupOffset+6 > len(data) {
		return nil, fmt.Errorf("lookup %d exceeds table", lookupIndex)
	}
	lookupType := readU16(data, lookupOffset)
	subtableCount := int(readU16(data, lookupOffset+4))
	if lookupOffset+6+subtableCount*2 > len(data) {
		return nil, fmt.Errorf("lookup %d subtable offsets exceed table", lookupIndex)
	}
	var all []ContextValueRecord
	for i := 0; i < subtableCount; i++ {
		so := lookupOffset + int(readU16(data, lookupOffset+6+i*2))
		if lookupType == 9 {
			so = decodeCPExtension(data, so, lookupType)
			if so == 0 {
				continue
			}
		}
		vals, err := decodeNestedSubtableValues(data, glyphOrder, lookupType, so, lookupIndex)
		if err != nil {
			return nil, err
		}
		all = append(all, vals...)
	}
	return all, nil
}

func decodeNestedSubtableValues(data []byte, glyphOrder []string, lookupType uint16, subtableOffset int, lookupIndex int) ([]ContextValueRecord, error) {
	if subtableOffset+2 > len(data) {
		return nil, nil
	}
	format := int(readU16(data, subtableOffset))
	actualType := lookupType
	if lookupType == 9 {
		if subtableOffset+8 > len(data) {
			return nil, nil
		}
		extType := readU16(data, subtableOffset+2)
		if extType == 1 || extType == 2 {
			actualType = extType
			extOff := int(readU32(data, subtableOffset+4))
			subtableOffset = subtableOffset + extOff
			if subtableOffset+2 > len(data) {
				return nil, nil
			}
			format = int(readU16(data, subtableOffset))
		}
	}
	switch actualType {
	case 1:
		return decodeNestedSinglePos(data, glyphOrder, subtableOffset, format, lookupIndex)
	case 2:
		return decodeNestedPairPos(data, glyphOrder, subtableOffset, format, lookupIndex)
	}
	return nil, nil
}

func decodeNestedSinglePos(data []byte, glyphOrder []string, offset int, format int, lookupIndex int) ([]ContextValueRecord, error) {
	switch format {
	case 1:
		return decodeNestedSingleFormat1(data, glyphOrder, offset, lookupIndex)
	case 2:
		return decodeNestedSingleFormat2(data, glyphOrder, offset, lookupIndex)
	}
	return nil, nil
}

func decodeNestedSingleFormat1(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]ContextValueRecord, error) {
	if offset+6 > len(data) {
		return nil, fmt.Errorf("nested SinglePos fmt1 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	valueFormat := readU16(data, offset+4)
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	vr, _, err := decodeValueRecord(data, offset+6, valueFormat)
	if err != nil {
		return nil, err
	}
	var vals []ContextValueRecord
	for _, gid := range coverage {
		glyph, ok := glyphName(glyphOrder, gid)
		if !ok {
			continue
		}
		vals = append(vals, ContextValueRecord{
			Glyph:       glyph,
			XPlacement:  vr.XPlacement,
			YPlacement:  vr.YPlacement,
			XAdvance:    vr.XAdvance,
			YAdvance:    vr.YAdvance,
			LookupIndex: lookupIndex,
			LookupType:  1,
			Format:      1,
		})
	}
	return vals, nil
}

func decodeNestedSingleFormat2(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]ContextValueRecord, error) {
	if offset+8 > len(data) {
		return nil, fmt.Errorf("nested SinglePos fmt2 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	valueFormat := readU16(data, offset+4)
	valueCount := int(readU16(data, offset+6))
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	if len(coverage) < valueCount {
		valueCount = len(coverage)
	}
	pos := offset + 8
	var vals []ContextValueRecord
	for i := 0; i < valueCount; i++ {
		vr, consumed, err := decodeValueRecord(data, pos, valueFormat)
		if err != nil {
			return nil, err
		}
		pos += consumed
		glyph, ok := glyphName(glyphOrder, coverage[i])
		if !ok {
			continue
		}
		vals = append(vals, ContextValueRecord{
			Glyph:       glyph,
			XPlacement:  vr.XPlacement,
			YPlacement:  vr.YPlacement,
			XAdvance:    vr.XAdvance,
			YAdvance:    vr.YAdvance,
			LookupIndex: lookupIndex,
			LookupType:  1,
			Format:      2,
		})
	}
	return vals, nil
}

func decodeNestedPairPos(data []byte, glyphOrder []string, offset int, format int, lookupIndex int) ([]ContextValueRecord, error) {
	switch format {
	case 1:
		return decodeNestedPairFormat1(data, glyphOrder, offset, lookupIndex)
	case 2:
		return decodeNestedPairFormat2(data, glyphOrder, offset, lookupIndex)
	}
	return nil, nil
}

func decodeNestedPairFormat1(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]ContextValueRecord, error) {
	if offset+10 > len(data) {
		return nil, fmt.Errorf("nested PairPos fmt1 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	valueFormat1 := readU16(data, offset+4)
	valueFormat2 := readU16(data, offset+6)
	pairSetCount := int(readU16(data, offset+8))
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	if len(coverage) < pairSetCount {
		pairSetCount = len(coverage)
	}
	if offset+10+pairSetCount*2 > len(data) {
		return nil, fmt.Errorf("nested PairPos fmt1 offsets exceed")
	}
	valueSize1 := valueRecordSize(valueFormat1)
	valueSize2 := valueRecordSize(valueFormat2)
	recordSize := 2 + valueSize1 + valueSize2
	var vals []ContextValueRecord
	for i := 0; i < pairSetCount; i++ {
		leftGID := coverage[i]
		leftGlyph, ok := glyphName(glyphOrder, leftGID)
		if !ok {
			continue
		}
		pso := int(readU16(data, offset+10+i*2))
		if pso+2 > len(data) {
			continue
		}
		pvc := int(readU16(data, pso))
		ro := pso + 2
		if ro+pvc*recordSize > len(data) {
			continue
		}
		for j := 0; j < pvc; j++ {
			rec := ro + j*recordSize
			rightGID := readU16(data, rec)
			rightGlyph, ok := glyphName(glyphOrder, rightGID)
			if !ok {
				continue
			}
			vr1, _, _ := decodeValueRecord(data, rec+2, valueFormat1)
			if vr1.XPlacement != 0 || vr1.YPlacement != 0 || vr1.XAdvance != 0 || vr1.YAdvance != 0 {
				vals = append(vals, ContextValueRecord{
					Glyph: leftGlyph, XPlacement: vr1.XPlacement, YPlacement: vr1.YPlacement,
					XAdvance: vr1.XAdvance, YAdvance: vr1.YAdvance,
					LookupIndex: lookupIndex, LookupType: 2, Format: 1,
				})
			}
			vr2, _, _ := decodeValueRecord(data, rec+2+valueSize1, valueFormat2)
			if vr2.XPlacement != 0 || vr2.YPlacement != 0 || vr2.XAdvance != 0 || vr2.YAdvance != 0 {
				vals = append(vals, ContextValueRecord{
					Glyph: rightGlyph, XPlacement: vr2.XPlacement, YPlacement: vr2.YPlacement,
					XAdvance: vr2.XAdvance, YAdvance: vr2.YAdvance,
					LookupIndex: lookupIndex, LookupType: 2, Format: 1,
				})
			}
		}
	}
	return vals, nil
}

func decodeNestedPairFormat2(data []byte, glyphOrder []string, offset int, lookupIndex int) ([]ContextValueRecord, error) {
	if offset+16 > len(data) {
		return nil, fmt.Errorf("nested PairPos fmt2 too short")
	}
	coverageOffset := int(readU16(data, offset+2))
	valueFormat1 := readU16(data, offset+4)
	valueFormat2 := readU16(data, offset+6)
	classDef1Offset := int(readU16(data, offset+8))
	classDef2Offset := int(readU16(data, offset+10))
	class1Count := int(readU16(data, offset+12))
	class2Count := int(readU16(data, offset+14))
	coverage, err := decodeCoverage(data, offset+coverageOffset)
	if err != nil {
		return nil, err
	}
	classDef1, err := decodeClassDef(data, offset+classDef1Offset)
	if err != nil {
		return nil, err
	}
	classDef2, err := decodeClassDef(data, offset+classDef2Offset)
	if err != nil {
		return nil, err
	}
	classGlyphs1 := make(map[int][]uint16)
	for _, gid := range coverage {
		classGlyphs1[classDef1[gid]] = append(classGlyphs1[classDef1[gid]], gid)
	}
	classGlyphs2 := make(map[int][]uint16)
	for gid, c := range classDef2 {
		classGlyphs2[c] = append(classGlyphs2[c], gid)
	}
	valueSize1 := valueRecordSize(valueFormat1)
	valueSize2 := valueRecordSize(valueFormat2)
	recordSize := valueSize1 + valueSize2
	recordsOffset := offset + 16
	if recordsOffset+class1Count*class2Count*recordSize > len(data) {
		return nil, fmt.Errorf("nested PairPos fmt2 records exceed")
	}
	var vals []ContextValueRecord
	for c1 := 0; c1 < class1Count; c1++ {
		gids1 := classGlyphs1[c1]
		for c2 := 0; c2 < class2Count; c2++ {
			recOff := recordsOffset + (c1*class2Count+c2)*recordSize
			vr1, _, _ := decodeValueRecord(data, recOff, valueFormat1)
			vr2, _, _ := decodeValueRecord(data, recOff+valueSize1, valueFormat2)
			hasVR1 := vr1.XPlacement != 0 || vr1.YPlacement != 0 || vr1.XAdvance != 0 || vr1.YAdvance != 0
			hasVR2 := vr2.XPlacement != 0 || vr2.YPlacement != 0 || vr2.XAdvance != 0 || vr2.YAdvance != 0
			if hasVR1 {
				for _, gid := range gids1 {
					glyph, ok := glyphName(glyphOrder, gid)
					if !ok {
						continue
					}
					vals = append(vals, ContextValueRecord{
						Glyph: glyph, XPlacement: vr1.XPlacement, YPlacement: vr1.YPlacement,
						XAdvance: vr1.XAdvance, YAdvance: vr1.YAdvance,
						LookupIndex: lookupIndex, LookupType: 2, Format: 2,
					})
				}
			}
			if hasVR2 {
				gids2 := classGlyphs2[c2]
				for _, gid := range gids2 {
					glyph, ok := glyphName(glyphOrder, gid)
					if !ok {
						continue
					}
					vals = append(vals, ContextValueRecord{
						Glyph: glyph, XPlacement: vr2.XPlacement, YPlacement: vr2.YPlacement,
						XAdvance: vr2.XAdvance, YAdvance: vr2.YAdvance,
						LookupIndex: lookupIndex, LookupType: 2, Format: 2,
					})
				}
			}
		}
	}
	return vals, nil
}
