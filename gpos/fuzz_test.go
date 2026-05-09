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
