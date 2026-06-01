// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package hvar

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

// HVAR holds the parsed HVAR (Horizontal Metrics Variations) table.
type HVAR struct {
	MajorVersion        uint16
	MinorVersion        uint16
	ItemVariationStore  ItemVariationStore
	AdvanceWidthMapping *DeltaSetIndexMap
	LsbMapping          *DeltaSetIndexMap
	RsbMapping          *DeltaSetIndexMap
}

// DecodeItemVariationStore decodes an ItemVariationStore from data at the given offset.
// This is the shared variation store format used by HVAR, VVAR, MVAR, and GDEF tables.
func DecodeItemVariationStore(data []byte, offset int) (ItemVariationStore, error) {
	return variation.DecodeItemVariationStore(data, offset)
}

// DecodeDeltaSetIndexMap decodes a DeltaSetIndexMap at the given offset.
func DecodeDeltaSetIndexMap(data []byte, offset int) (DeltaSetIndexMap, error) {
	return variation.DecodeDeltaSetIndexMap(data, offset)
}

// Decode parses an HVAR table from raw bytes.
func Decode(data []byte) (HVAR, error) {
	if len(data) < 12 {
		return HVAR{}, fmt.Errorf("HVAR header too short: %d bytes", len(data))
	}

	majorVersion := variation.ReadU16(data, 0)
	minorVersion := variation.ReadU16(data, 2)
	itemVarStoreOffset := variation.ReadU32(data, 4)
	advanceWidthMappingOffset := variation.ReadU32(data, 8)

	if majorVersion != 1 {
		return HVAR{}, fmt.Errorf("HVAR unsupported major version %d", majorVersion)
	}

	if itemVarStoreOffset == 0 {
		return HVAR{}, fmt.Errorf("HVAR ItemVariationStore offset is 0")
	}
	if int(itemVarStoreOffset)+4 > len(data) {
		return HVAR{}, fmt.Errorf("HVAR ItemVariationStore offset %d exceeds table", itemVarStoreOffset)
	}
	ivs, err := variation.DecodeItemVariationStore(data, int(itemVarStoreOffset))
	if err != nil {
		return HVAR{}, fmt.Errorf("HVAR ItemVariationStore: %w", err)
	}

	var awm *DeltaSetIndexMap
	if advanceWidthMappingOffset != 0 {
		if int(advanceWidthMappingOffset)+1 > len(data) {
			return HVAR{}, fmt.Errorf("HVAR AdvanceWidthMapping offset %d exceeds table", advanceWidthMappingOffset)
		}
		m, err := variation.DecodeDeltaSetIndexMap(data, int(advanceWidthMappingOffset))
		if err != nil {
			return HVAR{}, fmt.Errorf("HVAR AdvanceWidthMapping: %w", err)
		}
		awm = &m
	}

	var lsbMap *DeltaSetIndexMap
	if minorVersion >= 1 && len(data) >= 16 {
		lsbOffset := variation.ReadU32(data, 12)
		if lsbOffset != 0 {
			if int(lsbOffset)+1 > len(data) {
				return HVAR{}, fmt.Errorf("HVAR LsbMapping offset %d exceeds table", lsbOffset)
			}
			lm, err := variation.DecodeDeltaSetIndexMap(data, int(lsbOffset))
			if err != nil {
				return HVAR{}, fmt.Errorf("HVAR LsbMapping: %w", err)
			}
			lsbMap = &lm
		}
	}

	var rsbMap *DeltaSetIndexMap
	if minorVersion >= 2 && len(data) >= 20 {
		rsbOffset := variation.ReadU32(data, 16)
		if rsbOffset != 0 {
			if int(rsbOffset)+1 > len(data) {
				return HVAR{}, fmt.Errorf("HVAR RsbMapping offset %d exceeds table", rsbOffset)
			}
			rm, err := variation.DecodeDeltaSetIndexMap(data, int(rsbOffset))
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
