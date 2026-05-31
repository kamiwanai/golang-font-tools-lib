// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package opentype

// MaxCmapGroupCodepoints limits the number of codepoints expanded from a
// single cmap format 12 group. Groups spanning more codepoints than this
// limit are rejected. The default is large enough for any valid font.
var MaxCmapGroupCodepoints int = 0x110000

// MaxCmapFormat4Segments limits the number of segments processed in a
// cmap format 4 subtable.
var MaxCmapFormat4Segments int = 65536

// MaxPostCustomNames limits the number of custom glyph names read from
// the post table.
var MaxPostCustomNames int = 65536

// MaxNameRecords limits the number of name records read from the name table.
var MaxNameRecords int = 65536

// MaxCollectionFonts limits the number of fonts parsed from a TrueType/OpenType
// Collection (.ttc/.otc). Collections exceeding this limit are rejected.
var MaxCollectionFonts int = 8192

// MaxCFF2FDRanges limits the number of ranges in a CFF2 FDSelect format 3/4.
var MaxCFF2FDRanges int = 65536
