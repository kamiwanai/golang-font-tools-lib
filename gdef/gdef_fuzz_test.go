// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the MIT License. See LICENSE for details.
// Commercial licensing: kamiwanaiii@gmail.com

package gdef

import "testing"

func FuzzDecode(f *testing.F) {
	glyphOrder := []string{".notdef", "A", "V", "T"}
	f.Add([]byte{})
	f.Add([]byte{0, 1, 0, 0, 0, 10, 0, 0, 0, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := Decode(data, glyphOrder)
		if err != nil {
			return
		}
	})
}
