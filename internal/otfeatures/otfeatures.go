// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

// Package otfeatures decodes OpenType ScriptList/FeatureList/LangSys tables.
//
// The same layout is used by both GPOS and GSUB, so this code is shared
// between them. The caller passes a tag (e.g. "GPOS" or "GSUB") used in
// error messages to identify which table the error came from.
package otfeatures

import (
	"encoding/binary"
	"fmt"
)

// Script represents one script record from the ScriptList.
type Script struct {
	// Tag is the 4-character script tag (e.g. "latn", "arab", "cyrl").
	Tag string
	// DefaultLangSys is the default language system for this script, or nil
	// if the script has no default language.
	DefaultLangSys *LangSys
	// LangSys is the list of language-specific systems for this script.
	LangSys []LangSysRecord
}

// LangSysRecord associates a language tag with a LangSys.
type LangSysRecord struct {
	// Tag is the 4-character language tag (e.g. "dflt", "ENG ", "FRA ").
	Tag string
	// LangSys is the language system.
	LangSys LangSys
}

// LangSys describes one language system: which features are active.
type LangSys struct {
	// LookupOrder is reserved (always nil in practice).
	LookupOrder uint16
	// ReqFeatureIndex is the required feature index (0xFFFF if none).
	ReqFeatureIndex uint16
	// FeatureIndices lists feature indices active for this language.
	FeatureIndices []uint16
}

// Feature represents one feature record from the FeatureList.
type Feature struct {
	// Tag is the 4-character feature tag (e.g. "kern", "liga").
	Tag string
	// LookupListIndices lists lookup indices this feature activates.
	LookupListIndices []uint16
}

// DecodeScripts parses the ScriptList from data at the given offset.
// The tag argument is used in error messages (e.g. "GPOS" or "GSUB").
func DecodeScripts(data []byte, offset int, tag string) ([]Script, error) {
	return decodeScriptList(data, offset, tag)
}

// DecodeFeatures parses the FeatureList from data at the given offset.
// The tag argument is used in error messages (e.g. "GPOS" or "GSUB").
func DecodeFeatures(data []byte, offset int, tag string) ([]Feature, error) {
	return decodeFeatureList(data, offset, tag)
}

// DecodeScriptListOffset reads the ScriptList offset at the given header
// position (typically header offset 4) and returns the parsed scripts.
func DecodeScriptListOffset(data []byte, headerOffset int, tag string) ([]Script, error) {
	if len(data) < headerOffset+2 {
		return nil, fmt.Errorf("%s header too short: %d bytes", tag, len(data))
	}
	scriptListOffset := int(binary.BigEndian.Uint16(data[headerOffset : headerOffset+2]))
	return decodeScriptList(data, scriptListOffset, tag)
}

// DecodeFeatureListOffset reads the FeatureList offset at the given header
// position (typically header offset 6) and returns the parsed features.
func DecodeFeatureListOffset(data []byte, headerOffset int, tag string) ([]Feature, error) {
	if len(data) < headerOffset+2 {
		return nil, fmt.Errorf("%s header too short: %d bytes", tag, len(data))
	}
	featureListOffset := int(binary.BigEndian.Uint16(data[headerOffset : headerOffset+2]))
	return decodeFeatureList(data, featureListOffset, tag)
}

// ActiveLookups returns the set of lookup indices active for a given
// feature tag under the first matching script/language.
//
// If scriptTag is empty, it uses the first script. If langTag is empty,
// it uses the default language system. Returns nil if no match found.
func ActiveLookups(scripts []Script, features []Feature, scriptTag, langTag, featureTag string) ([]uint16, error) {
	// Find the script.
	var langSys *LangSys
	for _, s := range scripts {
		if scriptTag != "" && s.Tag != scriptTag {
			continue
		}
		if langTag == "" || langTag == "dflt" {
			if s.DefaultLangSys != nil {
				langSys = s.DefaultLangSys
				break
			}
		}
		for _, lr := range s.LangSys {
			if lr.Tag == langTag {
				langSys = &lr.LangSys
				break
			}
		}
		if langSys != nil {
			break
		}
	}
	if langSys == nil {
		return nil, nil
	}

	// Collect lookup indices from matching features.
	var lookups []uint16
	seen := make(map[uint16]bool)
	featureIndices := langSys.FeatureIndices
	if langSys.ReqFeatureIndex != 0xFFFF && langSys.ReqFeatureIndex < uint16(len(features)) {
		featureIndices = append([]uint16{langSys.ReqFeatureIndex}, featureIndices...)
	}
	for _, fi := range featureIndices {
		if int(fi) >= len(features) {
			continue
		}
		f := features[fi]
		if featureTag != "" && f.Tag != featureTag {
			continue
		}
		for _, li := range f.LookupListIndices {
			if !seen[li] {
				seen[li] = true
				lookups = append(lookups, li)
			}
		}
	}
	return lookups, nil
}

func decodeScriptList(data []byte, offset int, tag string) ([]Script, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("%s ScriptList offset exceeds table", tag)
	}
	scriptCount := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	if offset+2+scriptCount*6 > len(data) {
		return nil, fmt.Errorf("%s ScriptList records exceed table", tag)
	}
	scripts := make([]Script, 0, scriptCount)
	for i := 0; i < scriptCount; i++ {
		recOffset := offset + 2 + i*6
		tag := string(data[recOffset : recOffset+4])
		scriptOffset := int(binary.BigEndian.Uint16(data[recOffset+4 : recOffset+6]))
		script, err := decodeScript(data, offset+scriptOffset, tag)
		if err != nil {
			return nil, err
		}
		script.Tag = tag
		scripts = append(scripts, script)
	}
	return scripts, nil
}

func decodeScript(data []byte, offset int, tag string) (Script, error) {
	if offset+4 > len(data) {
		return Script{}, fmt.Errorf("%s Script exceeds table", tag)
	}
	script := Script{}
	defaultLangSysOffset := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	if defaultLangSysOffset != 0 {
		ls, err := decodeLangSys(data, offset+defaultLangSysOffset, tag)
		if err != nil {
			return Script{}, err
		}
		script.DefaultLangSys = &ls
	}
	langSysCount := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
	if offset+4+langSysCount*6 > len(data) {
		return Script{}, fmt.Errorf("%s LangSysRecords exceed table", tag)
	}
	script.LangSys = make([]LangSysRecord, 0, langSysCount)
	for i := 0; i < langSysCount; i++ {
		recOffset := offset + 4 + i*6
		tag := string(data[recOffset : recOffset+4])
		langSysOffset := int(binary.BigEndian.Uint16(data[recOffset+4 : recOffset+6]))
		ls, err := decodeLangSys(data, offset+langSysOffset, tag)
		if err != nil {
			return Script{}, err
		}
		script.LangSys = append(script.LangSys, LangSysRecord{Tag: tag, LangSys: ls})
	}
	return script, nil
}

func decodeLangSys(data []byte, offset int, tag string) (LangSys, error) {
	if offset+6 > len(data) {
		return LangSys{}, fmt.Errorf("%s LangSys exceeds table", tag)
	}
	lookupOrder := binary.BigEndian.Uint16(data[offset : offset+2])
	reqFeatureIndex := binary.BigEndian.Uint16(data[offset+2 : offset+4])
	featureIndexCount := int(binary.BigEndian.Uint16(data[offset+4 : offset+6]))
	if offset+6+featureIndexCount*2 > len(data) {
		return LangSys{}, fmt.Errorf("%s LangSys feature indices exceed table", tag)
	}
	indices := make([]uint16, featureIndexCount)
	for i := 0; i < featureIndexCount; i++ {
		indices[i] = binary.BigEndian.Uint16(data[offset+6+i*2 : offset+8+i*2])
	}
	return LangSys{
		LookupOrder:      lookupOrder,
		ReqFeatureIndex: reqFeatureIndex,
		FeatureIndices:   indices,
	}, nil
}

func decodeFeatureList(data []byte, offset int, tag string) ([]Feature, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("%s FeatureList offset exceeds table", tag)
	}
	featureCount := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	if offset+2+featureCount*6 > len(data) {
		return nil, fmt.Errorf("%s FeatureList records exceed table", tag)
	}
	features := make([]Feature, 0, featureCount)
	for i := 0; i < featureCount; i++ {
		recOffset := offset + 2 + i*6
		tag := string(data[recOffset : recOffset+4])
		featureOffset := int(binary.BigEndian.Uint16(data[recOffset+4 : recOffset+6]))
		feature, err := decodeFeature(data, offset+featureOffset, tag)
		if err != nil {
			return nil, err
		}
		feature.Tag = tag
		features = append(features, feature)
	}
	return features, nil
}

func decodeFeature(data []byte, offset int, tag string) (Feature, error) {
	if offset+4 > len(data) {
		return Feature{}, fmt.Errorf("%s Feature exceeds table", tag)
	}
	lookupCount := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
	if offset+4+lookupCount*2 > len(data) {
		return Feature{}, fmt.Errorf("%s Feature lookup indices exceed table", tag)
	}
	indices := make([]uint16, lookupCount)
	for i := 0; i < lookupCount; i++ {
		indices[i] = binary.BigEndian.Uint16(data[offset+4+i*2 : offset+6+i*2])
	}
	return Feature{LookupListIndices: indices}, nil
}
