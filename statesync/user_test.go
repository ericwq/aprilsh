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

package statesync

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/terminal"
	"github.com/rivo/uniseg"
)

func TestSubtract(t *testing.T) {
	sizes := []struct {
		width, height int
	}{
		{80, 40}, {132, 60}, {140, 70},
	}

	tc := []struct {
		name      string
		sizeB     bool // add sizes data
		keystroke string
		prefix    string
		remains   string
	}{
		{"subtract english keystroke from prefix", true, "Hello world", "Hello ", "world"},
		{"subtract chinese keystroke from prefix", false, "你好！中国", "你好！", "中国"},
		{"subtract equal keystroke from prefix", false, "equal prefix", "equal prefix", ""},
	}

	for _, v := range tc {

		u1 := UserStream{}

		// add user keystroke
		chs := []rune(v.keystroke)
		for i := range chs {
			// u1.pushBack(terminal.UserByte{Chs: []rune{chs[i]}})
			u1.pushBack([]rune{chs[i]})
		}
		// fmt.Printf("#test DiffFrom() base %s\n", &u1)

		// add size data
		if v.sizeB {
			for _, v := range sizes {
				// u1.pushBackResize(terminal.Resize{Width: v.width, Height: v.height})
				u1.pushBackResize(v.width, v.height)
			}
			// fmt.Printf("#test DiffFrom() base+size %s\n", &u1)
		}

		u2 := UserStream{}

		// add prefix user keystroke
		prefix := []rune(v.prefix)
		for i := range prefix {
			// u2.pushBack(terminal.UserByte{C: prefix[i]})
			u2.pushBack([]rune{prefix[i]})
		}
		// fmt.Printf("#test DiffFrom() prefix %s\n", &u2)

		// subtract the prefix from u1
		u1.Subtract(&u2)
		var output strings.Builder
		for _, v := range u1.actions {
			switch v.theType {
			case UserByteType:
				// output.WriteRune(v.userByte.C)
				output.WriteString(string(v.userByte.Chs))
			}
		}
		// fmt.Printf("#test DiffFrom() result %#v\n", &u1)

		// validate the result
		got := output.String()
		if got != v.remains {
			t.Errorf("%q expect %q, got %q\n", v.name, v.remains, got)
		}
	}
}

func TestUserEvent(t *testing.T) {
	e1 := NewUserEvent(terminal.UserByte{Chs: []rune("🇧🇷")})
	e2 := NewUserEvent(terminal.UserByte{Chs: []rune("🇧🇷")})

	if !reflect.DeepEqual(e1, e2) {
		t.Errorf("#test UserEvent equal should return true, %v, %v\n", e1, e2)
	}

	e1 = NewUserEventResize(terminal.Resize{Width: 80, Height: 40})
	e2 = NewUserEventResize(terminal.Resize{Width: 80, Height: 40})

	if !reflect.DeepEqual(e1, e2) {
		t.Errorf("#test UserEvent equal should return true, %v, %v\n", e1, e2)
	}
}

func TestApplyString(t *testing.T) {
	baseSize := []struct {
		width, height int
	}{
		{80, 40}, {132, 60}, {140, 70},
	}

	deltaSize := []struct {
		width, height int
	}{
		{80, 40}, {132, 60}, {140, 70},
	}

	tc := []struct {
		name      string
		keystroke string
		prefix    string
	}{
		{"diff & apply english keystroke from prefix", "Hello world", "Hello "},
		{"diff & apply chinese keystroke from prefix", "你好！中国", "你好！"},
		{"diff & apply equal prefix", "equal prefix", "equal prefix"},
		{"diff & apply flag", "Chin\u0308\u0308a 🏖 i国旗🇳🇱Fun 🌈with F🇧🇷lg", ""},
	}

	for _, v := range tc {

		u1 := UserStream{}
		// add user keystroke
		graphemes := uniseg.NewGraphemes(v.keystroke)
		for graphemes.Next() {
			chs := graphemes.Runes()
			u1.pushBack(chs)
			// if v.prefix == "" {
			// 	fmt.Printf("#test ApplyString() %c %q %x\n", chs, chs, chs)
			// }
		}
		// add base size data
		for _, v := range baseSize {
			// u1.pushBackResize(terminal.Resize{Width: v.width, Height: v.height})
			u1.pushBackResize(v.width, v.height)
		}
		// fmt.Printf("#test ApplyString() base+size %s len=%d\n", &u1, len(u1.actions))

		u2 := UserStream{}
		// add prefix user keystroke
		graphemes = uniseg.NewGraphemes(v.prefix)
		for graphemes.Next() {
			chs := graphemes.Runes()
			u2.pushBack(chs)
		}

		// add delta size data
		for _, v := range deltaSize {
			// u2.pushBackResize(terminal.Resize{Width: v.width, Height: v.height})
			u2.pushBackResize(v.width, v.height)
		}
		// fmt.Printf("#test ApplyString() prefix %s len=%d\n", &u2, len(u2.actions))

		diff := u1.DiffFrom(&u2)
		u1.Subtract(&u2) // after DiffFrom(), u1 is not affected.  Call subtract to modify it.
		// fmt.Printf("#test ApplyString() u1=%s diff len=%d\n", &u1, len(diff))

		u3 := UserStream{}
		u3.ApplyString(diff)
		// fmt.Printf("#test ApplyString() u3=%s\n\n", &u3)

		if !u1.Equal(&u3) {
			t.Errorf("%q expect \n%s, got \n%s\n", v.name, &u1, &u3)
		}
	}
}

func TestInitDiff(t *testing.T) {
	u3 := UserStream{}
	got := u3.InitDiff()
	expect := ""
	if expect != got {
		t.Errorf("#test InitDiff() expect %q, got %q\n", expect, got)
	}
}

func TestApplyStringFail(t *testing.T) {
	diff := "malformed diff"
	u3 := &UserStream{}
	if err := u3.ApplyString(diff); err == nil {
		t.Error("#test ApplyString() expect error, got nil")
	}
}

func TestString(t *testing.T) {
	tc := []struct {
		title     string
		keystroke string
		size      bool
		expect    string
	}{
		{"no size", "has keystroke, no size data", false, "Keystroke:\"has keystroke, no size data\", Resize:"},
		{"no keystroke", "", true, "Keystroke:\"\", Resize:(80,40),(132,60),(140,70),"},
		{"both keystroke and size", "has both keystroke and data", true, "Keystroke:\"has both keystroke and data\", Resize:(80,40),(132,60),(140,70),"},
		{"empty", "", false, "Keystroke:\"\", Resize:"},
	}

	sizes := []struct {
		width, height int
	}{
		{80, 40}, {132, 60}, {140, 70},
	}
	for _, v := range tc {

		u1 := UserStream{}

		// add user keystroke
		chs := []rune(v.keystroke)
		for i := range chs {
			u1.pushBack([]rune{chs[i]})
		}

		// add size data
		if v.size {
			for _, v := range sizes {
				u1.pushBackResize(v.width, v.height)
			}
		}

		got := u1.String()
		if v.expect != got {
			t.Errorf("%q expect [%s], got [%s]\n", v.title, v.expect, got)
		}

	}
}
