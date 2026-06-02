// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

// Package mvar decodes the OpenType MVAR (Metrics Variations) table.
//
// The MVAR table provides variation data for font-wide metrics in variable
// fonts. It uses an ItemVariationStore to hold delta data and ValueRecords
// that map metric tags (e.g., "hasc", "xhgt") to delta-set indices.
//
// Supported features:
//   - MVAR version 1.0
//   - ItemVariationStore with VariationRegionList and ItemVariationData
//   - Binary search for ValueRecord lookup by tag
//
// The main entry point is Decode(data).
package mvar
