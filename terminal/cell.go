package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// contents saved in strings.Builder
type Cell struct {
	contents   strings.Builder
	renditions Renditions
	wide       bool
	fallback   bool
	wrap       bool
}

func (c *Cell) Reset(bgColor uint32) {
	c.contents.Reset()
	c.renditions = Renditions{bgColor: bgColor}
	c.wide = false
	c.fallback = false
	c.wrap = false
}

var _output io.Writer

func init() {
	_output = os.Stderr
}

func (c Cell) Empty() bool {
	return c.contents.Len() == 0
}

// 32 seems like a reasonable limit on combining characters
func (c Cell) Full() bool {
	return c.contents.Len() >= 32
}

func (c *Cell) Clear() {
	c.contents.Reset()
}

func (c Cell) IsBlank() bool {
	return c.Empty() || c.contents.String() == " " || c.contents.String() == "\xC2\xA0"
}

func (c Cell) ContentsMatch(x Cell) bool {
	return (c.IsBlank() && x.IsBlank()) || c.contents.String() == x.contents.String()
}

func (c Cell) debugContents() string {
	if c.Empty() {
		return "'_' []"
	}
	var chars strings.Builder

	chars.WriteString("'")
	c.PrintGrapheme(&chars)
	chars.WriteString("' [")

	// convert string to bytes
	b2 := []byte(c.contents.String())

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
		fmt.Fprintf(_output, "Contents: %s (%d) vs. %s (%d)\n", c.debugContents(), c.contents.Len(), other.debugContents(), other.contents.Len())
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

func (c *Cell) Append(r rune) {
	// ASCII?  Cheat.
	if r < 0x7f {
		c.contents.WriteByte(byte(r))
		// set wide automaticlly?
		// c.wide = runeCount(c.contents.String())
		return
	}
	c.contents.WriteRune(r)
	// c.wide = runeCount(c.contents.String())
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
*/

// print grapheme to output
func (c Cell) PrintGrapheme(output *strings.Builder) {
	if c.Empty() {
		output.WriteString(" ")
		return
	}
	/*
	 * cells that begin with combining character get combiner
	 * attached to no-break space
	 */
	if c.fallback {
		output.WriteString("\xC2\xA0")
	}
	output.WriteString(c.contents.String())
}

func (c Cell) GetRenditions() Renditions {
	return c.renditions
}

func (c *Cell) SetRenditions(r Renditions) {
	c.renditions = r
}

func (c Cell) GetWide() bool {
	return c.wide
}

func (c Cell) GetWidth() uint {
	if c.wide {
		return 2
	} else {
		return 1
	}
}

func (c *Cell) SetWide(w bool) {
	c.wide = w
}

func (c Cell) GetFallback() bool {
	return c.fallback
}

func (c *Cell) SetFallback(f bool) {
	c.fallback = f
}

func (c Cell) GetWrap() bool {
	return c.wrap
}

func (c *Cell) SetWrap(w bool) {
	c.wrap = w
}
