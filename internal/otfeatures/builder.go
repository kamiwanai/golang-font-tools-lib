// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package otfeatures

import "encoding/binary"

// LangSysDef is a test-only struct used to build LangSys bytes.
type LangSysDef struct {
	ReqFeatureIndex uint16
	FeatureIndices  []uint16
}

// LangSysRecordDef is a test-only struct used to build LangSysRecord entries.
type LangSysRecordDef struct {
	Tag     string
	LangSys LangSysDef
}

// ScriptDef is a test-only struct used to build Script bytes.
type ScriptDef struct {
	Tag            string
	DefaultLangSys *LangSysDef
	Langs          []LangSysRecordDef
}

// FeatureDef is a test-only struct used to build Feature bytes.
type FeatureDef struct {
	Tag           string
	LookupIndices []uint16
}

// EncodeLangSys encodes a LangSys into raw bytes.
func EncodeLangSys(ls LangSysDef) []byte {
	size := 6 + len(ls.FeatureIndices)*2
	buf := make([]byte, size)
	binary.BigEndian.PutUint16(buf[0:2], 0) // lookupOrder
	reqIdx := ls.ReqFeatureIndex
	if reqIdx == 0 && len(ls.FeatureIndices) > 0 {
		reqIdx = 0xFFFF
	}
	binary.BigEndian.PutUint16(buf[2:4], reqIdx)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(ls.FeatureIndices)))
	for i, fi := range ls.FeatureIndices {
		binary.BigEndian.PutUint16(buf[6+i*2:8+i*2], fi)
	}
	return buf
}

// BuildTable assembles a GPOS/GSUB-like table from the given scripts and
// features. The returned bytes contain a 10-byte header (version 1.0 with
// ScriptList/FeatureList/LookupList offsets) followed by the lists.
func BuildTable(scripts []ScriptDef, features []FeatureDef) []byte {
	type scriptLayout struct {
		tag             string
		defaultLangSys  *LangSysDef
		langs           []LangSysRecordDef
		defaultLangData []byte
		langData        [][]byte
	}
	sLayouts := make([]scriptLayout, len(scripts))
	for i, s := range scripts {
		sLayouts[i].tag = s.Tag
		sLayouts[i].defaultLangSys = s.DefaultLangSys
		sLayouts[i].langs = s.Langs
		if s.DefaultLangSys != nil {
			sLayouts[i].defaultLangData = EncodeLangSys(*s.DefaultLangSys)
		}
		sLayouts[i].langData = make([][]byte, len(s.Langs))
		for j, l := range s.Langs {
			sLayouts[i].langData[j] = EncodeLangSys(l.LangSys)
		}
	}

	scriptListSize := 2 + len(scripts)*6
	for _, sl := range sLayouts {
		scriptSize := 4 + len(sl.langs)*6
		if sl.defaultLangData != nil {
			scriptSize += len(sl.defaultLangData)
		}
		for _, ld := range sl.langData {
			scriptSize += len(ld)
		}
		scriptListSize += scriptSize
	}

	featureListSize := 2 + len(features)*6
	for _, f := range features {
		featureListSize += 4 + len(f.LookupIndices)*2
	}

	scriptListOffset := 10
	featureListOffset := scriptListOffset + scriptListSize
	lookupListOffset := featureListOffset + featureListSize
	totalSize := lookupListOffset + 2
	data := make([]byte, totalSize)

	binary.BigEndian.PutUint16(data[0:2], 1) // majorVersion
	binary.BigEndian.PutUint16(data[2:4], 0) // minorVersion
	binary.BigEndian.PutUint16(data[4:6], uint16(scriptListOffset))
	binary.BigEndian.PutUint16(data[6:8], uint16(featureListOffset))
	binary.BigEndian.PutUint16(data[8:10], uint16(lookupListOffset))

	pos := scriptListOffset
	binary.BigEndian.PutUint16(data[pos:pos+2], uint16(len(scripts)))
	pos += 2

	scriptBodyStart := 2 + len(scripts)*6
	scriptOffsets := make([]int, len(scripts))
	{
		offset := scriptBodyStart
		for i, sl := range sLayouts {
			scriptOffsets[i] = offset
			sz := 4 + len(sl.langs)*6
			if sl.defaultLangData != nil {
				sz += len(sl.defaultLangData)
			}
			for _, ld := range sl.langData {
				sz += len(ld)
			}
			offset += sz
		}
	}

	for i, sl := range sLayouts {
		copy(data[pos:pos+4], sl.tag)
		binary.BigEndian.PutUint16(data[pos+4:pos+6], uint16(scriptOffsets[i]))
		pos += 6
	}

	for _, sl := range sLayouts {
		if sl.defaultLangData != nil {
			defaultOffset := 4 + len(sl.langs)*6
			binary.BigEndian.PutUint16(data[pos:pos+2], uint16(defaultOffset))
		} else {
			binary.BigEndian.PutUint16(data[pos:pos+2], 0)
		}
		pos += 2
		binary.BigEndian.PutUint16(data[pos:pos+2], uint16(len(sl.langs)))
		pos += 2

		langOffset := 4 + len(sl.langs)*6
		if sl.defaultLangData != nil {
			langOffset += len(sl.defaultLangData)
		}
		for j, l := range sl.langs {
			copy(data[pos:pos+4], l.Tag)
			binary.BigEndian.PutUint16(data[pos+4:pos+6], uint16(langOffset))
			pos += 6
			langOffset += len(sl.langData[j])
		}

		if sl.defaultLangData != nil {
			copy(data[pos:pos+len(sl.defaultLangData)], sl.defaultLangData)
			pos += len(sl.defaultLangData)
		}
		for _, ld := range sl.langData {
			copy(data[pos:pos+len(ld)], ld)
			pos += len(ld)
		}
	}

	pos = featureListOffset
	binary.BigEndian.PutUint16(data[pos:pos+2], uint16(len(features)))
	pos += 2

	featureBodyStart := 2 + len(features)*6
	featureOffsets := make([]int, len(features))
	{
		offset := featureBodyStart
		for i, f := range features {
			featureOffsets[i] = offset
			offset += 4 + len(f.LookupIndices)*2
		}
	}

	for i, f := range features {
		copy(data[pos:pos+4], f.Tag)
		binary.BigEndian.PutUint16(data[pos+4:pos+6], uint16(featureOffsets[i]))
		pos += 6
	}

	for _, f := range features {
		binary.BigEndian.PutUint16(data[pos:pos+2], 0) // featureParams (NULL)
		binary.BigEndian.PutUint16(data[pos+2:pos+4], uint16(len(f.LookupIndices)))
		pos += 4
		for _, li := range f.LookupIndices {
			binary.BigEndian.PutUint16(data[pos:pos+2], li)
			pos += 2
		}
	}

	pos = lookupListOffset
	binary.BigEndian.PutUint16(data[pos:pos+2], 0) // empty lookup list

	return data
}
