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
