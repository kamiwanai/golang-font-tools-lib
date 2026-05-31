// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package vvar

import "fmt"

func decodeItemVariationStore(data []byte, offset int) (ItemVariationStore, error) {
	if offset+8 > len(data) {
		return ItemVariationStore{}, fmt.Errorf("ItemVariationStore header too short")
	}
	format := readU16(data, offset)
	if format != 1 {
		return ItemVariationStore{}, fmt.Errorf("ItemVariationStore unsupported format %d", format)
	}
	varRegionListOffset := int(readU32(data, offset+2))
	itemVarDataCount := int(readU16(data, offset+6))

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
		ivdOff := int(readU32(data, ivdOffsetsStart+i*4))
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
	axisCount := int(readU16(data, offset))
	regionCount := int(readU16(data, offset+2))

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
				StartCoord: F2DOT14(int16(readU16(data, pos))),
				PeakCoord:  F2DOT14(int16(readU16(data, pos+2))),
				EndCoord:   F2DOT14(int16(readU16(data, pos+4))),
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
	itemCount := int(readU16(data, offset))
	wordDeltaCount := int(readU16(data, offset+2))
	regionIdxCount := int(readU16(data, offset+4))

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
		regionIndexes[i] = readU16(data, regionIdxStart+i*2)
	}

	bytesPerItem := wordDeltaCount*2 + (regionIdxCount - wordDeltaCount)*1
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
			deltas[j] = int32(int16(readU16(data, pos)))
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