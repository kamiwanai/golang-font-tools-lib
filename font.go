// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/golang-font-tools-lib/avar"
	"github.com/kamiwanai/golang-font-tools-lib/fvar"
	"github.com/kamiwanai/golang-font-tools-lib/gdef"
	"github.com/kamiwanai/golang-font-tools-lib/gpos"
	"github.com/kamiwanai/golang-font-tools-lib/gsub"
	"github.com/kamiwanai/golang-font-tools-lib/gvar"
	"github.com/kamiwanai/golang-font-tools-lib/hvar"
	"github.com/kamiwanai/golang-font-tools-lib/mvar"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
	"github.com/kamiwanai/golang-font-tools-lib/stat"
	"github.com/kamiwanai/golang-font-tools-lib/vvar"
)

// Font wraps a parsed OpenType font and provides typed table extractors as methods.
// All opentype.Font methods (Head, Hhea, OS2, GlyphOrder, etc.) are promoted through embedding.
type Font struct {
	*opentype.Font
}

// ParseFile opens and parses an OpenType font file.
func ParseFile(path string) (*Font, error) {
	f, err := opentype.ParseFile(path)
	if err != nil { return nil, err }
	return &Font{Font: f}, nil
}

// Parse parses font data from a byte slice.
func Parse(data []byte) (*Font, error) {
	f, err := opentype.Parse(data)
	if err != nil { return nil, err }
	return &Font{Font: f}, nil
}

// --- GPOS: Pair Positioning (kerning) ---

// GPOSKerning returns all supported GPOS PairPos kerning pairs.
func (f *Font) GPOSKerning() ([]gpos.PairValue, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := f.GPOS()
	if err != nil {
		return nil, err
	}
	return gpos.DecodePairPosLookups(gposData, glyphOrder)
}

// GPOSSinglePos returns all GPOS SinglePos (lookup type 1) positioning adjustments.
func (f *Font) GPOSSinglePos() ([]gpos.SinglePosValue, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := f.GPOS()
	if err != nil {
		return nil, err
	}
	return gpos.DecodeSinglePosLookups(gposData, glyphOrder)
}


// GPOSCursivePos returns all GPOS CursivePos (type 3) lookups.
func (f *Font) GPOSCursivePos() ([]gpos.CursiveEntry, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := f.GPOS()
	if err != nil {
		return nil, err
	}
	return gpos.DecodeCursivePosLookups(gposData, glyphOrder)
}

// GPOSMarkAttachments returns all GPOS mark attachments (MarkBasePos, MarkLigaturePos, MarkMarkPos).
func (f *Font) GPOSMarkAttachments() ([]gpos.MarkAttachment, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil {
		return nil, err
	}
	gposData, err := f.GPOS()
	if err != nil {
		return nil, err
	}
	return gpos.DecodeMarkLookups(gposData, glyphOrder)
}

// GPOSContextPos returns ContextPos (type 7) lookups.
func (f *Font) GPOSContextPos() ([]gpos.ContextPosLookup, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gposData, err := f.GPOS()
	if err != nil { return nil, err }
	return gpos.DecodeContextPosLookups(gposData, glyphOrder)
}

// GPOSChainContextPos returns ChainContextPos (type 8) lookups.
func (f *Font) GPOSChainContextPos() ([]gpos.ContextPosLookup, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gposData, err := f.GPOS()
	if err != nil { return nil, err }
	return gpos.DecodeChainContextPosLookups(gposData, glyphOrder)
}

// GPOSContextValueRecords extracts X/Y placement from nested lookups in ContextPos (type 7).
func (f *Font) GPOSContextValueRecords() ([]gpos.ContextValueRecord, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gposData, err := f.GPOS()
	if err != nil { return nil, err }
	return gpos.DecodeContextValueRecords(gposData, glyphOrder, 7)
}

// GPOSChainContextValueRecords extracts X/Y placement from nested lookups in ChainContextPos (type 8).
func (f *Font) GPOSChainContextValueRecords() ([]gpos.ContextValueRecord, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gposData, err := f.GPOS()
	if err != nil { return nil, err }
	return gpos.DecodeContextValueRecords(gposData, glyphOrder, 8)
}

// --- GSUB: Glyph Substitution ---

// GSUBSingleSubst returns all SingleSubst rules.
func (f *Font) GSUBSingleSubst() ([]gsub.SingleSubst, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.DecodeSingleSubstLookups(gsubData, glyphOrder)
}

// GSUBLigatureSubst returns all LigatureSubst rules.
func (f *Font) GSUBLigatureSubst() ([]gsub.LigatureSubst, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.DecodeLigatureSubstLookups(gsubData, glyphOrder)
}

// GSUBMultipleSubst returns all MultipleSubst rules.
func (f *Font) GSUBMultipleSubst() ([]gsub.MultipleSubst, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.DecodeMultipleSubstLookups(gsubData, glyphOrder)
}

// GSUBAlternateSubst returns all AlternateSubst rules.
func (f *Font) GSUBAlternateSubst() ([]gsub.AlternateSubst, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.DecodeAlternateSubstLookups(gsubData, glyphOrder)
}

// GSUBScripts returns the script list from the GSUB table.
func (f *Font) GSUBScripts() ([]gsub.Script, error) {
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.Scripts(gsubData)
}

// GSUBFeatures returns the feature list from the GSUB table.
func (f *Font) GSUBFeatures() ([]gsub.Feature, error) {
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.Features(gsubData)
}

// GSUBActiveLookups returns lookup indices for a script/language/feature combination.
func (f *Font) GSUBActiveLookups(scriptTag, langTag, featureTag string) ([]uint16, error) {
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.ActiveLookups(gsubData, scriptTag, langTag, featureTag)
}

// GSUBContextSubst returns ContextSubst (type 5) lookups.
func (f *Font) GSUBContextSubst() ([]gsub.ContextSubstLookup, error) {
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	return gsub.DecodeContextSubstLookups(gsubData, glyphOrder)
}

// GSUBChainContextSubst returns ChainContextSubst (type 6) lookups.
func (f *Font) GSUBChainContextSubst() ([]gsub.ContextSubstLookup, error) {
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	return gsub.DecodeChainContextSubstLookups(gsubData, glyphOrder)
}

// GSUBReverseSubst returns ReverseChainSingleSubst (type 8) lookups.
func (f *Font) GSUBReverseSubst() ([]gsub.ReverseSubst, error) {
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gsubData, err := f.GSUB()
	if err != nil { return nil, err }
	return gsub.DecodeReverseSubstLookups(gsubData, glyphOrder)
}

// --- GDEF: Glyph Definition ---

// GDEFTable returns the parsed GDEF table.
func (f *Font) GDEFTable() (gdef.GDEF, error) {
	gdefData, err := f.GDEF()
	if err != nil { return gdef.GDEF{}, err }
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return gdef.GDEF{}, err }
	return gdef.Decode(gdefData, glyphOrder)
}

// GDEFGlyphClasses returns glyph ID to GDEF class mapping (1=Base, 2=Ligature, 3=Mark, 4=Component).
func (f *Font) GDEFGlyphClasses() (map[uint16]uint16, error) {
	gdefData, err := f.GDEF()
	if err != nil { return nil, err }
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gd, err := gdef.Decode(gdefData, glyphOrder)
	if err != nil { return nil, err }
	return gd.GlyphClassMap, nil
}

// GDEFGlyphClassesByName returns glyph name to GDEF class mapping.
func (f *Font) GDEFGlyphClassesByName() (map[string]uint16, error) {
	gdefData, err := f.GDEF()
	if err != nil { return nil, err }
	glyphOrder, err := f.GlyphOrder()
	if err != nil { return nil, err }
	gd, err := gdef.Decode(gdefData, glyphOrder)
	if err != nil { return nil, err }
	return gd.ClassDefByNames(glyphOrder), nil
}

// --- FVAR: Font Variations ---

// FVARTable returns the parsed fvar (Font Variations) table.
func (f *Font) FVARTable() (fvar.FVAR, error) {
	fvarData, err := f.FVAR()
	if err != nil { return fvar.FVAR{}, err }
	return fvar.Decode(fvarData)
}

// FVARAxes returns the variation axes from the fvar table.
func (f *Font) FVARAxes() ([]fvar.Axis, error) {
	fvarData, err := f.FVAR()
	if err != nil { return nil, err }
	fv, err := fvar.Decode(fvarData)
	if err != nil { return nil, err }
	return fv.Axes, nil
}

// FVARInstances returns the named instances from the fvar table.
func (f *Font) FVARInstances() ([]fvar.Instance, error) {
	fvarData, err := f.FVAR()
	if err != nil { return nil, err }
	fv, err := fvar.Decode(fvarData)
	if err != nil { return nil, err }
	return fv.Instances, nil
}

// --- Other Variation Tables ---

// GVARTable returns the parsed gvar (Glyph Variations) table.
func (f *Font) GVARTable() (gvar.GVAR, error) {
	gvarData, err := f.TableData("gvar")
	if err != nil { return gvar.GVAR{}, err }
	numGlyphs, err := f.NumGlyphs()
	if err != nil { return gvar.GVAR{}, err }
	return gvar.Decode(gvarData, numGlyphs)
}

// STATTable returns the parsed STAT (Style Attributes) table.
func (f *Font) STATTable() (stat.STAT, error) {
	statData, err := f.TableData("STAT")
	if err != nil { return stat.STAT{}, err }
	return stat.Decode(statData)
}

// AVARTable returns the parsed avar (Axis Variations) table.
func (f *Font) AVARTable() (avar.AVAR, error) {
	avarData, err := f.TableData("avar")
	if err != nil { return avar.AVAR{}, err }
	return avar.Decode(avarData)
}

// HVARTable returns the parsed HVAR (Horizontal Metrics Variations) table.
func (f *Font) HVARTable() (hvar.HVAR, error) {
	hvarData, err := f.TableData("HVAR")
	if err != nil { return hvar.HVAR{}, err }
	return hvar.Decode(hvarData)
}

// VVARTable returns the parsed VVAR (Vertical Metrics Variations) table.
func (f *Font) VVARTable() (vvar.VVAR, error) {
	vvarData, err := f.TableData("VVAR")
	if err != nil { return vvar.VVAR{}, err }
	return vvar.Decode(vvarData)
}

// MVARTable returns the parsed MVAR (Metrics Variations) table.
func (f *Font) MVARTable() (mvar.MVAR, error) {
	mvarData, err := f.MVAR()
	if err != nil { return mvar.MVAR{}, err }
	return mvar.Decode(mvarData)
}

// --- Convenience wrappers (backward compatible) ---
// Each Extract*(path) parses the file and delegates to the Font method.
// New code should prefer ParseFile + method calls for single-parse efficiency.

func ExtractGPOSKerning(path string) ([]gpos.PairValue, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSKerning()
}

func ExtractGPOSSinglePos(path string) ([]gpos.SinglePosValue, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSSinglePos()
}

func ExtractGPOSCursivePos(path string) ([]gpos.CursiveEntry, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSCursivePos()
}

func ExtractGPOSMarkAttachments(path string) ([]gpos.MarkAttachment, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSMarkAttachments()
}

func ExtractGPOSContextPos(path string) ([]gpos.ContextPosLookup, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSContextPos()
}

func ExtractGPOSChainContextPos(path string) ([]gpos.ContextPosLookup, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSChainContextPos()
}

func ExtractContextValueRecords(path string) ([]gpos.ContextValueRecord, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSContextValueRecords()
}

func ExtractChainContextValueRecords(path string) ([]gpos.ContextValueRecord, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GPOSChainContextValueRecords()
}

func ExtractGSUBSingleSubst(path string) ([]gsub.SingleSubst, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBSingleSubst()
}

func ExtractGSUBLigatureSubst(path string) ([]gsub.LigatureSubst, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBLigatureSubst()
}

func ExtractGSUBMultipleSubst(path string) ([]gsub.MultipleSubst, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBMultipleSubst()
}

func ExtractGSUBAlternateSubst(path string) ([]gsub.AlternateSubst, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBAlternateSubst()
}

func ExtractGSUBScripts(path string) ([]gsub.Script, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBScripts()
}

func ExtractGSUBFeatures(path string) ([]gsub.Feature, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBFeatures()
}

func ExtractGSUBActiveLookups(path string, scriptTag, langTag, featureTag string) ([]uint16, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBActiveLookups(scriptTag, langTag, featureTag)
}

func ExtractGSUBContextSubst(path string) ([]gsub.ContextSubstLookup, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBContextSubst()
}

func ExtractGSUBChainContextSubst(path string) ([]gsub.ContextSubstLookup, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBChainContextSubst()
}

func ExtractGSUBReverseSubst(path string) ([]gsub.ReverseSubst, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GSUBReverseSubst()
}

func ExtractFVAR(path string) (fvar.FVAR, error) {
	font, err := ParseFile(path)
	if err != nil { return fvar.FVAR{}, err }
	return font.FVARTable()
}

func ExtractFVARAxes(path string) ([]fvar.Axis, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.FVARAxes()
}

func ExtractFVARInstances(path string) ([]fvar.Instance, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.FVARInstances()
}

func ExtractGDEF(path string) (gdef.GDEF, error) {
	font, err := ParseFile(path)
	if err != nil { return gdef.GDEF{}, err }
	return font.GDEFTable()
}

func ExtractGDEFGlyphClasses(path string) (map[uint16]uint16, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GDEFGlyphClasses()
}

func ExtractGDEFGlyphClassesByName(path string) (map[string]uint16, error) {
	font, err := ParseFile(path)
	if err != nil { return nil, err }
	return font.GDEFGlyphClassesByName()
}

func ExtractGVAR(path string) (gvar.GVAR, error) {
	font, err := ParseFile(path)
	if err != nil { return gvar.GVAR{}, err }
	return font.GVARTable()
}

func ExtractSTAT(path string) (stat.STAT, error) {
	font, err := ParseFile(path)
	if err != nil { return stat.STAT{}, err }
	return font.STATTable()
}

func ExtractAVAR(path string) (avar.AVAR, error) {
	font, err := ParseFile(path)
	if err != nil { return avar.AVAR{}, err }
	return font.AVARTable()
}

func ExtractHVAR(path string) (hvar.HVAR, error) {
	font, err := ParseFile(path)
	if err != nil { return hvar.HVAR{}, err }
	return font.HVARTable()
}

func ExtractVVAR(path string) (vvar.VVAR, error) {
	font, err := ParseFile(path)
	if err != nil { return vvar.VVAR{}, err }
	return font.VVARTable()
}

func ExtractMVAR(path string) (mvar.MVAR, error) {
	font, err := ParseFile(path)
	if err != nil { return mvar.MVAR{}, err }
	return font.MVARTable()
}
