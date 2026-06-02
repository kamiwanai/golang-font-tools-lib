// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package mvar

import "testing"

func FuzzDecode(f *testing.F) {
	f.Add([]byte{})
	// Minimal MVAR: version(2)+reserved(2)+valueRecordSize(2)+valueRecordCount(2)+storeOffset(2)+records
	f.Add([]byte{1, 0, 0, 0, 0, 8, 0, 0, 0, 10})

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := Decode(data)
		if err != nil {
			return
		}
	})
}
