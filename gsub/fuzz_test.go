// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gsub

import "testing"

func FuzzDecodeSingleSubstLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		subs, err := DecodeSingleSubstLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = subs
	})
}

func FuzzDecodeLigatureSubstLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W", "f", "i", "fi"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		ligs, err := DecodeLigatureSubstLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = ligs
	})
}

func FuzzDecodeMultipleSubstLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		subs, err := DecodeMultipleSubstLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = subs
	})
}

func FuzzDecodeAlternateSubstLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		alts, err := DecodeAlternateSubstLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = alts
	})
}

func FuzzScripts(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		scripts, err := Scripts(data)
		if err != nil {
			return
		}
		_ = scripts
	})
}

func FuzzFeatures(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		features, err := Features(data)
		if err != nil {
			return
		}
		_ = features
	})
}

func FuzzDecodeReverseSubstLookups(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T", "W"}
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		subs, err := DecodeReverseSubstLookups(data, glyphOrder)
		if err != nil {
			return
		}
		_ = subs
	})
}

func FuzzActiveLookups(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 10))

	f.Fuzz(func(t *testing.T, data []byte) {
		lookups, err := ActiveLookups(data, "latn", "dflt", "kern")
		if err != nil {
			return
		}
		_ = lookups
	})
}
