package gsub

import (
	"fmt"
)

// Script represents one script record from the GSUB ScriptList.
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

// Feature represents one feature record from the GSUB FeatureList.
type Feature struct {
	// Tag is the 4-character feature tag (e.g. "liga", "salt", "dlig").
	Tag string
	// LookupListIndices lists lookup indices this feature activates.
	LookupListIndices []uint16
}

// Scripts returns the ScriptList parsed from a GSUB table.
//
// The caller can use this to filter substitution lookups by script and language.
func Scripts(data []byte) ([]Script, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("GSUB header too short: %d bytes", len(data))
	}
	scriptListOffset := int(readU16(data, 4))
	return decodeScriptList(data, scriptListOffset)
}

// Features returns the FeatureList parsed from a GSUB table.
func Features(data []byte) ([]Feature, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("GSUB header too short: %d bytes", len(data))
	}
	featureListOffset := int(readU16(data, 6))
	return decodeFeatureList(data, featureListOffset)
}

// FeatureMap returns a mapping from feature tag to lookup indices derived from
// the GSUB ScriptList/FeatureList/LookupList. This allows the caller to ask
// "which lookups are active for script=latn, language=dflt?".
//
// It returns: scripts, features, error.
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

func decodeScriptList(data []byte, offset int) ([]Script, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("GSUB ScriptList offset exceeds table")
	}
	scriptCount := int(readU16(data, offset))
	if offset+2+scriptCount*6 > len(data) {
		return nil, fmt.Errorf("GSUB ScriptList records exceed table")
	}
	scripts := make([]Script, 0, scriptCount)
	for i := 0; i < scriptCount; i++ {
		recOffset := offset + 2 + i*6
		tag := string(data[recOffset : recOffset+4])
		scriptOffset := int(readU16(data, recOffset+4))
		script, err := decodeScript(data, offset+scriptOffset)
		if err != nil {
			return nil, err
		}
		script.Tag = tag
		scripts = append(scripts, script)
	}
	return scripts, nil
}

func decodeScript(data []byte, offset int) (Script, error) {
	if offset+4 > len(data) {
		return Script{}, fmt.Errorf("GSUB Script exceeds table")
	}
	script := Script{}
	defaultLangSysOffset := int(readU16(data, offset))
	if defaultLangSysOffset != 0 {
		ls, err := decodeLangSys(data, offset+defaultLangSysOffset)
		if err != nil {
			return Script{}, err
		}
		script.DefaultLangSys = &ls
	}
	langSysCount := int(readU16(data, offset+2))
	if offset+4+langSysCount*6 > len(data) {
		return Script{}, fmt.Errorf("GSUB LangSysRecords exceed table")
	}
	script.LangSys = make([]LangSysRecord, 0, langSysCount)
	for i := 0; i < langSysCount; i++ {
		recOffset := offset + 4 + i*6
		tag := string(data[recOffset : recOffset+4])
		langSysOffset := int(readU16(data, recOffset+4))
		ls, err := decodeLangSys(data, offset+langSysOffset)
		if err != nil {
			return Script{}, err
		}
		script.LangSys = append(script.LangSys, LangSysRecord{Tag: tag, LangSys: ls})
	}
	return script, nil
}

func decodeLangSys(data []byte, offset int) (LangSys, error) {
	if offset+6 > len(data) {
		return LangSys{}, fmt.Errorf("GSUB LangSys exceeds table")
	}
	lookupOrder := readU16(data, offset)
	reqFeatureIndex := readU16(data, offset+2)
	featureIndexCount := int(readU16(data, offset+4))
	if offset+6+featureIndexCount*2 > len(data) {
		return LangSys{}, fmt.Errorf("GSUB LangSys feature indices exceed table")
	}
	indices := make([]uint16, featureIndexCount)
	for i := 0; i < featureIndexCount; i++ {
		indices[i] = readU16(data, offset+6+i*2)
	}
	return LangSys{
		LookupOrder:      lookupOrder,
		ReqFeatureIndex: reqFeatureIndex,
		FeatureIndices:   indices,
	}, nil
}

func decodeFeatureList(data []byte, offset int) ([]Feature, error) {
	if offset+2 > len(data) {
		return nil, fmt.Errorf("GSUB FeatureList offset exceeds table")
	}
	featureCount := int(readU16(data, offset))
	if offset+2+featureCount*6 > len(data) {
		return nil, fmt.Errorf("GSUB FeatureList records exceed table")
	}
	features := make([]Feature, 0, featureCount)
	for i := 0; i < featureCount; i++ {
		recOffset := offset + 2 + i*6
		tag := string(data[recOffset : recOffset+4])
		featureOffset := int(readU16(data, recOffset+4))
		feature, err := decodeFeature(data, offset+featureOffset)
		if err != nil {
			return nil, err
		}
		feature.Tag = tag
		features = append(features, feature)
	}
	return features, nil
}

func decodeFeature(data []byte, offset int) (Feature, error) {
	if offset+4 > len(data) {
		return Feature{}, fmt.Errorf("GSUB Feature exceeds table")
	}
	lookupCount := int(readU16(data, offset+2))
	if offset+4+lookupCount*2 > len(data) {
		return Feature{}, fmt.Errorf("GSUB Feature lookup indices exceed table")
	}
	indices := make([]uint16, lookupCount)
	for i := 0; i < lookupCount; i++ {
		indices[i] = readU16(data, offset+4+i*2)
	}
	return Feature{LookupListIndices: indices}, nil
}