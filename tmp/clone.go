package main

import "fmt"

type Cloner[C any] interface {
	Clone() C
}

type CloneableSlice []int

func (c CloneableSlice) Clone() CloneableSlice {
	res := make(CloneableSlice, len(c))
	copy(res, c)
	return res
}

type CloneableMap map[int]int

func (c CloneableMap) Clone() CloneableMap {
	res := make(CloneableMap, len(c))
	for k, v := range c {
		res[k] = v
	}
	return res
}

func CloneAny[T Cloner[T]](c T) T {
	return c.Clone()
}

func main() {
	s := CloneableSlice{1, 2, 3, 4}
	fmt.Println(CloneAny(s))

	m := CloneableMap{1: 1, 2: 2, 3: 3, 4: 4}
	fmt.Println(CloneAny(m))
}
