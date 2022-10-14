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
	"math"

	pb "github.com/ericwq/aprilsh/protobufs/host"
	"github.com/ericwq/aprilsh/terminal"
	"google.golang.org/protobuf/proto"
)

const (
	ECHO_TIMEOUT = 50 // for late ack
)

type pair struct {
	frameNum  uint64 // remote frame number
	timestamp uint64
}

// Complete implements network.State[C any] interface
type Complete struct {
	terminal     *terminal.Emulator
	inputHistory []pair
	echoAck      uint64
}

func NewComplete(nCols, nRows, saveLines int) *Complete {
	c := &Complete{}
	c.terminal = terminal.NewEmulator3(nCols, nRows, saveLines)
	c.inputHistory = make([]pair, 0)
	c.echoAck = 0

	return c
}

// let the terminal parse and handle the data stream.
func (c *Complete) Act(str string) string {
	c.terminal.HandleStream(str)

	return c.terminal.ReadOctetsToHost()
}

// run the action in terminal, return the contents in terminalToHost.
func (c *Complete) ActOne(x terminal.ActOn) string {
	x.Handle(*c.terminal)
	return c.terminal.ReadOctetsToHost()
}

func (c *Complete) getFramebuffer() *terminal.Framebuffer {
	return c.terminal.GetFramebuffer()
}

func (c *Complete) resetInput() {
	c.terminal.GetParser().ResetInput()
}

func (c *Complete) getEchoAck() uint64 {
	return c.echoAck
}

func (c *Complete) setEchoAck(now uint64) (ret bool) {
	var newestEchoAck uint64 = 0
	for _, v := range c.inputHistory {
		if v.timestamp <= now-ECHO_TIMEOUT {
			newestEchoAck = v.frameNum
		}
	}

	// filter frame number which is less than newestEchoAck
	// filter without allocating
	// This trick uses the fact that a slice shares the same backing array
	// and capacity as the original, so the storage is reused for the filtered
	// slice. Of course, the original contents are modified.
	b := c.inputHistory[:0]
	for _, x := range c.inputHistory {
		if x.frameNum < newestEchoAck {
			b = append(b, x)
		}
	}
	c.inputHistory = b

	if c.echoAck != newestEchoAck {
		ret = true
	}

	c.echoAck = newestEchoAck
	return
}

func (c *Complete) registerInputFrame(num, now uint64) {
	c.inputHistory = append(c.inputHistory, pair{num, now})
}

func (c *Complete) waitTime(now uint64) int {
	if len(c.inputHistory) < 2 {
		return math.MaxInt
	}
	nextEchoAckTime := c.inputHistory[1].timestamp + ECHO_TIMEOUT

	if nextEchoAckTime <= now {
		return 0
	} else {
		return int(nextEchoAckTime - now)
	}
}

// implements network.State[C any] interface
// do nothing
func (c *Complete) Subtract(prefix *Complete) {
}

// implements network.State[C any] interface
func (c *Complete) DiffFrom(existing *Complete) string {
	hm := pb.HostMessage{}

	// TODO waiting for new_frame() and Action.act()
	output, _ := proto.Marshal(&hm)
	return string(output)
}
