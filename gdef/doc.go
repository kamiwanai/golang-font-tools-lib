// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

// Package gdef decodes the OpenType GDEF (Glyph Definition) table.
//
// It supports:
//   - GlyphClassDef (format 1 and format 2): maps glyph IDs to classes
//     (Base, Ligature, Mark, Component).
//   - AttachList: attachment points per glyph.
//   - LigCaretList: caret positions for ligature glyphs.
//   - MarkAttachClassDef (GDEF 1.2+): mark attachment class per glyph.
//   - MarkGlyphSetsDef (GDEF 1.3+): not decoded here yet.
//
// The main entry point is Decode(data, glyphOrder).
package gdef