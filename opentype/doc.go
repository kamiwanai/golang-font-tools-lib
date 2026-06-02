// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

// Package opentype reads a focused subset of OpenType font data.
//
// It parses table directories and exposes helpers for common metadata needed by
// kerning extraction: glyph count, glyph order, cmap mappings, name records,
// and key tables (head, hhea, hmtx, OS/2, kern).
//
// Font Collections (.ttc/.otc) are supported via ParseCollection and
// ParseCollectionFile. CFF glyph names are available for .otf fonts via
// CFFGlyphNames, and are used automatically by GlyphOrder when the post
// table does not provide names.
package opentype