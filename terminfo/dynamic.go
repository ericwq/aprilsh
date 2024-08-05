// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminfo

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type termcap struct {
	bools   map[string]bool
	nums    map[string]int
	strs    map[string]string
	name    string
	desc    string
	aliases []string
}

func (tc *termcap) getnum(s string) int {
	return (tc.nums[s])
}

func (tc *termcap) getflag(s string) bool {
	return (tc.bools[s])
}

func (tc *termcap) getstr(s string) string {
	return (tc.strs[s])
}

const (
	none = iota
	control
	escaped
)

// var errNotAddressable = errors.New("terminal not cursor addressable")

var nTerminfo *termcap

func unescape(s string) string {
	// Various escapes are in \x format.  Control codes are
	// encoded as ^M (carat followed by ASCII equivalent).
	// escapes are: \e, \E - escape
	//  \0 NULL, \n \l \r \t \b \f \s for equivalent C escape.
	buf := &bytes.Buffer{}
	esc := none

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch esc {
		case none:
			switch c {
			case '\\':
				esc = escaped
			case '^':
				esc = control
			default:
				buf.WriteByte(c)
			}
		case control:
			buf.WriteByte(c ^ 1<<6)
			esc = none
		case escaped:
			switch c {
			case 'E', 'e':
				buf.WriteByte(0x1b)
			case '0', '1', '2', '3', '4', '5', '6', '7':
				if i+2 < len(s) && s[i+1] >= '0' && s[i+1] <= '7' && s[i+2] >= '0' && s[i+2] <= '7' {
					buf.WriteByte(((c - '0') * 64) + ((s[i+1] - '0') * 8) + (s[i+2] - '0'))
					i = i + 2
				} else if c == '0' {
					buf.WriteByte(0)
				}
			case 'n':
				buf.WriteByte('\n')
			case 'r':
				buf.WriteByte('\r')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			case 'f':
				buf.WriteByte('\f')
			case 's':
				buf.WriteByte(' ')
			default:
				buf.WriteByte(c)
			}
			esc = none
		}
	}
	return (buf.String())
}

func (tc *termcap) setupterm(name string) error {
	cmd := exec.Command("infocmp", "-1", name)
	output := &bytes.Buffer{}
	cmd.Stdout = output
	cmd.Stderr = output

	tc.strs = make(map[string]string)
	tc.bools = make(map[string]bool)
	tc.nums = make(map[string]int)

	if err := cmd.Run(); err != nil {
		// this translaet the "exit status 1" into "infocmp: couldn't open terminfo file (null)."
		return errors.New(strings.TrimSpace(output.String()))
	}

	// Now parse the output.
	// We get comment lines (starting with "#"), followed by
	// a header line that looks like "<name>|<alias>|...|<desc>"
	// then capabilities, one per line, starting with a tab and ending
	// with a comma and newline.
	lines := strings.Split(output.String(), "\n")
	for len(lines) > 0 && strings.HasPrefix(lines[0], "#") {
		lines = lines[1:]
	}

	// Ditch trailing empty last line
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	header := lines[0]
	header = strings.TrimSuffix(header, ",")
	// if strings.HasSuffix(header, ",") {
	// 	header = header[:len(header)-1]
	// }
	names := strings.Split(header, "|")
	tc.name = names[0]
	names = names[1:]
	if len(names) > 0 {
		tc.desc = names[len(names)-1]
		names = names[:len(names)-1]
	}
	tc.aliases = names
	for _, val := range lines[1:] {
		if (!strings.HasPrefix(val, "\t")) ||
			(!strings.HasSuffix(val, ",")) {
			return (errors.New("malformed infocmp: " + val))
		}

		val = val[1:]
		val = val[:len(val)-1]

		if k := strings.SplitN(val, "=", 2); len(k) == 2 {
			tc.strs[k[0]] = unescape(k[1])
		} else if k := strings.SplitN(val, "#", 2); len(k) == 2 {
			u, err := strconv.ParseUint(k[1], 0, 0)
			if err != nil {
				return (err)
			}
			tc.nums[k[0]] = int(u)
		} else {
			tc.bools[val] = true
		}
	}
	return nil
}

func LookupTerminfo(capName string) (string, bool) {
	if v, ok := nTerminfo.nums[capName]; ok {
		return fmt.Sprintf("%d", v), true
	}

	if v, ok := nTerminfo.strs[capName]; ok {
		return v, true
	}

	if _, ok := nTerminfo.bools[capName]; ok {
		return "", true
	}

	return "", false
}

func init() {
	termName := os.Getenv("TERM")
	if termName == "" {
		fmt.Printf("not find TERM, please provide one.\n")
		os.Exit(1)
	}
	nTerminfo = &termcap{}
	err := nTerminfo.setupterm(termName)
	if err != nil {
		fmt.Printf("terminfo report:%s\n", err)
		panic(err)
	}

	/*
		https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Device-Control-functions

		A few special features are also recognized, which are not key names:

		o Co for termcap colors (or colors for terminfo colors), and
		o TN for termcap name (or name for terminfo name).
		o RGB for the ncurses direct-color extension.
		  Only a terminfo name is provided, since termcap
		  applications cannot use this information.

		 https://www.man7.org/linux/man-pages/man5/user_caps.5.html

		RGB
		boolean, number or string, used to assert that the
		set_a_foreground and set_a_background capabilities
		correspond to direct colors, using an RGB (red/green/blue)
		convention.  This capability allows the color_content
		function to return appropriate values without requiring the
		application to initialize colors using init_color.

		The capability type determines the values which ncurses sees:

		boolean
		implies that the number of bits for red, green and blue
		are the same.  Using the maximum number of colors,
		ncurses adds two, divides that sum by three, and assigns
		the result to red, green and blue in that order.

		If the number of bits needed for the number of colors is
		not a multiple of three, the blue (and green) components
		lose in comparison to red.

		number
		tells ncurses what result to add to red, green and blue.
		If ncurses runs out of bits, blue (and green) lose just
		as in the boolean case.

		string
		explicitly list the number of bits used for red, green
		and blue components as a slash-separated list of decimal
		integers.
	*/
	nTerminfo.nums["Co"] = nTerminfo.getnum("colors")
	nTerminfo.strs["TN"] = nTerminfo.name

	capName := "RGB"
	if _, ok := nTerminfo.nums[capName]; !ok {
		if _, ok := nTerminfo.bools[capName]; !ok {
			if _, ok := nTerminfo.strs[capName]; !ok {
				nTerminfo.strs[capName] = "8/8/8"
			}
		}
	}
}
