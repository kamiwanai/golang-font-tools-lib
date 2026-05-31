// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package vvar

// MaxItemVariationDataCount limits ItemVariationData subtables.
var MaxItemVariationDataCount int = 4096

// MaxVariationAxes limits the number of axes per variation region.
var MaxVariationAxes int = 256

// MaxVariationRegions limits the number of variation regions.
var MaxVariationRegions int = 65536

// MaxDeltaItems limits the number of items in ItemVariationData.
var MaxDeltaItems int = 65536

// MaxRegionIndexes limits region index count in ItemVariationData.
var MaxRegionIndexes int = 65536

// MaxDeltaSetIndexEntries limits entries in a DeltaSetIndexMap.
var MaxDeltaSetIndexEntries int = 65536