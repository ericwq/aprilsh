// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package statesync

import (
	"io"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
)

func TestCompleteSubtract(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)
	c.Subtract(c) // do nothing, just for coverage
	c.GetEmulator()
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

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	for _, v := range tc {
		c0, _ := NewComplete(80, 40, 40)
		c1, _ := NewComplete(80, 40, 40)

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
		if got := c0.DiffFrom(c1); got != "" {
			// if !c0.Equal(c1) {
			// got := c0.DiffFrom(c1)
			t.Errorf("%q expect empty result after ApplyString(), got %q\n", v.label, got)
		}
	}
}

func TestApplyString_Fail(t *testing.T) {
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
			c.RegisterInputFrame(p.frameNum, now+ts)
			// fmt.Printf("#test setEchoAck timestamp=%d, ts=%d\n", p.timestamp, ts)
		}

		// fmt.Printf("#test setEchoAck inputHistory = %v\n", c.inputHistory)

		got := c.SetEchoAck(now + ts)
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
			c.RegisterInputFrame(p.frameNum, now+ts)
			// fmt.Printf("#test setEchoAck timestamp=%d, ts=%d\n", p.timestamp, ts)
		}

		got := c.WaitTime(now + ts + v.time)
		if v.expect != got {
			t.Errorf("%q expect %d, got %d\n", v.label, v.expect, got)
		}
	}
}

func TestCompleteResetInput(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)

	c.ResetInput()
	if c.terminal.GetCursorCol() != 0 || c.terminal.GetCursorRow() != 0 {
		t.Errorf("#test after resetInput() the cursor should be in (0,0), got (%d,%d)\n",
			c.terminal.GetCursorRow(), c.terminal.GetCursorCol())
	}
}

func TestCompleteClone(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)
	clone := c.Clone()

	if !c.Equal(clone) {
		t.Errorf("#test clone expect %v, got %v\n", c, clone)
	}
}

func (c *Complete) equalDiffFrom(x *Complete) bool {
	// use DiffFrom to compare the state
	if diff := c.DiffFrom(x); diff != "" {
		return false
	}
	return true
	// return reflect.DeepEqual(c.terminal, x.terminal) && c.echoAck == x.echoAck
}

func (c *Complete) deepEqual(x *Complete) bool {
	return reflect.DeepEqual(c.terminal, x.terminal) && c.echoAck == x.echoAck
}

// check Equal mthod
// func (c *Complete) customEqual(x *Complete) bool {
// 	if c.echoAck != x.echoAck {
// 		return false
// 	}
//
// 	return c.terminal.Equal(x.terminal)
// }

// https://blog.logrocket.com/benchmarking-golang-improve-function-performance/
// https://coder.today/tech/2018-11-10_profiling-your-golang-app-in-3-steps/
// https://www.speedscope.app/
func BenchmarkEqualDiffFrom(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.equalDiffFrom(c1)
	}
}

func BenchmarkDeepEqual(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.deepEqual(c1)
	}
}

func BenchmarkCustomEqual(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.Equal(c1)
	}
}

func BenchmarkDiffFrom(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.DiffFrom(c1)
	}
}

func BenchmarkDiffFromFramebuffer(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.getFramebuffer().Equal(c1.getFramebuffer())
	}
}

func BenchmarkDiffFromNewFrame(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.display.NewFrame(true, c0.terminal, c1.terminal)
	}
}
