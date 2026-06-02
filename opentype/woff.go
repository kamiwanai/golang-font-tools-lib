// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package opentype

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// WOFF signature constants
const (
	woffSig  = 0x774F4646 // 'wOFF'
	woff2Sig = 0x774F4632 // 'wOF2'
)

// ParseWOFF decompresses a WOFF font and returns equivalent TTF/OTF bytes.
// WOFF uses per-table zlib compression.
func ParseWOFF(data []byte) ([]byte, error) {
	if len(data) < 44 {
		return nil, fmt.Errorf("WOFF header too short: %d bytes", len(data))
	}

	signature := binary.BigEndian.Uint32(data[0:4])
	if signature != woffSig {
		return nil, fmt.Errorf("not a WOFF file: signature %#08x", signature)
	}

	flavor := binary.BigEndian.Uint32(data[4:8])
	_ = binary.BigEndian.Uint32(data[8:12]) // totalLength
	numTables := int(binary.BigEndian.Uint16(data[12:14]))
	totalSfntSize := binary.BigEndian.Uint32(data[16:20])

	// Guard against unreasonable table counts
	if numTables > 256 || numTables < 1 {
		return nil, fmt.Errorf("WOFF: unreasonable table count %d", numTables)
	}

	// Build SFNT output
	result := make([]byte, totalSfntSize)

	// Offset table header
	binary.BigEndian.PutUint32(result[0:4], flavor)
	binary.BigEndian.PutUint16(result[4:6], uint16(numTables))

	// searchRange / entrySelector / rangeShift
	maxPower := uint16(1)
	log2 := uint16(0)
	for maxPower*2 <= uint16(numTables) {
		maxPower *= 2
		log2++
	}
	searchRange := maxPower * 16
	binary.BigEndian.PutUint16(result[6:8], searchRange)
	binary.BigEndian.PutUint16(result[8:10], log2)
	binary.BigEndian.PutUint16(result[10:12], uint16(numTables)*16-searchRange)

	// Table directory starts at byte 12, after 16*numTables bytes
	dataStart := uint32(12 + numTables*16)

	entriesStart := 44 // WOFF header is 44 bytes
	curOff := dataStart

	for i := 0; i < numTables; i++ {
		entryOff := entriesStart + i*20
		if entryOff+20 > len(data) {
			return nil, fmt.Errorf("WOFF: entry %d exceeds file", i)
		}

		tag := string(data[entryOff : entryOff+4])
		tableOff := binary.BigEndian.Uint32(data[entryOff+4 : entryOff+8])
		compLen := binary.BigEndian.Uint32(data[entryOff+8 : entryOff+12])
		origLen := binary.BigEndian.Uint32(data[entryOff+12 : entryOff+16])
		_ = binary.BigEndian.Uint32(data[entryOff+16 : entryOff+20]) // origChecksum

		if tableOff+compLen > uint32(len(data)) {
			return nil, fmt.Errorf("WOFF: table %q exceeds file", tag)
		}

		// Read table data (decompress if needed)
		var tableData []byte
		if compLen < origLen {
			// Zlib-compressed
			zr, err := zlib.NewReader(bytes.NewReader(data[tableOff : tableOff+compLen]))
			if err != nil {
				return nil, fmt.Errorf("WOFF: zlib decompress %q: %w", tag, err)
			}
			tableData = make([]byte, origLen)
			n, err := io.ReadFull(zr, tableData)
			zr.Close()
			if err != nil || uint32(n) != origLen {
				return nil, fmt.Errorf("WOFF: zlib decompress %q: read %d bytes, expected %d: %v", tag, n, origLen, err)
			}
		} else {
			// Uncompressed
			tableData = data[tableOff : tableOff+origLen]
		}

		// Write directory entry
		dirOff := uint32(12 + i*16)
		copy(result[dirOff:dirOff+4], tag)
		chk := calcTableChecksum(tableData)
		binary.BigEndian.PutUint32(result[dirOff+4:dirOff+8], chk)
		binary.BigEndian.PutUint32(result[dirOff+8:dirOff+12], curOff)
		binary.BigEndian.PutUint32(result[dirOff+12:dirOff+16], origLen)

		// Write table data
		copy(result[curOff:], tableData)

		// Pad to 4-byte boundary
		padded := (origLen + 3) & ^uint32(3)
		curOff += padded
	}

	// Recalculate head checksum adjustment
	var sum uint32
	for i := 0; int(i) < len(result); i += 4 {
		sum += binary.BigEndian.Uint32(result[i:])
	}
	for i := 0; i < numTables; i++ {
		dirOff := 12 + i*16
		if string(result[dirOff:dirOff+4]) == "head" {
			headOff := binary.BigEndian.Uint32(result[dirOff+8 : dirOff+12])
			binary.BigEndian.PutUint32(result[headOff+8:headOff+12], 0xB1B0AFBA-sum)
			break
		}
	}

	return result, nil
}


// calcTableChecksum computes the OpenType table checksum.
func calcTableChecksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i+3 < len(data); i += 4 {
		sum += binary.BigEndian.Uint32(data[i:])
	}
	rem := len(data) % 4
	if rem > 0 {
		var last uint32
		for i := len(data) - rem; i < len(data); i++ {
			last = (last << 8) | uint32(data[i])
		}
		last <<= (4 - rem) * 8
		sum += last
	}
	return sum
}

// IsWOFF reports whether data starts with the WOFF signature.
func IsWOFF(data []byte) bool {
	return len(data) >= 4 && binary.BigEndian.Uint32(data[0:4]) == woffSig
}

// IsWOFF2 reports whether data starts with the WOFF2 signature.
func IsWOFF2(data []byte) bool {
	return len(data) >= 4 && binary.BigEndian.Uint32(data[0:4]) == woff2Sig
}
