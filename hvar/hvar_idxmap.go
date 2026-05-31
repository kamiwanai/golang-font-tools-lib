// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package hvar

import "fmt"

// ---------- DeltaSetIndexMap decoder ----------

func decodeDeltaSetIndexMap(data []byte, offset int) (DeltaSetIndexMap, error) {
	if offset+1 > len(data) {
		return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap too short")
	}
	format := data[offset]
	if format != 0 && format != 1 {
		return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap unsupported format %d", format)
	}

	if offset+3 > len(data) {
		return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap header too short")
	}
	entryFormat := data[offset+1]

	var mapCount, dataStart int
	if format == 0 {
		mapCount = int(readU16(data, offset+2))
		dataStart = offset + 4
	} else {
		if offset+6 > len(data) {
			return DeltaSetIndexMap{}, fmt.Errorf("DeltaSetIndexMap format 1 header too short")
		}
		mapCount = int(readU32(data, offset+2))
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
			val = uint32(readU16(data, pos))
		case 3:
			val = uint32(data[pos])<<16 | uint32(readU16(data, pos+1))
		case 4:
			val = readU32(data, pos)
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