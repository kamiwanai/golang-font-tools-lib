package vvar

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

func putInt16(data []byte, offset int, v int16) {
	binary.BigEndian.PutUint16(data[offset:offset+2], uint16(v))
}

func buildMinimalVVAR() []byte {
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
	putInt16(data, 42, -30)
	data[44] = 0
	data[45] = 0x11
	putU16(data, 46, 3)
	putU16(data, 48, 0x0000)
	putU16(data, 50, 0x0002)
	putU16(data, 52, 0x0001)
	return data
}

func TestDecodeBasicVVAR(t *testing.T) {
	data := buildMinimalVVAR()
	vv, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if vv.MajorVersion != 1 {
		t.Fatalf("MajorVersion = %d, want 1", vv.MajorVersion)
	}
	if vv.MinorVersion != 0 {
		t.Fatalf("MinorVersion = %d, want 0", vv.MinorVersion)
	}
	ivs := vv.ItemVariationStore
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
	if ivd.DeltaSets[0][0] != -30 {
		t.Fatalf("delta[0][0] = %d, want -30", ivd.DeltaSets[0][0])
	}
	if vv.AdvanceHeightMapping == nil {
		t.Fatal("AdvanceHeightMapping should not be nil")
	}
	ahm := vv.AdvanceHeightMapping
	if ahm.Format != 0 {
		t.Fatalf("AHM format = %d, want 0", ahm.Format)
	}
	if len(ahm.Entries) != 3 {
		t.Fatalf("AHM entries = %d, want 3", len(ahm.Entries))
	}
}

func TestDecodeVVARTruncatedHeader(t *testing.T) {
	_, err := Decode(make([]byte, 4))
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestDecodeVVARBadVersion(t *testing.T) {
	data := make([]byte, 12)
	putU16(data, 0, 2)
	_, err := Decode(data)
	if err == nil {
		t.Fatal("expected error for bad version")
	}
}

func TestDecodeVVARVersion3AllMappings(t *testing.T) {
	data := make([]byte, 110)
	putU16(data, 0, 1)
	putU16(data, 2, 3)
	putU32(data, 4, 24)
	putU32(data, 8, 0)
	putU32(data, 12, 60)
	putU32(data, 16, 78)
	putU32(data, 20, 94)

	putU16(data, 24, 1)
	putU32(data, 26, 12)
	putU16(data, 30, 1)
	putU32(data, 32, 22)

	putU16(data, 36, 1)
	putU16(data, 38, 1)
	putF2DOT14(data, 40, -1.0)
	putF2DOT14(data, 42, 1.0)
	putF2DOT14(data, 44, 0.0)

	putU16(data, 46, 1)
	putU16(data, 48, 1)
	putU16(data, 50, 1)
	putU16(data, 52, 0)
	putU16(data, 54, 500)

	data[60] = 0
	data[61] = 0x11
	putU16(data, 62, 1)
	putU16(data, 64, 0x0000)

	data[78] = 0
	data[79] = 0x01
	putU16(data, 80, 2)
	data[82] = 0x01
	data[83] = 0x02

	data[94] = 0
	data[95] = 0x11
	putU16(data, 96, 1)
	putU16(data, 98, 0x0001)

	vv, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if vv.MinorVersion != 3 {
		t.Fatalf("MinorVersion = %d, want 3", vv.MinorVersion)
	}
	if vv.TsbMapping == nil {
		t.Fatal("TsbMapping should not be nil")
	}
	if len(vv.TsbMapping.Entries) != 1 {
		t.Fatalf("TsbMapping entries = %d, want 1", len(vv.TsbMapping.Entries))
	}
	if vv.BsbMapping == nil {
		t.Fatal("BsbMapping should not be nil")
	}
	if len(vv.BsbMapping.Entries) != 2 {
		t.Fatalf("BsbMapping entries = %d, want 2", len(vv.BsbMapping.Entries))
	}
	if vv.VOrgMapping == nil {
		t.Fatal("VOrgMapping should not be nil")
	}
	if len(vv.VOrgMapping.Entries) != 1 {
		t.Fatalf("VOrgMapping entries = %d, want 1", len(vv.VOrgMapping.Entries))
	}
}

func TestVVARFromRealFont(t *testing.T) {
	font, err := opentype.ParseFile("C:/Windows/Fonts/SegUIVar.ttf")
	if err != nil {
		t.Skipf("cannot open SegUIVar: %v", err)
	}
	vvarData, err := font.TableData("VVAR")
	if err != nil {
		t.Skipf("no VVAR: %v", err)
	}
	if len(vvarData) == 0 {
		t.Skip("VVAR data empty")
	}
	vv, err := Decode(vvarData)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	t.Logf("VVAR v%d.%d", vv.MajorVersion, vv.MinorVersion)
	t.Logf("VariationRegions: %d", len(vv.ItemVariationStore.VariationRegions))
	t.Logf("ItemVariationDatas: %d", len(vv.ItemVariationStore.ItemVariationDatas))
	if vv.AdvanceHeightMapping != nil {
		t.Logf("AdvanceHeightMapping: %d entries", len(vv.AdvanceHeightMapping.Entries))
	}
	if vv.TsbMapping != nil {
		t.Logf("TsbMapping: %d entries", len(vv.TsbMapping.Entries))
	}
	if vv.BsbMapping != nil {
		t.Logf("BsbMapping: %d entries", len(vv.BsbMapping.Entries))
	}
	if vv.VOrgMapping != nil {
		t.Logf("VOrgMapping: %d entries", len(vv.VOrgMapping.Entries))
	}
}

func FuzzVVARDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 12))
	f.Fuzz(func(t *testing.T, data []byte) {
		vv, err := Decode(data)
		if err != nil {
			return
		}
		_ = vv
	})
}
