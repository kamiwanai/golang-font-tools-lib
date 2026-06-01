// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package otfeatures

import "testing"

func TestEncodeLangSys(t *testing.T) {
	ls := LangSysDef{FeatureIndices: []uint16{0, 1, 2}}
	got := EncodeLangSys(ls)
	if len(got) != 6+3*2 {
		t.Fatalf("len = %d, want 12", len(got))
	}
	// lookupOrder = 0
	if got[0] != 0 || got[1] != 0 {
		t.Fatalf("lookupOrder = %v, want 0", got[0:2])
	}
	// reqFeatureIndex = 0xFFFF (since reqIdx==0 and features present)
	if got[2] != 0xFF || got[3] != 0xFF {
		t.Fatalf("reqFeatureIndex = %v, want 0xFFFF", got[2:4])
	}
	// featureIndexCount = 3
	if got[4] != 0 || got[5] != 3 {
		t.Fatalf("featureIndexCount = %v, want 3", got[4:6])
	}
}

func TestEncodeLangSysWithReqIdx(t *testing.T) {
	ls := LangSysDef{ReqFeatureIndex: 5, FeatureIndices: []uint16{1, 2}}
	got := EncodeLangSys(ls)
	if got[2] != 0 || got[3] != 5 {
		t.Fatalf("reqFeatureIndex = %v, want 5", got[2:4])
	}
}

func TestBuildTableRoundTrip(t *testing.T) {
	scripts := []ScriptDef{
		{
			Tag:            "latn",
			DefaultLangSys: &LangSysDef{FeatureIndices: []uint16{0}},
			Langs: []LangSysRecordDef{
				{Tag: "ENG ", LangSys: LangSysDef{FeatureIndices: []uint16{0, 1}}},
			},
		},
	}
	features := []FeatureDef{
		{Tag: "kern", LookupIndices: []uint16{0, 1}},
	}
	data := BuildTable(scripts, features)

	parsedScripts, err := DecodeScriptListOffset(data, 4, "TEST")
	if err != nil {
		t.Fatalf("DecodeScriptListOffset: %v", err)
	}
	if len(parsedScripts) != 1 {
		t.Fatalf("scripts = %d, want 1", len(parsedScripts))
	}
	if parsedScripts[0].Tag != "latn" {
		t.Fatalf("script tag = %q, want latn", parsedScripts[0].Tag)
	}

	parsedFeatures, err := DecodeFeatureListOffset(data, 6, "TEST")
	if err != nil {
		t.Fatalf("DecodeFeatureListOffset: %v", err)
	}
	if len(parsedFeatures) != 1 {
		t.Fatalf("features = %d, want 1", len(parsedFeatures))
	}
	if parsedFeatures[0].Tag != "kern" {
		t.Fatalf("feature tag = %q, want kern", parsedFeatures[0].Tag)
	}
}

func TestActiveLookupsMatchesExisting(t *testing.T) {
	scripts := []Script{
		{
			Tag: "latn",
			DefaultLangSys: &LangSys{
				ReqFeatureIndex: 0xFFFF,
				FeatureIndices:  []uint16{0},
			},
		},
	}
	features := []Feature{
		{Tag: "kern", LookupListIndices: []uint16{0, 1}},
	}
	lookups, err := ActiveLookups(scripts, features, "latn", "dflt", "kern")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(lookups) != 2 || lookups[0] != 0 || lookups[1] != 1 {
		t.Fatalf("lookups = %v, want [0 1]", lookups)
	}
}

func TestActiveLookupsNoMatch(t *testing.T) {
	scripts := []Script{
		{
			Tag:            "latn",
			DefaultLangSys: &LangSys{FeatureIndices: []uint16{0}},
		},
	}
	features := []Feature{
		{Tag: "kern", LookupListIndices: []uint16{0}},
	}
	lookups, err := ActiveLookups(scripts, features, "arab", "", "kern")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if lookups != nil {
		t.Fatalf("expected nil, got %v", lookups)
	}
}
