/*

MIT License

Copyright (c) 2022~2023 wangqi

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

This is a dual-license file, the original file is from tcell.
https://github.com/gdamore/tcell with some modification.
*/
package terminal

// Copyright 2018 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// package tcell

import (
	// ic "image/color"
	"testing"
)

func TestColorValues(t *testing.T) {
	values := []struct {
		color Color
		hex   int32
	}{
		{ColorRed, 0x00FF0000},
		{ColorGreen, 0x00008000},
		{ColorLime, 0x0000FF00},
		{ColorBlue, 0x000000FF},
		{ColorBlack, 0x00000000},
		{ColorWhite, 0x00FFFFFF},
		{ColorSilver, 0x00C0C0C0},
	}

	for _, tc := range values {
		if tc.color.Hex() != tc.hex {
			t.Errorf("Color: %x != %x", tc.color.Hex(), tc.hex)
		}
	}
}

/*
func TestColorFitting(t *testing.T) {
	pal := []Color{}
	for i := 0; i < 255; i++ {
		pal = append(pal, PaletteColor(i))
	}

	// Exact color fitting on ANSI colors
	for i := 0; i < 7; i++ {
		if FindColor(PaletteColor(i), pal[:8]) != PaletteColor(i) {
			t.Errorf("Color ANSI fit fail at %d", i)
		}
	}
	// Grey is closest to Silver
	if FindColor(PaletteColor(8), pal[:8]) != PaletteColor(7) {
		t.Errorf("Grey does not fit to silver")
	}
	// Color fitting of upper 8 colors.
	for i := 9; i < 16; i++ {
		if FindColor(PaletteColor(i), pal[:8]) != PaletteColor(i%8) {
			t.Errorf("Color fit fail at %d", i)
		}
	}
	// Imperfect fit
	if FindColor(ColorOrangeRed, pal[:16]) != ColorRed ||
		FindColor(ColorAliceBlue, pal[:16]) != ColorWhite ||
		FindColor(ColorPink, pal) != Color217 ||
		FindColor(ColorSienna, pal) != Color173 ||
		FindColor(GetColor("#00FD00"), pal) != ColorLime {
		t.Errorf("Imperfect color fit")
	}

}

*/
func TestColorNameLookup(t *testing.T) {
	values := []struct {
		name  string
		color Color
		rgb   bool
	}{
		{"#FF0000", ColorRed, true},
		{"black", ColorBlack, false},
		{"orange", ColorOrange, false},
		{"door", ColorDefault, false},
	}
	for _, v := range values {
		c := GetColor(v.name)
		if c.Hex() != v.color.Hex() {
			t.Errorf("Wrong color for %v: %v", v.name, c.Hex())
		}
		if v.rgb {
			if c&ColorIsRGB == 0 {
				t.Errorf("Color should have RGB")
			}
		} else {
			if c&ColorIsRGB != 0 {
				t.Errorf("Named color should not be RGB")
			}
		}

		if c.TrueColor().Hex() != v.color.Hex() {
			t.Errorf("TrueColor did not match")
		}
	}
}

func TestColorRGB(t *testing.T) {
	r, g, b := GetColor("#112233").RGB()
	if r != 0x11 || g != 0x22 || b != 0x33 {
		t.Errorf("RGB wrong (%x, %x, %x)", r, g, b)
	}
}

/*
func TestFromImageColor(t *testing.T) {
	red := ic.RGBA{0xFF, 0x00, 0x00, 0x00}
	white := ic.Gray{0xFF}
	cyan := ic.CMYK{0xFF, 0x00, 0x00, 0x00}

	if hex := FromImageColor(red).Hex(); hex != 0xFF0000 {
		t.Errorf("%v is not 0xFF0000", hex)
	}
	if hex := FromImageColor(white).Hex(); hex != 0xFFFFFF {
		t.Errorf("%v is not 0xFFFFFF", hex)
	}
	if hex := FromImageColor(cyan).Hex(); hex != 0x00FFFF {
		t.Errorf("%v is not 0x00FFFF", hex)
	}
}
*/
func TestColorString(t *testing.T) {
	tc := []struct {
		name  string
		color Color
		want  string
		isRGB bool
	}{
		{"RGB     color string", NewRGBColor(0x35, 0x33, 0x45), "rgb:3535/3333/4545", true},
		{"palette color string", PaletteColor(2), "rgb:0000/8080/0000", false},
		{"invalid color string", Color(2), "", false},
		{"outof range palette color string", Color(379 | ColorValid), "", false}, // any number >378 is undefined color index
		{"#345678 color string", GetColor("#345678"), "rgb:3434/5656/7878", true},
	}

	for _, v := range tc {
		got := v.color.String()
		if v.want != got {
			t.Errorf("%s: expect %s, got %s\n", v.name, v.want, got)
		}
		if v.color.IsRGB() != v.isRGB {
			t.Errorf("%s: expect %t, got %t\n", v.name, v.isRGB, v.color.IsRGB())
		}
	}

	// if Color(379|ColorValid).Hex() != -1 {
	// 	t.Errorf("Color(x).Hex() expect return -1, got %d\n ", Color(379|ColorValid).Hex())
	// }
}

func TestColorIndex(t *testing.T) {
	tc := []struct {
		name  string
		color Color
		index int
	}{
		{"ANSI 256 color index", Color100, 100},
		{"ANSI 8 color index", ColorBlack, 0},
		{"ANSI 16 color index", ColorRed, 9},
		{"default color", ColorDefault, -1},    // ColorDefault has no index
		{"RGB color", GetColor("#818181"), -1}, // RGB color has no index
	}

	for _, v := range tc {
		got := v.color.Index()
		if v.index != got {
			t.Errorf("%s: expect %d, got %d\n", v.name, v.index, v.color.Index())
		}
	}
}

func TestColorName(t *testing.T) {
	tc := []struct {
		name  string
		color Color
		want  string
	}{
		{"Balck        ", ColorBlack, "black"},
		{"Slate grey   ", ColorSlateGray, "slategrey"},
		{"Slate grey   ", ColorSlateGray, "slategray"},
		{"Indigo       ", ColorIndigo, "indigo"},
		{"Absense color", Color108, ""},
	}

	for _, v := range tc {
		names := v.color.Name()
		// t.Logf("TC:%q %s\n", v.name, names)
		found := false
		for _, name := range names {
			if name == v.want {
				found = true
			}
		}
		if v.want == "" && names == nil {
			continue
		}
		if !found {
			t.Errorf("%s: expect color name=%s, got nothing.\n", v.name, v.want)
		}

	}
}
