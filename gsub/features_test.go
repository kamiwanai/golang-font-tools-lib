// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package gsub

import (
	"testing"
)

func TestScriptsParsesFromGSUB(t *testing.T) {
	data := buildGSUBFeatures(
		[]gsubScriptDef{
			{
				tag:            "latn",
				defaultLangSys: &gsubLangSysDef{featureIndices: []uint16{0}},
				langs: []gsubLangSysRecordDef{
					{tag: "ENG ", langSys: gsubLangSysDef{featureIndices: []uint16{0, 1}}},
				},
			},
		},
		[]gsubFeatureDef{
			{tag: "liga", lookupIndices: []uint16{0, 1}},
		},
	)
	scripts, err := Scripts(data)
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

func TestScriptsDefaultOnlyGSUB(t *testing.T) {
	data := buildGSUBFeatures(
		[]gsubScriptDef{
			{tag: "latn", defaultLangSys: &gsubLangSysDef{featureIndices: []uint16{0}}},
		},
		[]gsubFeatureDef{
			{tag: "liga", lookupIndices: []uint16{0, 1}},
		},
	)
	scripts, err := Scripts(data)
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

func TestFeaturesParsesFromGSUB(t *testing.T) {
	data := buildGSUBFeatures(
		[]gsubScriptDef{
			{tag: "latn", defaultLangSys: &gsubLangSysDef{featureIndices: []uint16{0}}},
		},
		[]gsubFeatureDef{
			{tag: "liga", lookupIndices: []uint16{0, 1}},
		},
	)
	features, err := Features(data)
	if err != nil {
		t.Fatalf("Features returned error: %v", err)
	}
	if len(features) != 1 {
		t.Fatalf("feature count = %d, want 1", len(features))
	}
	if features[0].Tag != "liga" {
		t.Fatalf("feature tag = %q, want %q", features[0].Tag, "liga")
	}
	if len(features[0].LookupListIndices) != 2 {
		t.Fatalf("lookup indices count = %d, want 2", len(features[0].LookupListIndices))
	}
}

func TestActiveLookupsForLigaFeature(t *testing.T) {
	data := buildGSUBFeatures(
		[]gsubScriptDef{
			{tag: "latn", defaultLangSys: &gsubLangSysDef{featureIndices: []uint16{0}}},
		},
		[]gsubFeatureDef{
			{tag: "liga", lookupIndices: []uint16{0, 1}},
		},
	)
	lookups, err := ActiveLookups(data, "latn", "dflt", "liga")
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

func TestActiveLookupsNoMatchGSUB(t *testing.T) {
	data := buildGSUBFeatures(
		[]gsubScriptDef{
			{tag: "latn", defaultLangSys: &gsubLangSysDef{featureIndices: []uint16{0}}},
		},
		[]gsubFeatureDef{
			{tag: "liga", lookupIndices: []uint16{0}},
		},
	)
	lookups, err := ActiveLookups(data, "arab", "", "liga")
	if err != nil {
		t.Fatalf("ActiveLookups returned error: %v", err)
	}
	if lookups != nil {
		t.Fatalf("expected nil lookups, got %v", lookups)
	}
}

func TestActiveLookupsWithReqFeatureGSUB(t *testing.T) {
	data := buildGSUBFeatures(
		[]gsubScriptDef{
			{
				tag:            "latn",
				defaultLangSys: &gsubLangSysDef{reqFeatureIndex: 0, featureIndices: []uint16{0}},
			},
		},
		[]gsubFeatureDef{
			{tag: "liga", lookupIndices: []uint16{5}},
		},
	)
	lookups, err := ActiveLookups(data, "latn", "dflt", "liga")
	if err != nil {
		t.Fatalf("ActiveLookups returned error: %v", err)
	}
	if len(lookups) != 1 || lookups[0] != 5 {
		t.Fatalf("lookups = %v, want [5]", lookups)
	}
}

func TestScriptsRejectsTruncatedGSUB(t *testing.T) {
	_, err := Scripts([]byte{0, 1, 0, 0})
	if err == nil {
		t.Fatal("Scripts succeeded for truncated data")
	}
}

func TestFeaturesRejectsTruncatedGSUB(t *testing.T) {
	_, err := Features([]byte{0, 1, 0, 0})
	if err == nil {
		t.Fatal("Features succeeded for truncated data")
	}
}