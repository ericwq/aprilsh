// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

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
	EqualTrace(x C) bool // for test purpose

	// interface from code
	ResetInput()
	Reset()
	// SetLastRows(x int)
	// GetLastRows() int
	InitSize(y, x int)
	Clone() C
}

type TimestampedState[T State[T]] struct {
	timestamp int64
	num       uint64
	state     T
}

func (t *TimestampedState[T]) numEq(v uint64) bool {
	return t.num == v
}

func (t *TimestampedState[T]) numLt(v uint64) bool {
	return t.num < v
}

func (t *TimestampedState[T]) clone() TimestampedState[T] {
	clone := TimestampedState[T]{}

	clone.timestamp = t.timestamp
	clone.num = t.num
	clone.state = t.state.Clone()

	return clone
}

func (t *TimestampedState[T]) GetTimestamp() int64 {
	return t.timestamp
}

func (t *TimestampedState[T]) GetState() T {
	return t.state
}
