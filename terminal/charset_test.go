package terminal

import (
	"bufio"
	"bytes"
	"io"
	"testing"
)

var charsetG [4]*map[byte]rune

// disable this test
// this test show that ReadRune() will cause problem for iso 8859-1 GR area
func testCharsetMap(t *testing.T) {
	tx := map[byte]rune{
		0xA3: 0x00C0,
		0xA4: 0x00C1,
		0xA5: 0x00C2,
	}
	data := []byte{0xA3, 0xA4, 0xA5}

	for _, v := range data {
		t.Logf("%c %X @ 0x%2x, %U\n", tx[v], tx[v], v, v)
	}
	reader := bufio.NewReader(bytes.NewBuffer(data))

	for i := 0; i < len(data); i++ {
		r, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		t.Logf("% X %d\n", r, size)
	}
}

func TestVt_ISO_UK(t *testing.T) {
	charsetG[0] = &vt_ISO_UK

	str := "\x22\x23\x24"
	want := []rune{0x0022, 0x00A3, 0x0024}

	for i := 0; i < len(str); i++ {
		x := str[i]
		y := lookupTable(charsetG[0], x)
		if y != want[i] {
			t.Errorf("for %x expect %U , got %U \n", x, want[i], y)
		}
	}
}

func TestVt_DEC_Supplement(t *testing.T) {
	charsetG[2] = &vt_DEC_Supplement
	tc := []struct {
		str  string
		want []rune
	}{
		{"\xa0\xa1\xa6\xa8\xac\xad\xae\xaf", []rune{0x0020, 0x00a1, 0x0026, 0x00a4, 0x002c, 0x002d, 0x002e, 0x002f}},
		{"\xb4\xb8\xbe", []rune{0x0034, 0x0038, 0x003e}},
		{"\xd0\xd7\xdd\xde", []rune{0x0050, 0x0152, 0x0178, 0x005e}},
		{"\xf0\xf7\xfd\xfe\xff", []rune{0x0070, 0x0153, 0x00ff, 0x007e, 0x007f}},
	}

	for _, v := range tc {
		for i := 0; i < len(v.str); i++ {
			x := v.str[i]
			y := lookupTable(charsetG[2], x)
			if y != v.want[i] {
				t.Errorf("for %x expect %U, got %U\n", x, v.want[i], y)
			}
		}
	}
}
