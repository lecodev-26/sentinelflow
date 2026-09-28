package multimodalv5

import "testing"

func TestMultimodalValidation(t *testing.T) {
	r := Request{Parts: []Part{{Modality: Image, URI: "s3://x"}}}
	if !r.Has(Image) || r.Validate() != nil {
		t.Fatal(r)
	}
}
