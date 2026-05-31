// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/fonttools/gpos"
	"github.com/kamiwanai/fonttools/opentype"
)

// ExtractGPOSMarkAttachments reads a font file and returns all GPOS mark
// attachments (MarkBasePos, MarkLigaturePos, MarkMarkPos).
//
// This is the primary entry point for resolving diacritic positioning:
// given "a" + "acutecomb", it returns the anchor points that tell a renderer
// where to attach the accent.
func ExtractGPOSMarkAttachments(path string) ([]gpos.MarkAttachment, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractGPOSMarkAttachmentsFromFont(font)
}

// ExtractGPOSMarkAttachmentsFromFont returns all GPOS mark attachments from
// an already parsed font.
func ExtractGPOSMarkAttachmentsFromFont(font *opentype.Font) ([]gpos.MarkAttachment, error) {
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		return nil, err
	}
	return gpos.DecodeMarkLookups(gposData, glyphOrder)
}
