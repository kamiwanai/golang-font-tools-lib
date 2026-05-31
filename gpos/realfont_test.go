// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gpos

import (
	"testing"

	"github.com/kamiwanai/fonttools/opentype"
	"golang.org/x/image/font/gofont/goregular"
)

func TestDecodeMarkLookupsFromGoRegular(t *testing.T) {
	font, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder: %v", err)
	}
	gposData, err := font.TableData("GPOS")
	if err != nil {
		t.Skip("no GPOS table in Go Regular")
	}
	attachments, err := DecodeMarkLookups(gposData, glyphOrder)
	if err != nil {
		t.Fatalf("DecodeMarkLookups: %v", err)
	}
	t.Logf("DecodeMarkLookups returned %d attachments", len(attachments))
}
