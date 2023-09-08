// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package statesync

import (
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
	frameNum  int64 // remote frame number
	timestamp int64
}

// Complete implements network.State[C any] interface
type Complete struct {
	terminal     *terminal.Emulator
	inputHistory []pair
	echoAck      int64
	display      *terminal.Display
}

func NewComplete(nCols, nRows, saveLines int) (*Complete, error) {
	var err error

	c := &Complete{}
	c.terminal = terminal.NewEmulator3(nCols, nRows, saveLines)
	c.inputHistory = make([]pair, 0)
	c.echoAck = 0
	c.display, err = terminal.NewDisplay(false)

	return c, err
}

// let the terminal parse and handle the data stream.
func (c *Complete) Act(str string) string {
	c.terminal.HandleStream(str)

	return c.terminal.ReadOctetsToHost()
}

// run the action in terminal, return the contents in terminalToHost.
func (c *Complete) ActOne(x terminal.ActOn) string {
	x.Handle(c.terminal)
	return c.terminal.ReadOctetsToHost()
}

func (c *Complete) getFramebuffer() *terminal.Framebuffer {
	return c.terminal.GetFramebuffer()
}

func (c *Complete) GetEmulator() *terminal.Emulator {
	return c.terminal
}

func (c *Complete) GetEchoAck() int64 {
	return c.echoAck
}

// shrink input history according to timestamp. return true if newestEchoAck changed.
// update echoAck if find the newest state.
func (c *Complete) SetEchoAck(now int64) (ret bool) {
	var newestEchoAck int64 = 0
	for _, v := range c.inputHistory {
		// fmt.Printf("#setEchoAck timestamp=%d, now-ECHO_TIMEOUT=%d, condition is %t\n",
		// 	v.timestamp, now-ECHO_TIMEOUT, v.timestamp <= now-ECHO_TIMEOUT)
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
		if x.frameNum >= newestEchoAck {
			b = append(b, x)
		}
	}
	c.inputHistory = b

	if c.echoAck != newestEchoAck {
		// fmt.Printf("#setEchoAck echoAck=%d, newestEchoAck=%d\n", c.echoAck, newestEchoAck)
		ret = true
	}

	// fmt.Printf("#setEchoAck echoAck changed(%t) to %d\n", ret, c.echoAck)
	c.echoAck = newestEchoAck
	return
}

// register the latest remote state number and time.
// the latest remote state is the client input state.
func (c *Complete) RegisterInputFrame(num, now int64) {
	c.inputHistory = append(c.inputHistory, pair{num, now})
}

// return 0 if history frame state + 50ms < now. That means now is larger than 50ms.
// return max int if there is only one history frame.
// return normal gap between now and history frame + 50ms.
func (c *Complete) WaitTime(now int64) int {
	if len(c.inputHistory) < 2 {
		return math.MaxInt
	}
	// start from the second
	nextEchoAckTime := c.inputHistory[1].timestamp + ECHO_TIMEOUT

	// util.Log.With("now", now).
	// 	With("nextEchoAckTime", nextEchoAckTime).With("gap", nextEchoAckTime-now)

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

	if existing.GetEchoAck() != c.GetEchoAck() {
		echoack := c.GetEchoAck()
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

	update := c.display.NewFrame(true, existing.terminal, c.terminal)
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
	blank, _ := NewComplete(c.terminal.GetWidth(), c.terminal.GetHeight(), c.terminal.GetSaveLines())
	return c.DiffFrom(blank)
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
			// server never interrogates client terminal
			c.Act(string(input.Instruction[i].Hostbytes.Hoststring))
			// terminalToHost := c.Act(string(input.Instruction[i].Hostbytes.Hoststring))
			// if terminalToHost != "" {
			// 	fmt.Printf("warn: terminalToHost=%s\n", terminalToHost)
			// }
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
	// use DiffFrom to compare the state
	// if diff := c.DiffFrom(x); diff != "" {
	// 	return false
	// }
	// return true
	return reflect.DeepEqual(c.terminal, x.terminal) && c.echoAck == x.echoAck
}

// implements network.State[C any] interface
func (c *Complete) ResetInput() {
	c.terminal.GetParser().ResetInput()
}

// implements network.State[C any] interface
func (c *Complete) Clone() *Complete {
	clone := Complete{}

	clone = *c
	clone.display = c.display.Clone()
	clone.terminal = c.terminal.Clone()

	clone.inputHistory = make([]pair, len(c.inputHistory))
	copy(clone.inputHistory, c.inputHistory)

	return &clone
}
