package gsub

import (
	"encoding/binary"
	"testing"
)

func buildGSUBTable(lookupType uint16, lookupData []byte) []byte {
	header := make([]byte, 10)
	binary.BigEndian.PutUint32(header[0:4], 0x00010000)
	binary.BigEndian.PutUint16(header[4:6], 0)
	binary.BigEndian.PutUint16(header[6:8], 0)
	binary.BigEndian.PutUint16(header[8:10], 10)

	lookupHeader := make([]byte, 6)
	binary.BigEndian.PutUint16(lookupHeader[0:2], lookupType)
	binary.BigEndian.PutUint16(lookupHeader[2:4], 0)
	binary.BigEndian.PutUint16(lookupHeader[4:6], 1)

	subOff := uint16(6 + 2)
	subOffBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(subOffBytes[0:2], subOff)

	lookupOff := uint16(4)
	lookupList := make([]byte, 2+2)
	binary.BigEndian.PutUint16(lookupList[0:2], 1)
	binary.BigEndian.PutUint16(lookupList[2:4], lookupOff)

	result := append(header, lookupList...)
	result = append(result, lookupHeader...)
	result = append(result, subOffBytes...)
	result = append(result, lookupData...)
	return result
}

func TestDecodeContextSubstFormat1(t *testing.T) {
	coverage := []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x05}
	// SubRule: GlyphCount=2, SubstCount=1, Input[0]=10, Lookup(seq=0,idx=3)
	subRule := []byte{
		0x00, 0x02, // glyphCount
		0x00, 0x01, // substCount
		0x00, 0x0A, // input[0] = glyph 10
		0x00, 0x00, 0x00, 0x03, // seq=0, idx=3
	}
	// SubRuleSet: count=1, offset=4 (past count+offset header)
	subRuleSet := []byte{0x00, 0x01, 0x00, 0x04}
	subRuleSet = append(subRuleSet, subRule...)

	// ContextSubst Fmt1: format + coverageOff + ruleSetCount + ruleSetOff
	covOff := uint16(2 + 2 + 2 + 2)       // 8
	rsOff := covOff + uint16(len(coverage)) // 8+6=14
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], 1)
	binary.BigEndian.PutUint16(header[2:4], covOff)
	binary.BigEndian.PutUint16(header[4:6], 1)
	binary.BigEndian.PutUint16(header[6:8], rsOff)

	lookupData := append(header, coverage...)
	lookupData = append(lookupData, subRuleSet...)

	data := buildGSUBTable(5, lookupData)
	glyphOrder := make([]string, 20)
	for i := range glyphOrder {
		glyphOrder[i] = string(rune('A' + i))
	}

	lookups, err := DecodeContextSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodeContextSubstLookups: %v", err)
	}
	if len(lookups) != 1 {
		t.Fatalf("expected 1 lookup, got %d", len(lookups))
	}
	l := lookups[0]
	if l.Format != 1 {
		t.Errorf("Format = %d, want 1", l.Format)
	}
	if len(l.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(l.Rules))
	}
	r := l.Rules[0]
	if r.Input[0] != 5 || r.Input[1] != 10 {
		t.Errorf("Input = %v, want [5,10]", r.Input)
	}
	if r.NestedLookups[0].SequenceIndex != 0 || r.NestedLookups[0].LookupListIndex != 3 {
		t.Errorf("NestedLookup = %+v", r.NestedLookups[0])
	}
}

func TestDecodeContextSubstFormat2(t *testing.T) {
	classDef := []byte{
		0x00, 0x02, 0x00, 0x02,
		0x00, 0x05, 0x00, 0x09, 0x00, 0x01,
		0x00, 0x0A, 0x00, 0x0E, 0x00, 0x02,
	}
	coverage := []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x05}
	// SubClassRule: GlyphCount=2, SubstCount=1, Class=2, Lookup(seq=1,idx=4)
	subClassRule := []byte{0x00, 0x02, 0x00, 0x01, 0x00, 0x02, 0x00, 0x01, 0x00, 0x04}
	subClassRuleSet := []byte{0x00, 0x01, 0x00, 0x04}
	subClassRuleSet = append(subClassRuleSet, subClassRule...)

	covOff := uint16(2 + 2 + 2 + 2 + 2) // format + covOff + classDefOff + ruleSetCount + ruleSetOff
	classDefOff := covOff + uint16(len(coverage))
	rsOff := classDefOff + uint16(len(classDef))
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], 2)
	binary.BigEndian.PutUint16(header[2:4], covOff)
	binary.BigEndian.PutUint16(header[4:6], classDefOff)
	binary.BigEndian.PutUint16(header[6:8], 1)

	ruleSetOffBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(ruleSetOffBytes[0:2], rsOff)

	lookupData := append(header, ruleSetOffBytes...)
	lookupData = append(lookupData, coverage...)
	lookupData = append(lookupData, classDef...)
	lookupData = append(lookupData, subClassRuleSet...)

	data := buildGSUBTable(5, lookupData)
	glyphOrder := make([]string, 20)
	for i := range glyphOrder {
		glyphOrder[i] = string(rune('A' + i))
	}

	lookups, err := DecodeContextSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodeContextSubstLookups: %v", err)
	}
	l := lookups[0]
	if l.Format != 2 {
		t.Errorf("Format = %d, want 2", l.Format)
	}
	if len(l.Rules) != 25 {
		t.Fatalf("expected 25 expanded rules, got %d", len(l.Rules))
	}
	for _, r := range l.Rules {
		if r.Input[1] < 10 || r.Input[1] > 14 {
			t.Errorf("second glyph = %d, want 10..14", r.Input[1])
		}
	}
}

func TestDecodeContextSubstFormat3(t *testing.T) {
	coverage1 := []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x05}
	coverage2 := []byte{0x00, 0x01, 0x00, 0x01, 0x00, 0x06}

	off1 := uint16(2 + 2 + 2 + 4 + 4) // format + glyphCount + substCount + 2 coverageOff + lookupRec
	off2 := off1 + uint16(len(coverage1))
	header := make([]byte, 6)
	binary.BigEndian.PutUint16(header[0:2], 3)
	binary.BigEndian.PutUint16(header[2:4], 2)
	binary.BigEndian.PutUint16(header[4:6], 1)

	covOffsets := make([]byte, 4)
	binary.BigEndian.PutUint16(covOffsets[0:2], off1)
	binary.BigEndian.PutUint16(covOffsets[2:4], off2)

	lookupRec := make([]byte, 4)
	binary.BigEndian.PutUint16(lookupRec[0:2], 0)
	binary.BigEndian.PutUint16(lookupRec[2:4], 7)

	lookupData := append(header, covOffsets...)
	lookupData = append(lookupData, lookupRec...)
	lookupData = append(lookupData, coverage1...)
	lookupData = append(lookupData, coverage2...)

	data := buildGSUBTable(5, lookupData)
	glyphOrder := make([]string, 20)
	for i := range glyphOrder {
		glyphOrder[i] = string(rune('A' + i))
	}

	lookups, err := DecodeContextSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("DecodeContextSubstLookups: %v", err)
	}
	l := lookups[0]
	if l.Format != 3 {
		t.Errorf("Format = %d, want 3", l.Format)
	}
	if l.Rules[0].NestedLookups[0].LookupListIndex != 7 {
		t.Errorf("LookupListIndex = %d, want 7", l.Rules[0].NestedLookups[0].LookupListIndex)
	}
}

func TestDecodeContextSubstTruncated(t *testing.T) {
	glyphOrder := []string{"A", "B", "C"}
	_, err := DecodeContextSubstLookups([]byte{0x00}, glyphOrder)
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

func TestDecodeContextSubstNoLookups(t *testing.T) {
	glyphOrder := []string{"A", "B", "C"}
	singleSubst := []byte{0x00, 0x01, 0x00, 0x04, 0x00, 0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00}
	data := buildGSUBTable(1, singleSubst)
	lookups, err := DecodeContextSubstLookups(data, glyphOrder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lookups) != 0 {
		t.Errorf("expected 0 lookups, got %d", len(lookups))
	}
}

func FuzzDecodeContextSubstLookups(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A})
	f.Fuzz(func(t *testing.T, data []byte) {
		glyphOrder := []string{"A", "B", "C", "D", "E"}
		DecodeContextSubstLookups(data, glyphOrder)
	})
}

func FuzzDecodeChainContextSubstLookups(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A})
	f.Fuzz(func(t *testing.T, data []byte) {
		glyphOrder := []string{"A", "B", "C", "D", "E"}
		DecodeChainContextSubstLookups(data, glyphOrder)
	})
}
