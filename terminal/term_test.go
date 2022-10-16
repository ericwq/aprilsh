package terminal

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ericwq/terminfo"
	_ "github.com/ericwq/terminfo/base"
	"github.com/ericwq/terminfo/dynamic"
)

func TestTermCapability(t *testing.T) {
	name := "alacritty"
	ti, e := terminfo.LookupTerminfo(name)
	if e != nil {
		fmt.Printf("#test lookup failed. %s\n", e)
		ti, _, e := dynamic.LoadTerminfo(name)
		if e != nil {
			t.Errorf("#test %s %s\n", name, e)
		}
		fmt.Printf("#test dynamic success. %v\n", ti)
		terminfo.AddTerminfo(ti)
	}

	fmt.Printf("#test ti=%v\n", ti)

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
