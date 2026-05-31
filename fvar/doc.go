// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

// Package fvar decodes the OpenType fvar (Font Variations) table.
//
// It supports:
//   - Variation axes (tag, min/default/max values, flags, name ID)
//   - Named instances (subfamily name, flags, coordinates, optional PostScript name ID)
//   - Axis lookup by tag (AxisByTag)
//
// The main entry point is Decode(data).
package fvar