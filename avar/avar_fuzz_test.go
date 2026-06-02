// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: inari1337@gmail.com

package avar

import "testing"

func FuzzDecode(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 0, 0, 0, 0}) // major=1, minor=0, reserved=0

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := Decode(data)
		if err != nil {
			return
		}
	})
}
