package gpos

import (
	"encoding/binary"
	"testing"
)

func TestCursivePosFormat1(t *testing.T) {
	glyphOrder := []string{".notdef", "alef", "beh", "meem"}
	enc := binary.BigEndian
	ec := 3
	cs := 4 + ec*2
	hs := 6 + ec*4
	as := ec * 6
	ss := hs + cs + as*2
	sub := make([]byte, ss)
	enc.PutUint16(sub[0:2], 1)
	enc.PutUint16(sub[2:4], uint16(hs))
	enc.PutUint16(sub[4:6], uint16(ec))
	cS := hs
	enc.PutUint16(sub[cS:cS+2], 1)
	enc.PutUint16(sub[cS+2:cS+4], uint16(ec))
	enc.PutUint16(sub[cS+4:cS+6], 1)
	enc.PutUint16(sub[cS+6:cS+8], 2)
	enc.PutUint16(sub[cS+8:cS+10], 3)
	eS := cS + cs
	xS := eS + as
	for i := 0; i < ec; i++ {
		ro := 6 + i*4
		enc.PutUint16(sub[ro:ro+2], uint16(eS+i*6))
		enc.PutUint16(sub[ro+2:ro+4], uint16(xS+i*6))
	}
	for i := 0; i < ec; i++ {
		pos := eS + i*6
		enc.PutUint16(sub[pos:pos+2], 1)
		enc.PutUint16(sub[pos+2:pos+4], 0)
		enc.PutUint16(sub[pos+4:pos+6], 500)
		pos = xS + i*6
		enc.PutUint16(sub[pos:pos+2], 1)
		enc.PutUint16(sub[pos+2:pos+4], 0)
		enc.PutUint16(sub[pos+4:pos+6], 0xFF9C)
	}
	llo := 10
	luo := llo + 4
	sto := luo + 8
	total := sto + ss
	data := make([]byte, total)
	enc.PutUint16(data[0:2], 1)
	enc.PutUint16(data[2:4], 0)
	enc.PutUint16(data[8:10], uint16(llo))
	enc.PutUint16(data[llo:llo+2], 1)
	enc.PutUint16(data[llo+2:llo+4], uint16(luo-llo))
	enc.PutUint16(data[luo:luo+2], 3)
	enc.PutUint16(data[luo+2:luo+4], 0)
	enc.PutUint16(data[luo+4:luo+6], 1)
	enc.PutUint16(data[luo+6:luo+8], uint16(sto-luo))
	copy(data[sto:], sub)
	entries, err := DecodeCursivePosLookups(data, glyphOrder)
	if err != nil { t.Fatalf("err: %v", err) }
	if len(entries) != 3 { t.Fatalf("want 3, got %d", len(entries)) }
	if entries[0].Glyph != "alef" { t.Errorf("glyph: %s", entries[0].Glyph) }
}
