// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gpos

import (
	"encoding/binary"
	"testing"
)

func BenchmarkDecodePairPosLookups(b *testing.B) {
	glyphOrder := []string{".notdef", "A", "T", "V", "W"}
	gpos := make([]byte, 22)
	binary.BigEndian.PutUint16(gpos[0:2], 1)
	binary.BigEndian.PutUint16(gpos[8:10], 10)
	binary.BigEndian.PutUint16(gpos[10:12], 1)
	binary.BigEndian.PutUint16(gpos[12:14], 4)
	binary.BigEndian.PutUint16(gpos[14:16], 2)
	binary.BigEndian.PutUint16(gpos[18:20], 1)
	binary.BigEndian.PutUint16(gpos[20:22], 8)
	gpos = append(gpos, buildPairPosFormat2Fixture()...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePairPosLookups(gpos, glyphOrder)
	}
}

func BenchmarkDecodePairPosFormat1(b *testing.B) {
	glyphOrder := makeGlyphOrder(500)
	data := buildFormat1Fixture(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePairPosFormat1(data, glyphOrder)
	}
}

func BenchmarkDecodePairPosFormat2(b *testing.B) {
	glyphOrder := []string{".notdef", "A", "T", "V", "W"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePairPosFormat2(buildPairPosFormat2Fixture(), glyphOrder)
	}
}

func makeGlyphOrder(n int) []string {
	names := make([]string, n)
	names[0] = ".notdef"
	for i := 1; i < n; i++ {
		names[i] = "glyphName"
	}
	return names
}

func buildFormat1Fixture(numPairs int) []byte {
	const valSize = 2
	pairSetRecords := 2 + numPairs*(2+valSize+0)
	subtableSize := 10 + 2 + 4 + 2 + 2 + pairSetRecords
	data := make([]byte, subtableSize)
	binary.BigEndian.PutUint16(data[0:2], 1)
	binary.BigEndian.PutUint16(data[2:4], 16)
	binary.BigEndian.PutUint16(data[4:6], 0x0004)
	binary.BigEndian.PutUint16(data[6:8], 0)
	binary.BigEndian.PutUint16(data[8:10], 1)
	binary.BigEndian.PutUint16(data[10:12], 18)
	binary.BigEndian.PutUint16(data[12:14], 1)
	binary.BigEndian.PutUint16(data[14:16], 1)
	binary.BigEndian.PutUint16(data[16:18], 1)
	binary.BigEndian.PutUint16(data[18:20], uint16(numPairs))
	for i := 0; i < numPairs; i++ {
		off := 20 + i*(2+valSize+0)
		binary.BigEndian.PutUint16(data[off:off+2], uint16(1+(i%400)))
		binary.BigEndian.PutUint16(data[off+2:off+4], 0xffec) // -20
	}
	return data
}
