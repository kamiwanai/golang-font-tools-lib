// Package gvar decodes the OpenType gvar (Glyph Variations) table.
//
// It supports:
//   - Header parsing (version, axis count, flags)
//   - Shared tuple records (F2DOT14 coordinates)
//   - Per-glyph variation data (tuple variation headers + serialized deltas)
//   - Packed point number decoding
//   - Packed delta decoding
//
// The main entry point is Decode(data, glyphCount).
package gvar
