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
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE. */

package terminal

//
// import "fmt"
//
// const (
// 	ACTION_IGNORE       = "Ignore"
// 	ACTION_PRINT        = "Print"
// 	ACTION_EXECUTE      = "Execute"
// 	ACTION_CLEAR        = "Clear"
// 	ACTION_COLLECT      = "Collect"
// 	ACTION_PARAM        = "Param"
// 	ACTION_ESC_DISPATCH = "ESC_Dispatch"
// 	ACTION_CSI_DISPATCH = "CSI_Dispatch"
// 	ACTION_HOOK         = "Hook"
// 	ACTION_PUT          = "Put"
// 	ACTION_UNHOOK       = "Unhook"
// 	ACTION_OSC_START    = "OSC_Start"
// 	ACTION_OSC_PUT      = "OSC_Put"
// 	ACTION_OSC_END      = "OSC_End"
// 	ACTION_USER_BYTE    = "UserByte"
// 	ACTION_RESIZE       = "Resize"
// )
//
// // action implement Action interface
// type action struct {
// 	ch      rune
// 	present bool
// }
//
// func (act action) Ignore() bool       { return false } // default: do not ignore this action
// func (act action) Name() string       { return "" }    // default name: empty
// func (act action) ActOn(t Emulator)   {}               // default action: do nothing
// func (act *action) SetChar(r rune)    { act.ch = r }
// func (act *action) SetPresent(b bool) { act.present = b }
// func (act *action) GetChar() rune     { return act.ch }
// func (act *action) IsPresent() bool   { return act.present }
//
// type ignore struct {
// 	action
// }
//
// func (act ignore) Ignore() bool { return true } // ignore this action
// func (act ignore) Name() string { return ACTION_IGNORE }
//
// type print struct {
// 	action
// }
//
// func (act print) Name() string { return ACTION_PRINT }
// func (act print) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_PRINT, act.ch, act.present)
// }
//
// func (act print) ActOn(emu Emulator) {
// 	emu.Print(&act)
// }
//
// type execute struct {
// 	action
// }
//
// func (act execute) Name() string { return ACTION_EXECUTE }
// func (act execute) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_EXECUTE, act.ch, act.present)
// }
//
// func (act execute) ActOn(emu Emulator) {
// 	emu.Execute(&act)
// }
//
// type clear struct {
// 	action
// }
//
// func (act clear) Name() string { return ACTION_CLEAR }
// func (act clear) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_CLEAR, act.ch, act.present)
// }
//
// func (act clear) ActOn(emu Emulator) {
// 	emu.Dispatch().clear(&act)
// }
//
// type collect struct {
// 	action
// }
//
// func (act collect) Name() string { return ACTION_COLLECT }
// func (act collect) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_COLLECT, act.ch, act.present)
// }
//
// func (act collect) ActOn(emu Emulator) {
// 	emu.Dispatch().collect(&act)
// }
//
// type param struct {
// 	action
// }
//
// func (act param) Name() string { return ACTION_PARAM }
// func (act param) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_PARAM, act.ch, act.present)
// }
//
// func (act param) ActOn(emu Emulator) {
// 	emu.Dispatch().newParamChar(&act)
// }
//
// type escDispatch struct {
// 	action
// }
//
// func (act escDispatch) Name() string { return ACTION_ESC_DISPATCH }
// func (act escDispatch) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_ESC_DISPATCH, act.ch, act.present)
// }
//
// func (act escDispatch) ActOn(emu Emulator) {
// 	emu.ESCdispatch(&act)
// }
//
// type csiDispatch struct {
// 	action
// }
//
// func (act csiDispatch) Name() string { return ACTION_CSI_DISPATCH }
// func (act csiDispatch) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_CSI_DISPATCH, act.ch, act.present)
// }
//
// func (act csiDispatch) ActOn(emu Emulator) {
// 	emu.CSIdispatch(&act)
// }
//
// type hook struct {
// 	action
// }
//
// func (act hook) Name() string { return ACTION_HOOK }
// func (act hook) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_HOOK, act.ch, act.present)
// }
//
// type put struct {
// 	action
// }
//
// func (act put) Name() string { return ACTION_PUT }
// func (act put) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_PUT, act.ch, act.present)
// }
//
// type unhook struct {
// 	action
// }
//
// func (act unhook) Name() string { return ACTION_UNHOOK }
// func (act unhook) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_UNHOOK, act.ch, act.present)
// }
//
// type oscStart struct {
// 	action
// }
//
// func (act oscStart) Name() string { return ACTION_OSC_START }
// func (act oscStart) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_OSC_START, act.ch, act.present)
// }
//
// func (act oscStart) ActOn(emu Emulator) {
// 	emu.Dispatch().oscStart(&act)
// }
//
// type oscPut struct {
// 	action
// }
//
// func (act oscPut) Name() string { return ACTION_OSC_PUT }
// func (act oscPut) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_OSC_PUT, act.ch, act.present)
// }
//
// func (act oscPut) ActOn(emu Emulator) {
// 	emu.Dispatch().oscPut(&act)
// }
//
// type oscEnd struct {
// 	action
// }
//
// func (act oscEnd) Name() string { return ACTION_OSC_END }
// func (act oscEnd) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_OSC_END, act.ch, act.present)
// }
//
// func (act oscEnd) ActOn(emu Emulator) {
// 	emu.OSCend(&act)
// }
//

// func (act UserByte) Name() string { return ACTION_USER_BYTE }
// func (act UserByte) String() string {
// 	return fmt.Sprintf("%s(0x%02X,%t)", ACTION_USER_BYTE, act.c, act.present)
// }

