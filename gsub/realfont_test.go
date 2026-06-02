// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gsub

import (
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

func TestGSUBGoRegularFixture(t *testing.T) {
	font, err := opentype.Parse(goregular.TTF)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	gsubData, err := font.GSUB()
	if err != nil {
		// Go Regular may not have a GSUB table — skip in that case
		t.Skipf("font has no GSUB table: %v", err)
	}

	glyphOrder, err := font.GlyphOrder()
	if err != nil {
		t.Fatalf("GlyphOrder returned error: %v", err)
	}

	t.Run("SingleSubst", func(t *testing.T) {
		subs, err := DecodeSingleSubstLookups(gsubData, glyphOrder)
		if err != nil {
			t.Fatalf("DecodeSingleSubstLookups error: %v", err)
		}
		// Just verify it doesn't crash and produces valid data
		for _, s := range subs {
			if s.From == "" || s.To == "" {
				t.Errorf("empty glyph name in SingleSubst: From=%q To=%q", s.From, s.To)
			}
		}
		t.Logf("SingleSubst: %d rules", len(subs))
	})

	t.Run("LigatureSubst", func(t *testing.T) {
		ligs, err := DecodeLigatureSubstLookups(gsubData, glyphOrder)
		if err != nil {
			t.Fatalf("DecodeLigatureSubstLookups error: %v", err)
		}
		for _, l := range ligs {
			if len(l.Components) < 2 {
				t.Errorf("LigatureSubst with <2 components: %v", l)
			}
		}
		t.Logf("LigatureSubst: %d rules", len(ligs))
	})

	t.Run("MultipleSubst", func(t *testing.T) {
		subs, err := DecodeMultipleSubstLookups(gsubData, glyphOrder)
		if err != nil {
			t.Fatalf("DecodeMultipleSubstLookups error: %v", err)
		}
		for _, s := range subs {
			if len(s.To) == 0 {
				t.Errorf("MultipleSubst with empty To: From=%s", s.From)
			}
		}
		t.Logf("MultipleSubst: %d rules", len(subs))
	})

	t.Run("AlternateSubst", func(t *testing.T) {
		alts, err := DecodeAlternateSubstLookups(gsubData, glyphOrder)
		if err != nil {
			t.Fatalf("DecodeAlternateSubstLookups error: %v", err)
		}
		for _, a := range alts {
			if len(a.Alternates) == 0 {
				t.Errorf("AlternateSubst with empty Alternates: From=%s", a.From)
			}
		}
		t.Logf("AlternateSubst: %d rules", len(alts))
	})

	t.Run("Scripts", func(t *testing.T) {
		scripts, err := Scripts(gsubData)
		if err != nil {
			t.Fatalf("Scripts error: %v", err)
		}
		t.Logf("Scripts: %d entries", len(scripts))
	})

	t.Run("Features", func(t *testing.T) {
		features, err := Features(gsubData)
		if err != nil {
			t.Fatalf("Features error: %v", err)
		}
		t.Logf("Features: %d entries", len(features))
		for _, f := range features {
			t.Logf("  feature %q → lookups %v", f.Tag, f.LookupListIndices)
		}
	})

	t.Run("ActiveLookups", func(t *testing.T) {
		lookups, err := ActiveLookups(gsubData, "latn", "dflt", "liga")
		if err != nil {
			t.Fatalf("ActiveLookups error: %v", err)
		}
		t.Logf("ActiveLookups(latn, dflt, liga): %v", lookups)
	})
}