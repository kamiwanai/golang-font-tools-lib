// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package mvar

import (
	"encoding/binary"
	"fmt"

	"github.com/kamiwanai/fonttools/hvar"
)

// ---------- Value Tag Constants ----------

// Metric value tags as defined in OpenType 1.9.1.
const (
	TagHasc = "hasc" // Horizontal ascender
	TagHdsc = "hdsc" // Horizontal descender
	TagHlgp = "hlgp" // Horizontal line gap
	TagHcla = "hcla" // Horizontal clipping ascent
	TagHcld = "hcld" // Horizontal clipping descent
	TagVasc = "vasc" // Vertical ascender
	TagVdsc = "vdsc" // Vertical descender
	TagVlgp = "vlgp" // Vertical line gap
	TagHcrs = "hcrs" // Horizontal caret rise
	TagHcrn = "hcrn" // Horizontal caret run
	TagHcof = "hcof" // Horizontal caret offset
	TagVcrs = "vcrs" // Vertical caret rise
	TagVcrn = "vcrn" // Vertical caret run
	TagVcof = "vcof" // Vertical caret offset
	TagXhgt = "xhgt" // X-height
	TagCpht = "cpht" // Cap height
	TagSbxo = "sbxo" // Subscript em x-offset
	TagSbyo = "sbyo" // Subscript em y-offset
	TagSbxs = "sbxs" // Subscript em x-size
	TagSbys = "sbys" // Subscript em y-size
	TagSpxo = "spxo" // Superscript em x-offset
	TagSpyo = "spyo" // Superscript em y-offset
	TagSpxs = "spxs" // Superscript em x-size
	TagSpys = "spys" // Superscript em y-size
	TagStrs = "strs" // Strikeout size
	TagStro = "stro" // Strikeout offset
	TagUnds = "unds" // Underline size
	TagUndo = "undo" // Underline offset
	TagGsp0 = "gsp0" // GaspRange[0]
	TagGsp1 = "gsp1" // GaspRange[1]
	TagGsp2 = "gsp2" // GaspRange[2]
	TagGsp3 = "gsp3" // GaspRange[3]
	TagGsp4 = "gsp4" // GaspRange[4]
	TagGsp5 = "gsp5" // GaspRange[5]
	TagGsp6 = "gsp6" // GaspRange[6]
	TagGsp7 = "gsp7" // GaspRange[7]
	TagGsp8 = "gsp8" // GaspRange[8]
	TagGsp9 = "gsp9" // GaspRange[9]
)

// ---------- Types ----------

// ValueRecord identifies a font-wide metric and its delta-set index
// within the ItemVariationStore.
type ValueRecord struct {
	ValueTag      string // 4-byte tag (e.g., "hasc", "hdsc")
	DeltaSetOuter uint16 // Outer index into ItemVariationData subtable
	DeltaSetInner uint16 // Inner index into delta-set row
}

// MVAR holds the parsed MVAR (Metrics Variations) table.
type MVAR struct {
	MajorVersion       uint16
	MinorVersion       uint16
	ValueRecords       []ValueRecord
	ItemVariationStore hvar.ItemVariationStore
}

// ---------- Helpers ----------

func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readTag(data []byte, offset int) string {
	return string(data[offset : offset+4])
}

// ---------- Decode ----------

// Decode parses an MVAR table from raw bytes.
func Decode(data []byte) (MVAR, error) {
	if len(data) < 12 {
		return MVAR{}, fmt.Errorf("MVAR header too short: %d bytes", len(data))
	}

	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)
	// offset 4-5: reserved
	valueRecordSize := readU16(data, 6)
	valueRecordCount := readU16(data, 8)
	itemVarStoreOffset := readU16(data, 10)

	if majorVersion != 1 {
		return MVAR{}, fmt.Errorf("MVAR unsupported major version %d", majorVersion)
	}

	if valueRecordCount > uint16(MaxValueRecords) {
		return MVAR{}, fmt.Errorf("MVAR valueRecordCount %d exceeds limit %d", valueRecordCount, MaxValueRecords)
	}

	// Parse value records
	recordsStart := 12
	recordsEnd := recordsStart + int(valueRecordCount)*int(valueRecordSize)
	if recordsEnd > len(data) {
		return MVAR{}, fmt.Errorf("MVAR value records exceed table")
	}

	records := make([]ValueRecord, valueRecordCount)
	for i := 0; i < int(valueRecordCount); i++ {
		pos := recordsStart + i*int(valueRecordSize)
		if pos+8 > len(data) {
			return MVAR{}, fmt.Errorf("MVAR value record %d truncated", i)
		}
		records[i] = ValueRecord{
			ValueTag:      readTag(data, pos),
			DeltaSetOuter: readU16(data, pos+4),
			DeltaSetInner: readU16(data, pos+6),
		}
	}

	// Parse ItemVariationStore
	if valueRecordCount > 0 && itemVarStoreOffset == 0 {
		return MVAR{}, fmt.Errorf("MVAR has %d value records but ItemVariationStore offset is 0", valueRecordCount)
	}

	var ivs hvar.ItemVariationStore
	if itemVarStoreOffset != 0 {
		var err error
		ivs, err = hvar.DecodeItemVariationStore(data, int(itemVarStoreOffset))
		if err != nil {
			return MVAR{}, fmt.Errorf("MVAR ItemVariationStore: %w", err)
		}
	}

	return MVAR{
		MajorVersion:       majorVersion,
		MinorVersion:       minorVersion,
		ValueRecords:       records,
		ItemVariationStore: ivs,
	}, nil
}

// ---------- Lookup ----------

// RecordByTag returns the ValueRecord for the given tag, or nil if not found.
// Tags are sorted alphabetically per the spec; binary search is used.
func (mv *MVAR) RecordByTag(tag string) *ValueRecord {
	lo, hi := 0, len(mv.ValueRecords)
	for lo < hi {
		mid := (lo + hi) / 2
		rec := &mv.ValueRecords[mid]
		if rec.ValueTag == tag {
			return rec
		}
		if rec.ValueTag < tag {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return nil
}

// HasTag returns true if the MVAR table has a value record for the given tag.
func (mv *MVAR) HasTag(tag string) bool {
	return mv.RecordByTag(tag) != nil
}
