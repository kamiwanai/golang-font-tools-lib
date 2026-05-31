// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import (
	"github.com/kamiwanai/golang-font-tools-lib/fvar"
	"github.com/kamiwanai/golang-font-tools-lib/opentype"
)

// ExtractFVAR reads a font file and returns the parsed fvar (Font Variations) table.
// Returns an error if the font has no fvar table or if decoding fails.
func ExtractFVAR(path string) (fvar.FVAR, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return fvar.FVAR{}, err
	}
	return ExtractFVARFromFont(font)
}

// ExtractFVARFromFont returns the parsed fvar table from an already parsed font.
// Returns an error if the font has no fvar table or if decoding fails.
func ExtractFVARFromFont(font *opentype.Font) (fvar.FVAR, error) {
	fvarData, err := font.FVAR()
	if err != nil {
		return fvar.FVAR{}, err
	}
	return fvar.Decode(fvarData)
}

// ExtractFVARAxes reads a font file and returns the variation axes.
// Returns nil if the font has no fvar table.
func ExtractFVARAxes(path string) ([]fvar.Axis, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractFVARAxesFromFont(font)
}

// ExtractFVARAxesFromFont returns the variation axes from an already parsed font.
// Returns nil if the font has no fvar table.
func ExtractFVARAxesFromFont(font *opentype.Font) ([]fvar.Axis, error) {
	fvarData, err := font.FVAR()
	if err != nil {
		return nil, nil
	}
	fv, err := fvar.Decode(fvarData)
	if err != nil {
		return nil, err
	}
	return fv.Axes, nil
}

// ExtractFVARInstances reads a font file and returns the named instances.
// Returns nil if the font has no fvar table.
func ExtractFVARInstances(path string) ([]fvar.Instance, error) {
	font, err := opentype.ParseFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractFVARInstancesFromFont(font)
}

// ExtractFVARInstancesFromFont returns the named instances from an already parsed font.
// Returns nil if the font has no fvar table.
func ExtractFVARInstancesFromFont(font *opentype.Font) ([]fvar.Instance, error) {
	fvarData, err := font.FVAR()
	if err != nil {
		return nil, nil
	}
	fv, err := fvar.Decode(fvarData)
	if err != nil {
		return nil, err
	}
	return fv.Instances, nil
}