/*

MIT License

Copyright (c) 2022 wangqi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package terminal

// import (
// 	"bufio"
// 	"io"
// 	"os"
// 	"reflect"
// 	"testing"
// 	"unicode/utf8"
// )
//
// // only reflect.DeepEqual can compare interface value with pointer receiver.
// func compareActions(a []Action, b []Action) bool {
// 	return reflect.DeepEqual(a, b)
// }
//
// func TestParserParse(t *testing.T) {
// 	tc := []struct {
// 		name string
// 		raw  string
// 		want []Action
// 	}{
// 		{"ground ISO 8859-1", "sun", []Action{
// 			&print{action{'s', true}}, &print{action{'u', true}}, &print{action{'n', true}},
// 		}},
// 		{"ground chinese", "s世a", []Action{
// 			&print{action{'s', true}}, &print{action{'世', true}}, &print{action{'a', true}},
// 		}},
// 		{"ground chinese", "s世a\x9C", []Action{
// 			&print{action{'s', true}}, &print{action{'世', true}}, &print{action{'a', true}},
// 		}},
// 		{"Control BEL", "\x07", []Action{
// 			&execute{action{'\x07', true}},
// 		}},
// 		{"Control  LF", "\x0A", []Action{
// 			&execute{action{'\x0A', true}},
// 		}},
// 		{"ESC E", "\x1BE", []Action{
// 			&clear{}, &escDispatch{action{'E', true}},
// 		}},
// 		{"ESC D", "\x1BD", []Action{
// 			&clear{}, &escDispatch{action{'D', true}},
// 		}},
// 		{"ESC 7", "\x1B7", []Action{
// 			&clear{}, &escDispatch{action{'7', true}},
// 		}},
// 		{"ESC #8", "\x1B#8", []Action{
// 			&clear{}, &collect{action{'#', true}}, &escDispatch{action{'8', true}},
// 		}},
// 		{"CSI !P", "\x1B[!P", []Action{
// 			&clear{}, &clear{}, &collect{action{'!', true}}, &csiDispatch{action{'P', true}},
// 		}},
// 		{"CSI Ps;PsH", "\x1B[23;12H", []Action{
// 			&clear{}, &clear{}, &param{action{'2', true}}, &param{action{'3', true}},
// 			&param{action{';', true}}, &param{action{'1', true}}, &param{action{'2', true}},
// 			&csiDispatch{action{'H', true}},
// 		}},
// 		{"CSI ? Pm h", "\x1B[?1;9;1000h", []Action{
// 			&clear{}, &clear{}, &collect{action{'?', true}},
// 			&param{action{'1', true}}, &param{action{';', true}}, &param{action{'9', true}},
// 			&param{action{';', true}}, &param{action{'1', true}}, &param{action{'0', true}},
// 			&param{action{'0', true}}, &param{action{'0', true}}, &csiDispatch{action{'h', true}},
// 		}},
// 		{"CSI 38:2:Pr:Pg,Pbm", "\x1B[38:2:25:90:12m", []Action{
// 			&clear{}, &clear{}, &param{action{'3', true}}, &param{action{'8', true}},
// 			&param{action{':', true}}, &param{action{'2', true}}, &param{action{':', true}},
// 			&param{action{'2', true}}, &param{action{'5', true}}, &param{action{':', true}},
// 			&param{action{'9', true}}, &param{action{'0', true}}, &param{action{':', true}},
// 			&param{action{'1', true}}, &param{action{'2', true}}, &csiDispatch{action{'m', true}},
// 		}},
// 		{"OSC 0;Pt BEL", "\x1B]0;ada\x07", []Action{
// 			&clear{}, &oscStart{}, &oscPut{action{'0', true}}, &oscPut{action{';', true}},
// 			&oscPut{action{'a', true}}, &oscPut{action{'d', true}}, &oscPut{action{'a', true}}, &oscEnd{},
// 		}},
// 		{"OSC 0;Pt ST", "\x1B]0;ada\x9c", []Action{
// 			&clear{}, &oscStart{}, &oscPut{action{'0', true}}, &oscPut{action{';', true}},
// 			&oscPut{action{'a', true}}, &oscPut{action{'d', true}}, &oscPut{action{'a', true}}, &oscEnd{},
// 		}},
// 		{"OSC 0;Pt ST chinese", "\x1B]0;a仁a\x9c", []Action{
// 			&clear{}, &oscStart{}, &oscPut{action{'0', true}}, &oscPut{action{';', true}},
// 			&oscPut{action{'a', true}}, &oscPut{action{'仁', true}}, &oscPut{action{'a', true}}, &oscEnd{},
// 		}},
// 	}
//
// 	p := NewParser()
// 	for _, v := range tc {
// 		p.reset()
//
// 		actions := make([]Action, 0, 8)
// 		for _, ch := range v.raw {
// 			// if ch == '\uFFFD' {
// 			// 	t.Errorf("read from raw got 0x%X\n", ch)
// 			// }
// 			actions = p.parse(actions, ch)
// 		}
// 		if !compareActions(v.want, actions) {
// 			t.Errorf("%s \nexpect\t %s\ngot:\t %s\n", v.name, v.want, actions)
// 		}
// 	}
// }
//
// // disable this test
// func testMixUnicodeFile(t *testing.T) {
// 	fileName := "mix_unicode.out"
//
// 	f, err := os.Open(fileName)
// 	if err != nil {
// 		t.Errorf("open file %s return error: %s", fileName, err)
// 	}
//
// 	reader := bufio.NewReader(f)
//
// 	for {
// 		r, size, err := reader.ReadRune()
// 		buf := make([]byte, 3)
// 		n := utf8.EncodeRune(buf, r)
//
// 		t.Logf("%q 0x%X size=%d 0x%X s=%d\n", r, r, size, buf, n)
// 		if err == io.EOF {
// 			break
// 		}
// 	}
// }
