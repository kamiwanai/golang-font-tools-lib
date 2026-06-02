// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package gpos

import "github.com/kamiwanai/golang-font-tools-lib/internal/otfeatures"

// Type aliases for the shared test builder. The canonical definitions live in
// internal/otfeatures.
type (
	langSysDef       = otfeatures.LangSysDef
	langSysRecordDef = otfeatures.LangSysRecordDef
	scriptDef        = otfeatures.ScriptDef
	featureDef       = otfeatures.FeatureDef
)

// buildGPOSFeatures builds a GPOS-like table with the given scripts and features.
func buildGPOSFeatures(scripts []scriptDef, features []featureDef) []byte {
	return otfeatures.BuildTable(scripts, features)
}
