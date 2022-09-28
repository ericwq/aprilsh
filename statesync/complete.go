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
	pb "github.com/ericwq/aprilsh/protobufs/host"
	"github.com/ericwq/aprilsh/terminal"
	"google.golang.org/protobuf/proto"
)

const (
	ECHO_TIMEOUT = 50 // for late ack
)

// Complete implements network.State[C any] interface
type Complete struct {
	parser   *terminal.Parser
	terminal *terminal.Emulator

	actions      []*terminal.Handler
	inputHistory []string
	echoAck      uint64
}

func NewComplete(nCols, nRows, saveLines int) *Complete {
	c := new(Complete)
	c.parser = terminal.NewParser()
	c.terminal = terminal.NewEmulator3(nCols, nRows, saveLines)
	c.echoAck = 0

	return c
}

// let the terminal parse and handle the data stream.
func (c *Complete) Act(str string) string {
	c.terminal.HandleStream(str)

	return c.terminal.ReadOctetsToHost()
}

func (c *Complete) getEchoAck() uint64 {
	return c.echoAck
}

func (c *Complete) getFB() *terminal.Framebuffer {
	return c.terminal.GetFramebuffer()
}

func (c *Complete) ResetInput() {
	c.parser.ResetInput()
}

// do nothing
func (c *Complete) Subtract(prefix *Complete) {
}

func (c *Complete) DiffFrom(existing *Complete) string {
	hm := pb.HostMessage{}

	// TODO waiting for new_frame() and Action.act()
	output, _ := proto.Marshal(&hm)
	return string(output)
}
