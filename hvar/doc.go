// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

// Package hvar decodes the OpenType HVAR (Horizontal Metrics Variations) table.
//
// The HVAR table provides variation data for horizontal metrics (advance widths,
// left/right side bearings) in variable fonts. It uses an ItemVariationStore for
// delta data and DeltaSetIndexMaps to map glyph indices to delta-set indices.
//
// Supported features:
//   - HVAR version 1.0, 1.1, 1.2
//   - ItemVariationStore with VariationRegionList and ItemVariationData
//   - DeltaSetIndexMap formats 0 and 1
//   - AdvanceWidthMapping, LsbMapping, RsbMapping
//
// The main entry point is Decode(data).
package hvar