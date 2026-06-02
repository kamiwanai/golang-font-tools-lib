// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gvar

import (
	"encoding/binary"
	"fmt"
)

// F2DOT14 is a signed 16-bit fixed-point number with 14 fraction bits.
// To convert to float64, divide by 16384.0.
type F2DOT14 int16

// Float returns the float64 representation of this F2DOT14.
func (f F2DOT14) Float() float64 {
	return float64(f) / 16384.0
}

// ---------- Header types ----------

// GVAR holds the parsed gvar (Glyph Variations) table.
type GVAR struct {
	MajorVersion uint16
	MinorVersion uint16
	AxisCount    uint16
	SharedTuples [][]F2DOT14
	Flags        uint16
	Glyphs       []*GlyphVariationData
}

// GlyphVariationData holds the variation data for one glyph.
type GlyphVariationData struct {
	Tuples             []TupleVariation
	SharedPointNumbers []uint16
}

// TupleVariation describes one variation tuple for a glyph.
type TupleVariation struct {
	Peak         []F2DOT14
	Start        []F2DOT14
	End          []F2DOT14
	PointNumbers []uint16
	DeltasX      []int16
	DeltasY      []int16
}

// ---------- Constants ----------

const (
	embeddedPeakTuple   = 0x8000
	intermediateRegion  = 0x4000
	privatePointNumbers = 0x2000
	tupleIndexMask      = 0x0FFF

	tuplesSharePointNumbers = 0x8000
	tupleCountMask          = 0x0FFF

	deltasAreZero    = 0x80
	deltasAreWords   = 0x40
	deltasSizeMask   = 0xC0
	deltaRunCountMask = 0x3F

	pointsAreWords    = 0x80
	pointRunCountMask = 0x7F
)

// ---------- Decode ----------

func Decode(data []byte, glyphCount int) (GVAR, error) {
	if len(data) < 20 {
		return GVAR{}, fmt.Errorf("gvar header too short: %d bytes", len(data))
	}

	majorVersion := readU16(data, 0)
	minorVersion := readU16(data, 2)
	axisCount := int(readU16(data, 4))
	sharedTupleCount := int(readU16(data, 6))
	sharedTuplesOffset := readU32(data, 8)
	hdrGlyphCount := int(readU16(data, 12))
	flags := readU16(data, 14)
	offsetToGlyphVarData := readU32(data, 16)

	if majorVersion != 1 {
		return GVAR{}, fmt.Errorf("gvar: unsupported major version %d", majorVersion)
	}
	if hdrGlyphCount != glyphCount {
		return GVAR{}, fmt.Errorf("gvar: glyphCount %d != maxp numGlyphs %d", hdrGlyphCount, glyphCount)
	}
	if sharedTupleCount > MaxSharedTuples {
		return GVAR{}, fmt.Errorf("gvar: sharedTupleCount %d exceeds limit %d", sharedTupleCount, MaxSharedTuples)
	}
	if axisCount > 255 {
		return GVAR{}, fmt.Errorf("gvar: axisCount %d too large", axisCount)
	}

	sharedTuples := make([][]F2DOT14, sharedTupleCount)
	if sharedTupleCount > 0 {
		tupleStart := int(sharedTuplesOffset)
		tupleEnd := tupleStart + sharedTupleCount*axisCount*2
		if tupleEnd > len(data) {
			return GVAR{}, fmt.Errorf("gvar: shared tuples exceed table")
		}
		for i := 0; i < sharedTupleCount; i++ {
			coords := make([]F2DOT14, axisCount)
			for j := 0; j < axisCount; j++ {
				off := tupleStart + i*axisCount*2 + j*2
				coords[j] = F2DOT14(int16(readU16(data, off)))
			}
			sharedTuples[i] = coords
		}
	}
	// Determine offset table format.
	// Per Apple spec / fonttools convention:
	//   flags & 1 == 0: short offsets (uint16, stored as offset/2)
	//   flags & 1 != 0: long offsets (uint32)
	useLongOffsets := flags&1 != 0

	var offsetStride int
	if useLongOffsets {
		offsetStride = 4
	} else {
		offsetStride = 2
	}

	offsetsStart := 20
	offsetsEnd := offsetsStart + (glyphCount+1)*offsetStride
	if offsetsEnd > len(data) {
		return GVAR{}, fmt.Errorf("gvar: offset array exceeds table")
	}

	offsets := make([]int, glyphCount+1)
	for i := 0; i <= glyphCount; i++ {
		if useLongOffsets {
			offsets[i] = int(readU32(data, offsetsStart+i*4))
		} else {
			offsets[i] = int(readU16(data, offsetsStart+i*2)) * 2
		}
	}

	glyphs := make([]*GlyphVariationData, glyphCount)
	for gid := 0; gid < glyphCount; gid++ {
		start := offsets[gid]
		end := offsets[gid+1]
		if start >= end {
			continue
		}
		absStart := int(offsetToGlyphVarData) + start
		absEnd := int(offsetToGlyphVarData) + end
		if absEnd > len(data) {
			return GVAR{}, fmt.Errorf("gvar: glyph %d variation data exceeds table", gid)
		}
		glyphData := data[absStart:absEnd]
		gvd, err := decodeGlyphVariationData(glyphData, axisCount, sharedTuples)
		if err != nil {
			return GVAR{}, fmt.Errorf("gvar: glyph %d: %w", gid, err)
		}
		glyphs[gid] = gvd
	}

	return GVAR{
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
		AxisCount:    uint16(axisCount),
		SharedTuples: sharedTuples,
		Flags:        flags,
		Glyphs:       glyphs,
	}, nil
}

func decodeGlyphVariationData(data []byte, axisCount int, sharedTuples [][]F2DOT14) (*GlyphVariationData, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("GlyphVariationData too short: %d bytes", len(data))
	}

	tupleVarCount := readU16(data, 0)
	dataOffset := readU16(data, 2)
	dataStart := int(dataOffset)
	if dataStart > len(data) {
		return nil, fmt.Errorf("dataOffset %d exceeds data length %d", dataOffset, len(data))
	}

	hasSharedPoints := tupleVarCount&tuplesSharePointNumbers != 0
	tupleCount := int(tupleVarCount & tupleCountMask)
	if tupleCount > MaxTupleVariations {
		return nil, fmt.Errorf("tupleVarCount %d exceeds limit %d", tupleCount, MaxTupleVariations)
	}

	var sharedPointNumbers []uint16
	serializedPos := dataStart

	if hasSharedPoints && serializedPos < len(data) {
		var err error
		sharedPointNumbers, serializedPos, err = decompilePoints(data, serializedPos)
		if err != nil {
			return nil, fmt.Errorf("shared points: %w", err)
		}
	}

	tuples := make([]TupleVariation, 0, tupleCount)
	headerPos := 4

	for i := 0; i < tupleCount; i++ {
		if headerPos+4 > len(data) {
			return nil, fmt.Errorf("tuple %d header exceeds data", i)
		}

		variationDataSize := int(readU16(data, headerPos))
		tupleIndex := readU16(data, headerPos+2)

		hasPeak := tupleIndex&embeddedPeakTuple != 0
		hasIntermed := tupleIndex&intermediateRegion != 0
		hasPrivate := tupleIndex&privatePointNumbers != 0
		index := int(tupleIndex & tupleIndexMask)

		headerSize := 4
		if hasPeak {
			headerSize += axisCount * 2
		}
		if hasIntermed {
			headerSize += axisCount * 2 * 2
		}
		if headerPos+headerSize > len(data) {
			return nil, fmt.Errorf("tuple %d header exceeds data", i)
		}

		coordsOff := headerPos + 4

		var peak []F2DOT14
		var start, end []F2DOT14

		if hasPeak {
			peak = make([]F2DOT14, axisCount)
			for j := 0; j < axisCount; j++ {
				peak[j] = F2DOT14(int16(readU16(data, coordsOff+j*2)))
			}
			coordsOff += axisCount * 2
		} else {
			if index >= len(sharedTuples) {
				return nil, fmt.Errorf("tuple %d: shared tuple index %d out of range (have %d)",
					i, index, len(sharedTuples))
			}
			peak = sharedTuples[index]
		}

		if hasIntermed {
			start = make([]F2DOT14, axisCount)
			for j := 0; j < axisCount; j++ {
				start[j] = F2DOT14(int16(readU16(data, coordsOff+j*2)))
			}
			coordsOff += axisCount * 2
			end = make([]F2DOT14, axisCount)
			for j := 0; j < axisCount; j++ {
				end[j] = F2DOT14(int16(readU16(data, coordsOff+j*2)))
			}
			coordsOff += axisCount * 2
		}

		serializedEnd := serializedPos + variationDataSize
		if serializedEnd > len(data) {
			return nil, fmt.Errorf("tuple %d: serialized data exceeds table", i)
		}

		tupleSerialized := data[serializedPos:serializedEnd]
		serializedPos = serializedEnd

		var points []uint16
		var deltasX, deltasY []int16
		sp := 0

		if hasPrivate {
			var err error
			points, sp, err = decompilePoints(tupleSerialized, sp)
			if err != nil {
				return nil, fmt.Errorf("tuple %d private points: %w", i, err)
			}
		} else if hasSharedPoints {
			points = sharedPointNumbers
		}

		var err error
		deltasX, sp, err = decompileDeltas(tupleSerialized, sp, points)
		if err != nil {
			return nil, fmt.Errorf("tuple %d X deltas: %w", i, err)
		}

		deltasY, sp, err = decompileDeltas(tupleSerialized, sp, points)
		if err != nil {
			return nil, fmt.Errorf("tuple %d Y deltas: %w", i, err)
		}

		tuples = append(tuples, TupleVariation{
			Peak:         peak,
			Start:        start,
			End:          end,
			PointNumbers: points,
			DeltasX:      deltasX,
			DeltasY:      deltasY,
		})

		headerPos += headerSize
	}

	return &GlyphVariationData{
		Tuples:             tuples,
		SharedPointNumbers: sharedPointNumbers,
	}, nil
}

func decompilePoints(data []byte, offset int) ([]uint16, int, error) {
	if offset >= len(data) {
		return nil, offset, fmt.Errorf("no data for points")
	}

	pos := offset
	count := int(data[pos])
	pos++
	if count&pointsAreWords != 0 {
		if pos >= len(data) {
			return nil, offset, fmt.Errorf("truncated point count")
		}
		count = (count&pointRunCountMask)<<8 | int(data[pos])
		pos++
	}
	if count == 0 {
		return nil, pos, nil
	}
	if count > MaxPointDeltaCount {
		return nil, offset, fmt.Errorf("point count %d exceeds limit %d", count, MaxPointDeltaCount)
	}

	result := make([]uint16, 0, count)
	for len(result) < count {
		if pos >= len(data) {
			return nil, offset, fmt.Errorf("truncated point run header")
		}
		runHeader := data[pos]
		pos++
		runCount := int(runHeader&pointRunCountMask) + 1
		if len(result)+runCount > count {
			return nil, offset, fmt.Errorf("point run exceeds total count")
		}
		if runHeader&pointsAreWords != 0 {
			if pos+runCount*2 > len(data) {
				return nil, offset, fmt.Errorf("truncated 16-bit point run")
			}
			for j := 0; j < runCount; j++ {
				result = append(result, readU16(data, pos))
				pos += 2
			}
		} else {
			if pos+runCount > len(data) {
				return nil, offset, fmt.Errorf("truncated 8-bit point run")
			}
			for j := 0; j < runCount; j++ {
				result = append(result, uint16(data[pos]))
				pos++
			}
		}
	}

	current := uint16(0)
	for i := range result {
		current += result[i]
		result[i] = current
	}

	return result, pos, nil
}

func decompileDeltas(data []byte, offset int, pointNumbers []uint16) ([]int16, int, error) {
	numDeltas := 0
	if pointNumbers != nil {
		numDeltas = len(pointNumbers)
	}

	pos := offset
	result := make([]int16, 0)
	for numDeltas == 0 || len(result) < numDeltas {
		if pos >= len(data) {
			if numDeltas == 0 {
				break
			}
			return nil, offset, fmt.Errorf("truncated delta run header")
		}
		runHeader := data[pos]
		pos++
		runCount := int(runHeader&deltaRunCountMask) + 1
		if numDeltas > 0 && len(result)+runCount > numDeltas {
			return nil, offset, fmt.Errorf("delta run exceeds expected count %d", numDeltas)
		}

		switch runHeader & deltasSizeMask {
		case deltasAreZero:
			for j := 0; j < runCount; j++ {
				result = append(result, 0)
			}
		case deltasAreWords:
			if pos+runCount*2 > len(data) {
				return nil, offset, fmt.Errorf("truncated 16-bit delta run")
			}
			for j := 0; j < runCount; j++ {
				result = append(result, int16(readU16(data, pos)))
				pos += 2
			}
		default:
			if runHeader&deltasSizeMask == 0xC0 {
				if pos+runCount*4 > len(data) {
					return nil, offset, fmt.Errorf("truncated 32-bit delta run")
				}
				for j := 0; j < runCount; j++ {
					result = append(result, int16(int32(readU32(data, pos))))
					pos += 4
				}
			} else {
				if pos+runCount > len(data) {
					return nil, offset, fmt.Errorf("truncated 8-bit delta run")
				}
				for j := 0; j < runCount; j++ {
					result = append(result, int16(int8(data[pos])))
					pos++
				}
			}
		}

		if numDeltas == 0 && pos >= len(data) {
			break
		}
	}

	return result, pos, nil
}

func readU16(data []byte, offset int) uint16 {
	return binary.BigEndian.Uint16(data[offset : offset+2])
}

func readU32(data []byte, offset int) uint32 {
	return binary.BigEndian.Uint32(data[offset : offset+4])
}
