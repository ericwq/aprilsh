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
	"testing"
	"time"
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
	// tc := []struct {
	// 	label  string
	// 	seq    string
	// 	expect string
	// }{}
	//
	// for _, v := range tc {
	// 	c1, _ := NewComplete(80, 40, 40)
	// }
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
