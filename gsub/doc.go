// Package gsub decodes supported GSUB substitution lookups from OpenType fonts.
//
// It supports lookup type 1 (SingleSubst), type 2 (MultipleSubst),
// type 3 (AlternateSubst), type 4 (LigatureSubst), and extension lookup type 9.
// It also provides Script, Feature, and ActiveLookups filtering via the
// GSUB ScriptList/FeatureList.
package gsub