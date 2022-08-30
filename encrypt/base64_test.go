package encrypt

import (
	"testing"
)

func TestPrng(t *testing.T) {
	tc := []int{0, 1, 2, 4, 8, 16, 32}

	for _, v := range tc {
		got := prngFill(v)
		if v != len(got) {
			t.Errorf("prngFill got %#v\n", got)
		}
	}

	for i := 0; i < 8; i++ {
		got := prngUint8()
		if got == 0 {
			t.Errorf("prngUint8 got %#v\n", got)
		}
	}
}
