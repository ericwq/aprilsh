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
			t.Errorf("stack oversize: expect %d pop %c, got %c\n", i, expect[i], got)
		}
	}

	// empty pop
	_, err := s.Pop()
	if !errors.Is(err, ErrEmptyStack) {
		t.Errorf("stack empty pop: expect %s, got %s\n", ErrEmptyStack, err)
	}

	// push one item into stack
	s.Push('x')

	x, err := s.Pop()
	if !errors.Is(err, ErrLastItem) {
		t.Errorf("stack pop last item: expect %s, got %s\n", ErrLastItem, err)
	}
	if x != 'x' {
		t.Errorf("stack pop last item: expect %c, got %c\n", 'x', x)
	}
}

func TestStack_Equal(t *testing.T) {
	data := []string{"first", "second", "third"}
	s := NewStack[string](len(data))
	for i := range data {
		s.Push(data[i])
	}

	s2 := s.Clone()

	// same stack compare
	if !s.Equal(s2) {
		t.Errorf("stack equal: expect true, got false\n")
	}

	// empty stack compare
	s3 := NewStack[string](len(data))
	if s.Equal(s3) {
		t.Errorf("stack equal: expect false, got true\n")
	}

	// different item compare
	s2.data[2] = "forth"
	if s.Equal(s2) {
		t.Errorf("stack equal: expect false, got true\n")
	}

	// diff max compare
	s4 := NewStack[string](len(data) - 1)
	if s.Equal(s4) {
		t.Errorf("stack equal: expect false, got true\n")
	}
}

func TestStack_Peek(t *testing.T) {
	data := []float64{1.1, 2.2, 3.3}
	s := NewStack[float64](len(data))
	for i := range data {
		s.Push(data[i])
	}
	// fmt.Printf("after init: %v\n", s.data)

	// validate get peek
	d2 := s.GetPeek()
	if d2 != data[2] {
		t.Errorf("stack get peek: expect %f, got %f\n", data[2], d2)
	}

	// validate update peek
	newf := 4.5
	s.UpdatePeek(newf)
	d2 = s.GetPeek()
	if d2 != newf {
		t.Errorf("stack get peek: expect %f, got %f\n", newf, d2)
	}

	// fmt.Printf("after test: %v\n", s.data)
}
