// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

// Package variation decodes OpenType ItemVariationStore, DeltaSetIndexMap,
// and related shared types used by HVAR, VVAR, MVAR, and GDEF v2.
//
// The same format is used by all these tables, so this code is shared
// between them. The caller wraps the result in a table-specific struct.
package variation

import (
	"encoding/binary"
	"fmt"
)

// F2DOT14 is a signed 16-bit fixed-point number with 14 fraction bits.
type F2DOT14 int16

// Float returns the float64 representation of this F2DOT14.
func (f F2DOT14) Float() float64 {
	return float64(f) / 16384.0
}

// RegionAxis defines one axis range for a variation region.
type RegionAxis struct {
	StartCoord F2DOT14
	PeakCoord  F2DOT14
	EndCoord   F2DOT14
}

// VariationRegion defines one region of applicability for a delta set.
type VariationRegion struct {
	RegionAxes []RegionAxis
}

// ItemVariationData holds delta data for a set of items across multiple regions.
type ItemVariationData struct {
	ItemCount      uint16
	WordDeltaCount uint16
	RegionIndexes  []uint16
	DeltaSets      [][]int32
}

// ItemVariationStore stores variation delta data shared across tables
// (HVAR, VVAR, MVAR, GDEF v2).
type ItemVariationStore struct {
	Format             uint16
	VariationRegions   []VariationRegion
	ItemVariationDatas []ItemVariationData
}

// DeltaSetIndexEntry is one entry in a DeltaSetIndexMap.
// It maps a glyph (or other item) to a delta-set (outer, inner) pair.
type DeltaSetIndexEntry struct {
	OuterIndex uint16
	InnerIndex uint16
}

// DeltaSetIndexMap is an indirect mapping from item indices to delta-set indices.
// Supports format 0 (uint16 mapCount) and format 1 (uint32 mapCount).
type DeltaSetIndexMap struct {
	Format      uint8
	EntryFormat uint8
	Entries     []DeltaSetIndexEntry
}

// Limits. These can be lowered for stricter validation or raised for
// permissive parsing, but should not be removed.

var MaxItemVariationDataCount int = 4096
var MaxVariationAxes int = 256
var MaxVariationRegions int = 65536
var MaxDeltaItems int = 65536
var MaxRegionIndexes int = 65536
var MaxDeltaSetIndexEntries int = 65536

// ReadU16 reads a big-endian uint16 from data at offset.
func ReadU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

// ReadU32 reads a big-endian uint32 from data at offset.
func ReadU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}

// DecodeItemVariationStore decodes an ItemVariationStore at the given offset.
func DecodeItemVariationStore(data []byte, offset int) (ItemVariationStore, error) {
	return decodeItemVariationStore(data, offset)
}

func decodeItemVariationStore(data []byte, offset int) (ItemVariationStore, error) {
	if offset+8 > len(data) {
		return ItemVariationStore{}, fmt.Errorf("ItemVariationStore header too short")
	}
	format := ReadU16(data, offset)
	if format != 1 {
		return ItemVariationStore{}, fmt.Errorf("ItemVariationStore unsupported format %d", format)
	}
	varRegionListOffset := int(ReadU32(data, offset+2))
	itemVarDataCount := int(ReadU16(data, offset+6))

	if itemVarDataCount > MaxItemVariationDataCount {
		return ItemVariationStore{}, fmt.Errorf("ItemVariationData count %d exceeds limit %d", itemVarDataCount, MaxItemVariationDataCount)
	}

	absRegionOffset := offset + varRegionListOffset
	regions, err := decodeVariationRegionList(data, absRegionOffset)
	if err != nil {
		return ItemVariationStore{}, fmt.Errorf("VariationRegionList: %w", err)
	}

	ivdOffsetsStart := offset + 8
	ivdOffsetsEnd := ivdOffsetsStart + itemVarDataCount*4
	if ivdOffsetsEnd > len(data) {
		return ItemVariationStore{}, fmt.Errorf("ItemVariationData offsets exceed table")
	}

	ivds := make([]ItemVariationData, itemVarDataCount)
	for i := 0; i < itemVarDataCount; i++ {
		ivdOff := int(ReadU32(data, ivdOffsetsStart+i*4))
		if ivdOff == 0 {
			continue
		}
		absOff := offset + ivdOff
		ivd, err := decodeItemVariationData(data, absOff)
		if err != nil {
			return ItemVariationStore{}, fmt.Errorf("ItemVariationData[%d]: %w", i, err)
		}
		ivds[i] = ivd
	}

	return ItemVariationStore{
		Format:             format,
		VariationRegions:   regions,
		ItemVariationDatas: ivds,
	}, nil
}

func decodeVariationRegionList(data []byte, offset int) ([]VariationRegion, error) {
	if offset+4 > len(data) {
		return nil, fmt.Errorf("VariationRegionList too short")
	}
	axisCount := int(ReadU16(data, offset))
	regionCount := int(ReadU16(data, offset+2))

	if axisCount > MaxVariationAxes {
		return nil, fmt.Errorf("axisCount %d exceeds limit %d", axisCount, MaxVariationAxes)
	}
	if regionCount > MaxVariationRegions {
		return nil, fmt.Errorf("regionCount %d exceeds limit %d", regionCount, MaxVariationRegions)
	}

	regionSize := axisCount * 6
	dataStart := offset + 4
	dataEnd := dataStart + regionCount*regionSize
	if dataEnd > len(data) {
		return nil, fmt.Errorf("VariationRegionList data exceeds table")
	}

	regions := make([]VariationRegion, regionCount)
	for i := 0; i < regionCount; i++ {
		axes := make([]RegionAxis, axisCount)
		for j := 0; j < axisCount; j++ {
			pos := dataStart + i*regionSize + j*6
			axes[j] = RegionAxis{
				StartCoord: F2DOT14(int16(ReadU16(data, pos))),
				PeakCoord:  F2DOT14(int16(ReadU16(data, pos+2))),
				EndCoord:   F2DOT14(int16(ReadU16(data, pos+4))),
			}
		}
		regions[i] = VariationRegion{RegionAxes: axes}
	}

	return regions, nil
}

func decodeItemVariationData(data []byte, offset int) (ItemVariationData, error) {
	if offset+6 > len(data) {
		return ItemVariationData{}, fmt.Errorf("ItemVariationData too short")
	}
	itemCount := int(ReadU16(data, offset))
	wordDeltaCount := int(ReadU16(data, offset+2))
	regionIdxCount := int(ReadU16(data, offset+4))

	if itemCount > MaxDeltaItems {
		return ItemVariationData{}, fmt.Errorf("itemCount %d exceeds limit %d", itemCount, MaxDeltaItems)
	}
	if regionIdxCount > MaxRegionIndexes {
		return ItemVariationData{}, fmt.Errorf("regionIdxCount %d exceeds limit %d", regionIdxCount, MaxRegionIndexes)
	}

	regionIdxStart := offset + 6
	regionIdxEnd := regionIdxStart + regionIdxCount*2
	if regionIdxEnd > len(data) {
		return ItemVariationData{}, fmt.Errorf("region indexes exceed table")
	}
	regionIndexes := make([]uint16, regionIdxCount)
	for i := 0; i < regionIdxCount; i++ {
		regionIndexes[i] = ReadU16(data, regionIdxStart+i*2)
	}

	bytesPerItem := wordDeltaCount*2 + (regionIdxCount-wordDeltaCount)*1
	deltaStart := regionIdxEnd
	deltaEnd := deltaStart + itemCount*bytesPerItem
	if deltaEnd > len(data) {
		return ItemVariationData{}, fmt.Errorf("delta sets exceed table")
	}

	deltaSets := make([][]int32, itemCount)
	for i := 0; i < itemCount; i++ {
		deltas := make([]int32, regionIdxCount)
		pos := deltaStart + i*bytesPerItem
		for j := 0; j < wordDeltaCount; j++ {
			deltas[j] = int32(int16(ReadU16(data, pos)))
			pos += 2
		}
		for j := wordDeltaCount; j < regionIdxCount; j++ {
			deltas[j] = int32(int8(data[pos]))
			pos++
		}
		deltaSets[i] = deltas
	}

	return ItemVariationData{
		ItemCount:      uint16(itemCount),
		WordDeltaCount: uint16(wordDeltaCount),
		RegionIndexes:  regionIndexes,
		DeltaSets:      deltaSets,
	}, nil
}

// DecodeDeltaSetIndexMap decodes a DeltaSetIndexMap at the given offset.
func DecodeDeltaSetIndexMap(data []byte, offset int) (DeltaSetIndexMap, error) {
	return decodeDeltaSetIndexMap(data, offset)
}

func decodeDeltaSetIndexMap(data []byte, offset int) (DeltaSetIndexMap, error) {
	if offset+1 > len(data) {
		return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap too short")
	}
	format := data[offset]
	if format != 0 && format != 1 {
		return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap unsupported format %d", format)
	}

	if offset+4 > len(data) {
		return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap header too short")
	}
	entryFormat := data[offset+1]

	var mapCount, dataStart int
	if format == 0 {
		mapCount = int(ReadU16(data, offset+2))
		dataStart = offset + 4
	} else {
		if offset+6 > len(data) {
			return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap format 1 header too short")
		}
		mapCount = int(ReadU32(data, offset+2))
		dataStart = offset + 6
	}

	if mapCount > MaxDeltaSetIndexEntries {
		return DeltaSetIndexMap{}, fmt.Errorf("mapCount %d exceeds limit %d", mapCount, MaxDeltaSetIndexEntries)
	}

	innerBitCount := int(entryFormat & 0x0F)
	entrySize := int(entryFormat>>4) + 1

	dataEnd := dataStart + mapCount*entrySize
	if dataEnd > len(data) {
		return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap data exceeds table")
	}

	innerMask := uint32((1 << innerBitCount) - 1)
	entries := make([]DeltaSetIndexEntry, mapCount)
	for i := 0; i < mapCount; i++ {
		pos := dataStart + i*entrySize
		var val uint32
		switch entrySize {
		case 1:
			val = uint32(data[pos])
		case 2:
			val = uint32(ReadU16(data, pos))
		case 3:
			val = uint32(data[pos])<<16 | uint32(ReadU16(data, pos+1))
		case 4:
			val = ReadU32(data, pos)
		default:
			return DeltaSetIndexMap{}, fmt.Errorf("unsupported entrySize %d", entrySize)
		}
		entries[i] = DeltaSetIndexEntry{
			OuterIndex: uint16(val >> innerBitCount),
			InnerIndex: uint16(val & innerMask),
		}
	}

	return DeltaSetIndexMap{
		Format:      format,
		EntryFormat: entryFormat,
		Entries:     entries,
	}, nil
}
