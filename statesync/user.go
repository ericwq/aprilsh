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
	"reflect"

	"github.com/ericwq/aprilsh/terminal"
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

// https://appliedgo.com/blog/generic-interface-functions
// google: golang interface return self
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
func (u *UserStream) Subtract(prefix *UserStream) {
	// if we are subtracting ourself from ourself, just clear the deque
	if u.equal(prefix) {
		u.actions = make([]UserEvent, 0)
		return
	}

	for i := range prefix.actions {
		if len(u.actions) > 1 && u.actions[0] == prefix.actions[i] {
			u.actions = u.actions[1:]
		}
	}
}

func (u *UserStream) diffFrom(prefix *UserStream) string {
	fmt.Println("#UserStream subtract")
	return ""
}

func (u *UserStream) initDiff() string {
	return ""
}

func (u *UserStream) equal(x *UserStream) bool {
	return reflect.DeepEqual(u.actions, x.actions)
}

// TODO move it to another file
type CompleteTerminal struct {
	action []string
}

func (u *CompleteTerminal) subtract(prefix *CompleteTerminal) {
	fmt.Println("#CompleteTerminal subtract")
}
