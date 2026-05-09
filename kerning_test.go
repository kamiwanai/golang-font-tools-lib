package fonttools

import "testing"

func TestExtractGPOSKerningReturnsParseError(t *testing.T) {
	_, err := ExtractGPOSKerning("does-not-exist.ttf")
	if err == nil {
		t.Fatal("ExtractGPOSKerning succeeded for missing file")
	}
}
