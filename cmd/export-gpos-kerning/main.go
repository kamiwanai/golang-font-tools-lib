package main

import (
	"fmt"
	"log"
	"os"

	"github.com/kamiwanai/fonttools"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <fontfile>\n", os.Args[0])
		os.Exit(1)
	}

	pairs, err := fonttools.ExtractGPOSKerning(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	for _, pair := range pairs {
		fmt.Printf("%s %s %d\n", pair.LeftGlyph, pair.RightGlyph, pair.Value)
	}
}
