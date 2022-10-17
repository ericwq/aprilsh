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

import (
	"errors"
	"os"
	"testing"
)

func TestDisplay(t *testing.T) {
	tc := []struct {
		label    string
		useEnv   bool
		termEnv  string
		err      error
		hasECH   bool
		hasBCE   bool
		hasTitle bool
	}{
		{"useEnvironment, base TERM", true, "alacritty", nil, true, true, false},
		{"useEnvironment, base TERM, title support", true, "xterm", nil, true, true, true},
		{"useEnvironment, dynamic TERM", true, "sun", nil, true, true, false}, // we choose sun, because sun fade out from the market
		{"wrong TERM", true, "stranger", errors.New("infocmp: couldn't open terminfo file (null)."), false, false, false},
	}

	for _, v := range tc {
		os.Setenv("TERM", v.termEnv)
		d, e := NewDisplay(v.useEnv)

		if e == nil {

			if d.hasBCE != v.hasBCE {
				t.Errorf("%q expect bce %t, got %t\n", v.label, v.hasBCE, d.hasBCE)
			}
			if d.hasECH != v.hasECH {
				t.Errorf("%q expect ech %t, got %t\n", v.label, v.hasECH, d.hasECH)
			}
			if d.hasTitle != v.hasTitle {
				t.Errorf("%q expect title %t, got %t\n", v.label, v.hasTitle, d.hasTitle)
			}
		} else {
			if e.Error() != v.err.Error() {
				t.Errorf("%q expect err %q, got %q\n", v.label, v.err, e)
			}
		}
	}
}
