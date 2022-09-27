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
	"reflect"

	pb "github.com/ericwq/aprilsh/protobufs/user"
	"github.com/ericwq/aprilsh/terminal"
	"google.golang.org/protobuf/proto"
)

type UserEventType int

const (
	UserByteType UserEventType = iota
	ResizeType
)

type UserEvent struct {
	theType  UserEventType
	userByte terminal.UserByte // Parser::UserByte
	resize   terminal.Resize   // Parser::Resize
}

func NewUserByte(userByte terminal.UserByte) (u UserEvent) {
	u = UserEvent{}

	u.theType = UserByteType
	u.userByte = userByte

	return u
}

func NewUserResize(resize terminal.Resize) (u UserEvent) {
	u = UserEvent{}

	u.theType = ResizeType
	u.resize = resize

	return u
}

// UserStream implements network.State[C any] interface
type UserStream struct {
	actions []UserEvent
}

func (u *UserStream) pushBack(userByte terminal.UserByte) {
	u.actions = append(u.actions, NewUserByte(userByte))
}

// in go, we can only use different method name for pushBack()
func (u *UserStream) pushBackResize(resize terminal.Resize) {
	u.actions = append(u.actions, NewUserResize(resize))
}

func (u *UserStream) ResetInput() {}

// interface for network.State[C any]

// Subtract() the prefix UserStream from current UserStream
func (u *UserStream) Subtract(prefix *UserStream) {
	// if we are subtracting ourself from ourself, just clear the deque
	if u.Equal(prefix) {
		u.actions = make([]UserEvent, 0)
		return
	}

	for i := range prefix.actions {
		if len(u.actions) > 1 && u.actions[0] == prefix.actions[i] {
			u.actions = u.actions[1:]
		}
	}
}

func (u *UserStream) DiffFrom(existing *UserStream) string {
	// remove the existing part
	pos := 0
	for i := range existing.actions {
		if u.actions[pos] == existing.actions[i] {
			if pos+1 <= len(u.actions[pos:])-1 {
				pos++
			}
		}
	}

	// create the UserMessage based on content in UserStream
	um := pb.UserMessage{}
	for _, ue := range u.actions[pos:] {
		switch ue.theType {
		case UserByteType:
			theByte := ue.userByte.GetRune()
			pos := len(um.Instruction) // TODO the last one?

			if len(um.Instruction) > 0 && um.Instruction[pos].Keystroke != nil {
				// append Keys for Keystroke
				keys := []byte(string(theByte))

				um.Instruction[pos].Keystroke.Keys = append(um.Instruction[0].Keystroke.Keys, keys...)
			} else {
				// create a new Instruction for Keystroke
				um.Instruction = make([]*pb.Instruction, 0)

				inst := pb.Instruction{
					Keystroke: &pb.Keystroke{Keys: []byte(string(theByte))},
				}
				um.Instruction = append(um.Instruction, &inst)
			}
		case ResizeType:
			// create a new Instruction for ResizeMessage
			inst := pb.Instruction{
				Resize: &pb.ResizeMessage{Width: int32(ue.resize.GetWidth()), Height: int32(ue.resize.GetHeight())},
			}
			um.Instruction = append(um.Instruction, &inst)
		default:
			break
		}
	}

	// get the wire-format encoding of UserMessage
	output, _ := proto.Marshal(&um)
	// if err != nil {
	// 	panic(fmt.Sprintf("#DiffFrom marshal %s ", err))
	// }

	return string(output)
}

func (u *UserStream) InitDiff() string {
	// this should not be called
	return ""
}

func (u *UserStream) ApplyString(diff string) {
	// parse the wire-format encoding of UserMessage
	input := pb.UserMessage{}
	err := proto.Unmarshal([]byte(diff), &input)
	if err != nil {
		return
	}

	// create the UserStream based on content of UserMessage
	for i := range input.Instruction {
		if input.Instruction[i].Keystroke != nil {
			for _, v := range input.Instruction[i].Keystroke.Keys {
				u.actions = append(u.actions, NewUserByte(terminal.UserByte{C: rune(v)}))
			}
		} else if input.Instruction[i].Resize != nil {
			w := input.Instruction[i].Resize
			u.actions = append(u.actions, NewUserResize(terminal.Resize{Width: int(w.Width), Height: int(w.Height)}))
		}
	}
}

func (u *UserStream) Equal(x *UserStream) bool {
	return reflect.DeepEqual(u.actions, x.actions)
}
