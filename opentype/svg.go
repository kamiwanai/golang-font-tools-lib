// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package opentype

import (
	"encoding/binary"
	"fmt"
)

type SVGDocument struct {
	StartGlyphID uint16
	EndGlyphID   uint16
	Data         string
}

func (font *Font) SVGDocuments() ([]SVGDocument, error) {
	data, err := font.TableData("SVG ")
	if err != nil { return nil, nil }
	if len(data) < 10 { return nil, fmt.Errorf("SVG table too short") }
	ver := binary.BigEndian.Uint16(data[0:])
	if ver != 0 { return nil, fmt.Errorf("unsupported SVG version %d", ver) }
	dlOff := int(binary.BigEndian.Uint32(data[2:]))
	if dlOff > len(data) { return nil, fmt.Errorf("SVG doc list offset exceeds table") }
	pos := dlOff
	if pos+2 > len(data) { return nil, fmt.Errorf("SVG doc list too short") }
	n := int(binary.BigEndian.Uint16(data[pos:]))
	pos += 2
	var docs []SVGDocument
	for i := 0; i < n; i++ {
		if pos+12 > len(data) { return nil, fmt.Errorf("SVG entry %d exceeds table", i) }
		sg := binary.BigEndian.Uint16(data[pos:])
		eg := binary.BigEndian.Uint16(data[pos+2:])
		so := int(binary.BigEndian.Uint32(data[pos+4:]))
		sl := int(binary.BigEndian.Uint32(data[pos+8:]))
		pos += 12
		if so+sl > len(data) { return nil, fmt.Errorf("SVG doc %d exceeds table", i) }
		svgStr := string(data[so : so+sl])
		d := SVGDocument{StartGlyphID: sg, EndGlyphID: eg, Data: svgStr}
		docs = append(docs, d)
	}
	return docs, nil
}