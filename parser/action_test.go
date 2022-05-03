package parser

import (
	"testing"
)

func TestIgnore(t *testing.T) {
	ig := ignore{action{ch: 0, present: false}}

	if ig.Ignore() == false {
		t.Errorf("%s expect ture, got %t\n", ig.Name(), ig.Ignore())
	}

	ig2 := ignore{}
	if ig != ig2 {
		t.Errorf("expect %v, got %v\n", ig, ig2)
	}
}

func TestUserByte(t *testing.T) {
	u1 := UserByte{c: 'f'}

	if u1.Ignore() {
		t.Errorf("%s expect false, got %t\n", u1.Name(), u1.Ignore())
	}

	u2 := UserByte{c: 'f'}
	if u1 != u2 {
		t.Errorf("expect %v, got %v\n", u1, u2)
	}
}
