// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

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

// gsubEncodeLangSys builds raw bytes for one LangSys.
func gsubEncodeLangSys(ls gsubLangSysDef) []byte {
	return otfeatures.EncodeLangSys(ls)
}

// buildGSUBFeatures builds a GSUB-like table with the given scripts and features.
func buildGSUBFeatures(scripts []gsubScriptDef, features []gsubFeatureDef) []byte {
	return otfeatures.BuildTable(scripts, features)
}
