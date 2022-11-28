/*

MIT License

Copyright (c) 2022~2023 wangqi

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

import (
	"bytes"
	"testing"

	"github.com/ericwq/terminfo"
	_ "github.com/ericwq/terminfo/base"
	"github.com/ericwq/terminfo/dynamic"
)

func TestTerminfo_bce_ech(t *testing.T) {
	name := "xterm-256color" // xterm-256color support bce and ech on Mac
	ti, e := terminfo.LookupTerminfo(name)
	if e != nil {
		// fmt.Printf("#test lookup failed. %s\n", e)
		ti, _, e = dynamic.LoadTerminfo(name)
		if e != nil {
			t.Fatalf("#test can't find terminfo for %s, %s\n", name, e)
		}
		// fmt.Printf("#test dynamic success. %p\n", ti)
		terminfo.AddTerminfo(ti)
	}

	buf := bytes.NewBuffer(nil)
	ti.TPuts(buf, ti.Bell)
	got := string(buf.Bytes())
	if got != "\x07" {
		t.Errorf("#test TPuts %q expect %q, got %q\n", ti.Bell, "\x07", got)
	}

	if !ti.BackColorErase {
		t.Errorf("#test expect bce exist, got %t\n", ti.BackColorErase)
	}

	if ti.EraseChars == "" {
		t.Errorf("#test expect ech %q, got empty.\n", ti.EraseChars)
	}
}
