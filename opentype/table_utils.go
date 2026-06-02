// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
	"sort"
)


func SortedTableTags(font *Font) []string {
	tags := make([]string, 0, len(font.Tables))
	for tag := range font.Tables {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func SerializeFont(order []string, tables map[string][]byte) ([]byte, error) {
	numTables := len(order)
	headerSize := 12
	dirSize := numTables * 16
	dataStart := headerSize + dirSize
	offsets := make(map[string]uint32)
	curOff := uint32(dataStart)
	for _, tag := range order {
		offsets[tag] = curOff
		padded := (len(tables[tag]) + 3) & ^3
		curOff += uint32(padded)
	}
	result := make([]byte, curOff)
	sfVersion := uint32(0x00010000)
	if _, ok := tables["CFF "]; ok {
		sfVersion = 0x4F54544F
	}
	binary.BigEndian.PutUint32(result[0:4], sfVersion)
	binary.BigEndian.PutUint16(result[4:6], uint16(numTables))
	maxPower := uint16(1)
	log2 := uint16(0)
	for maxPower*2 <= uint16(numTables) {
		maxPower *= 2
		log2++
	}
	searchRange := maxPower * 16
	binary.BigEndian.PutUint16(result[6:8], searchRange)
	binary.BigEndian.PutUint16(result[8:10], log2)
	binary.BigEndian.PutUint16(result[10:12], uint16(numTables)*16-searchRange)
	for i, tag := range order {
		dirOff := headerSize + i*16
		padded := make([]byte, 4)
		copy(padded, tag)
		copy(result[dirOff:dirOff+4], padded)
		data := tables[tag]
		binary.BigEndian.PutUint32(result[dirOff+4:dirOff+8], calcChecksum(data))
		binary.BigEndian.PutUint32(result[dirOff+8:dirOff+12], offsets[tag])
		binary.BigEndian.PutUint32(result[dirOff+12:dirOff+16], uint32(len(data)))
		copy(result[offsets[tag]:], data)
		paddedLen := (len(data) + 3) & ^3
		for j := len(data); j < paddedLen; j++ {
			result[offsets[tag]+uint32(j)] = 0
		}
	}
	var sum uint32
	for i := 0; i < len(result); i += 4 {
		sum += binary.BigEndian.Uint32(result[i:])
	}
	headOff := offsets["head"]
	if headOff > 0 {
		binary.BigEndian.PutUint32(result[headOff+8:headOff+12], 0xB1B0AFBA-sum)
	}
	return result, nil
}

func RecalcTableChecksum(data []byte, tag string) {
	font, err := Parse(data)
	if err != nil {
		return
	}
	table, ok := font.Tables[tag]
	if !ok {
		return
	}
	td := data[table.Offset : table.Offset+table.Length]
	chk := calcChecksum(td)
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	for i := 0; i < numTables; i++ {
		off := 12 + i*16
		if string(data[off:off+4]) == tag {
			binary.BigEndian.PutUint32(data[off+4:off+8], chk)
			break
		}
	}
}

func RecalcHeadChecksum(data []byte) {
	var sum uint32
	for i := 0; i+3 < len(data); i += 4 {
		sum += binary.BigEndian.Uint32(data[i:])
	}
	font, _ := Parse(data)
	if font == nil {
		return
	}
	table, ok := font.Tables["head"]
	if !ok {
		return
	}
	binary.BigEndian.PutUint32(data[table.Offset+8:table.Offset+12], 0xB1B0AFBA-sum)
}

func calcChecksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i+3 < len(data); i += 4 {
		sum += binary.BigEndian.Uint32(data[i:])
	}
	rem := len(data) % 4
	if rem > 0 {
		var last uint32
		for i := len(data) - rem; i < len(data); i++ {
			last = (last << 8) | uint32(data[i])
		}
		last <<= (4 - rem) * 8
		sum += last
	}
	return sum
}

func WalkGPOSLookups(data []byte, fn func(lookupIndex int, lookupData []byte) bool) {
	if len(data) < 10 {
		return
	}
	llOff := int(ReadU16(data, 8))
	if llOff+2 > len(data) {
		return
	}
	lookupCount := int(ReadU16(data, llOff))
	for i := 0; i < lookupCount; i++ {
		if llOff+2+(i+1)*2 > len(data) {
			break
		}
		lkOff := int(ReadU16(data, llOff+2+i*2))
		if lkOff >= len(data) {
			continue
		}
		if !fn(i, data[lkOff:]) {
			break
		}
	}
}

func DecodeCov(data []byte, offset int) ([]uint16, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("coverage too short")
	}
	if ReadU16(data, offset) != 1 {
		return nil, fmt.Errorf("unsupported coverage format")
	}
	count := int(ReadU16(data, offset+2))
	if offset+4+count*2 > len(data) {
		return nil, fmt.Errorf("coverage exceeds data")
	}
	glyphs := make([]uint16, count)
	for i := 0; i < count; i++ {
		glyphs[i] = ReadU16(data, offset+4+i*2)
	}
	return glyphs, nil
}

