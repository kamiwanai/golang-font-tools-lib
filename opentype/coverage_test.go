// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package opentype

import (
	"encoding/binary"
	"testing"
)

func putU16(data []byte, offset int, v uint16) {
	binary.BigEndian.PutUint16(data[offset:offset+2], v)
}

func TestDecodeCoverageFormat1(t *testing.T) {
	// Format 1, 3 glyphs: 5, 10, 15
	data := []byte{0, 1, 0, 3, 0, 5, 0, 10, 0, 15}
	got, err := DecodeCoverage(data, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []uint16{5, 10, 15}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestDecodeCoverageFormat2(t *testing.T) {
	// Format 2, 2 ranges: 1-3, 7-7. Each range is 6 bytes (start, end, startCoverageIndex).
	data := []byte{
		0, 2, 0, 2, // format=2, rangeCount=2
		0, 1, 0, 3, 0, 0, // range[0]: start=1, end=3, startCoverageIndex=0
		0, 7, 0, 7, 0, 4, // range[1]: start=7, end=7, startCoverageIndex=4
	}
	got, err := DecodeCoverage(data, 0)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := []uint16{1, 2, 3, 7}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestDecodeCoverageFormat2EndBeforeStart(t *testing.T) {
	// Format 2, 1 range with end < start
	data := []byte{0, 2, 0, 1, 0, 5, 0, 3, 0, 0}
	if _, err := DecodeCoverage(data, 0); err == nil {
		t.Fatal("expected error for range end < start")
	}
}

func TestDecodeCoverageRejectsOversizedFormat1(t *testing.T) {
	saved := MaxCoverageGlyphs
	defer func() { MaxCoverageGlyphs = saved }()
	MaxCoverageGlyphs = 4
	// Format 1, glyphCount=5 > limit
	data := []byte{0, 1, 0, 5, 0, 1, 0, 2, 0, 3, 0, 4, 0, 5}
	if _, err := DecodeCoverage(data, 0); err == nil {
		t.Fatal("expected error for glyphCount > MaxCoverageGlyphs")
	}
}

func TestDecodeCoverageRejectsOversizedFormat2Ranges(t *testing.T) {
	saved := MaxCoverageRanges
	defer func() { MaxCoverageRanges = saved }()
	MaxCoverageRanges = 1
	// Format 2, rangeCount=2 > limit
	data := []byte{0, 2, 0, 2, 0, 1, 0, 1, 0, 2, 0, 2}
	if _, err := DecodeCoverage(data, 0); err == nil {
		t.Fatal("expected error for rangeCount > MaxCoverageRanges")
	}
}

func TestDecodeCoverageRejectsTruncated(t *testing.T) {
	if _, err := DecodeCoverage([]byte{0, 1, 0}, 0); err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestDecodeCoverageRejectsUnknownFormat(t *testing.T) {
	data := []byte{0, 7, 0, 0}
	if _, err := DecodeCoverage(data, 0); err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestDecodeCoverageOffsetBeyondTable(t *testing.T) {
	data := []byte{0, 1, 0, 0}
	if _, err := DecodeCoverage(data, 100); err == nil {
		t.Fatal("expected error for offset beyond table")
	}
}
