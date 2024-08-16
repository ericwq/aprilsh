// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"errors"
	"sync"
)

var (
	ErrEmptyStack = errors.New("empty stack")
	ErrLastItem   = errors.New("last item in stack")
)

type stack[V comparable] struct {
	data []V
	max  int
	sync.Mutex
}

// create LIFO stack with max items
func NewStack[V comparable](max int) *stack[V] {
	s := &stack[V]{}
	s.max = max
	s.data = make([]V, 0, max)
	return s
}

// push new item input stack.
//
// If a push request is received and the stack is full, the oldest entry from
// the stack is evicted.
func (s *stack[V]) Push(v V) int {
	s.Lock()
	defer s.Unlock()

	if len(s.data) >= s.max {
		s.data = append(s.data[1:], v)
	} else {
		s.data = append(s.data, v)
	}

	return len(s.data)
}

// pop last item from stack.
//
// If a pop request is received that empties the stack, report ErrLastData.
// if a pop request is received and the stack is empty, report ErrEmptyStack.
func (s *stack[V]) Pop() (last V, err error) {
	s.Lock()
	defer s.Unlock()

	pos := len(s.data)
	if pos == 0 {
		return last, ErrEmptyStack
	}

	last = s.data[pos-1]
	s.data = s.data[:pos-1]
	if len(s.data) == 0 {
		err = ErrLastItem
	}

	return last, err
}

func (s *stack[V]) GetPeek() (v V) {
	return s.data[len(s.data)-1]
}

func (s *stack[V]) UpdatePeek(v V) {
	s.data[len(s.data)-1] = v
}

func (s *stack[V]) Clone() *stack[V] {
	clone := NewStack[V](s.max)
	clone.data = make([]V, len(s.data))
	copy(clone.data, s.data)
	return clone
}

func (s *stack[V]) Equal(c *stack[V]) bool {
	if s.max != c.max {
		return false
	}
	if len(s.data) != len(c.data) {
		return false
	}
	for i := range s.data {
		if s.data[i] != c.data[i] {
			return false
		}
	}

	return true
}
