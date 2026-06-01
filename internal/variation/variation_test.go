// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package variation

import (
	"encoding/binary"
	"testing"
)

func putU16(data []byte, offset int, v uint16) {
	binary.BigEndian.PutUint16(data[offset:offset+2], v)
}

func putU32(data []byte, offset int, v uint32) {
	binary.BigEndian.PutUint32(data[offset:offset+4], v)
}

func putF2DOT14(data []byte, offset int, v float64) {
	putU16(data, offset, uint16(int16(v*16384)))
}

// Build a minimal IVS with 1 region, 1 axis, 1 IVD, 1 item, 1 word delta.
// Layout:
//   0-1:  format = 1
//   2-5:  varRegionListOffset = 12
//   6-7:  itemVarDataCount = 1
//   8-11: ivdOffsets[0] = 22
//   12-13: axisCount = 1
//   14-15: regionCount = 1
//   16-17: startCoord = -1.0
//   18-19: peakCoord = 1.0
//   20-21: endCoord = 0.0
//   22-23: itemCount = 1
//   24-25: wordDeltaCount = 1
//   26-27: regionIdxCount = 1
//   28-29: regionIndexes[0] = 0
//   30-31: delta[0] = 42 (signed)
func buildMinimalIVS() []byte {
	data := make([]byte, 32)
	putU16(data, 0, 1)
	putU32(data, 2, 12)
	putU16(data, 6, 1)
	putU32(data, 8, 22)
	putU16(data, 12, 1)
	putU16(data, 14, 1)
	putF2DOT14(data, 16, -1.0)
	putF2DOT14(data, 18, 1.0)
	putF2DOT14(data, 20, 0.0)
	putU16(data, 22, 1)
	putU16(data, 24, 1)
	putU16(data, 26, 1)
	putU16(data, 28, 0)
	putU16(data, 30, 42)
	return data
}

func TestDecodeItemVariationStore(t *testing.T) {
	data := buildMinimalIVS()
	ivs, err := DecodeItemVariationStore(data, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ivs.Format != 1 {
		t.Fatalf("format = %d, want 1", ivs.Format)
	}
	if len(ivs.VariationRegions) != 1 {
		t.Fatalf("regions = %d, want 1", len(ivs.VariationRegions))
	}
	if len(ivs.VariationRegions[0].RegionAxes) != 1 {
		t.Fatalf("axes = %d, want 1", len(ivs.VariationRegions[0].RegionAxes))
	}
	if len(ivs.ItemVariationDatas) != 1 {
		t.Fatalf("ivds = %d, want 1", len(ivs.ItemVariationDatas))
	}
	ivd := ivs.ItemVariationDatas[0]
	if ivd.ItemCount != 1 {
		t.Fatalf("itemCount = %d, want 1", ivd.ItemCount)
	}
	if ivd.DeltaSets[0][0] != 42 {
		t.Fatalf("delta = %d, want 42", ivd.DeltaSets[0][0])
	}
}

func TestDecodeItemVariationStoreTruncatedHeader(t *testing.T) {
	if _, err := DecodeItemVariationStore([]byte{0, 1, 0, 0}, 0); err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestDecodeItemVariationStoreUnsupportedFormat(t *testing.T) {
	data := make([]byte, 8)
	putU16(data, 0, 99) // format 99
	putU32(data, 2, 0)
	putU16(data, 6, 0)
	if _, err := DecodeItemVariationStore(data, 0); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}

func TestDeltaSetIndexMapFormat0(t *testing.T) {
	// Format 0, 2 entries, entryFormat=0x01 (innerBitCount=1, entrySize=1)
	data := []byte{0, 0x01, 0, 2, 0, 0}
	m, err := DecodeDeltaSetIndexMap(data, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if m.Format != 0 {
		t.Fatalf("format = %d, want 0", m.Format)
	}
	if len(m.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(m.Entries))
	}
}

func TestDeltaSetIndexMapFormat1(t *testing.T) {
	// Format 1, 1 entry, entryFormat=0x21 (innerBitCount=1, entrySize=3)
	data := make([]byte, 12)
	data[0] = 1
	data[1] = 0x21
	putU32(data, 2, 1)
	data[6] = 0
	data[7] = 0
	data[8] = 0x03
	m, err := DecodeDeltaSetIndexMap(data, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if m.Format != 1 {
		t.Fatalf("format = %d, want 1", m.Format)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(m.Entries))
	}
	if m.Entries[0].OuterIndex != 1 || m.Entries[0].InnerIndex != 1 {
		t.Fatalf("entry = (%d, %d), want (1, 1)", m.Entries[0].OuterIndex, m.Entries[0].InnerIndex)
	}
}

func TestF2DOT14Float(t *testing.T) {
	cases := []struct {
		in   F2DOT14
		want float64
	}{
		{0, 0.0},
		{0x4000, 1.0},
		{-0x4000, -1.0},
	}
	for _, c := range cases {
		got := c.in.Float()
		diff := got - c.want
		if diff < -1e-6 || diff > 1e-6 {
			t.Errorf("F2DOT14(%d).Float() = %f, want %f", c.in, got, c.want)
		}
	}
}
