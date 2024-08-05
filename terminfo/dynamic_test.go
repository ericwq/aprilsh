// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminfo

import (
	"io"
	"os"
	"testing"

	"github.com/ericwq/aprilsh/util"
)

func TestUnescape(t *testing.T) {
	dynamicInit()
	expectStr := "\x1b[%i%p1%d;%p2%dH"
	if nTerminfo.getstr("cup") != expectStr {
		t.Errorf("cup expect %q, got %q\n", expectStr, nTerminfo.getstr("cup"))
	}

	expectBool := true
	if nTerminfo.getflag("am") != expectBool {
		t.Errorf("am expect %t, got %t\n", expectBool, nTerminfo.getflag("am"))
	}

	expectNum := 80
	if nTerminfo.getnum("cols") != expectNum {
		t.Errorf("cols expect %d, got %d\n", expectNum, nTerminfo.getnum("cols"))
	}

	tc := []struct {
		label string
		data  string
		value []rune
	}{
		{
			"special escape",
			"\\0\\n\\r\\t\\b\\f\\s",
			[]rune{'\x00', '\n', '\r', '\t', '\b', '\f', ' '},
		},
	}
	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			got := unescape(v.data)
			if got != string(v.value) {
				t.Errorf("%s require %q, got %q\b", v.label, string(v.value), got)
			}
		})
	}
}

func TestSetupterm(t *testing.T) {
	badTc := &terminfo{}
	err := badTc.setupterm("badTerm")
	if err == nil {
		t.Errorf("setupterm expect errors got \n%s\n", err)
	}
}

func TestInit_BadTerm(t *testing.T) {
	term := os.Getenv("TERM")
	os.Setenv("TERM", "badTerm@$%")

	defer func() {
		if p := recover(); p != nil {
			os.Setenv("TERM", term)
		}
	}()
	dynamicInit()
	t.Errorf("should panic")
}

func TestInit_NoTerm(t *testing.T) {
	term := os.Getenv("TERM")
	os.Unsetenv("TERM")

	defer func() {
		if p := recover(); p != nil {
			os.Setenv("TERM", term)
		}
	}()
	dynamicInit()
	t.Errorf("should panic")
}

func TestLookup(t *testing.T) {
	tc := []struct {
		label  string
		names  []string
		values []string
		ok     []bool
	}{
		{
			"special capability",
			[]string{"TN", "Co", "RGB"},
			[]string{os.Getenv("TERM"), "256", "8/8/8"},
			[]bool{true, true, true},
		},
		{
			"number capability",
			[]string{"colors", "cols"},
			[]string{"256", "80"},
			[]bool{true, true},
		},
		{
			"string capability",
			[]string{"cup", "setrgbf", "setrgbb"},
			[]string{"\x1b[%i%p1%d;%p2%dH", "", ""},
			[]bool{true, false, false},
		},
		{
			"bool capability",
			[]string{"am"},
			[]string{""},
			[]bool{true},
		},
	}

	util.Logger.CreateLogger(io.Discard, false, util.LevelTrace)
	nTerminfo = nil

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			for i, name := range v.names {
				value, ok := Lookup(name)
				if v.values[i] != value || ok != v.ok[i] {
					t.Errorf("%s name:%-9s expect %q got %q,ok=%t",
						v.label, name, v.values[i], value, ok)
				}
			}
		})
	}
}
