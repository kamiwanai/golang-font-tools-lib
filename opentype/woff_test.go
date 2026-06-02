// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestIsWOFF(t *testing.T) {
	data := []byte{0x77, 0x4F, 0x46, 0x46}
	if !IsWOFF(data) {
		t.Error("IsWOFF should return true for wOFF signature")
	}
	if IsWOFF([]byte{0x77, 0x4F}) {
		t.Error("IsWOFF should return false for short data")
	}
	if IsWOFF([]byte{0x00, 0x01, 0x02, 0x03}) {
		t.Error("IsWOFF should return false for wrong signature")
	}
}

func TestIsWOFF2(t *testing.T) {
	data := []byte{0x77, 0x4F, 0x46, 0x32}
	if !IsWOFF2(data) {
		t.Error("IsWOFF2 should return true for wOF2 signature")
	}
	if IsWOFF2([]byte{0x77, 0x4F, 0x46, 0x46}) {
		t.Error("IsWOFF2 should return false for wOFF (WOFF1) signature")
	}
}

func buildMinimalWOFF(ttfData []byte) []byte {
	flavor := binary.BigEndian.Uint32(ttfData[0:4])
	numTables := int(binary.BigEndian.Uint16(ttfData[4:6]))

	var buf bytes.Buffer
	buf.Write([]byte{0x77, 0x4F, 0x46, 0x46})
	binary.Write(&buf, binary.BigEndian, flavor)
	binary.Write(&buf, binary.BigEndian, uint32(0))
	binary.Write(&buf, binary.BigEndian, uint16(numTables))
	binary.Write(&buf, binary.BigEndian, uint16(0))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	binary.Write(&buf, binary.BigEndian, uint16(0))
	binary.Write(&buf, binary.BigEndian, uint16(0))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	binary.Write(&buf, binary.BigEndian, uint32(0))

	entries := make([][]byte, numTables)
	dataStart := uint32(44 + numTables*20)
	curOff := dataStart

	sfntHeaderSize := uint32(12 + numTables*16)
	totalSfntSize := sfntHeaderSize

	for i := 0; i < numTables; i++ {
		dirOff := 12 + i*16
		tag := string(ttfData[dirOff : dirOff+4])
		origLen := binary.BigEndian.Uint32(ttfData[dirOff+12 : dirOff+16])
		ttfTableOff := binary.BigEndian.Uint32(ttfData[dirOff+8 : dirOff+12])

		tableData := ttfData[ttfTableOff : ttfTableOff+origLen]

		var compressed bytes.Buffer
		w := zlib.NewWriter(&compressed)
		w.Write(tableData)
		w.Close()
		compLen := uint32(compressed.Len())

		entry := make([]byte, 20)
		copy(entry[0:4], []byte(tag))
		binary.BigEndian.PutUint32(entry[4:8], curOff)
		binary.BigEndian.PutUint32(entry[8:12], compLen)
		binary.BigEndian.PutUint32(entry[12:16], origLen)
		entries[i] = entry

		padded := (compLen + 3) & ^uint32(3)
		curOff += padded
		totalSfntSize += (origLen + 3) & ^uint32(3)
	}

	for _, e := range entries {
		buf.Write(e)
	}

	for i := 0; i < numTables; i++ {
		ttfTableOff := binary.BigEndian.Uint32(ttfData[12+i*16+8 : 12+i*16+12])
		origLen := binary.BigEndian.Uint32(ttfData[12+i*16+12 : 12+i*16+16])
		tableData := ttfData[ttfTableOff : ttfTableOff+origLen]

		var compressed bytes.Buffer
		w := zlib.NewWriter(&compressed)
		w.Write(tableData)
		w.Close()

		buf.Write(compressed.Bytes())
		for buf.Len()%4 != 0 {
			buf.WriteByte(0)
		}
	}

	result := buf.Bytes()
	binary.BigEndian.PutUint32(result[8:12], uint32(len(result)))
	binary.BigEndian.PutUint32(result[16:20], totalSfntSize)

	return result
}

func TestParseWOFF(t *testing.T) {
	_, err := ParseWOFF([]byte{1, 2, 3, 4})
	if err == nil {
		t.Fatal("expected error for non-WOFF data")
	}
	_, err = ParseWOFF([]byte{0x77, 0x4F, 0x46, 0x46, 0, 0, 0, 0})
	if err == nil {
		t.Fatal("expected error for too-short WOFF header")
	}
}

func TestParseWOFFRoundtrip(t *testing.T) {
	ttf, err := Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	woff := buildMinimalWOFF(goregular.TTF)
	if !IsWOFF(woff) {
		t.Fatal("buildMinimalWOFF did not produce valid WOFF")
	}

	result, err := ParseWOFF(woff)
	if err != nil {
		t.Fatalf("ParseWOFF: %v", err)
	}

	resultFont, err := Parse(result)
	if err != nil {
		t.Fatalf("Parse result: %v", err)
	}

	if len(resultFont.Tables) != len(ttf.Tables) {
		t.Errorf("table count mismatch: got %d, want %d", len(resultFont.Tables), len(ttf.Tables))
	}

	_, err = resultFont.Head()
	if err != nil {
		t.Errorf("Head table not readable after WOFF roundtrip: %v", err)
	}
}

func TestCalcTableChecksum(t *testing.T) {
	if chk := calcTableChecksum([]byte{0, 0, 0, 1}); chk != 1 {
		t.Errorf("checksum([0,0,0,1]) = %d, want 1", chk)
	}
	chk := calcTableChecksum([]byte{0, 0, 1})
	if chk == 0 {
		t.Error("checksum for non-aligned data should not be 0")
	}
}
