// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gsub

import "github.com/kamiwanai/golang-font-tools-lib/internal/otfeatures"

// Type aliases for the shared test builder. The canonical definitions live in
// internal/otfeatures.
type (
	gsubLangSysDef       = otfeatures.LangSysDef
	gsubLangSysRecordDef = otfeatures.LangSysRecordDef
	gsubScriptDef        = otfeatures.ScriptDef
	gsubFeatureDef       = otfeatures.FeatureDef
)

// buildGSUBFeatures builds a GSUB-like table with the given scripts and features.
func buildGSUBFeatures(scripts []gsubScriptDef, features []gsubFeatureDef) []byte {
	return otfeatures.BuildTable(scripts, features)
}
