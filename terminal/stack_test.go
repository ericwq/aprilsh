// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"errors"
	"testing"
)

func TestStack(t *testing.T) {
	tc := []struct {
		label string
		data  []int
	}{
		{"int stack", []int{3, 4, 5, 6}},
	}

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			s := NewStack[int](len(v.data))
			for i := range v.data {
				s.Push(v.data[i])
			}

			for i := len(v.data) - 1; i >= 0; i-- {
				got, _ := s.Pop()
				if got != v.data[i] {
					t.Errorf("%s expect %d pop %d, got %d\n", v.label, i, v.data[i], got)
				}
			}
		})
	}
}

func TestStack_Oversie_Empty(t *testing.T) {
	// init data
	data := []rune{'a', 'b', 'c'}
	s := NewStack[rune](len(data))
	for i := range data {
		s.Push(data[i])
	}

	// oversize push
	s.Push('d')

	expect := []rune{'b', 'c', 'd'}

	for i := len(expect) - 1; i >= 0; i-- {
		got, _ := s.Pop()
		if got != expect[i] {
			t.Errorf("stack oversize expect %d pop %c, got %c\n", i, expect[i], got)
		}
	}

	// empty pop
	_, err := s.Pop()
	if !errors.Is(err, ErrEmptyStack) {
		t.Errorf("stack empty pop expect %s, got %s\n", ErrEmptyStack, err)
	}
}
