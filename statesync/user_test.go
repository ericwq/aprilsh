// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package statesync

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ericwq/aprilsh/terminal"
	"github.com/rivo/uniseg"
)

func TestUserStreamSubtract(t *testing.T) {
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
			u1.PushBack([]rune{chs[i]})
			// fmt.Printf("#test Subtract() PushBack %q into u1\n", chs[i])
		}
		// fmt.Printf("#test Subtract() base %s\n", &u1)

		// add size data
		if v.sizeB {
			for _, v := range sizes {
				u1.PushBackResize(v.width, v.height)
			}
			// fmt.Printf("#test DiffFrom() base+size %s\n", &u1)
		}

		u2 := UserStream{}

		// add prefix user keystroke
		prefix := []rune(v.prefix)
		for i := range prefix {
			u2.PushBack([]rune{prefix[i]})
			// fmt.Printf("#test Subtract() PushBack %q into u2\n", prefix[i])
		}
		// fmt.Printf("#test Subtract() prefix %s\n", &u2)

		// subtract the prefix from u1
		u1.Subtract(&u2)

		// only collect the UserByteType part
		var output strings.Builder
		for _, v := range u1.actions {
			switch v.theType {
			case UserByteType:
				output.WriteString(string(v.userByte.Chs))
			}
		}
		// fmt.Printf("#test Subtract() result %s\n", &u1)

		// validate the result
		got := output.String()
		if got != v.remains {
			t.Errorf("%q expect %q, got %q\n", v.name, v.remains, got)
		}
	}
}

func TestUserStreamUserEvent(t *testing.T) {
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

func TestUserStreamApplyString(t *testing.T) {
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
			u1.PushBack(chs)
			// if v.prefix == "" {
			// 	fmt.Printf("#test ApplyString() %c %q %x\n", chs, chs, chs)
			// }
		}
		// add base size data
		for _, v := range baseSize {
			// u1.pushBackResize(terminal.Resize{Width: v.width, Height: v.height})
			u1.PushBackResize(v.width, v.height)
		}
		// fmt.Printf("#test ApplyString() base+size %s len=%d\n", &u1, len(u1.actions))

		u2 := UserStream{}
		// add prefix user keystroke
		graphemes = uniseg.NewGraphemes(v.prefix)
		for graphemes.Next() {
			chs := graphemes.Runes()
			u2.PushBack(chs)
		}

		// add delta size data
		for _, v := range deltaSize {
			// u2.pushBackResize(terminal.Resize{Width: v.width, Height: v.height})
			u2.PushBackResize(v.width, v.height)
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

func TestUserStreamInitDiff(t *testing.T) {
	u3 := UserStream{}
	got := u3.InitDiff()
	expect := ""
	if expect != got {
		t.Errorf("#test InitDiff() expect %q, got %q\n", expect, got)
	}
}

func TestUserStreamApplyStringFail(t *testing.T) {
	diff := "malformed diff"
	u3 := &UserStream{}
	if err := u3.ApplyString(diff); err == nil {
		t.Error("#test ApplyString() expect error, got nil")
	}
}

func TestUserStreamString(t *testing.T) {
	tc := []struct {
		title     string
		keystroke string
		size      bool
		expect    string
	}{
		{"no size", "has keystroke, no size data", false, "Keystroke:\"has keystroke, no size data\", Resize:, size=27"},
		{"no keystroke", "", true, "Keystroke:\"\", Resize:(80,40),(132,60),(140,70),, size=3"},
		{
			"both keystroke and size", "has both keystroke and data", true,
			"Keystroke:\"has both keystroke and data\", Resize:(80,40),(132,60),(140,70),, size=30",
		},
		{"empty", "", false, "Keystroke:\"\", Resize:, size=0"},
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
			u1.PushBack([]rune{chs[i]})
		}

		// add size data
		if v.size {
			for _, v := range sizes {
				u1.PushBackResize(v.width, v.height)
			}
		}

		got := u1.String()
		if v.expect != got {
			t.Errorf("%q expect [%s], got [%s]\n", v.title, v.expect, got)
		}
	}
}

func TestUserStreamGetAction(t *testing.T) {
	tc := []struct {
		title        string
		keystrokeStr string
		addSizeItem  bool
		expectSize   int
		idx01        int
		item1        terminal.UserByte
		idx02        int
		item2        terminal.Resize
		idx03        int
		item3        terminal.ActOn
	}{
		{
			"english keystroke and size", "has both keystroke and data", true, 30,
			6,
			terminal.UserByte{Chs: []rune{'t'}},
			28,
			terminal.Resize{Width: 132, Height: 60},
			31, nil,
		},
		{
			"chinese keystroke and size", "包含用户输入和窗口大小调整数据", true, 18,
			6,
			terminal.UserByte{Chs: []rune("和")},
			15,
			terminal.Resize{Width: 80, Height: 40},
			18, nil,
		},
	}

	sizes := []struct {
		width, height int
	}{
		{80, 40}, {132, 60}, {140, 70},
	}
	for _, v := range tc {

		us := UserStream{}

		// add user keystroke
		chs := []rune(v.keystrokeStr)
		for i := range chs {
			us.PushBack([]rune{chs[i]})
		}

		// add size data
		if v.addSizeItem {
			for _, v := range sizes {
				us.PushBackResize(v.width, v.height)
			}
		}

		// validate size
		if v.expectSize != us.Size() {
			t.Errorf("%q expect size %d, got %d\n", v.title, v.expectSize, us.Size())
		}

		// validate user byte item
		if !reflect.DeepEqual(v.item1, us.GetAction(v.idx01)) {
			t.Errorf("%q expect index %d contains %q, got %q\n", v.title, v.idx01, v.item1, us.GetAction(v.idx01))
		}

		// validate size item
		if !reflect.DeepEqual(v.item2, us.GetAction(v.idx02)) {
			t.Errorf("%q expect index %d contains %q, got %q\n", v.title, v.idx02, v.item2, us.GetAction(v.idx02))
		}

		// validate out-of-range item
		if us.GetAction(v.idx03) != v.item3 {
			t.Errorf("%q getAction() expect %q, got %q\n", v.title, v.item3, us.GetAction(v.idx03))
		}
	}
}

func TestUserStreamClone(t *testing.T) {
	us := &UserStream{}

	// prepare user input data
	keystrokeStr := "data for clone"
	chs := []rune(keystrokeStr)
	for i := range chs {
		us.PushBack([]rune{chs[i]})
	}

	// prepare resize data
	sizes := []struct {
		width, height int
	}{
		{80, 40}, {132, 60}, {140, 70},
	}
	for _, v := range sizes {
		us.PushBackResize(v.width, v.height)
	}

	clone := us.Clone()

	if !reflect.DeepEqual(us, clone) {
		t.Errorf("#test expect %v, got %v\n", us, clone)
	}

	clone.ResetInput() // just for coverage
}

func TestUserStreamEmpty(t *testing.T) {
	us := &UserStream{}

	// prepare user input data
	keystrokeStr := "data for clone"
	chs := []rune(keystrokeStr)
	for i := range chs {
		us.PushBack([]rune{chs[i]})
	}

	if us.Empty() {
		t.Errorf("#test expect false, got %t\n", us.Empty())
	}
}
