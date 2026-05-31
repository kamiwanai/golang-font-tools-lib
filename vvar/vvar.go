package vvar

import (
	"encoding/binary"
	"fmt"
)

// ---------- F2DOT14 ----------

type F2DOT14 int16

func (f F2DOT14) Float() float64 {
	return float64(f) / 16384.0
}

// ---------- RegionAxis / VariationRegion ----------

type RegionAxis struct {
	StartCoord F2DOT14
	PeakCoord  F2DOT14
	EndCoord   F2DOT14
}

type VariationRegion struct {
	RegionAxes []RegionAxis
}

// ---------- ItemVariationData / ItemVariationStore ----------

type ItemVariationData struct {
	ItemCount      uint16
	WordDeltaCount uint16
	RegionIndexes  []uint16
	DeltaSets      [][]int32
}

type ItemVariationStore struct {
	Format             uint16
	VariationRegions   []VariationRegion
	ItemVariationDatas []ItemVariationData
}

// ---------- DeltaSetIndexMap ----------

type DeltaSetIndexEntry struct {
	OuterIndex uint16
	InnerIndex uint16
}

type DeltaSetIndexMap struct {
	Format      uint8
	EntryFormat uint8
	Entries     []DeltaSetIndexEntry
}

// ---------- VVAR ----------

// VVAR holds the parsed VVAR (Vertical Metrics Variations) table.
type VVAR struct {
	MajorVersion         uint16
	MinorVersion         uint16
	ItemVariationStore   ItemVariationStore
	AdvanceHeightMapping *DeltaSetIndexMap  // nil if offset was 0
	TsbMapping           *DeltaSetIndexMap  // nil if absent
	BsbMapping           *DeltaSetIndexMap  // nil if absent
	VOrgMapping          *DeltaSetIndexMap  // nil if absent
}

// ---------- Helpers ----------

func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}

// ---------- Decode ----------

// Decode parses a VVAR table from raw bytes.
func Decode(data []byte) (VVAR, error) {
	if len(data) < 8 {
		return VVAR{}, fmt.Errorf("VVAR header too short: %d bytes", len(data))
	}

	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)
	itemVarStoreOffset := readU32(data, 4)
	advanceHeightMappingOffset := readU32(data, 8)

	if majorVersion != 1 {
		return VVAR{}, fmt.Errorf("VVAR unsupported major version %d", majorVersion)
	}

	if itemVarStoreOffset == 0 {
		return VVAR{}, fmt.Errorf("VVAR ItemVariationStore offset is 0")
	}
	if int(itemVarStoreOffset)+4 > len(data) {
		return VVAR{}, fmt.Errorf("VVAR ItemVariationStore offset %d exceeds table", itemVarStoreOffset)
	}
	ivs, err := decodeItemVariationStore(data, int(itemVarStoreOffset))
	if err != nil {
		return VVAR{}, fmt.Errorf("VVAR ItemVariationStore: %w", err)
	}

	var ahm *DeltaSetIndexMap
	if advanceHeightMappingOffset != 0 {
		if int(advanceHeightMappingOffset)+1 > len(data) {
			return VVAR{}, fmt.Errorf("VVAR AdvanceHeightMapping offset exceeds table")
		}
		m, err := decodeDeltaSetIndexMap(data, int(advanceHeightMappingOffset))
		if err != nil {
			return VVAR{}, fmt.Errorf("VVAR AdvanceHeightMapping: %w", err)
		}
		ahm = &m
	}

	var tsbMap *DeltaSetIndexMap
	if minorVersion >= 1 && len(data) >= 16 {
		tsbOffset := readU32(data, 12)
		if tsbOffset != 0 {
			if int(tsbOffset)+1 > len(data) {
				return VVAR{}, fmt.Errorf("VVAR TsbMapping offset exceeds table")
			}
			tm, err := decodeDeltaSetIndexMap(data, int(tsbOffset))
			if err != nil {
				return VVAR{}, fmt.Errorf("VVAR TsbMapping: %w", err)
			}
			tsbMap = &tm
		}
	}

	var bsbMap *DeltaSetIndexMap
	if minorVersion >= 2 && len(data) >= 20 {
		bsbOffset := readU32(data, 16)
		if bsbOffset != 0 {
			if int(bsbOffset)+1 > len(data) {
				return VVAR{}, fmt.Errorf("VVAR BsbMapping offset exceeds table")
			}
			bm, err := decodeDeltaSetIndexMap(data, int(bsbOffset))
			if err != nil {
				return VVAR{}, fmt.Errorf("VVAR BsbMapping: %w", err)
			}
			bsbMap = &bm
		}
	}

	var vorgMap *DeltaSetIndexMap
	if minorVersion >= 3 && len(data) >= 24 {
		vorgOffset := readU32(data, 20)
		if vorgOffset != 0 {
			if int(vorgOffset)+1 > len(data) {
				return VVAR{}, fmt.Errorf("VVAR VOrgMapping offset exceeds table")
			}
			vm, err := decodeDeltaSetIndexMap(data, int(vorgOffset))
			if err != nil {
				return VVAR{}, fmt.Errorf("VVAR VOrgMapping: %w", err)
			}
			vorgMap = &vm
		}
	}

	return VVAR{
		MajorVersion:         majorVersion,
		MinorVersion:         minorVersion,
		ItemVariationStore:   ivs,
		AdvanceHeightMapping: ahm,
		TsbMapping:           tsbMap,
		BsbMapping:           bsbMap,
		VOrgMapping:          vorgMap,
	}, nil
}