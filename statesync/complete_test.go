// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package statesync

import (
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	"golang.org/x/exp/slog"
)

func TestCompleteSubtract(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)
	c.Subtract(c) // do nothing, just for coverage
	c.GetEmulator()
}

func TestCompleteInitDiff(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)
	got := c.InitDiff()

	expect := ""
	if expect != got {
		t.Errorf("#test InitDiff() expect %q, got %q\n", expect, got)
	}
}

func TestCompleteApplyString(t *testing.T) {
	tc := []struct {
		label         string
		seq           string
		width, height int
		echoAck       uint64
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", 0, 0, 0},
		{"fill one row and resize", "\x1B[6;67HLAST", 70, 30, 0},
		{"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)

	for _, v := range tc {
		c0, _ := NewComplete(80, 40, 40)
		c1, _ := NewComplete(80, 40, 40)

		// resize new state if necessary
		if v.height != 0 && v.width != 0 {
			r := terminal.Resize{Width: v.width, Height: v.height}
			emu := c1.terminal
			r.Handle(emu)
		}

		// print some data on screen
		c1.terminal.HandleStream(v.seq)

		// validate the equal is false
		if c1.Equal(c0) {
			t.Errorf("%q expect false equal(), got true", v.label)
		}

		// set echoAck for new state
		if v.echoAck != 0 {
			c1.echoAck = v.echoAck
		}

		// new state calculate difference with old state as parameter
		diff := c1.DiffFrom(c0)

		// apply to the old state
		c0.ApplyString(diff)

		// validate the result
		if got := c0.DiffFrom(c1); got != "" {
			// if !c0.Equal(c1) {
			// got := c0.DiffFrom(c1)
			t.Errorf("%q expect empty result after ApplyString(), got %q\n", v.label, got)
		}
	}
}

func TestApplyString_Fail(t *testing.T) {
	diff := "mislead\n\x04:\x02@\x03\n2\x120\"."

	c, _ := NewComplete(80, 40, 40)
	if err := c.ApplyString(diff); err == nil {
		t.Error("#test feed ApplyString with wrong parameter, expect error.")
	}
}

func TestCompleteSetEchoAck(t *testing.T) {
	tc := []struct {
		label  string
		data   []pair
		expect bool
	}{
		{"find two states", []pair{{1, 49}, {2, 43}, {3, 52}}, true},
		{"too quick to find the latest state", []pair{{1, 9}, {2, 13}, {3, 12}}, false},
	}

	c, _ := NewComplete(8, 4, 4)
	now := time.Now().UnixMilli()

	for _, v := range tc {
		// reset history
		c.inputHistory = make([]pair, 0)
		c.echoAck = 0

		// register the frame number and time
		var ts int64 = 0
		for _, p := range v.data {

			ts += p.timestamp
			// note: the timestamp is delta value in ms.
			c.RegisterInputFrame(p.frameNum, now+ts)
			// fmt.Printf("#test setEchoAck timestamp=%d, ts=%d\n", p.timestamp, ts)
		}

		// fmt.Printf("#test setEchoAck inputHistory = %v\n", c.inputHistory)

		got := c.SetEchoAck(now + ts)
		// fmt.Printf("#test setEchoAck inputHistory = %v\n", c.inputHistory)
		if v.expect != got {
			t.Errorf("%q expect %t, got %t\n", v.label, v.expect, got)
		}
	}
}

func TestCompleteWaitTime(t *testing.T) {
	tc := []struct {
		label  string
		data   []pair
		time   int64
		expect int
	}{
		{"history size <2", []pair{{1, 49}}, 0, math.MaxInt},
		{"now < last +50 ", []pair{{1, 49}, {2, 43}}, 9, 50 - 9},
		{"last +50 <= now", []pair{{1, 49}, {2, 43}}, 50, 0},
	}

	c, _ := NewComplete(8, 4, 4)
	now := time.Now().UnixMilli()

	for _, v := range tc {
		// reset history
		c.inputHistory = make([]pair, 0)
		c.echoAck = 0

		// register the frame number and time
		var ts int64 = 0
		for _, p := range v.data {

			ts += p.timestamp
			// note: the timestamp is delta value in ms.
			c.RegisterInputFrame(p.frameNum, now+ts)
			// fmt.Printf("#test setEchoAck timestamp=%d, ts=%d\n", p.timestamp, ts)
		}

		got := c.WaitTime(now + ts + v.time)
		if v.expect != got {
			t.Errorf("%q expect %d, got %d\n", v.label, v.expect, got)
		}
	}
}

func TestCompleteResetInput(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)

	c.ResetInput()
	if c.terminal.GetCursorCol() != 0 || c.terminal.GetCursorRow() != 0 {
		t.Errorf("#test after resetInput() the cursor should be in (0,0), got (%d,%d)\n",
			c.terminal.GetCursorRow(), c.terminal.GetCursorCol())
	}
}

func TestCompleteClone(t *testing.T) {
	c, _ := NewComplete(8, 4, 4)
	clone := c.Clone()

	if !c.Equal(clone) {
		t.Errorf("#test clone expect %v, got %v\n", c, clone)
	}
}

func TestDiffFrom(t *testing.T) {
	tc := []struct {
		label        string
		nCols, nRows int      // screen size
		seq1         []string // sequence before action
		seq2         []string // sequence after action
		resp         string
	}{
		{"simple case", 80, 40, []string{},
			[]string{
				"nvide:0.8.9\r\n",
				"\r\nLua, C/C++ and Golang Integrated Development Environment.\r\nPowered by neovim, luals, gopls and clangd.\r\n", "\x1b]0;aprilsh\a",
				"ide@openrc-nvide:~/develop $ \x1b[6n"}, "\x1b[5;30R"},
		{"vi and quit", 80, 40,
			[]string{
				"\x1b]0;aprilsh\a", // set title to avoid title stack warning
				/*vi start*/ "\x1b[?1049h\x1b[22;0;0t\x1b[?1h\x1b=\x1b[H\x1b[2J\x1b]11;?\a\x1b[?2004h\x1b[?u\x1b[c\x1b[?25h",
				// "\x1b]11;rgb:0000/0000/0000\x1b\\\x1b[?64;1;9;15;21;22c"
				/*clear screen*/ "\x1b[?25l\x1b(B\x1b[m\x1b[H\x1b[2J\x1b[>4;2m\x1b]112\a\x1b[2 q\x1b[?1002h\x1b[?1006h\x1b[38;2;233;233;244m\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[J\x1b[H",
				/*vi file*/ "\x1b(B\x1b[m\x1b[38;2;98;100;131m  \x1b(B\x1b[m\x1b[38;2;248;248;242m1 \x1b(B\x1b[m\x1b[38;2;233;233;244m\x1b[48;2;45;48;62mgo 1.19                                                                                                                                                         \r\n\x1b(B\x1b[m\x1b[38;2;98;100;131m  \x1b(B\x1b[m\x1b[38;2;94;95;105m2 \x1b(B\x1b[m\x1b[38;2;233;233;244m\x1b[K\r\n\x1b(B\x1b[m\x1b[38;2;98;100;131m  \x1b(B\x1b[m\x1b[38;2;94;95;105m3 \x1b(B\x1b[m\x1b[38;2;233;233;244muse (\x1b[K\r\n\x1b(B\x1b[m\x1b[38;2;98;100;131m  \x1b(B\x1b[m\x1b[38;2;94;95;105m4 \x1b(B\x1b[m\x1b[38;2;233;233;244m   ./aprilsh\x1b[K\r\n\x1b(B\x1b[m\x1b[38;2;98;100;131m  \x1b(B\x1b[m\x1b[38;2;94;95;105m5 \x1b(B\x1b[m\x1b[38;2;233;233;244m   ./terminfo\x1b[K\r\n\x1b(B\x1b[m\x1b[38;2;98;100;131m  \x1b(B\x1b[m\x1b[38;2;94;95;105m6 \x1b(B\x1b[m\x1b[38;2;233;233;244m)\x1b[K\x1b(B\x1b[m\x1b[38;2;98;100;131m\r\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b[K\n\x1b(B\x1b[0;1m\x1b[38;2;40;42;54m\x1b[48;2;139;155;205m \ue7c5\x1b[39;3H NORMAL\x1b(B\x1b[m\x1b[38;2;139;155;205m\x1b[48;2;94;95;105m\ue0bc\x1b[39;11H \x1b(B\x1b[m\x1b[38;2;94;95;105m\x1b[48;2;65;67;79m\ue0bc\x1b[39;13H \x1b(B\x1b[m\x1b[38;2;248;248;242m\x1b[48;2;65;67;79m go.work \x1b(B\x1b[m\x1b[38;2;65;67;79m\ue0bc\x1b[39;24H                                                                                                                         \x1b(B\x1b[m\x1b[38;2;255;112;112m\ue0b6\x1b[39;146H\x1b(B\x1b[m\x1b[38;2;55;56;68m\x1b[48;2;255;112;112m\U000f024b\x1b[39;147H \x1b(B\x1b[m\x1b[38;2;248;248;242m\x1b[48;2;65;67;79m develop \x1b(B\x1b[m\x1b[38;2;80;250;123m\x1b[48;2;65;67;79m\ue0b6\x1b[39;158H\x1b(B\x1b[m\x1b[38;2;40;42;54m\x1b[48;2;80;250;123m\ue612\x1b[39;159H \x1b(B\x1b[m\x1b[38;2;80;250;123m\x1b[48;2;65;67;79m Top \x1b(B\x1b[m\x1b[38;2;233;233;244m\r\n\x1b[J\x1b]112\a\x1b[2 q\x1b[1;5H\x1b[?25h",
				/*screen border*/ "\x1b[?25l\n\n\n\x1b(B\x1b[m\x1b[38;2;60;61;73m│\x1b[4;6H  \x1b[5;5H│\x1b[5;6H  \x1b[1;5H\x1b[?25h",
				/*loading*/ "\x1b[?25l\x1b[39;52H\x1b(B\x1b[m\x1b[38;2;80;250;123m \U000f0aa2\x1b[39;54H Setting up workspace Loading packages... (0%)                             \x1b(B\x1b[m\x1b[38;2;139;155;205m \uf085\x1b[39;131H  LSP ~ gopls \x1b[1;5H\x1b[?25h",
				/*loading*/ "\x1b[?25l\x1b[39;49H\x1b(B\x1b[m\x1b[38;2;80;250;123m \U000f0aa2\x1b[39;51H Setting up workspace Finished loading packages. (0%)\x1b[1;5H\x1b[?25h",
				/*loading*/ "\x1b[?25l\x1b[39;49H\x1b(B\x1b[m\x1b[38;2;65;67;79m                                                                                \x1b[1;5H\x1b[?25h",
			},
			[]string{
				/*1st sequence after :q*/ "\x1b[?25l\r\x1b[40;1H\x1b[?25h",
				/*2nd sequence after :q*/ "\x1b[?25l\x1b]112\a\x1b[2 q\x1b[?25h",
				/*3rd sequence after :q*/ "\x1b[?25l\x1b]112\a\x1b[2 q\x1b[?1002l\x1b[?1006l\x1b(B\x1b[m\x1b[?25h\x1b[?1l\x1b>\x1b[>4;0m\x1b[?1049l\x1b[23;0;0t\x1b[?2004l\x1b[?1004l\x1b[?25h",
				/*4th sequence after :q*/ "ide@openrc-nvide:~/develop $ \x1b[6n",
			}, "\x1b[1;30R"},
		{"screen with content then vi utf-8 file", 80, 40,
			[]string{
				"\x1b]0;aprilsh\a", // set title to avoid title stack warning
				"ide@openrc-nvide:~/develop $ \x1b[6n"},
			[]string{
				"\x1b[?1049h\x1b[22;0;0t\x1b[?1h\x1b=\x1b[H\x1b[2J\x1b]11;?\a\x1b[?2004h\x1b[?u\x1b[c\x1b[?25h",
			}, "\x1b]11;rgb:0000/0000/0000\x1b\\\x1b[?64;1;9;15;21;22c"},
		{"cat file larger than screen rows", 80, 40,
			[]string{
				"nvide:0.8.9\r\n\r\nLua, C/C++ and Golang Integrated Development Environment.\r\nPowered by neovim, luals, gopls and clangd.\r\n",
				"\x1b]0;aprilsh\a",
				"ide@openrc-nvide:~ $ cd develop\r\n",
				"ide@openrc-nvide:~/develop $ ls\r\n",
				"\x1b[1;34mNvChad\x1b[m                     \x1b[0;0mgit.md\x1b[m                     \x1b[1;34mgolangIDE\x1b[m                  \x1b[1;34mneovim-lua\x1b[m                 \x1b[1;34ms6\x1b[m\r\n\x1b[1;34maprilsh\x1b[m                    \x1b[0;0mgo.work\x1b[m                    \x1b[1;34mmosh\x1b[m                       \x1b[1;34mnvide\x1b[m                      \x1b[1;34mtelescope.nvim\x1b[m\r\n\x1b[1;34mdotfiles\x1b[m                   \x1b[0;0mgo.work.sum\x1b[m                \x1b[0;0mmosh-1.3.2.tar.gz\x1b[m          \x1b[0;0mpersonal-access-token.txt\x1b[m  \x1b[1;34mterminfo\x1b[m\r\n",
				"ide@openrc-nvide:~/develop $ cat tokens.txt\r\n",
			},
			[]string{"111\r\naaa\r\nbbb\r\n\r\nccc\r\nddd\r\n\r\neeee\r\nffff\r\n\r\nggg\r\nhhh#\r\n\r\napp\r\niii\r\n\r\njjj\r\nkkk#\r\n\r\nlll\r\nmmm\r\n\r\n#nnn \r\n\r\nooo.\r\nppp\r\n\r\nqqq\r\nrrr\r\nsss\r\nttt\r\n\r\n隐uuu\r\n方式vvv\r\niwww\r\nxxx\r\n\r\nyyy\r\nzzz\r\n\r\n专用aaa\r\nbbb\r\nccc\r\nddd\r\n\r\neee\r\nfff\r\nggg\r\n"}, // total 48 \n
			""},
		{"return on last row", 101, 24,
			[]string{"nvide:0.8.9\r\n",
				"\r\nLua, C/C++ and Golang Integrated Development Environment.\r\nPowered by neovim, luals, gopls and clangd.\r\n",
				"\x1b]0;aprilsh\a",
				"ide@openrc-nvide:~ $ ls\r\n",
				"\x1b[1;34mdevelop\x1b[m  \x1b[1;34mproj\x1b[m\r\n",
				"ide@openrc-nvide:~ $ cd develop\r\n",
				"ide@openrc-nvide:~/develop $ ls -al\r\n",
				"total 1696\r\ndrwxr-xr-x   22 ide      develop        704 Oct 26 20:28 \x1b[1;34m.\x1b[m\r\ndrwxr-sr-x    1 ide      develop       4096 Oct 26 21:08 \x1b[1;34m..\x1b[m\r\n-rw-r--r--    1 ide      develop       8196 Sep 15 22:10 \x1b[0;0m.DS_Store\x1b[m\r\ndrwxr-xr-x   19 ide      develop        608 Oct 27 19:13 \x1b[1;34maprilsh\x1b[m\r\ndrwxr-xr-x   14 ide      develop        448 May 26  2022 \x1b[1;34mdocTerminal\x1b[m\r\ndrwxr-xr-x   29 ide      develop        928 Aug  7 22:02 \x1b[1;34mexamples\x1b[m\r\n-rw-r--r--    1 ide      develop         50 May 21 14:35 \x1b[0;0mgo.work\x1b[m\r\n-rw-r--r--    1 ide      develop        694 Aug  5 22:36 \x1b[0;0mgo.work.sum\x1b[m\r\ndrwxr-xr-x   18 ide      develop        576 Jun 24 22:36 \x1b[1;34mgoutmp\x1b[m\r\ndrwxr-xr-x    9 ide      develop        288 Dec 17  2020 \x1b[1;34mgskills\x1b[m\r\n-rwxr-xr-x    1 ide      develop    1484717 Nov 15  2021 \x1b[1;32mgskills.key\x1b[m\r\ndrwxr-xr-x    6 ide      develop        192 Nov 15  2021 \x1b[1;34mimages\x1b[m\r\ndrwxr-sr-x   32 ide      develop       1024 May  9  2021 \x1b[1;34mldd3\x1b[m\r\ndrwxr-xr-x   16 ide      develop        512 Jun 15 21:48 \x1b[1;34mlibutempter\x1b[m\r\ndrwxr-xr-x   51 ide      develop       1632 Apr 30  2022 \x1b[1;34mncurses-6.3\x1b[m\r\ndrwxr-xr-x   14 ide      develop        448 Sep 27 22:01 \x1b[1;34mnvide\x1b[m\r\n-rw-rw-rw-    1 ide      develop        782 Oct 26 20:27 \x1b[0;0mpersonal-access-token.txt\x1b[m\r\ndrwxr-xr-x   10 ide      develop        320 May 30 18:59 \x1b[1;34ms6\x1b[m\r\ndrwxr-xr-x   33 ide      develop       1056 Jun 28 06:17 \x1b[1;34mterminfo\x1b[m\r\ndrwxr-xr-x   82 ide      develop       2624 May 11  2022 \x1b[1;34mvttest-20220215\x1b[m\r\n-rw-r--r--    1 ide      develop     216949 May 11  2022 \x1b[0;0mvttest.tar.gz\x1b[m\r\ndrwxr-xr-x    5 ide      develop        160 May  2  2022 \x1b[1;34mworkspace\x1b[m\r\nide@openrc-nvide:~/develop $ ",
			},
			[]string{"\r\n"},
			""},
		{"write on ring buffer", 8, 4,
			[]string{"8888888\r\na\r\nb\r\nc\r\n",
				"a1\r\na2\r\na3\r\na4\r\n",
				"b2\r\nb2\r\nb3\r\nb4\r\n",
				"c2\r\nc2\r\nc3\r\nc4\r\n",
			},
			[]string{"x1\r\nx2\r\nx3\r\nx4\r\n"}, ""},
	}

	defer util.Log.Restore()
	util.Log.SetOutput(io.Discard)
	// util.Log.SetLevel(slog.LevelDebug)
	// util.Log.SetOutput(os.Stderr)

	for _, v := range tc {
		t.Run(v.label, func(t *testing.T) {
			savedLines := v.nRows * 3

			a, _ := NewComplete(v.nCols, v.nRows, savedLines)
			c, _ := NewComplete(v.nCols, v.nRows, savedLines)

			// fmt.Printf("#TestDiffFrom point=%d\n", 333)
			// assumed state prepare
			var t2 strings.Builder
			var ss strings.Builder
			for i := range v.seq1 {
				ss.WriteString(v.seq1[i])
			}
			t2.WriteString(a.Act(ss.String()))
			// fmt.Printf("#TestDiffFrom point=%d\n", 444)
			c.Act(ss.String())

			if !c.EqualTrace(a) {
				t.Errorf("%s: prepare stage error\n", v.label)
			}

			// current state changed after :q command
			t2.Reset()
			ss.Reset()
			for i := range v.seq2 {
				ss.WriteString(v.seq2[i])
			}

			t2.WriteString(c.Act(ss.String()))
			if v.resp != t2.String() {
				t.Errorf("%s: terminal response expect %q, got %q\n", v.label, v.resp, t2.String())
			}

			util.Log.With("point", 601).Debug("TestDiffFrom")
			diff := c.DiffFrom(a)
			util.Log.With("point", 602).Debug("TestDiffFrom")

			n := a.Clone()
			n.ApplyString(diff)
			if !c.EqualTrace(n) {
				t.Errorf("%s: round-trip Instruction verification failed!", v.label)
				// t.Logf("%s: diff=%q", v.label, diff)
			}

			util.Log.With("point", 603).Debug("TestDiffFrom")
			cd := c.InitDiff()
			util.Log.With("point", 604).Debug("TestDiffFrom")
			nd := n.InitDiff()
			util.Log.With("point", 605).Debug("TestDiffFrom")
			if cd != nd {
				t.Errorf("%s: target state Instruction verification failed!", v.label)
				// t.Logf("current state diff=%q", cd)
				// t.Logf("new     state diff=%q", nd)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"custom equal", "\x1B[6;67HLAST\x1B[1;7H", "\x1B[6;67HLAST\x1B[1;7H"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	if !c0.Equal(c1) {
		t.Errorf("%q expect not equal object\n", v.label)
	}
}

func (c *Complete) equalDiffFrom(x *Complete) bool {
	// use DiffFrom to compare the state
	if diff := c.DiffFrom(x); diff != "" {
		return false
	}
	return true
	// return reflect.DeepEqual(c.terminal, x.terminal) && c.echoAck == x.echoAck
}

func (c *Complete) deepEqual(x *Complete) bool {
	return reflect.DeepEqual(c.terminal, x.terminal) && c.echoAck == x.echoAck
}

// check Equal mthod
// func (c *Complete) customEqual(x *Complete) bool {
// 	if c.echoAck != x.echoAck {
// 		return false
// 	}
//
// 	return c.terminal.Equal(x.terminal)
// }

// https://blog.logrocket.com/benchmarking-golang-improve-function-performance/
// https://coder.today/tech/2018-11-10_profiling-your-golang-app-in-3-steps/
// https://www.speedscope.app/
func BenchmarkDiffFromEqual(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[6;67HLAST", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.equalDiffFrom(c1)
	}
}

func BenchmarkDeepEqual(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[6;67HLAST", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.deepEqual(c1)
	}
}

func BenchmarkEqual(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[6;67HLAST", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.Equal(c1)
	}
}

func BenchmarkDiffFrom(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.DiffFrom(c1)
	}
}

func BenchmarkFramebuffer_Equal(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.getFramebuffer().Equal(c1.getFramebuffer())
	}
}

func BenchmarkNewFrame(b *testing.B) {
	tc := []struct {
		label string
		seq0  string
		seq1  string
	}{
		{"fill one row with string", "\x1B[4;4HErase to the end of line\x1B[0K.", "\x1B[6;67HLAST"},
		// {"fill one row and set ack", "\x1B[7;7H左边\x1B[7;77H中文", 0, 0, 3},
	}
	v := tc[0]
	c0, _ := NewComplete(80, 40, 40)
	c1, _ := NewComplete(80, 40, 40)

	c0.terminal.HandleStream(v.seq0)
	c1.terminal.HandleStream(v.seq1)

	for i := 0; i < b.N; i++ {
		c0.display.NewFrame(true, c0.terminal, c1.terminal)
	}
}
