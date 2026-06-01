// Copyright (c) 2026 kamiwanai. All rights reserved.
// Licensed under the GNU Affero General Public License v3.0 (AGPL-3.0).
// See LICENSE for details. Commercial licensing: inari1337@gmail.com

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kamiwanai/golang-font-tools-lib"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <fontfile>\n", os.Args[0])
		os.Exit(1)
	}

	// Parse once — all extractions reuse the same in-memory data.
	font, err := fonttools.ParseFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	pairs, err := font.GPOSKerning()
	if err != nil {
		log.Fatal(err)
	}

	for _, pair := range pairs {
		fmt.Printf("%s %s %d\n", pair.LeftGlyph, pair.RightGlyph, pair.Value)
	}
}
