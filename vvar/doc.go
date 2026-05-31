// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

// Package vvar decodes the OpenType VVAR (Vertical Metrics Variations) table.
//
// The VVAR table provides variation data for vertical metrics (advance heights,
// top/bottom side bearings, vertical origin) in variable fonts. It uses an
// ItemVariationStore for delta data and DeltaSetIndexMaps to map glyph indices
// to delta-set indices.
//
// Supported features:
//   - VVAR version 1.0, 1.1, 1.2, 1.3
//   - ItemVariationStore with VariationRegionList and ItemVariationData
//   - DeltaSetIndexMap formats 0 and 1
//   - AdvanceHeightMapping, TsbMapping, BsbMapping, VOrgMapping
//
// The main entry point is Decode(data).
package vvar