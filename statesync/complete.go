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
	"fmt"
	"math"
	"reflect"

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
	display      terminal.Display
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
func (c *Complete) Subtract(prefix *Complete) {
	// do nothing
}

// implements network.State[C any] interface
// compare two Complete value and build the seralized HostMessage: difference content.
func (c *Complete) DiffFrom(existing *Complete) string {
	hm := pb.HostMessage{}

	if existing.getEchoAck() != c.getEchoAck() {
		echoack := c.getEchoAck()
		instEcho := pb.Instruction{Echoack: &pb.EchoAck{EchoAckNum: &echoack}}
		hm.Instruction = append(hm.Instruction, &instEcho)
	}

	if !reflect.DeepEqual(existing.getFramebuffer(), c.getFramebuffer()) {
		if existing.terminal.GetWidth() != c.terminal.GetWidth() ||
			existing.terminal.GetHeight() != c.terminal.GetHeight() {
			w := int32(c.terminal.GetWidth())
			h := int32(c.terminal.GetHeight())
			instResize := pb.Instruction{Resize: &pb.ResizeMessage{Width: &w, Height: &h}}
			hm.Instruction = append(hm.Instruction, &instResize)
		}
	}

	update := c.display.NewFrame(true, existing.getFramebuffer(), c.getFramebuffer())
	if len(update) > 0 {
		instBytes := pb.Instruction{Hostbytes: &pb.HostBytes{Hoststring: []byte(update)}}
		hm.Instruction = append(hm.Instruction, &instBytes)
	}

	output, _ := proto.Marshal(&hm)
	return string(output)
}

// implements network.State[C any] interface
// get difference between this Complete and a new one.
func (c *Complete) InitDiff() string {
	// TODO what about saveLines? use 0 or other value?
	return c.DiffFrom(NewComplete(c.terminal.GetWidth(), c.terminal.GetHeight(), 0))
}

// implements network.State[C any] interface
// convert differene content into HostMessage, and apply the instructions to the terminal.
func (c *Complete) ApplyString(diff string) error {
	// parse the wire-format encoding of UserMessage
	input := pb.HostMessage{}
	err := proto.Unmarshal([]byte(diff), &input)
	if err != nil {
		return err
	}

	for i := range input.Instruction {
		if input.Instruction[i].Hostbytes != nil {
			terminalToHost := c.Act(string(input.Instruction[i].Hostbytes.Hoststring))
			// server never interrogates client terminal
			if terminalToHost != "" {
				fmt.Printf("warn: terminalToHost=%s\n", terminalToHost)
			}
		} else if input.Instruction[i].Resize != nil {
			newSize := terminal.Resize{
				Height: int(input.Instruction[i].Resize.GetHeight()),
				Width:  int(input.Instruction[i].Resize.GetWidth()),
			}
			c.ActOne(newSize)
		} else if input.Instruction[i].Echoack != nil {
			instEchoAckNum := input.Instruction[i].Echoack.GetEchoAckNum()
			c.echoAck = instEchoAckNum
		}
	}

	return nil
}

// implements network.State[C any] interface
func (c *Complete) Equal(x *Complete) bool {
	return false
}
