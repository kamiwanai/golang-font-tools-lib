// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package fonttools

import "testing"

func TestExtractGPOSKerningReturnsParseError(t *testing.T) {
	_, err := ExtractGPOSKerning("does-not-exist.ttf")
	if err == nil {
		t.Fatal("ExtractGPOSKerning succeeded for missing file")
	}
}
