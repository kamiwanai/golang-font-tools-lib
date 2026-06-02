// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gsub

import (
	"github.com/kamiwanai/golang-font-tools-lib/internal/otfeatures"
)

// Public type aliases preserve the existing API. The canonical definitions
// live in internal/otfeatures.

type Script = otfeatures.Script
type LangSysRecord = otfeatures.LangSysRecord
type LangSys = otfeatures.LangSys
type Feature = otfeatures.Feature

const tableTag = "GSUB"

// Scripts returns the ScriptList parsed from a GSUB table.
//
// The caller can use this to filter substitution lookups by script and language.
func Scripts(data []byte) ([]Script, error) {
	return otfeatures.DecodeScriptListOffset(data, 4, tableTag)
}

// Features returns the FeatureList parsed from a GSUB table.
func Features(data []byte) ([]Feature, error) {
	return otfeatures.DecodeFeatureListOffset(data, 6, tableTag)
}

// FeatureMap returns the ScriptList and FeatureList from a GSUB table.
func FeatureMap(data []byte) ([]Script, []Feature, error) {
	scripts, err := Scripts(data)
	if err != nil {
		return nil, nil, err
	}
	features, err := Features(data)
	if err != nil {
		return nil, nil, err
	}
	return scripts, features, nil
}

// ActiveLookups returns the set of lookup indices active for a given
// feature tag under the first matching script/language.
//
// If scriptTag is empty, it uses the first script. If langTag is empty,
// it uses the default language system. Returns nil if no match found.
func ActiveLookups(data []byte, scriptTag, langTag, featureTag string) ([]uint16, error) {
	scripts, features, err := FeatureMap(data)
	if err != nil {
		return nil, err
	}
	return otfeatures.ActiveLookups(scripts, features, scriptTag, langTag, featureTag)
}
