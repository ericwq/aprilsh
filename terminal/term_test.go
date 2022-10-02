package terminal

import (
	"bytes"
	"testing"

	"github.com/ericwq/terminfo"
	_ "github.com/ericwq/terminfo/base"
	"github.com/ericwq/terminfo/dynamic"
)

func TestTermCapability(t *testing.T) {
	name := "xterm-256color"
	ti, e := terminfo.LookupTerminfo(name)
	if e != nil {
		ti, _, e := dynamic.LoadTerminfo(name)
		if e != nil {
			t.Errorf("#test %s %s\n", name, e)
		}
		terminfo.AddTerminfo(ti)
	}

	buf := bytes.NewBuffer(nil)
	ti.TPuts(buf, ti.Bell)
	got := string(buf.Bytes())
	if got != "\x07" {
		t.Errorf("#test TPuts %q expect %q, got %q\n", ti.Bell, "\x07", got)
	}
}
