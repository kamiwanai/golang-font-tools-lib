package hvar

import (
	"encoding/binary"
	"fmt"
)

// ---------- F2DOT14 ----------

// F2DOT14 is a signed 16-bit fixed-point number with 14 fraction bits.
type F2DOT14 int16

// Float returns the float64 representation of this F2DOT14.
func (f F2DOT14) Float() float64 {
	return float64(f) / 16384.0
}

// ---------- RegionAxis ----------

// RegionAxis defines one axis range for a variation region.
type RegionAxis struct {
	StartCoord F2DOT14
	PeakCoord  F2DOT14
	EndCoord   F2DOT14
}

// ---------- VariationRegion ----------

// VariationRegion defines one region of applicability for a delta set.
type VariationRegion struct {
	RegionAxes []RegionAxis
}

// ---------- ItemVariationData ----------

// ItemVariationData holds delta data for a set of items across multiple regions.
type ItemVariationData struct {
	ItemCount         uint16
	WordDeltaCount     uint16  // number of 16-bit deltas (remainder are 8-bit)
	RegionIndexes      []uint16
	DeltaSets          [][]int32  // itemCount x len(RegionIndexes)
}

// ---------- ItemVariationStore ----------

// ItemVariationStore stores variation delta data shared across tables (HVAR, VVAR, MVAR, GDEF).
type ItemVariationStore struct {
	Format              uint16
	VariationRegions    []VariationRegion
	ItemVariationDatas  []ItemVariationData
}

// ---------- DeltaSetIndexMap ----------

// DeltaSetIndexEntry is one entry in a DeltaSetIndexMap.
// It maps a glyph (or other item) to a delta-set (outer, inner) pair.
type DeltaSetIndexEntry struct {
	OuterIndex uint16
	InnerIndex uint16
}

// DeltaSetIndexMap is an indirect mapping from item indices to delta-set indices.
// Supports format 0 (uint16 mapCount) and format 1 (uint32 mapCount).
type DeltaSetIndexMap struct {
	Format      uint8   // 0 or 1
	EntryFormat uint8   // packed: innerBitCount (low nibble) + entrySize (high nibble)
	Entries     []DeltaSetIndexEntry
}

// ---------- HVAR ----------

// HVAR holds the parsed HVAR (Horizontal Metrics Variations) table.
type HVAR struct {
	MajorVersion        uint16
	MinorVersion        uint16
	ItemVariationStore  ItemVariationStore
	AdvanceWidthMapping *DeltaSetIndexMap  // nil if offset was 0
	LsbMapping          *DeltaSetIndexMap  // nil if absent (minorVersion < 1 or offset 0)
	RsbMapping          *DeltaSetIndexMap  // nil if absent (minorVersion < 2 or offset 0)
}

// ---------- Helpers ----------

func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}

// ---------- Decode ----------

// Decode parses an HVAR table from raw bytes.
func Decode(data []byte) (HVAR, error) {
	if len(data) < 8 {
		return HVAR{}, fmt.Errorf("HVAR header too short: %d bytes", len(data))
	}

	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)
	itemVarStoreOffset := readU32(data, 4)
	advanceWidthMappingOffset := readU32(data, 8)

	if majorVersion != 1 {
		return HVAR{}, fmt.Errorf("HVAR unsupported major version %d", majorVersion)
	}

	// Parse ItemVariationStore
	if itemVarStoreOffset == 0 {
		return HVAR{}, fmt.Errorf("HVAR ItemVariationStore offset is 0")
	}
	if int(itemVarStoreOffset)+4 > len(data) {
		return HVAR{}, fmt.Errorf("HVAR ItemVariationStore offset %d exceeds table", itemVarStoreOffset)
	}
	ivs, err := decodeItemVariationStore(data, int(itemVarStoreOffset))
	if err != nil {
		return HVAR{}, fmt.Errorf("HVAR ItemVariationStore: %w", err)
	}

	// Parse AdvanceWidthMapping
	var awm *DeltaSetIndexMap
	if advanceWidthMappingOffset != 0 {
		if int(advanceWidthMappingOffset)+1 > len(data) {
			return HVAR{}, fmt.Errorf("HVAR AdvanceWidthMapping offset %d exceeds table", advanceWidthMappingOffset)
		}
		m, err := decodeDeltaSetIndexMap(data, int(advanceWidthMappingOffset))
		if err != nil {
			return HVAR{}, fmt.Errorf("HVAR AdvanceWidthMapping: %w", err)
		}
		awm = &m
	}

	// Parse optional LsbMapping (minorVersion >= 1)
	var lsbMap *DeltaSetIndexMap
	if minorVersion >= 1 && len(data) >= 16 {
		lsbOffset := readU32(data, 12)
		if lsbOffset != 0 {
			if int(lsbOffset)+1 > len(data) {
				return HVAR{}, fmt.Errorf("HVAR LsbMapping offset %d exceeds table", lsbOffset)
			}
			lm, err := decodeDeltaSetIndexMap(data, int(lsbOffset))
			if err != nil {
				return HVAR{}, fmt.Errorf("HVAR LsbMapping: %w", err)
			}
			lsbMap = &lm
		}
	}

	// Parse optional RsbMapping (minorVersion >= 2)
	var rsbMap *DeltaSetIndexMap
	if minorVersion >= 2 && len(data) >= 20 {
		rsbOffset := readU32(data, 16)
		if rsbOffset != 0 {
			if int(rsbOffset)+1 > len(data) {
				return HVAR{}, fmt.Errorf("HVAR RsbMapping offset %d exceeds table", rsbOffset)
			}
			rm, err := decodeDeltaSetIndexMap(data, int(rsbOffset))
			if err != nil {
				return HVAR{}, fmt.Errorf("HVAR RsbMapping: %w", err)
			}
			rsbMap = &rm
		}
	}

	return HVAR{
		MajorVersion:        majorVersion,
		MinorVersion:        minorVersion,
		ItemVariationStore:  ivs,
		AdvanceWidthMapping: awm,
		LsbMapping:          lsbMap,
		RsbMapping:          rsbMap,
	}, nil
}