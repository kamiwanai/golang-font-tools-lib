// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gsub

import (
	"encoding/binary"
	"testing"
)

func TestReverseSubstFormat1(t *testing.T) {
	glyphOrder := []string{".notdef", "A", "B", "C"}
	enc := binary.BigEndian

	covSize := 4 + 2*2  // format1 + count=2 + 2 glyphs
	btCovSize := 4 + 1*2 // format1 + count=1 + 1 glyph
	headerSize := 16      // format+coverageOff+btCnt+btOff+laCnt+glyphCnt+sub[2]
	totalSize := headerSize + covSize + btCovSize
	data := make([]byte, totalSize)

	enc.PutUint16(data[0:2], 1)
	enc.PutUint16(data[2:4], uint16(headerSize)) // cov offset
	enc.PutUint16(data[4:6], 1)                  // backtrack count
	enc.PutUint16(data[6:8], uint16(headerSize+covSize)) // backtrack[0] offset
	enc.PutUint16(data[8:10], 0)                 // lookahead count
	enc.PutUint16(data[10:12], 2)                // glyph count
	enc.PutUint16(data[12:14], 3)                // A -> C
	enc.PutUint16(data[14:16], 2)                // B -> B

	// Coverage
	enc.PutUint16(data[headerSize:headerSize+2], 1)
	enc.PutUint16(data[headerSize+2:headerSize+4], 2)
	enc.PutUint16(data[headerSize+4:headerSize+6], 1)
	enc.PutUint16(data[headerSize+6:headerSize+8], 2)

	// Backtrack
	btOff := headerSize + covSize
	enc.PutUint16(data[btOff:btOff+2], 1)
	enc.PutUint16(data[btOff+2:btOff+4], 1)
	enc.PutUint16(data[btOff+4:btOff+6], 3)

	llo := 10; luo := llo + 4; sto := luo + 8
	fullSize := sto + totalSize
	full := make([]byte, fullSize)
	enc.PutUint16(full[0:2], 1)
	enc.PutUint16(full[2:4], 0)
	enc.PutUint16(full[8:10], uint16(llo))
	enc.PutUint16(full[llo:llo+2], 1)
	enc.PutUint16(full[llo+2:llo+4], uint16(luo-llo))
	enc.PutUint16(full[luo:luo+2], 8)
	enc.PutUint16(full[luo+2:luo+4], 0)
	enc.PutUint16(full[luo+4:luo+6], 1)
	enc.PutUint16(full[luo+6:luo+8], uint16(sto-luo))
	copy(full[sto:], data)

	subs, err := DecodeReverseSubstLookups(full, glyphOrder)
	if err != nil { t.Fatalf("err: %v", err) }
	if len(subs) != 2 { t.Fatalf("want 2, got %d", len(subs)) }
	if subs[0].From != "A" || subs[0].To != "C" { t.Errorf("A->C: got %s->%s", subs[0].From, subs[0].To) }
}
