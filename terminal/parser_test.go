package terminal

import (
	"bufio"
	"io"
	"os"
	"reflect"
	"testing"
	"unicode/utf8"
)

// only reflect.DeepEqual can compare interface value with pointer receiver.
func compareActions(a []Action, b []Action) bool {
	return reflect.DeepEqual(a, b)
}

func TestParserGroundParse(t *testing.T) {
	tc := []struct {
		name string
		raw  string
		want []Action
	}{
		{"ground ISO 8859-1", "sun", []Action{&print{action{'s', true}}, &print{action{'u', true}}, &print{action{'n', true}}}},
		{"ground chinese", "s世a", []Action{&print{action{'s', true}}, &print{action{'世', true}}, &print{action{'a', true}}}},
	}

	p := NewParser()
	for _, v := range tc {
		p.reset()

		actions := make([]Action, 0, 8)
		for _, ch := range v.raw {
			actions = p.parse(actions, ch)
		}
		if !compareActions(v.want, actions) {
			t.Errorf("%s \nexpect\t %s\ngot:\t %s\n", v.name, v.want, actions)
		}
	}
}

// disable this test
func testMixUnicodeFile(t *testing.T) {
	fileName := "mix_unicode.out"

	f, err := os.Open(fileName)
	if err != nil {
		t.Errorf("open file %s return error: %s", fileName, err)
	}

	reader := bufio.NewReader(f)

	for {
		r, size, err := reader.ReadRune()
		buf := make([]byte, 3)
		n := utf8.EncodeRune(buf, r)

		t.Logf("%q 0x%X size=%d 0x%X s=%d\n", r, r, size, buf, n)
		if err == io.EOF {
			break
		}
	}
}
