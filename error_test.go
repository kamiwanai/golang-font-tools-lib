// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package fonttools

import "testing"

func TestError_EmptyData(t *testing.T) {
	_, err := Parse([]byte{})
	if err == nil {
		t.Fatal("Expected error for empty data")
	}
}

func TestError_TooShort(t *testing.T) {
	_, err := Parse([]byte{0, 0, 0, 0})
	if err == nil {
		t.Fatal("Expected error for < 12 bytes")
	}
}

func TestError_ZeroTables(t *testing.T) {
	data := make([]byte, 12)
	data[3] = 1
	_, err := Parse(data)
	if err == nil {
		t.Fatal("Expected error for zero tables")
	}
}

func TestError_InvalidScalerType(t *testing.T) {
	data := make([]byte, 12)
	data[0] = 0xFF
	data[1] = 0xFF
	data[2] = 0xFF
	data[3] = 0xFF
	_, err := Parse(data)
	if err == nil {
		t.Fatal("Expected error for invalid scalerType")
	}
}

func TestError_TruncatedTableRecord(t *testing.T) {
	data := make([]byte, 32)
	data[3] = 1
	data[5] = 2
	_, err := Parse(data)
	if err == nil {
		t.Fatal("Expected error for truncated table record")
	}
}

func TestError_TableOffsetBeyondData(t *testing.T) {
	data := make([]byte, 28)
	data[3] = 1
	data[5] = 1
	data[12] = 't'
	data[13] = 'e'
	data[14] = 's'
	data[15] = 't'
	data[20] = 0x01
	data[21] = 0x86
	data[22] = 0xA0
	data[23] = 0x00
	data[24] = 0
	data[25] = 0
	data[26] = 0
	data[27] = 4

	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.TableData("test")
	if err == nil {
		t.Fatal("Expected error for out-of-bounds table offset")
	}
}

func TestError_MissingHead(t *testing.T) {
	data := make([]byte, 28)
	data[3] = 1
	data[5] = 1
	data[12] = 't'
	data[13] = 'e'
	data[14] = 's'
	data[15] = 't'
	data[20] = 0
	data[21] = 0
	data[22] = 0
	data[23] = 28
	data[24] = 0
	data[25] = 0
	data[26] = 0
	data[27] = 0

	font, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = font.Head()
	if err == nil {
		t.Fatal("Expected error for missing head table")
	}
}

func TestError_ParseFile_Nonexistent(t *testing.T) {
	_, err := ParseFile("testdata/__nonexistent__.ttf")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

func TestError_EmptyFile(t *testing.T) {
	font, err := Parse([]byte{})
	if font != nil || err == nil {
		t.Fatal("Expected nil font and non-nil error for empty input")
	}
}

func TestError_NilData(t *testing.T) {
	_, err := Parse(nil)
	if err == nil {
		t.Fatal("Expected error for nil data")
	}
}
