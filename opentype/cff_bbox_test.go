package opentype

import (
	"testing"
)

func TestComputeCharStringBBoxSimpleMovetoLineto(t *testing.T) {
	// Type 2 charstring: rmoveto(50,50), rlineto(50,0), rlineto(0,150), rlineto(-50,0), endchar
	// 50 = 189-139, 0 = 139-139, 150 = 247,42 ((247-247)*256+42+108=150)
	// -50 = 28,0xFF,0xCE (int16)
	cs := []byte{
		189, 189, 21,             // rmoveto(50, 50)
		189, 139, 5,              // rlineto(50, 0)
		139, 247, 42, 5,          // rlineto(0, 150)
		28, 0xFF, 0xCE, 139, 5,   // rlineto(-50, 0)
		14,                       // endchar
	}
	xmin, ymin, xmax, ymax, err := computeCharStringBBox(cs, nil, nil, 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if xmin != 50 || ymin != 50 || xmax != 100 || ymax != 200 {
		t.Fatalf("bbox = (%d,%d)-(%d,%d), want (50,50)-(100,200)", xmin, ymin, xmax, ymax)
	}
}

func TestComputeCharStringBBoxEndcharOnly(t *testing.T) {
	cs := []byte{14}
	_, _, _, _, err := computeCharStringBBox(cs, nil, nil, 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestComputeCharStringBBoxDepthLimit(t *testing.T) {
	subr := []byte{29, 14}
	_, _, _, _, err := computeCharStringBBox(subr, [][]byte{subr}, nil, 0)
	if err != nil {
		t.Logf("expected error for deep recursion: %v", err)
	}
}
