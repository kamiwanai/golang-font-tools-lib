// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gpos

import (
	"testing"
)

func TestScriptsParsesFromGPOS(t *testing.T) {
	gpos := buildGPOSFeatures(
		[]scriptDef{
			{
				Tag:            "latn",
				DefaultLangSys: &langSysDef{FeatureIndices: []uint16{0}},
				Langs: []langSysRecordDef{
					{Tag: "ENG ", LangSys: langSysDef{FeatureIndices: []uint16{0, 1}}},
				},
			},
		},
		[]featureDef{
			{Tag: "kern", LookupIndices: []uint16{0, 1}},
		},
	)
	scripts, err := Scripts(gpos)
	if err != nil {
		t.Fatalf("Scripts returned error: %v", err)
	}
	if len(scripts) != 1 {
		t.Fatalf("script count = %d, want 1", len(scripts))
	}
	if scripts[0].Tag != "latn" {
		t.Fatalf("script tag = %q, want %q", scripts[0].Tag, "latn")
	}
	if scripts[0].DefaultLangSys == nil {
		t.Fatal("expected DefaultLangSys, got nil")
	}
	if len(scripts[0].DefaultLangSys.FeatureIndices) != 1 || scripts[0].DefaultLangSys.FeatureIndices[0] != 0 {
		t.Fatalf("DefaultLangSys.FeatureIndices = %v, want [0]", scripts[0].DefaultLangSys.FeatureIndices)
	}
	if len(scripts[0].LangSys) != 1 {
		t.Fatalf("LangSys count = %d, want 1", len(scripts[0].LangSys))
	}
	if scripts[0].LangSys[0].Tag != "ENG " {
		t.Fatalf("LangSys tag = %q, want %q", scripts[0].LangSys[0].Tag, "ENG ")
	}
	if len(scripts[0].LangSys[0].LangSys.FeatureIndices) != 2 {
		t.Fatalf("ENG FeatureIndices count = %d, want 2", len(scripts[0].LangSys[0].LangSys.FeatureIndices))
	}
}

func TestScriptsDefaultOnly(t *testing.T) {
	gpos := buildGPOSFeatures(
		[]scriptDef{
			{Tag: "latn", DefaultLangSys: &langSysDef{FeatureIndices: []uint16{0}}},
		},
		[]featureDef{
			{Tag: "kern", LookupIndices: []uint16{0, 1}},
		},
	)
	scripts, err := Scripts(gpos)
	if err != nil {
		t.Fatalf("Scripts returned error: %v", err)
	}
	if scripts[0].DefaultLangSys == nil {
		t.Fatal("expected DefaultLangSys, got nil")
	}
	if len(scripts[0].DefaultLangSys.FeatureIndices) != 1 {
		t.Fatalf("DefaultLangSys.FeatureIndices = %v, want [0]", scripts[0].DefaultLangSys.FeatureIndices)
	}
}

func TestFeaturesParsesFromGPOS(t *testing.T) {
	gpos := buildGPOSFeatures(
		[]scriptDef{
			{Tag: "latn", DefaultLangSys: &langSysDef{FeatureIndices: []uint16{0}}},
		},
		[]featureDef{
			{Tag: "kern", LookupIndices: []uint16{0, 1}},
		},
	)
	features, err := Features(gpos)
	if err != nil {
		t.Fatalf("Features returned error: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("feature count = %d, want 1", len(features))
	}
	if features[0].Tag != "kern" {
		t.Fatalf("feature tag = %q, want %q", features[0].Tag, "kern")
	}
	if len(features[0].LookupListIndices) != 2 {
		t.Fatalf("lookup indices count = %d, want 2", len(features[0].LookupListIndices))
	}
}

func TestActiveLookupsForKernFeature(t *testing.T) {
	gpos := buildGPOSFeatures(
		[]scriptDef{
			{Tag: "latn", DefaultLangSys: &langSysDef{FeatureIndices: []uint16{0}}},
		},
		[]featureDef{
			{Tag: "kern", LookupIndices: []uint16{0, 1}},
		},
	)
	lookups, err := ActiveLookups(gpos, "latn", "dflt", "kern")
	if err != nil {
		t.Fatalf("ActiveLookups returned error: %v", err)
	}
	if len(lookups) != 2 {
		t.Fatalf("lookups count = %d, want 2", len(lookups))
	}
	if lookups[0] != 0 || lookups[1] != 1 {
		t.Fatalf("lookups = %v, want [0, 1]", lookups)
	}
}

func TestActiveLookupsNoMatch(t *testing.T) {
	gpos := buildGPOSFeatures(
		[]scriptDef{
			{Tag: "latn", DefaultLangSys: &langSysDef{FeatureIndices: []uint16{0}}},
		},
		[]featureDef{
			{Tag: "kern", LookupIndices: []uint16{0}},
		},
	)
	lookups, err := ActiveLookups(gpos, "arab", "", "kern")
	if err != nil {
		t.Fatalf("ActiveLookups returned error: %v", err)
	}
	if lookups != nil {
		t.Fatalf("expected nil lookups, got %v", lookups)
	}
}

func TestActiveLookupsWithReqFeature(t *testing.T) {
	gpos := buildGPOSFeatures(
		[]scriptDef{
			{
				Tag:            "latn",
				DefaultLangSys: &langSysDef{ReqFeatureIndex: 0, FeatureIndices: []uint16{0}},
			},
		},
		[]featureDef{
			{Tag: "kern", LookupIndices: []uint16{5}},
		},
	)
	lookups, err := ActiveLookups(gpos, "latn", "dflt", "kern")
	if err != nil {
		t.Fatalf("ActiveLookups returned error: %v", err)
	}
	if len(lookups) != 1 || lookups[0] != 5 {
		t.Fatalf("lookups = %v, want [5]", lookups)
	}
}

func TestScriptsRejectsTruncated(t *testing.T) {
	_, err := Scripts([]byte{0, 1, 0, 0})
	if err == nil {
		t.Fatal("Scripts succeeded for truncated data")
	}
}

func TestFeaturesRejectsTruncated(t *testing.T) {
	_, err := Features([]byte{0, 1, 0, 0})
	if err == nil {
		t.Fatal("Features succeeded for truncated data")
	}
}
