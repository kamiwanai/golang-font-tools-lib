// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package vvar

import (
	"fmt"

	"github.com/kamiwanai/golang-font-tools-lib/internal/variation"
)

// Public type aliases. Canonical definitions live in internal/variation.

type F2DOT14 = variation.F2DOT14
type RegionAxis = variation.RegionAxis
type VariationRegion = variation.VariationRegion
type ItemVariationData = variation.ItemVariationData
type ItemVariationStore = variation.ItemVariationStore
type DeltaSetIndexEntry = variation.DeltaSetIndexEntry
type DeltaSetIndexMap = variation.DeltaSetIndexMap

// VVAR holds the parsed VVAR (Vertical Metrics Variations) table.
type VVAR struct {
	MajorVersion         uint16
	MinorVersion         uint16
	ItemVariationStore   ItemVariationStore
	AdvanceHeightMapping *DeltaSetIndexMap
	TsbMapping           *DeltaSetIndexMap
	BsbMapping           *DeltaSetIndexMap
	VOrgMapping          *DeltaSetIndexMap
}

// DecodeItemVariationStore decodes an ItemVariationStore from data at the given offset.
func DecodeItemVariationStore(data []byte, offset int) (ItemVariationStore, error) {
	return variation.DecodeItemVariationStore(data, offset)
}

// Decode parses a VVAR table from raw bytes.
func Decode(data []byte) (VVAR, error) {
	if len(data) < 8 {
		return VVAR{}, fmt.Errorf("VVAR header too short: %d bytes", len(data))
	}

	majorVersion := variation.ReadU16(data, 0)
	minorVersion := variation.ReadU16(data, 2)
	itemVarStoreOffset := variation.ReadU32(data, 4)
	advanceHeightMappingOffset := variation.ReadU32(data, 8)

	if majorVersion != 1 {
		return VVAR{}, fmt.Errorf("VVAR unsupported major version %d", majorVersion)
	}

	if itemVarStoreOffset == 0 {
		return VVAR{}, fmt.Errorf("VVAR ItemVariationStore offset is 0")
	}
	if int(itemVarStoreOffset)+4 > len(data) {
		return VVAR{}, fmt.Errorf("VVAR ItemVariationStore offset %d exceeds table", itemVarStoreOffset)
	}
	ivs, err := variation.DecodeItemVariationStore(data, int(itemVarStoreOffset))
	if err != nil {
		return VVAR{}, fmt.Errorf("VVAR ItemVariationStore: %w", err)
	}

	var ahm *DeltaSetIndexMap
	if advanceHeightMappingOffset != 0 {
		if int(advanceHeightMappingOffset)+1 > len(data) {
			return VVAR{}, fmt.Errorf("VVAR AdvanceHeightMapping offset exceeds table")
		}
		m, err := variation.DecodeDeltaSetIndexMap(data, int(advanceHeightMappingOffset))
		if err != nil {
			return VVAR{}, fmt.Errorf("VVAR AdvanceHeightMapping: %w", err)
		}
		ahm = &m
	}

	var tsbMap *DeltaSetIndexMap
	if minorVersion >= 1 && len(data) >= 16 {
		tsbOffset := variation.ReadU32(data, 12)
		if tsbOffset != 0 {
			if int(tsbOffset)+1 > len(data) {
				return VVAR{}, fmt.Errorf("VVAR TsbMapping offset exceeds table")
			}
			tm, err := variation.DecodeDeltaSetIndexMap(data, int(tsbOffset))
			if err != nil {
				return VVAR{}, fmt.Errorf("VVAR TsbMapping: %w", err)
			}
			tsbMap = &tm
		}
	}

	var bsbMap *DeltaSetIndexMap
	if minorVersion >= 2 && len(data) >= 20 {
		bsbOffset := variation.ReadU32(data, 16)
		if bsbOffset != 0 {
			if int(bsbOffset)+1 > len(data) {
				return VVAR{}, fmt.Errorf("VVAR BsbMapping offset exceeds table")
			}
			bm, err := variation.DecodeDeltaSetIndexMap(data, int(bsbOffset))
			if err != nil {
				return VVAR{}, fmt.Errorf("VVAR BsbMapping: %w", err)
			}
			bsbMap = &bm
		}
	}

	var vorgMap *DeltaSetIndexMap
	if minorVersion >= 3 && len(data) >= 24 {
		vorgOffset := variation.ReadU32(data, 20)
		if vorgOffset != 0 {
			if int(vorgOffset)+1 > len(data) {
				return VVAR{}, fmt.Errorf("VVAR VOrgMapping offset exceeds table")
			}
			vm, err := variation.DecodeDeltaSetIndexMap(data, int(vorgOffset))
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
