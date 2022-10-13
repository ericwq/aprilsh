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

package terminal

import (
	"strings"
)

/* contents saved in strings.Builder
 * Don't set the fields directly
 *
 * Cell is comparable
 */
type Cell struct {
	contents   string
	renditions Renditions
	// fallback   bool
	dirty      bool
	wrap       bool
	dwidth     bool
	dwidthCont bool
}

func (c Cell) IsBlank() bool {
	if c.dwidth || c.dwidthCont {
		return false
	}
	return c.Empty() || c.contents == " " || c.contents == "\xC2\xA0"
}

func (c Cell) GetRenditions() Renditions {
	return c.renditions
}

func (c Cell) String() string {
	return c.contents
}

// 32 seems like a reasonable limit on combining characters
// here we only counting the bytes number
func (c *Cell) full() bool {
	return len(c.contents) >= 32
}

// reset cell with default renditions and contents
// the default contents is empty string
/*
func (c *Cell) Reset() {
	c.contents = ""
	c.renditions = Renditions{}
	c.fallback = false
	c.wrap = false
}
*/

// reset cell with specified renditions
// TODO : the default contents is " "?
func (c *Cell) Reset2(attrs Cell) {
	c.contents = attrs.contents
	c.renditions = attrs.renditions
	c.dwidth = false
	c.dwidthCont = false
}

/*
// ease of testing
var _output io.Writer

func init() {
	_output = os.Stderr
}
*/

func (c *Cell) Empty() bool {
	return len(c.contents) == 0
}

// clear contents
func (c *Cell) Clear() {
	c.contents = ""
}

func (c *Cell) ContentsMatch(x Cell) bool {
	return (c.IsBlank() && x.IsBlank()) || c.contents == x.contents
}

/*
func (c *Cell) debugContents() string {
	if c.Empty() {
		return "'_' []"
	}
	var chars strings.Builder

	chars.WriteString("'")
	c.PrintGrapheme(&chars)
	chars.WriteString("' [")

	// convert string to bytes
	b2 := []byte(c.contents)

	// print each byte in hex
	comma := ""
	for i := 0; i < len(b2); i++ {
		fmt.Fprintf(&chars, "%s0x%02x, ", comma, b2[i])
		comma = ", "
	}
	chars.WriteString("]")

	return chars.String()
}

func (c Cell) Compare(other Cell) bool {
	var ret bool
	var grapheme strings.Builder
	var other_grapheme strings.Builder

	c.PrintGrapheme(&grapheme)
	other.PrintGrapheme(&other_grapheme)

	if grapheme.String() != other_grapheme.String() {
		ret = true
		fmt.Fprintf(_output, "Graphemes: '%s' vs. '%s'\n", grapheme.String(), other_grapheme.String())
	}

	if !c.ContentsMatch(other) {
		fmt.Fprintf(_output, "Contents: %s (%d) vs. %s (%d)\n", c.debugContents(), len(c.contents), other.debugContents(), len(other.contents))
	}

	if c.fallback != other.fallback {
		fmt.Fprintf(_output, "fallback: %t vs. %t\n", c.fallback, other.fallback)
	}

	if c.wide != other.wide {
		ret = true
		fmt.Fprintf(_output, "width: %t vs. %t\n", c.wide, other.wide)
	}

	if c.renditions != other.renditions {
		ret = true
		fmt.Fprintf(_output, "renditions differ\n")
	}

	if c.wrap != other.wrap {
		ret = true
		fmt.Fprintf(_output, "wrap: %t vs. %t\n", c.wrap, other.wrap)
	}
	return ret
}
*/

// append to the contents
func (c *Cell) Append(r rune) {
	var builder strings.Builder
	if !c.Empty() {
		builder.WriteString(c.contents)
	}

	// ASCII?  Cheat.
	if r < 0x7f {
		builder.WriteByte(byte(r))
	} else {
		builder.WriteRune(r)
	}
	c.contents = builder.String()
	// set wide automaticlly?
}

// replace the contents
func (c *Cell) SetContents(chs []rune) {
	c.contents = string(chs)
}

func (c *Cell) GetContents() string {
	return c.contents
}

func (c *Cell) SetRenditions(r Renditions) {
	c.renditions = r
}

func (c *Cell) SetDoubleWidth(value bool) {
	c.dwidth = value
}

func (c *Cell) SetDoubleWidthCont(value bool) {
	c.dwidthCont = value
}

func (c *Cell) IsDoubleWidth() bool {
	return c.dwidth
}

func (c *Cell) IsDoubleWidthCont() bool {
	return c.dwidthCont
}

/*
hide it

	func runeCount(s string) bool {
		if utf8.RuneCountInString(s)>1 {
			return true
		} else {
			return false
		}
	}

// print grapheme to output

	func (c *Cell) PrintGrapheme(output *strings.Builder) {
		if c.Empty() {
			output.WriteString(" ")
			return
		}
		 // * cells that begin with combining character get combiner
		 // * attached to no-break space
		if c.fallback {
			output.WriteString("\xC2\xA0")
		}
		output.WriteString(c.contents)
	}
*/
func (c *Cell) SetUnderline(underline bool) {
	c.renditions.underline = underline
}

// func (c Cell) GetWide() bool {
// 	return c.wide
// }

// return cell grapheme width: 0,1,2
func (c *Cell) GetWidth() int {
	if c.dwidthCont { // it's a place holder for wide grapheme
		return 0
	}

	if c.dwidth { // it's a wide grapheme
		return 2
	} else {
		return 1
	}
}

// func (c *Cell) SetWide(w bool) {
// 	c.wide = w
// }

/*
func (c *Cell) GetFallback() bool {
	return c.fallback
}

func (c *Cell) SetFallback(f bool) {
	c.fallback = f
}

func (c *Cell) GetWrap() bool {
	return c.wrap
}

func (c *Cell) SetWrap(w bool) {
	c.wrap = w
}
*/

/*
// Is this a printing ISO 8859-1 character?
func IsPrintISO8859_1(r rune) bool {
	return (r <= 0xff && r >= 0xa0) || (r <= 0x7e && r >= 0x20)
	// return unicode.IsGraphic(r)
	// return unicode.In(r, unicode.Latin, unicode.Number, unicode.Punct)
}
func AppendToStr(dest *strings.Builder, r rune) {
	// ASCII?  Cheat.
	if r < 0x7f {
		dest.WriteByte(byte(r))
		return
	}
	dest.WriteRune(r)
}
*/
