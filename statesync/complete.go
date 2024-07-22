// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package statesync

import (
	"fmt"
	"math"
	"strings"
	"time"

	pb "github.com/ericwq/aprilsh/protobufs/host"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	"google.golang.org/protobuf/proto"
)

const (
	ECHO_TIMEOUT = 50 // for late ack
)

type pair struct {
	frameNum  uint64 // remote frame number
	timestamp int64
}

// Complete implements network.State[C any] interface
type Complete struct {
	terminal     *terminal.Emulator
	display      *terminal.Display
	remainsBuf   strings.Builder
	diffBuf      strings.Builder
	inputHistory []pair // user input history
	echoAck      uint64 // which user input is echoed?
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
func (c *Complete) ActLarge(str string, feed chan string) string {
	// if there is remains, append the new one
	c.remainsBuf.WriteString(str)

	// consume the data stream
	diff, remains := c.terminal.HandleLargeStream(c.remainsBuf.String())
	c.diffBuf.WriteString(diff)
	c.remainsBuf.Reset()

	// save remains if we got
	if len(remains) > 0 {
		c.remainsBuf.WriteString(remains)
		util.Logger.Debug("ActLarge", "remains", remains)
		go func() {
			// a little bit late is good enough, the main loop will block
			// until it send the previous content.
			time.Sleep(time.Millisecond * 10)
			feed <- ""
			util.Logger.Debug("ActLarge", "schedule", "send remains")
		}()
	}

	// util.Log.Debug("ActLarge","diff", c.diffBuf.String())
	return c.terminal.ReadOctetsToHost()
}

func (c *Complete) InitSize(nCols, nRows int) {
	newSize := terminal.Resize{Height: nRows, Width: nCols}
	c.ActOne(newSize)
}

// let the terminal parse and handle the data stream.
func (c *Complete) Act(str string) string {
	// c.terminal.ResetDamage()
	_, diff := c.terminal.HandleStream(str)
	c.diffBuf.WriteString(diff)

	// util.Logger.Debug("Act", "input", str, "diff", diff, "diffBuf", c.diffBuf.String())
	return c.terminal.ReadOctetsToHost()
}

func (c *Complete) GetDiff() string {
	ret := c.diffBuf.String()
	c.Reset()
	return ret
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

func (c *Complete) GetEchoAck() uint64 {
	return c.echoAck
}

// shrink input history according to timestamp. return true if newestEchoAck changed.
// update echoAck if find the newest state.
func (c *Complete) SetEchoAck(now int64, inputEchoDone bool) (ret bool) {
	newestEchoAck := c.echoAck
	var st int64 = ECHO_TIMEOUT
	if inputEchoDone {
		st = 0
	}
	for _, v := range c.inputHistory {
		if v.timestamp <= now-st {
			newestEchoAck = v.frameNum
			// util.Logger.Debug("SetEchoAck", "frameNum", v.frameNum, "timestamp", v.timestamp%10000)
		}
		// combine with RegisterInputFrame, if there is any user input
		// the echo ack will be updated and sent back to client.
	}

	// only keep frame number which is greate than newestEchoAck
	b := c.inputHistory[:0]
	for _, x := range c.inputHistory {
		if x.frameNum >= newestEchoAck {
			b = append(b, x)
		}
	}
	c.inputHistory = b

	if c.echoAck != newestEchoAck {
		ret = true
	}

	util.Logger.Debug("SetEchoAck", "newestEchoAck", newestEchoAck, "return", ret,
		"now", now, "inputHistory", b)

	c.echoAck = newestEchoAck
	return
}

// register the latest remote state number and time.
// the latest remote state is the client input state.
func (c *Complete) RegisterInputFrame(num uint64, now int64) {
	c.inputHistory = append(c.inputHistory, pair{num, now})
}

// return 0 if history frame state + 50ms < now. That means now is larger than 50ms.
// return max int if there is only one history frame.
// return normal gap between now and history frame + 50ms.
//
// if the frame is not acked after ECHO_TIMEOUT, give short WaitTime to accelerate
// ack actions.
func (c *Complete) WaitTime(now int64) int {
	// defer util.Log.Debug("Complete WaitTime", "inputHistory length", len(c.inputHistory))
	if len(c.inputHistory) < 2 {
		return math.MaxInt
	}
	// start from the second
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
	return c.diffFrom(existing, true)
}

// implements network.State[C any] interface
// get difference between this Complete and a new one.
func (c *Complete) InitDiff() string {
	blank, _ := NewComplete(c.terminal.GetWidth(), c.terminal.GetHeight(), c.terminal.GetSaveLines())
	return c.diffFrom(blank, false)
}

func (c *Complete) diffFrom(existing *Complete, diffBuf bool) string {
	hm := pb.HostMessage{}

	if existing.GetEchoAck() != c.GetEchoAck() {
		echoack := c.GetEchoAck()
		instEcho := pb.Instruction{Echoack: &pb.EchoAck{EchoAckNum: &echoack}}
		hm.Instruction = append(hm.Instruction, &instEcho)
	}

	// if !reflect.DeepEqual(existing.getFramebuffer(), c.getFramebuffer()) {
	// if !c.getFramebuffer().Equal(existing.getFramebuffer()) {
	if !c.Equal(existing) {
		if existing.terminal.GetWidth() != c.terminal.GetWidth() ||
			existing.terminal.GetHeight() != c.terminal.GetHeight() {
			w := int32(c.terminal.GetWidth())
			h := int32(c.terminal.GetHeight())
			instResize := pb.Instruction{Resize: &pb.ResizeMessage{Width: &w, Height: &h}}
			hm.Instruction = append(hm.Instruction, &instResize)
		}

		// the following part consider the cursor movement.
		// update := c.display.NewFrame(true, existing.terminal, c.terminal)
		var update string
		if diffBuf {
			update = c.diffBuf.String()
		} else {
			update = c.display.NewFrame(true, existing.terminal, c.terminal)
		}
		if len(update) > 0 {
			instBytes := pb.Instruction{Hostbytes: &pb.HostBytes{Hoststring: []byte(update)}}
			hm.Instruction = append(hm.Instruction, &instBytes)
		}
		// util.Log.Debug("DiffFrom","diff", c.diffBuf.String())
		// c.Reset()
	}

	output, _ := proto.Marshal(&hm)
	return string(output)
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
			// util.Logger.Trace("ApplyString", "after", "apply",
			// 	"cursor.row", c.terminal.GetCursorRow(), "cursor.col", c.terminal.GetCursorCol())
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
	if c.echoAck != x.echoAck {
		return false
	}

	return c.terminal.Equal(x.terminal)
}

// implements network.State[C any] interface
func (c *Complete) ResetInput() {
	// c.terminal.GetParser().ResetInput()
	//
	// NOTE: to prevent broken control sequence, keep the parser state between
	// continuously output from host.
	// such as "\x1b", "H"
}

// implements network.State[C any] interface
func (c *Complete) Reset() {
	c.diffBuf.Reset()
	c.terminal.SetLastRows(0)
}

// implements network.State[C any] interface
func (c *Complete) Clone() *Complete {
	clone := Complete{}

	clone = *c
	clone.display = c.display.Clone()
	clone.terminal = c.terminal.Clone()
	clone.diffBuf.Reset()
	clone.remainsBuf.Reset()

	clone.inputHistory = make([]pair, len(c.inputHistory))
	copy(clone.inputHistory, c.inputHistory)

	return &clone
}

// fot test purpose
// implements network.State[C any] interface
func (c *Complete) EqualTrace(x *Complete) bool {
	if c.echoAck != x.echoAck {
		msg := fmt.Sprintf("echoAck=(%d,%d)", c.echoAck, x.echoAck)
		util.Logger.Warn(msg)
		return false
	}

	ret := c.terminal.EqualTrace(x.terminal)
	return ret
}
