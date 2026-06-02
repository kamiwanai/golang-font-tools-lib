// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

// Package fonttools provides a high-level API for OpenType font inspection.
//
// The Font type wraps a parsed OpenType font and exposes typed table
// extractors as methods. Parse once, then call any number of extractors
// without repeated file I/O:
//
//	font, _ := fonttools.ParseFile("FiraCode-Regular.ttf")
//	kerning, _ := font.GPOSKerning()
//	axes, _ := font.FVARAxes()
//
// Lower-level table access is available through the subpackages (opentype,
// gpos, gsub, fvar, etc.).
package fonttools
