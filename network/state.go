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

package network

// State is implemented by UserSteam or CompleteTerminal. The type parameter is
// required to meet the requirement: the concrete type, such as UserSteam or CompleteTerminal,
// can use the concrete type for method parameter or return type instead of interface.
// self reference in method parameter and return type is not common, pay attention to it.
// [ref](https://appliedgo.com/blog/generic-interface-functions)
// The meaning of [C any]:
// the following methods requires a quite unspecified type C - basically, it can be anything.
type State[C any] interface {
	// interface for Network::Transport
	Subtract(x C)
	DiffFrom(x C) string
	InitDiff() string
	ApplyString(diff string) error
	Equal(x C) bool

	// interface from code
	ResetInput()
}

type TimestampedState[T State[T]] struct {
	timestamp int64
	num       int64
	state     T
}

// func NewTimestampedState2() *TimestampedState[UserStream] {
// 	ts := TimestampedState[UserStream]{}
// 	return &ts
// }

func (t *TimestampedState[T]) numEq(v int64) bool {
	return t.num == v
}

func (t *TimestampedState[T]) numLt(v int64) bool {
	return t.num < v
}
