// Package fvar decodes the OpenType fvar (Font Variations) table.
//
// It supports:
//   - Variation axes (tag, min/default/max values, flags, name ID)
//   - Named instances (subfamily name, flags, coordinates, optional PostScript name ID)
//   - Axis lookup by tag (AxisByTag)
//
// The main entry point is Decode(data).
package fvar