package hvar

import (
	"encoding/binary"
	"testing"

	"github.com/kamiwanai/fonttools/opentype"
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

func buildMinimalHVAR() []byte {
	data := make([]byte, 54)
	putU16(data, 0, 1)
	putU16(data, 2, 0)
	putU32(data, 4, 12)
	putU32(data, 8, 44)
	putU16(data, 12, 1)
	putU32(data, 14, 12)
	putU16(data, 18, 1)
	putU32(data, 20, 22)
	putU16(data, 24, 1)
	putU16(data, 26, 1)
	putF2DOT14(data, 28, -1.0)
	putF2DOT14(data, 30, 1.0)
	putF2DOT14(data, 32, 0.0)
	putU16(data, 34, 1)
	putU16(data, 36, 1)
	putU16(data, 38, 1)
	putU16(data, 40, 0)
	putU16(data, 42, 50)
	data[44] = 0
	data[45] = 0x11
	putU16(data, 46, 3)
	putU16(data, 48, 0x0000)
	putU16(data, 50, 0x0002)
	putU16(data, 52, 0x0001)
	return data
}

func TestDecodeBasicHVAR(t *testing.T) {
	data := buildMinimalHVAR()
	hv, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if hv.MajorVersion != 1 {
		t.Fatalf("MajorVersion = %d, want 1", hv.MajorVersion)
	}
	if hv.MinorVersion != 0 {
		t.Fatalf("MinorVersion = %d, want 0", hv.MinorVersion)
	}
	ivs := hv.ItemVariationStore
	if len(ivs.VariationRegions) != 1 {
		t.Fatalf("regions = %d, want 1", len(ivs.VariationRegions))
	}
	if len(ivs.ItemVariationDatas) != 1 {
		t.Fatalf("ivds = %d, want 1", len(ivs.ItemVariationDatas))
	}
	ivd := ivs.ItemVariationDatas[0]
	if ivd.ItemCount != 1 {
		t.Fatalf("itemCount = %d, want 1", ivd.ItemCount)
	}
	if len(ivd.DeltaSets) != 1 {
		t.Fatalf("deltaSets = %d, want 1", len(ivd.DeltaSets))
	}
	if ivd.DeltaSets[0][0] != 50 {
		t.Fatalf("delta[0][0] = %d, want 50", ivd.DeltaSets[0][0])
	}
	if hv.AdvanceWidthMapping == nil {
		t.Fatal("AdvanceWidthMapping should not be nil")
	}
	awm := hv.AdvanceWidthMapping
	if awm.Format != 0 {
		t.Fatalf("AWM format = %d, want 0", awm.Format)
	}
	if len(awm.Entries) != 3 {
		t.Fatalf("AWM entries = %d, want 3", len(awm.Entries))
	}
	if awm.Entries[0].OuterIndex != 0 || awm.Entries[0].InnerIndex != 0 {
		t.Fatalf("AWM[0] = (%d, %d), want (0, 0)", awm.Entries[0].OuterIndex, awm.Entries[0].InnerIndex)
	}
}

func TestDecodeHVARTruncatedHeader(t *testing.T) {
	_, err := Decode(make([]byte, 4))
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestDecodeHVARBadVersion(t *testing.T) {
	data := make([]byte, 12)
	putU16(data, 0, 2)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
}

func TestDeltaSetIndexMapFormat1(t *testing.T) {
	data := make([]byte, 12)
	data[0] = 1
	data[1] = 0x21
	putU32(data, 2, 1)
	data[6] = 0
	data[7] = 0
	data[8] = 0x03
	m, err := decodeDeltaSetIndexMap(data, 0)
	if err != nil {
		t.Fatalf("decodeDeltaSetIndexMap error: %v", err)
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

func TestHVARFromRealFont(t *testing.T) {
	font, err := opentype.ParseFile("C:/Windows/Fonts/SegUIVar.ttf")
	if err != nil {
		t.Skipf("cannot open SegUIVar: %v", err)
	}
	hvarData, err := font.TableData("HVAR")
	if err != nil {
		t.Skipf("no HVAR: %v", err)
	}
	if len(hvarData) == 0 {
		t.Skip("HVAR data empty")
	}
	hv, err := Decode(hvarData)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	t.Logf("HVAR v%d.%d", hv.MajorVersion, hv.MinorVersion)
	t.Logf("VariationRegions: %d", len(hv.ItemVariationStore.VariationRegions))
	t.Logf("ItemVariationDatas: %d", len(hv.ItemVariationStore.ItemVariationDatas))
	if hv.AdvanceWidthMapping != nil {
		t.Logf("AdvanceWidthMapping: %d entries", len(hv.AdvanceWidthMapping.Entries))
	}
	if hv.LsbMapping != nil {
		t.Logf("LsbMapping: %d entries", len(hv.LsbMapping.Entries))
	}
	if hv.RsbMapping != nil {
		t.Logf("RsbMapping: %d entries", len(hv.RsbMapping.Entries))
	}
}

func FuzzHVARDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 12))
	f.Fuzz(func(t *testing.T, data []byte) {
		hv, err := Decode(data)
		if err != nil {
			return
		}
		_ = hv
	})
}
