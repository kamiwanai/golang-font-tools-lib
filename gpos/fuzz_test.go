// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gpos

import (
	"testing"
)

func FuzzDecodePairPosLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(makeMinimalGpos())

	f.Fuzz(func(t *testing.T, data []byte) {
		pairs, err := DecodePairPosLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = pairs
	})
}

func FuzzDecodePairPosFormat1(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(makeMinimalFormat1())

	f.Fuzz(func(t *testing.T, data []byte) {
		pairs, err := DecodePairPosFormat1(data, glyphOrder)
		if err != nil {
			return
		}
		_ = pairs
	})
}

func FuzzDecodePairPosFormat2(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(buildPairPosFormat2Fixture())

	f.Fuzz(func(t *testing.T, data []byte) {
		pairs, err := DecodePairPosFormat2(data, glyphOrder)
		if err != nil {
			return
		}
		_ = pairs
	})
}

func makeMinimalGpos() []byte {
	data := make([]byte, 12)
	data[8] = 10
	return data
}

func makeMinimalFormat1() []byte {
	data := make([]byte, 2)
	data[0] = 0
	data[1] = 1
	return data
}


func FuzzDecodeSinglePosLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		vals, err := DecodeSinglePosLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = vals
	})
}

func FuzzDecodeCursivePosLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		entries, err := DecodeCursivePosLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = entries
	})
}

func FuzzDecodeMarkLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		attachments, err := DecodeMarkLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = attachments
	})
}

func FuzzDecodeContextPosLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		lookups, err := DecodeContextPosLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = lookups
	})
}

func FuzzDecodeChainContextPosLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		lookups, err := DecodeChainContextPosLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = lookups
	})
}

func FuzzDecodeContextValueRecords(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		records, err := DecodeContextValueRecords(data, glyphOrder, 7)
		if err != nil {
			return
		}
		_ = records
	})
}

func FuzzDecodeChainContextValueRecords(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		records, err := DecodeContextValueRecords(data, glyphOrder, 8)
		if err != nil {
			return
		}
		_ = records
	})
}
