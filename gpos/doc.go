// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

// Package gpos decodes supported GPOS PairPos kerning subtables.
//
// It supports lookup type 2, extension lookup type 9 targeting PairPos, and
// PairPos formats 1 and 2. Decoded pairs are returned as expanded glyph-name
// pairs with optional source provenance.
package gpos
