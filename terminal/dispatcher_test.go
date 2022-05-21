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
//
// import (
// 	"testing"
// )
//
// func TestDispatcherGetParam(t *testing.T) {
// 	tc := []struct {
// 		name   string
// 		params string
// 		want   []int
// 	}{
// 		// the default value is 0
// 		{"normal", "4;57", []int{4, 57}},
// 		{"malform", ";12;45;", []int{0, 12, 45, 0}},
// 		{"too large", "65536;4500;", []int{0, 4500, 0}},
// 		{"semicolon", "5:67", []int{5, 67}},
// 	}
//
// 	for _, v := range tc {
// 		// prepare the test case
// 		d := Dispatcher{}
// 		d.params.WriteString(v.params)
// 		// t.Logf("%v\n", d.params)
//
// 		if len(v.want) != d.getParamCount() {
// 			t.Errorf("%s expect %d result, got %d result.\n", v.name, len(v.want), d.getParamCount())
// 		} else {
// 			for i := range d.parsedParams {
// 				got := d.getParam(i, 0)
// 				if v.want[i] == got {
// 					continue
// 				} else {
// 					t.Errorf("%s:\t case:%s\t [%02d parameter]: expect %d, got %d\n", v.name, v.params, i, v.want[i], got)
// 				}
// 			}
// 		}
// 	}
// }
//
// func TestDispatcherNewParamChar(t *testing.T) {
// 	tc := []struct {
// 		name   string
// 		params string
// 		want   []int
// 	}{
// 		// the default value is 13
// 		{"normal", "9;21", []int{9, 21}},
// 		{"malform", ";19;121;", []int{13, 19, 121, 13}},
// 		{"too large", "65536;210", []int{13, 210}},
// 	}
//
// 	d := Dispatcher{}
// 	for _, v := range tc {
// 		d.clear(&clear{})
//
// 		// fill data with newParamChar
// 		for _, ch := range v.params {
// 			d.newParamChar(&param{action{ch, true}})
// 		}
//
// 		// check the expect data
// 		for i, w := range v.want {
// 			got := d.getParam(i, 13)
// 			if got != w {
// 				t.Errorf("%s:\t %q [%02d parameter]: expect %d, got %d\n", v.name, v.params, i, w, got)
// 			}
// 		}
// 		if len(v.want) != d.getParamCount() {
// 			t.Errorf("%s expect %d result, got %d result.\n", v.name, len(v.want), d.getParamCount())
// 		}
// 	}
// }
//
// func TestDispatcherCollect(t *testing.T) {
// 	tc := []struct {
// 		name   string
// 		params string
// 		want   string
// 	}{
// 		// 0x20-0x2F
// 		{"normal", "#$%&'()", "#$%&'()"},
// 		{"over size", " !\"#$%&'()*+,-./", " !\"#$%&'"},
// 	}
//
// 	d := Dispatcher{}
// 	for _, v := range tc {
// 		d.clear(&clear{})
// 		for _, ch := range v.params {
// 			d.collect(&collect{action{ch, true}})
// 		}
// 		if v.want != d.getDispatcherChars() {
// 			t.Errorf("%s:\t expect %q, got %q\n", v.name, v.want, d.getDispatcherChars())
// 		}
// 	}
// }
//
// func TestDispatcherOSCput(t *testing.T) {
// 	// over size title
// 	a := "Stop all the clocks, cut off the telephone, Prevent the dog from barking with a juicy bone, Silence the pianos and with muffled drum Bring out the coffin, let the mourners come. Let aeroplanes circle moaning overhead Scribbling on the sky the message He Is Dead, Put crepe bows round the white necks of the public doves, Let the traffic policemen wear black cotton gloves. "
//
// 	// chinese title
// 	b := "北国风光，千里冰封，万里雪飘。望长城内外，惟余莽莽；大河上下，顿失滔滔。山舞银蛇，原驰蜡象，欲与天公试比高。须晴日，看红装素裹，分外妖娆。江山如此多娇，引无数英雄竞折腰。惜秦皇汉武，略输文采；唐宗宋祖，稍逊风骚。一代天骄，成吉思汗，只识弯弓射大雕。俱往矣，数风流人物，还看今朝。"
//
// 	// got english title
// 	a1 := "Stop all the clocks, cut off the telephone, Prevent the dog from barking with a juicy bone, Silence the pianos and with muffled drum Bring out the coffin, let the mourners come. Let aeroplanes circle moaning overhead Scribbling on the sky the message He Is"
//
// 	// got chinese title
// 	b1 := "北国风光，千里冰封，万里雪飘。望长城内外，惟余莽莽；大河上下，顿失滔滔。山舞银蛇，原驰蜡象，欲与天公试比高。须晴日，看红装素裹，分外妖娆。江山如此多娇，引无数英雄竞折腰。惜"
//
// 	tc := []struct {
// 		name string
// 		osc  string
// 		want string
// 	}{
// 		{"normal", a, a1},
// 		{"chinese", b, b1},
// 	}
//
// 	d := Dispatcher{}
// 	for _, v := range tc {
// 		// clear the content
// 		d.oscStart(&oscStart{})
//
// 		// fill in the osc string
// 		for _, ch := range v.osc {
// 			d.oscPut(&oscPut{action{ch, true}})
// 		}
//
// 		if v.want != d.getOSCstring() {
// 			t.Errorf("%s:\t osc string size expect %d, got %d\n", v.name, len(v.want), len(d.getOSCstring()))
// 		}
// 	}
// }
//
// func TestDispatcherOSCdispatch(t *testing.T) {
// 	tc := []struct {
// 		name    string
// 		osc     string
// 		command int
// 		want    string
// 	}{
// 		{"normanl", ";title", 0, "title"},
// 		{"icon", "1;icon name", 1, "icon name"},
// 		{"window", "2;window title name", 2, "window title name"},
// 		{"unsupport", "52;window title name", 3, "window title name"},
// 	}
//
// 	for _, v := range tc {
// 		d := Dispatcher{}
// 		fb := NewFramebuffer(4, 4)
//
// 		// fill in the osc string
// 		for _, ch := range v.osc {
// 			d.oscPut(&oscPut{action{ch, true}})
// 		}
//
// 		d.oscDispatch(&oscEnd{}, fb)
//
// 		switch v.command {
// 		case 0:
// 			if fb.IsTitleInitialized() && fb.GetIconName() == v.want && fb.GetWindowTitle() == v.want {
// 				continue
// 			} else {
// 				t.Errorf("%s:\t osc=%q expect %q, got %q and %q\n", v.name, v.osc, v.want, fb.GetIconName(), fb.GetWindowTitle())
// 			}
// 		case 1:
// 			if fb.IsTitleInitialized() && fb.GetIconName() == v.want {
// 				continue
// 			} else {
// 				t.Errorf("%s:\t osc=%q expect %q, got %q \n", v.name, v.osc, v.want, fb.GetIconName())
// 			}
// 		case 2:
// 			if fb.IsTitleInitialized() && fb.GetWindowTitle() == v.want {
// 				continue
// 			} else {
// 				t.Errorf("%s:\t osc=%q expect %q, got %q \n", v.name, v.osc, v.want, fb.GetWindowTitle())
// 			}
// 		default:
// 			if !fb.IsTitleInitialized() {
// 				continue
// 			} else {
// 				t.Errorf("%s:\t osc=%q expect %q, got %q and %q", v.name, v.osc, v.want, fb.GetIconName(), fb.GetWindowTitle())
// 			}
// 		}
// 	}
// }
