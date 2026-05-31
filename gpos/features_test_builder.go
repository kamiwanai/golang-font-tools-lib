package gpos

import (
	"encoding/binary"
)

type langSysDef struct {
	reqFeatureIndex uint16
	featureIndices  []uint16
}

type langSysRecordDef struct {
	tag     string
	langSys langSysDef
}

type scriptDef struct {
	tag            string
	defaultLangSys *langSysDef
	langs          []langSysRecordDef
}

type featureDef struct {
	tag           string
	lookupIndices []uint16
}

func encodeLangSys(ls langSysDef) []byte {
	size := 6 + len(ls.featureIndices)*2
	buf := make([]byte, size)
	binary.BigEndian.PutUint16(buf[0:2], 0) // lookupOrder
	reqIdx := ls.reqFeatureIndex
	if reqIdx == 0 && len(ls.featureIndices) > 0 {
		reqIdx = 0xFFFF
	}
	binary.BigEndian.PutUint16(buf[2:4], reqIdx)
	binary.BigEndian.PutUint16(buf[4:6], uint16(len(ls.featureIndices)))
	for i, fi := range ls.featureIndices {
		binary.BigEndian.PutUint16(buf[6+i*2:8+i*2], fi)
	}
	return buf
}

func buildGPOSFeatures(scripts []scriptDef, features []featureDef) []byte {
	type scriptLayout struct {
		tag             string
		defaultLangSys  *langSysDef
		langs           []langSysRecordDef
		defaultLangData []byte
		langData        [][]byte
	}
	sLayouts := make([]scriptLayout, len(scripts))
	for i, s := range scripts {
		sLayouts[i].tag = s.tag
		sLayouts[i].defaultLangSys = s.defaultLangSys
		sLayouts[i].langs = s.langs
		if s.defaultLangSys != nil {
			sLayouts[i].defaultLangData = encodeLangSys(*s.defaultLangSys)
		}
		sLayouts[i].langData = make([][]byte, len(s.langs))
		for j, l := range s.langs {
			sLayouts[i].langData[j] = encodeLangSys(l.langSys)
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
		featureListSize += 4 + len(f.lookupIndices)*2
	}

	scriptListOffset := 10
	featureListOffset := scriptListOffset + scriptListSize
	lookupListOffset := featureListOffset + featureListSize
	totalSize := lookupListOffset + 2
	data := make([]byte, totalSize)

	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 0)
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
			copy(data[pos:pos+4], l.tag)
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
			offset += 4 + len(f.lookupIndices)*2
		}
	}

	for i, f := range features {
		copy(data[pos:pos+4], f.tag)
		binary.BigEndian.PutUint16(data[pos+4:pos+6], uint16(featureOffsets[i]))
		pos += 6
	}

	for _, f := range features {
		binary.BigEndian.PutUint16(data[pos:pos+2], 0) // featureParams (NULL)
		binary.BigEndian.PutUint16(data[pos+2:pos+4], uint16(len(f.lookupIndices)))
		pos += 4
		for _, li := range f.lookupIndices {
			binary.BigEndian.PutUint16(data[pos:pos+2], li)
			pos += 2
		}
	}

	pos = lookupListOffset
	binary.BigEndian.PutUint16(data[pos:pos+2], 0)

	return data
}