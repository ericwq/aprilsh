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
	"io"
	"math"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/terminal"
)

func TestCompleteSubtract(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)
	c.Subtract(c) // do nothing, just for coverage
}

func TestCompleteInitDiff(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)
	got := c.InitDiff()

	expect := ""
	if expect != got {
		t.Errorf("#test InitDiff() expect %q, got %q\n", expect, got)
	}
}

func TestCompleteApplyString(t *testing.T) {
	tc := []struct {
		label         string
		seq           string
		width, height int
		echoAck       int64
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", 0, 0, 0},
		{"fill one row and resize", "\x1B[6;67HLAST", 70, 30, 0},
		{"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}

	for _, v := range tc {
		c0, _ := NewComplete(80, 40, 40)
		c1, _ := NewComplete(80, 40, 40)

		// disable log trace
		c0.terminal.SetLogTraceOutput(io.Discard)
		c1.terminal.SetLogTraceOutput(io.Discard)

		// resize new state if necessary
		if v.height != 0 && v.width != 0 {
			r := terminal.Resize{Width: v.width, Height: v.height}
			emu := c1.terminal
			r.Handle(emu)
		}

		// print some data on screen
		c1.terminal.HandleStream(v.seq)

		// validate the equal is false
		if c1.Equal(c0) {
			t.Errorf("%q expect false equal(), got true", v.label)
		}

		// set echoAck for new state
		if v.echoAck != 0 {
			c1.echoAck = v.echoAck
		}

		// new state calculate difference with old state as parameter
		diff := c1.DiffFrom(c0)

		// apply to the old state
		c0.ApplyString(diff)

		// validate the result
		// if got := c0.DiffFrom(c1); got != "" {
		if !c0.Equal(c1) {
			got := c0.DiffFrom(c1)
			t.Errorf("%q expect empty result after ApplyString(), got %q\n", v.label, got)
		}
	}
}

func TestCompleteApplyString_Fail(t *testing.T) {
	diff := "mislead\n\x04:\x02@\x03\n2\x120\"."

	c, _ := NewComplete(80, 40, 40)
	if err := c.ApplyString(diff); err == nil {
		t.Error("#test feed ApplyString with wrong parameter, expect error.")
	}
}

func TestCompleteSetEchoAck(t *testing.T) {
	tc := []struct {
		label  string
		data   []pair
		expect bool
	}{
		{"find two states", []pair{{1, 49}, {2, 43}, {3, 52}}, true},
		{"too quick to find the latest state", []pair{{1, 9}, {2, 13}, {3, 12}}, false},
	}

	c, _ := NewComplete(8, 4, 4)
	now := time.Now().UnixMilli()

	for _, v := range tc {
		// reset history
		c.inputHistory = make([]pair, 0)
		c.echoAck = 0

		// register the frame number and time
		var ts int64 = 0
		for _, p := range v.data {

			ts += p.timestamp
			// note: the timestamp is delta value in ms.
			c.registerInputFrame(p.frameNum, now+ts)
			// fmt.Printf("#test setEchoAck timestamp=%d, ts=%d\n", p.timestamp, ts)
		}

		// fmt.Printf("#test setEchoAck inputHistory = %v\n", c.inputHistory)

		got := c.setEchoAck(now + ts)
		// fmt.Printf("#test setEchoAck inputHistory = %v\n", c.inputHistory)
		if v.expect != got {
			t.Errorf("%q expect %t, got %t\n", v.label, v.expect, got)
		}
	}
}

func TestCompleteWaitTime(t *testing.T) {
	tc := []struct {
		label  string
		data   []pair
		time   int64
		expect int
	}{
		{"history size <2", []pair{{1, 49}}, 0, math.MaxInt},
		{"now < last +50 ", []pair{{1, 49}, {2, 43}}, 9, 50 - 9},
		{"last +50 <= now", []pair{{1, 49}, {2, 43}}, 50, 0},
	}

	c, _ := NewComplete(8, 4, 4)
	now := time.Now().UnixMilli()

	for _, v := range tc {
		// reset history
		c.inputHistory = make([]pair, 0)
		c.echoAck = 0

		// register the frame number and time
		var ts int64 = 0
		for _, p := range v.data {

			ts += p.timestamp
			// note: the timestamp is delta value in ms.
			c.registerInputFrame(p.frameNum, now+ts)
			// fmt.Printf("#test setEchoAck timestamp=%d, ts=%d\n", p.timestamp, ts)
		}

		got := c.waitTime(now + ts + v.time)
		if v.expect != got {
			t.Errorf("%q expect %d, got %d\n", v.label, v.expect, got)
		}
	}
}

func TestCompleteResetInput(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)

	c.resetInput()
	if c.terminal.GetCursorCol() != 0 || c.terminal.GetCursorRow() != 0 {
		t.Errorf("#test after resetInput() the cursor should be in (0,0), got (%d,%d)\n",
			c.terminal.GetCursorRow(), c.terminal.GetCursorCol())
	}
}
