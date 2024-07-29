// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package terminal

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ericwq/aprilsh/util"
	"github.com/rivo/uniseg"
)

/*
 * 64 - VT420 family
 *  1 - 132 columns
 *  9 - National Replacement Character-sets
 * 15 - DEC technical set
 * 21 - horizontal scrolling
 * 22 - color
 */
const (
	DEVICE_ID = "64;1;9;15;21;22c"
)

const (
	unused_handlerID = iota
	C0_BEL
	C0_CR
	C0_HT
	C0_SI
	C0_SO
	CSI_CBT
	CSI_CHA
	CSI_CHT
	CSI_CNL
	CSI_CPL
	CSI_CUB
	CSI_CUD
	CSI_CUF
	CSI_CUP
	CSI_CUU
	CSI_DCH
	CSI_DECIC
	CSI_DECDC
	CSI_privRM
	CSI_DECSCL
	CSI_DECSCUSR
	CSI_privSM
	CSI_DECRQM
	CSI_DECSTBM
	CSI_DECSTR
	CSI_ECMA48_SL
	CSI_ECMA48_SR
	CSI_FocusIn
	CSI_FocusOut
	CSI_DL
	CSI_DSR
	CSI_ECH
	CSI_ED
	CSI_EL
	CSI_HPA
	CSI_HPR
	CSI_ICH
	CSI_IL
	CSI_priDA
	CSI_REP
	CSI_RM
	CSI_secDA
	CSI_SD
	CSI_SM
	CSI_SU
	CSI_SCORC
	CSI_SLRM_SCOSC
	CSI_SGR
	CSI_TBC
	CSI_VPA
	CSI_VPR
	CSI_XTMODKEYS
	CSI_XTWINOPS
	CSI_U
	DCS_DECRQSS
	DCS_XTGETTCAP
	ESC_BI
	ESC_DCS
	ESC_DECALN
	ESC_DECANM
	ESC_DECKPAM
	ESC_DECKPNM
	ESC_DECRC
	ESC_DECSC
	ESC_DOCS_UTF8
	ESC_DOCS_ISO8859_1
	ESC_FI
	ESC_HTS
	ESC_IND
	ESC_LS1R
	ESC_LS2
	ESC_LS2R
	ESC_LS3
	ESC_LS3R
	ESC_NEL
	ESC_RI
	ESC_RIS
	ESC_SS2
	ESC_SS3
	Graphemes
	OSC_4
	OSC_52
	OSC_0_1_2
	OSC_10_11_12_17_19
	OSC_112
	OSC_8
	VT52_EGM
	VT52_ID
)

var strHandlerID = [...]string{
	"",
	"c0_bel",
	"c0_cr",
	"c0_ht",
	"c0_si",
	"c0_so",
	"csi_cbt",
	"csi_cha",
	"csi_cht",
	"csi_cnl",
	"csi_cpl",
	"csi_cub",
	"csi_cud",
	"csi_cuf",
	"csi_cup",
	"csi_cuu",
	"csi_dch",
	"csi_decic",
	"csi_decdc",
	"csi_decrst",
	"csi_decscl",
	"csi_decscusr",
	"csi_decset",
	"csi_decrqm",
	"csi_decstbm",
	"csi_decstr",
	"csi_ecma48_SL",
	"csi_ecma48_SR",
	"csi_focus_in",
	"csi_focus_out",
	"csi_dl",
	"csi_dsr",
	"csi_ech",
	"csi_ed",
	"csi_el",
	"csi_hpa",
	"csi_hpr",
	"csi_ich",
	"csi_il",
	"csi_priDA",
	"csi_rep",
	"csi_rm",
	"csi_secDA",
	"csi_sd",
	"csi_sm",
	"csi_su",
	"csi_scorc",
	"csi_slrm_scosc",
	"csi_sgr",
	"csi_tbc",
	"csi_vpa",
	"csi_vpr",
	"csi_xtmodkeys",
	"csi_xtwinops",
	"csi_u",
	"dcs_decrqss",
	"dcs_xtgettcap",
	"esc_bi",
	"esc_dcs",
	"esc_decaln",
	"esc_decanm",
	"esc_deckpam",
	"esc_deckpnm",
	"esc_decrc",
	"esc_decsc",
	"esc_docs_utf_8",
	"esc_docs_iso8859_1",
	"esc_fi",
	"esc_hts",
	"esc_ind",
	"esc_ls1r",
	"esc_ls2",
	"esc_ls2r",
	"esc_ls3",
	"esc_ls3r",
	"esc_nel",
	"esc_ri",
	"esc_ris",
	"esc_ss2",
	"esc_ss3",
	"graphemes",
	"osc_4",
	"osc_52",
	"osc_0_1_2",
	"osc_10_11_12_17_19",
	"osc_112",
	"osc_8",
	"vt52_egm",
	"vt52_id",
}

// Handler is the outcome of parsering input, it can be used to perform control sequence on emulator.
type Handler struct {
	handle   func(emu *Emulator) // handle function that will perform control sequnce on emulator
	sequence string              // control sequence
	id       int                 // handler ID
	ch       rune                // the last byte
}

func (h *Handler) GetId() int {
	return h.id
}

func (h *Handler) GetCh() rune {
	return h.ch
}

func restoreSequence(hds []*Handler) string {
	var b strings.Builder

	for i := range hds {
		b.WriteString(hds[i].sequence)
	}

	return b.String()
}

/*
func (h *Handler) GetSequence() string {
	return h.sequence
}

func (h *Handler) Handle(emu *Emulator) {
	h.handle(emu)
}
*/

// In the loop, national flag's width got 1+1=2.
/*
func RunesWidth(runes []rune) (width int) {
	// return uniseg.StringWidth(string(runes))

	// quick pass for iso8859-1
	if len(runes) == 1 && runes[0] < 0x00fe {
		return 1
	}

	cond := runewidth.NewCondition()
	cond.StrictEmojiNeutral = false
	cond.EastAsianWidth = true

	// loop for multi rune
	width = 0
	for i := 0; i < len(runes); i++ {
		width += cond.RuneWidth(runes[i])
	}

	return width
}
*/

// print the graphic char to the emulator
// https://henvic.dev/posts/go-utf8/
// https://pkg.go.dev/golang.org/x/text/encoding/charmap
// https://github.com/rivo/uniseg
func hdl_graphemes(emu *Emulator, chs ...rune) {
	w := uniseg.StringWidth(string(chs))
	if len(chs) == 1 && emu.charsetState.vtMode {
		chs[0] = emu.lookupCharset(chs[0])
	}
	// fmt.Printf("#hdl_graphemes %q, %U, %t w=%d (%d/%d)\n", chs, chs, emu.lastCol, w, emu.posY, emu.posX)

	// the first condition deal with the new graphemes should wrap on next row
	// the second condition deal with widh graphemes in special position: posX = nColsEff-1 and width is 2
	if (emu.autoWrapMode && emu.lastCol) || (w == 2 && emu.posX == emu.nColsEff-1) {
		lastCell := emu.cf.getCellPtr(emu.posY, emu.posX)
		if w == 2 && emu.posX == emu.nColsEff-1 {
			lastCell.SetEarlyWrap(true) // mark the last chinese cell early wrap case
		}
		lastCell.wrap = true
		hdl_c0_cr(emu)
		hdl_c0_lf(emu)
	}

	// validate lastCol
	// fmt.Printf("#hdl_graphemes lastCol   at (%d,%d) lastCol=%t\n", emu.posY, emu.posX, emu.lastCol)
	// if emu.lastCol && emu.posX != emu.nColsEff-1 {
	// 	emu.lastCol = false
	// }

	// insert a blank cell for insert mode
	if emu.insertMode {
		hdl_csi_ich(emu, 1)
	}

	// print grapheme in current cursor position with default renditions.
	c := emu.cf.getCellPtr(emu.posY, emu.posX)
	*c = emu.attrs
	c.SetContents(chs)
	// util.Logger.Trace("hdl_graphemes", "col", emu.posX, "row", emu.posY, "ch", c)

	/// for double width graphemes
	if w == 2 && emu.posX < emu.nColsEff-1 {
		// set double width flag
		c.SetDoubleWidth(true)
		// the cell after double width cell
		// set double width continue flag
		emu.posX++
		emu.cf.getCellPtr(emu.posY, emu.posX).SetDoubleWidthCont(true)
	}

	// prepare for the next graphemes, move the cursor or set last column flag
	// posX is used by current grapheme.
	if emu.posX == emu.nColsEff-1 {
		emu.lastCol = true
	} else {
		emu.posX++
	}
	// fmt.Printf("#hdl_graphemes next      at (%d,%d) lastCol=%t\n", emu.posY, emu.posX, emu.lastCol)
}

// func hdl_userbyte(emu *emulator, u UserByte) {
// 	ret := emu.user.parse(u, emu.framebuffer.DS.ApplicationModeCursorKeys)
// 	emu.dispatcher.terminalToHost.WriteString(ret)
// }
//
// func hdl_resize(emu *emulator, width, height int) {
// 	emu.resize(width, height)
// }

// Horizontal Tab (HTS  is Ctrl-I).
// move cursor to the next tab position
func hdl_c0_ht(emu *Emulator) {
	if emu.posX < emu.nColsEff-1 {
		emu.jumpToNextTabStop()
	}
}

// Bell (BEL  is Ctrl-G).
// ring the bell
func hdl_c0_bel(emu *Emulator) {
	emu.ringBell()
}

// Carriage Return (CR  is Ctrl-M).
// move cursor to the head of the same row
func hdl_c0_cr(emu *Emulator) {
	if emu.originMode == OriginMode_Absolute && emu.posX < emu.hMargin {
		emu.posX = 0
	} else {
		emu.posX = emu.hMargin
	}
	emu.lastCol = false
}

// Line Feed
// move cursor to the next row, scroll down if necessary.
// if the screen scrolled, erase the line from current position.
func hdl_c0_lf(emu *Emulator) {
	if hdl_esc_ind(emu) {
		emu.cf.eraseInRow(emu.posY, emu.posX, emu.nColsEff-emu.posX, emu.attrs)
	}
}

// SI       Switch to Standard Character Set (Ctrl-O is Shift In or LS0).
//
//	This invokes the G0 character set (the default) as GL.
//	VT200 and up implement LS0.
func hdl_c0_si(emu *Emulator) {
	emu.charsetState.gl = 0
}

// SO       Switch to Alternate Character Set (Ctrl-N is Shift Out or
//
//	LS1).  This invokes the G1 character set as GL.
//	VT200 and up implement LS1.
func hdl_c0_so(emu *Emulator) {
	emu.charsetState.gl = 1
}

// VT52: switch gl to charset DEC special
func hdl_vt52_egm(emu *Emulator) {
	resetCharsetState(&emu.charsetState)
	emu.charsetState.g[emu.charsetState.gl] = &vt_DEC_Special
}

// VT52: return device id for vt52 emulator
// For a terminal emulating VT52, the identifying sequence should be ESC / Z.
func hdl_vt52_id(emu *Emulator) {
	emu.writePty("\x1B/Z")
}

// FF, VT same as LF a.k.a IND Index
// move cursor to the next row, scroll up if necessary.
func hdl_esc_ind(emu *Emulator) (scrolled bool) {
	if emu.posY == emu.marginBottom-1 {
		// text up, viewpoint down if it reaches the last row in active area
		hdl_csi_su(emu, 1)
		scrolled = true
	} else if emu.posY < emu.nRows-1 {
		emu.posY++
		emu.lastCol = false
	}
	return
}

// ESC N
//
//	Single Shift Select of G2 Character Set (SS2  is 0x8e), VT220.
//	This affects next character only.
func hdl_esc_ss2(emu *Emulator) {
	emu.charsetState.ss = 2
}

// ESC O
//
//	Single Shift Select of G3 Character Set (SS3  is 0x8f), VT220.
//	This affects next character only.
func hdl_esc_ss3(emu *Emulator) {
	emu.charsetState.ss = 3
}

// ESC ~     Invoke the G1 Character Set as GR (LS1R), VT100.
func hdl_esc_ls1r(emu *Emulator) {
	emu.charsetState.gr = 1
}

// ESC n     Invoke the G2 Character Set as GL (LS2).
func hdl_esc_ls2(emu *Emulator) {
	emu.charsetState.gl = 2
}

// ESC }     Invoke the G2 Character Set as GR (LS2R).
func hdl_esc_ls2r(emu *Emulator) {
	emu.charsetState.gr = 2
}

// ESC o     Invoke the G3 Character Set as GL (LS3).
func hdl_esc_ls3(emu *Emulator) {
	emu.charsetState.gl = 3
}

// ESC |     Invoke the G3 Character Set as GR (LS3R).
func hdl_esc_ls3r(emu *Emulator) {
	emu.charsetState.gr = 3
}

// ESC % G   Select UTF-8 character set, ISO 2022.
// https://en.wikipedia.org/wiki/ISO/IEC_2022#Interaction_with_other_coding_systems
func hdl_esc_docs_utf8(emu *Emulator) {
	resetCharsetState(&emu.charsetState)
}

// ESC % @   Select default character set.  That is ISO 8859-1 (ISO 2022).
// https://www.cl.cam.ac.uk/~mgk25/unicode.html#utf-8
func hdl_esc_docs_iso8859_1(emu *Emulator) {
	resetCharsetState(&emu.charsetState)
	emu.charsetState.g[emu.charsetState.gr] = &vt_ISO_8859_1 // Charset_IsoLatin1
	emu.charsetState.vtMode = true
}

// Select G0 ~ G3 character set based on parameter
func hdl_esc_dcs(emu *Emulator, index int, charset *map[byte]rune) {
	emu.charsetState.g[index] = charset
	if charset != nil {
		emu.charsetState.vtMode = true
	}
}

// ESC H Tab Set (HTS is 0x88).
// Sets a tab stop in the current column the cursor is in.
func hdl_esc_hts(emu *Emulator) {
	emu.tabStops = append(emu.tabStops, emu.posX)
	sort.Ints(emu.tabStops)
}

// ESC M  Reverse Index (RI  is 0x8d).
// reverse index -- like a backwards line feed
func hdl_esc_ri(emu *Emulator) {
	if emu.posY == emu.marginTop {
		// scroll down 1 row
		hdl_csi_sd(emu, 1)
	} else if emu.posY > 0 {
		emu.posY--
		emu.lastCol = false
	}
}

// ESC E  Next Line (NEL  is 0x85).
func hdl_esc_nel(emu *Emulator) {
	hdl_esc_ind(emu)
	hdl_c0_cr(emu)
}

// ESC c     Full Reset (RIS), VT100.
// reset the screen
func hdl_esc_ris(emu *Emulator) {
	emu.resetTerminal()
}

// ESC 7     Save Cursor (DECSC), VT100.
func hdl_esc_decsc(emu *Emulator) {
	emu.savedCursor_DEC.posX = emu.posX
	emu.savedCursor_DEC.posY = emu.posY
	emu.savedCursor_DEC.lastCol = emu.lastCol
	emu.savedCursor_DEC.attrs = emu.attrs
	emu.savedCursor_DEC.originMode = emu.originMode
	emu.savedCursor_DEC.charsetState = emu.charsetState
	emu.savedCursor_DEC.isSet = true
	// fmt.Printf("esc_decsc 501: save DEC: (%d,%d) isSet=%t\n", emu.posY, emu.posX, emu.savedCursor_DEC.isSet)
}

// ESC 8     Restore Cursor (DECRC), VT100.
func hdl_esc_decrc(emu *Emulator) {
	if !emu.savedCursor_DEC.isSet {
		// emu.logI.Println("Asked to restore cursor (DECRC) but it has not been saved.")
		util.Logger.Warn("Asked to restore cursor (DECRC) but it has not been saved")
	} else {
		emu.posX = emu.savedCursor_DEC.posX
		emu.posY = emu.savedCursor_DEC.posY
		emu.normalizeCursorPos()
		emu.lastCol = emu.savedCursor_DEC.lastCol
		emu.attrs = emu.savedCursor_DEC.attrs
		emu.originMode = emu.savedCursor_DEC.originMode
		emu.charsetState = emu.savedCursor_DEC.charsetState
		emu.savedCursor_DEC.isSet = false
		// fmt.Printf("esc_decsc 518: restore DEC: (%d,%d) isSet=%t\n", emu.posY, emu.posX, emu.savedCursor_DEC.isSet)
	}
}

// ESC # 8   DEC Screen Alignment Test (DECALN), VT100.
// fill the screen with 'E'
func hdl_esc_decaln(emu *Emulator) {
	// Save current attrs
	origAttrs := emu.attrs

	origFg := emu.attrs.renditions.fgColor
	origBg := emu.attrs.renditions.bgColor

	emu.resetAttrs()
	emu.fillScreen('E')

	emu.fg = origFg
	emu.bg = origBg
	emu.attrs = origAttrs
}

// DECFI—Forward Index
// This control function moves the cursor forward one column. If the cursor is
// at the right margin, then all screen data within the margins moves one column
// to the left. The column shifted past the left margin is lost.
func hdl_esc_fi(emu *Emulator) {
	arg := 1
	if emu.posX < emu.nColsEff-1 {
		hdl_csi_cuf(emu, arg)
	} else {
		hdl_csi_ecma48_SL(emu, arg)
	}
}

// DECBI—Back Index
// This control function moves the cursor backward one column. If the cursor is
// at the left margin, then all screen data within the margin moves one column
// to the right. The column that shifted past the right margin is lost.
func hdl_esc_bi(emu *Emulator) {
	arg := 1
	if emu.posX > emu.hMargin {
		hdl_csi_cub(emu, arg)
	} else {
		hdl_csi_ecma48_SR(emu, arg)
	}
}

// DECKPAM—Keypad Application Mode
// DECKPAM enables the numeric keypad to send application sequences to the host.
// DECKPNM enables the numeric keypad to send numeric characters.
// ESC = Send application sequences.
func hdl_esc_deckpam(emu *Emulator) {
	emu.keypadMode = KeypadMode_Application
}

// DECKPNM—Keypad Numeric Mode
// DECKPNM enables the keypad to send numeric characters to the host.
// DECKPAM enables the keypad to send application sequences.
// ESC > Send numeric keypad characters.
func hdl_esc_deckpnm(emu *Emulator) {
	emu.keypadMode = KeypadMode_Normal
}

// DECANM—ANSI Mode
func hdl_esc_decanm(emu *Emulator, cl CompatibilityLevel) {
	emu.setCompatLevel(cl)
}

// CSI Ps g  Tab Clear (TBC).
//
//	Ps = 0  ⇒  Clear Current Column (default).
//	Ps = 3  ⇒  Clear All.
func hdl_csi_tbc(emu *Emulator, cmd int) {
	switch cmd {
	case 0: // clear this tab stop
		idx := sort.SearchInts(emu.tabStops, emu.posX)
		if idx < len(emu.tabStops) && emu.tabStops[idx] == emu.posX {
			// posX is present at tabStops[idx]
			emu.tabStops = RemoveIndex(emu.tabStops, idx)
		}
	case 3: // clear all tab stops
		emu.tabStops = make([]int, 0)
	default:
	}
}

// CSI Ps I  Cursor Forward Tabulation Ps tab stops (default = 1) (CHT).
// Advance the cursor to the next column (in the same row) with a tab stop.
// If there are no more tab stops, move to the last column in the row.
func hdl_csi_cht(emu *Emulator, count int) {
	if count == 1 {
		hdl_c0_ht(emu)
	} else {
		for k := 0; k < count; k++ {
			emu.jumpToNextTabStop()
		}
	}
}

// CSI Ps Z  Cursor Backward Tabulation Ps tab stops (default = 1) (CBT).
// Move the cursor to the previous column (in the same row) with a tab stop.
// If there are no more tab stops, moves the cursor to the first column.
// If the cursor is in the first column, doesn’t move the cursor.
func hdl_csi_cbt(emu *Emulator, count int) {
	for k := 0; k < count; k++ {
		if len(emu.tabStops) == 0 {
			if emu.posX > 0 && emu.posX%8 == 0 {
				emu.posX -= 8
			} else {
				emu.posX = (emu.posX / 8) * 8
			}
		} else {
			// Set posX to previous tab stop
			nextTabIdx := LowerBound(emu.tabStops, emu.posX)
			if nextTabIdx-1 >= 0 {
				emu.posX = emu.tabStops[nextTabIdx-1]
			} else {
				emu.posX = 0
			}

			emu.lastCol = false
		}
	}
}

// CSI Ps @  Insert Ps (Blank) Character(s) (default = 1) (ICH).
func hdl_csi_ich(emu *Emulator, arg int) {
	if emu.isCursorInsideMargins() {
		length := emu.nColsEff - emu.posX
		arg = min(arg, length)
		length -= arg

		if emu.cf.getCell(emu.posY, emu.posX+arg+length-1).wrap {
			// maintain wrap bit invariance at EOL
			emu.cf.getCellPtr(emu.posY, emu.posX+arg+length-1).wrap = false

			if length != 0 {
				emu.cf.getCellPtr(emu.posY, emu.posX+length-1).wrap = true
			} // TODO add logic for length ==0
		}

		emu.cf.moveInRow(emu.posY, emu.posX+arg, emu.posX, length)
		emu.cf.eraseInRow(emu.posY, emu.posX, arg, emu.attrs)
	}
	emu.lastCol = false
}

// CSI Ps J Erase in Display (ED), VT100.
// * Ps = 0  ⇒  Erase Below (default).
// * Ps = 1  ⇒  Erase Above.
// * Ps = 2  ⇒  Erase All.
// * Ps = 3  ⇒  Erase Saved Lines, xterm.
func hdl_csi_ed(emu *Emulator, cmd int) {
	emu.normalizeCursorPos()
	switch cmd {
	case 0: // clear from cursor to end of screen
		emu.cf.eraseInRow(emu.posY, emu.posX, emu.nCols-emu.posX, emu.attrs)
		for pY := emu.posY + 1; pY < emu.nRows; pY++ {
			emu.eraseRow(pY)
		}
	case 1: // clear from beginning of screen to cursor
		for pY := 0; pY < emu.posY; pY++ {
			emu.eraseRow(pY)
		}
		emu.cf.eraseInRow(emu.posY, 0, emu.posX+1, emu.attrs)
	case 3: // clear entire screen including scrollback buffer (xterm)
		emu.cf.dropScrollbackHistory()
		fallthrough
	case 2: // clear entire screen
		for pY := 0; pY < emu.nRows; pY++ {
			emu.eraseRow(pY)
		}
	default:
		// emu.logI.Printf("Erase in Display with illegal param: %d\n", cmd)
		util.Logger.Info("Erase in Display with illegal param", "cmd", cmd)
	}
}

// CSI Ps K Erase in Line (EL), VT100.
// * Ps = 0  ⇒  Erase to Right (default).
// * Ps = 1  ⇒  Erase to Left.
// * Ps = 2  ⇒  Erase All.
func hdl_csi_el(emu *Emulator, cmd int) {
	emu.normalizeCursorPos()
	switch cmd {
	case 0: // clear from cursor to end of line
		emu.cf.eraseInRow(emu.posY, emu.posX, emu.nCols-emu.posX, emu.attrs)
	case 1: // clear from cursor to beginning of line
		emu.cf.eraseInRow(emu.posY, 0, emu.posX+1, emu.attrs)
	case 2: // clear entire line
		emu.cf.eraseInRow(emu.posY, 0, emu.nCols, emu.attrs)
	default:
		// emu.logI.Printf("Erase in Line with illegal param: %d\n", cmd)
		util.Logger.Info("Erase in Line with illegal param", "cmd", cmd)
	}
}

// CSI Ps L  Insert Ps Line(s) (default = 1) (IL).
// insert N lines in cursor position
func hdl_csi_il(emu *Emulator, lines int) {
	if emu.isCursorInsideMargins() {
		lines = min(lines, emu.marginBottom-emu.posY)
		emu.insertRows(emu.posY, lines)
		hdl_c0_cr(emu)
	}
}

// CSI Ps M  Delete Ps Line(s) (default = 1) (DL).
// delete N lines in cursor position
func hdl_csi_dl(emu *Emulator, lines int) {
	if emu.isCursorInsideMargins() {
		lines = min(lines, emu.marginBottom-emu.posY)
		emu.deleteRows(emu.posY, lines)
		hdl_c0_cr(emu)
	}
}

// CSI I FocusIn
// CSI O FocusOut
func hdl_csi_focus(emu *Emulator, hasFocus bool) {
	if emu.mouseTrk.focusEventMode {
		// if hasFocus {
		// 	emu.writePty("\x1B[I")
		// } else {
		// 	emu.writePty("\x1B[O")
		// }
		emu.setHasFocus(hasFocus)
	}
}

// CSI Ps P  Delete Ps Character(s) (default = 1) (DCH).
func hdl_csi_dch(emu *Emulator, arg int) {
	if emu.isCursorInsideMargins() {
		length := emu.nColsEff - emu.posX
		arg = min(arg, length)
		// arg = calculateCellNum(emu, arg)
		length -= arg

		// fmt.Printf("#hdl_csi_dch posX=%d, arg=%d\n", emu.posX, arg)
		emu.cf.moveInRow(emu.posY, emu.posX, emu.posX+arg, length)
		emu.cf.eraseInRow(emu.posY, emu.posX+length, arg, emu.attrs)
	}
	emu.lastCol = false
}

// CSI Ps S  Scroll up Ps lines (default = 1) (SU), VT420, ECMA-48.
func hdl_csi_su(emu *Emulator, arg int) {
	if emu.horizMarginMode {
		arg = min(arg, emu.marginBottom-emu.marginTop)
		emu.deleteRows(emu.marginTop, arg)
	} else {
		emu.cf.scrollUp(arg)
		emu.eraseRows(emu.marginBottom-arg, arg)
		emu.lastCol = false
	}
}

// CSI Ps T  Scroll down Ps lines (default = 1) (SD), VT420.
func hdl_csi_sd(emu *Emulator, arg int) {
	if emu.horizMarginMode {
		arg = min(arg, emu.marginBottom-emu.marginTop)
		emu.insertRows(emu.marginTop, arg)
	} else {
		emu.cf.scrollDown(arg)
		emu.eraseRows(emu.marginTop, arg)
		emu.lastCol = false
	}
}

// CSI Ps X  Erase Ps Character(s) (default = 1) (ECH).
func hdl_csi_ech(emu *Emulator, arg int) {
	length := emu.nColsEff - emu.posX
	arg = min(arg, length)

	emu.cf.eraseInRow(emu.posY, emu.posX, arg, emu.attrs)
	emu.lastCol = false
}

// CSI Ps c  Send Device Attributes (Primary DA).
// CSI ? 6 2 ; Ps c  ("VT220")
// DA response
func hdl_csi_priDA(emu *Emulator) {
	// mosh only reply "\x1B[?62c" plain vt220
	resp := fmt.Sprintf("\x1B[?%s", DEVICE_ID)
	emu.writePty(resp)
}

// CSI > Ps c Send Device Attributes (Secondary DA).
// Ps = 0  or omitted ⇒  request the terminal's identification code.
// CSI > Pp ; Pv ; Pc c
// Pp = 1  ⇒  "VT220".
// Pv is the firmware version.
// Pc indicates the ROM cartridge registration number and is always zero.
func hdl_csi_secDA(emu *Emulator) {
	// mosh only reply "\033[>1;10;0c" plain vt220
	resp := "\x1B[>64;0;0c" // VT520
	emu.writePty(resp)
}

// CSI Ps d  Line Position Absolute  [row] (default = [1,column]) (VPA).
// Move cursor to line Pn.
func hdl_csi_vpa(emu *Emulator, row int) {
	row = max(1, min(row, emu.nRows))
	emu.posY = row - 1
	emu.lastCol = false
}

// CSI Ps e  Line Position Relative  [rows] (default = [row+1,column]) (VPR).
// Move cursor to the n-th line relative to active row
func hdl_csi_vpr(emu *Emulator, row int) {
	row += emu.posY + 1
	row = max(1, min(row, emu.nRows))
	emu.posY = row - 1
	emu.lastCol = false
}

// Move the active position to the n-th character of the active line.
// CSI Ps G  Cursor Character Absolute  [column] (default = [row,1]) (CHA).
func hdl_csi_cha(emu *Emulator, count int) {
	count = max(1, min(count, emu.nCols))
	emu.posX = count - 1
	emu.lastCol = false
}

// Move the active position to the n-th character of the active line.
// CSI Ps `  Character Position Absolute  [column] (default = [row,1]) (HPA).
// same as CHA
func hdl_csi_hpa(emu *Emulator, count int) {
	hdl_csi_cha(emu, count)
}

// CSI Ps a  Character Position Relative  [columns] (default = [row,col+1]) (HPR).
// move to the n-th character relative to the active position
func hdl_csi_hpr(emu *Emulator, arg int) {
	hdl_csi_cha(emu, emu.posX+arg+1)
}

// CSI Ps A  Cursor Up Ps Times (default = 1) (CUU).
func hdl_csi_cuu(emu *Emulator, num int) {
	if emu.posY >= emu.marginTop {
		num = min(num, emu.posY-emu.marginTop)
	} else {
		num = min(num, emu.posY)
	}
	emu.posY -= num
	emu.lastCol = false
}

// CSI Ps B  Cursor Down Ps Times (default = 1) (CUD).
func hdl_csi_cud(emu *Emulator, num int) {
	if emu.posY < emu.marginBottom {
		num = min(num, emu.marginBottom-emu.posY-1)
	} else {
		num = min(num, emu.nRows-emu.posY-1)
	}
	emu.posY += num
	emu.lastCol = false
}

// CSI Ps C  Cursor Forward Ps Times (default = 1) (CUF).
func hdl_csi_cuf(emu *Emulator, num int) {
	num = min(num, emu.nColsEff-emu.posX-1)
	emu.posX += num
	// emu.posX += calculateCellNum(emu, num)
	emu.lastCol = false
}

// CSI Ps D  Cursor Backward Ps Times (default = 1) (CUB).
func hdl_csi_cub(emu *Emulator, num int) {
	// fmt.Printf("hdl_csi_cub num=%d, posX=%d, hMargin=%d\n", num, emu.posX, emu.hMargin)
	if emu.posX >= emu.hMargin {
		num = min(num, emu.posX-emu.hMargin)
	} else {
		num = min(num, emu.posX)
	}
	if emu.posX == emu.nColsEff {
		num = min(num+1, emu.posX)
	}
	emu.posX -= num
	// fmt.Printf("hdl_csi_cub -num=%d\n", -num)
	// emu.posX += calculateCellNum(emu, -num)
	emu.lastCol = false
}

/*
// calculate raw cell number with the consideration of wide grapheme and
// regular grapheme. one wide grapheme takes two raw cells. one regular
// grapheme takes one cell, count is the number of graphemes.
//
// count >0 , count raw cell to right.
// count <0 , count raw cell to left.
func calculateCellNum(emu *Emulator, count int) int {
	oldX := emu.posX
	currentX := emu.posX // the start position
	// var cell Cell

	for i := 0; i < Abs(count); i++ {
		// fmt.Printf("#calculateCellNum currentX=%d\n", currentX)
		if count > 0 { // calculate to the right
			if currentX >= emu.nColsEff-1 {
				currentX = emu.nColsEff
				break
			}
			// emu.GetCell(emu.posY, currentX+1)
			// cell = emu.GetCell(emu.posY, currentX+1)
			// if cell.dwidth || cell.dwidthCont {
			// 	currentX += 2
			// } else {
			// 	currentX++
			// }
			currentX++
		} else { // calculate to the left
			// fmt.Printf("#calculateCellNum currentX=%d, count=%d, emu.hMargin=%d\n", currentX, count, emu.hMargin)
			// if currentX <= emu.hMargin {
			// 	currentX = emu.hMargin
			// 	break
			// }
			// emu.GetCell(emu.posY, currentX-1)
			// cell = emu.GetCell(emu.posY, currentX-1)
			// if cell.dwidthCont || cell.dwidth {
			// 	currentX -= 2
			// } else {
			// 	currentX--
			// }
			currentX--
		}
	}

	return currentX - oldX
}
*/

// CSI Ps ; Ps H Cursor Position [row;column] (default = [1,1]) (CUP).
func hdl_csi_cup(emu *Emulator, row int, col int) {
	switch emu.originMode {
	case OriginMode_Absolute:
		row = max(1, min(row, emu.nRows)) - 1
	case OriginMode_ScrollingRegion:
		row = max(1, min(row, emu.marginBottom)) - 1
		row += emu.marginTop
	}
	col = max(1, min(col, emu.nCols)) - 1

	emu.posX = col
	emu.posY = row
	emu.lastCol = false

	// emu.logT.Printf("Cursor positioned to (%d,%d)\n", emu.posY, emu.posX)
}

// CSI Ps n  Device Status Report (DSR).
//
//	Ps = 5  ⇒  Status Report. Result ("OK") is CSI 0 n
//	Ps = 6  ⇒  Report Cursor Position (CPR) [row;column]. Result is CSI r ; c R
func hdl_csi_dsr(emu *Emulator, cmd int) {
	switch cmd {
	case 5: // device status report requested
		emu.writePty("\x1B[0n") // device OK
	case 6: // report of active position requested
		resp := ""
		if emu.originMode == OriginMode_Absolute {
			resp = fmt.Sprintf("\x1B[%d;%dR", emu.posY+1, emu.posX+1)
		} else {
			// scrolling region mode
			resp = fmt.Sprintf("\x1B[%d;%dR", emu.posY-emu.marginTop+1, emu.posX+1)
		}
		emu.writePty(resp)
	default:
	}
}

// CSI Pm m  Character Attributes (SGR).
// select graphics rendition -- e.g., bold, blinking, etc.
// support 8, 16, 256 color, RGB color.
//
// https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
// https://wezfurlong.org/wezterm/faq.html#how-do-i-enable-undercurl-curly-underlines
func hdl_csi_sgr(emu *Emulator, params []int, seps ...rune) {
	// we need to change the field, get the field pointer
	rend := &emu.attrs.renditions
	for k := 0; k < len(params); k++ {
		// fmt.Printf("hdl_csi_sgr k=%2d, params[k]=%3d, params=%v, iterate\n", k, params[k], params)

		switch params[k] {
		case 38:
			if k >= len(params)-1 {
				break
			}
			switch params[k+1] {
			case 5:
				if k+1 >= len(params)-1 {
					k += len(params) - 1 - k
					break
				}
				rend.SetForegroundColor(params[k+2])
				k += 2
			case 2:
				if k+1 >= len(params)-3 {
					k += len(params) - 1 - k
					break
				}
				red := params[k+2]
				green := params[k+3]
				blue := params[k+4]
				rend.SetFgColor(red, green, blue)
				k += 4
			default:
				k += len(params) - 1 - k
			}
		case 48:
			if k >= len(params)-1 {
				break
			}
			switch params[k+1] {
			case 5:
				if k+1 >= len(params)-1 {
					k += len(params) - 1 - k
					break
				}
				rend.SetBackgroundColor(params[k+2])
				k += 2
			case 2:
				if k+1 >= len(params)-3 {
					k += len(params) - 1 - k
					break
				}
				red := params[k+2]
				green := params[k+3]
				blue := params[k+4]
				rend.SetBgColor(red, green, blue)
				k += 4
			default:
				k += len(params) - 1 - k
			}
		case 4:
			/*
				<ESC>[4:0m  # no underline
				<ESC>[4:1m  # straight underline
				<ESC>[4:2m  # double underline
				<ESC>[4:3m  # curly underline
				<ESC>[4:4m  # dotted underline
				<ESC>[4:5m  # dashed underline
				<ESC>[4m    # straight underline (for backwards compat)
				<ESC>[24m   # no underline (for backwards compat)
			*/
			// fmt.Printf("hdl_csi_sgr seps=%c params=%d\n", seps, params)
			// if k > len(params)-1 {
			// 	break
			// }
			if k+1 <= len(seps)-1 && seps[k] == ':' {
				// fmt.Printf("hdl_csi_sgr k=%d, seps[k]=%c, params[k+1]=%d\n", k, seps[k], params[k+1])
				switch params[k+1] {
				case 0:
					rend.underline = false
					rend.setUnderlineStyle(charAttribute(params[k+1]))
					k += 1
				case 1, 2, 3, 4, 5:
					rend.underline = true
					rend.setUnderlineStyle(charAttribute(params[k+1]))
					k += 1
				default:
					k += len(params) - 1 - k
				}
			} else {
				rend.underline = true
				rend.setUnderlineStyle(ULS_SINGLE)
			}
			// fmt.Printf("hdl_csi_sgr k=%d rend.underline=%t, rend.ulColor=%d, rend.ulStyle=%d\n",
			// 	k, rend.underline, rend.ulColor, rend.ulStyle)
		case 58:
			/*
			 CSI 58:2::R:G:B m   -> set underline color to specified true color RGB
			 CSI 58:5:I m        -> set underline color to palette index I (0-255)
			 CSI 59              -> restore underline color to default
			*/
			if k >= len(params)-1 {
				break
			}
			switch params[k+1] {
			case 5:
				if k+1 >= len(params)-1 {
					k += len(params) - 1 - k
					break
				}
				rend.setUnderlineColor(params[k+2])
				k += 2
			case 2:
				if k+1 >= len(params)-4 {
					k += len(params) - 1 - k
					break
				}
				red := params[k+3]
				green := params[k+4]
				blue := params[k+5]
				rend.setUnderlineRGBColor(red, green, blue)
				k += 5
			default:
				k += len(params) - 1 - k
			}
		case 59:
			rend.ulColor = ColorDefault
		default:
			/*
			 process the 8-color set, 16-color set and default color
			 CSI 24 m   -> No underline
			*/
			if !rend.buildRendition(params[k]) {
				util.Logger.Warn("attribute not supported", "unimplement", "CSI SGR", "params", params)
			}
		}
	}
}

/*
OSC Ps ; Pt ST

	The 10 colors (below) which may be set or queried using 1 0
	through 1 9  are denoted dynamic colors, since the
	corresponding control sequences were the first means for
	setting xterm's colors dynamically, i.e., after it was
	started.  They are not the same as the ANSI colors (however,
	the dynamic text foreground and background colors are used
	when ANSI colors are reset using SGR 3 9  and 4 9 ,
	respectively).  These controls may be disabled using the
	allowColorOps resource.  At least one parameter is expected
	for Pt.  Each successive parameter changes the next color in
	the list.  The value of Ps tells the starting point in the
	list.  The colors are specified by name or RGB specification
	as per XParseColor.

	If a "?" is given rather than a name or RGB specification,
	xterm replies with a control sequence of the same form which
	can be used to set the corresponding dynamic color.  Because
	more than one pair of color number and specification can be
	given in one control sequence, xterm can make more than one
	reply.

	  Ps = 1 0  ⇒  Change VT100 text foreground color to Pt.
	  Ps = 1 1  ⇒  Change VT100 text background color to Pt.
	  Ps = 1 2  ⇒  Change text cursor color to Pt.
	  Ps = 1 3  ⇒  Change pointer foreground color to Pt.
	  Ps = 1 4  ⇒  Change pointer background color to Pt.
	  Ps = 1 5  ⇒  Change Tektronix foreground color to Pt.
	  Ps = 1 6  ⇒  Change Tektronix background color to Pt.
	  Ps = 1 7  ⇒  Change highlight background color to Pt.
	  Ps = 1 8  ⇒  Change Tektronix cursor color to Pt.
	  Ps = 1 9  ⇒  Change highlight foreground color to Pt.
*/
func hdl_osc_10x(emu *Emulator, cmd int, arg string) {
	arg = fmt.Sprintf("%d;%s", cmd, arg) // add the cmd back to the arg
	count := strings.Count(arg, ";")
	if count > 0 && count%2 == 1 { // color pair has 2n-1 ';'
		pairs := (count + 1) / 2 // count the color pair
		if pairs >= 5 {          // limit the color pair up to 5
			pairs = 5
		}

		args := strings.Split(arg, ";")
		idx := 0
		for i := 0; i < pairs; i++ {

			idx = i * 2
			color := args[idx]
			action := args[idx+1]

			// we only support query for the time being.
			if action == "?" {
				colorIdx, err := strconv.Atoi(color)
				if err != nil {
					// emu.logW.Printf("OSC 10x: can't parse color index. %q\n", arg)
					util.Logger.Warn("OSC 10x: can't parse color index", "arg", arg)
					return
				}

				color := ColorDefault
				switch colorIdx {
				case 11, 17: // 11: VT100 text background color  17: highlight background color
					color = emu.attrs.renditions.bgColor
				case 10, 19: // 10: VT100 text foreground color; 19: highlight foreground color
					color = emu.attrs.renditions.fgColor
				case 12: // 12: text cursor color
					color = emu.cf.cursor.color
				}
				response := fmt.Sprintf("\x1B]%d;%s\x1B\\", colorIdx, color) // the String() method of Color will be called.
				emu.writePty(response)
			}
		}
	} else {
		// emu.logW.Printf("OSC 10x: malformed argument, missing ';'. %q\n", arg)
		util.Logger.Warn("OSC 10x: malformed argument, missing ';'", "arg", arg)
		return
	}
}

/*
 * OSC Ps ; Pt ST
 * Ps = 5 2  ⇒ Manipulate Selection Data. The parameter Pt is parsed as
 *
 * 		Pc ; Pd
 *
 * The first, Pc, may contain zero or more characters from the
 * set c , p , q , s , 0 , 1 , 2 , 3 , 4 , 5 , 6 , and 7 .  It is
 * used to construct a list of selection parameters for
 * clipboard, primary, secondary, select, or cut-buffers 0
 * through 7 respectively, in the order given.  If the parameter
 * is empty, xterm uses s 0 , to specify the configurable
 * primary/clipboard selection and cut-buffer 0.
 *
 * The second parameter, Pd, gives the selection data.  Normally
 * this is a string encoded in base64 (RFC-4648).  The data
 * becomes the new selection, which is then available for pasting
 * by other applications.
 *
 * If the second parameter is a ? , xterm replies to the host
 * with the selection data encoded using the same protocol.  It
 * uses the first selection found by asking successively for each
 * item from the list of selection parameters.
 *
 * If the second parameter is neither a base64 string nor ? ,
 * then the selection is cleared.
 *
 * 1. use one of the following commands to encode the original text and set the system clipboard
 * - % echo -e "\033]52;c;$(base64 <<< hello)\a"
 * - % echo "Hello Russia!" | base64
 * - SGVsbG8gUnVzc2lhIQo=
 * - % echo  -e "\033]52;p;SGVsbG8gUnVzc2lhIQo=\a"
 * 2. press the paste hot-eky, you will see the original text in step 1.
 * - % hello
 * - % Hello Russia!
 */
func hdl_osc_52(emu *Emulator, cmd int, arg string) {
	// parse Pc:Pd
	pos := strings.Index(arg, ";")
	if pos == -1 {
		// emu.logW.Printf("OSC 52: can't find Pc parameter. %q\n", arg)
		util.Logger.Warn("OSC 52: can't find Pc parameter", "arg", arg)
		return
	}
	Pc := arg[:pos]
	Pd := arg[pos+1:]

	/*
		If the parameter is empty, xterm uses s 0 , to specify the configurable
		primary/clipboard selection and cut-buffer 0.
	*/
	if Pc == "" {
		Pc = "pc" // zutty support pc instead of xterm s0
	}

	// validate Pc
	if !osc52InRange(Pc) {
		// emu.logW.Printf("OSC 52: invalid Pc parameters. %q\n", Pc)
		util.Logger.Warn("OSC 52: invalid Pc parameters", "Pc", Pc)
		return
	}

	if Pd == "?" {
		/*
			If the second parameter is a ? , xterm replies to the host
			with the selection data encoded using the same protocol.  It
			uses the first selection found by asking successively for each
			item from the list of selection parameters.
		*/
		for _, ch := range Pc {
			if data, ok := emu.selectionStore[ch]; ok && data != "" {
				// resp to the host
				resp := fmt.Sprintf("\x1B]%d;%c;%s\x1B\\", cmd, ch, data)
				emu.writePty(resp)
				break
			}
		}
	} else {
		// TODO please consider the race condition
		_, err := base64.StdEncoding.DecodeString(Pd)
		set := false
		// fmt.Printf("#hdl_osc_52 Pd=%q, err==nil is %t\n", Pd, err==nil)
		if err == nil && len(Pd) != 0 { // it's a base64 string
			/*
			   The second parameter, Pd, gives the selection data.  Normally
			   this is a string encoded in base64 (RFC-4648).  The data
			   becomes the new selection, which is then available for pasting
			   by other applications.
			*/
			for _, ch := range Pc {
				if _, ok := emu.selectionStore[ch]; ok { // make sure Pc exist
					// update the new selection in local cache
					emu.selectionStore[ch] = Pd
					set = true
				}
			}
			if set {
				// store the selection data, later it will be sent to terminal.
				emu.selectionData = fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", cmd, Pc, Pd)
			}
		} else {
			/*
			   If the second parameter is neither a base64 string nor ? ,
			   then the selection is cleared.
			*/
			for _, ch := range Pc {
				if _, ok := emu.selectionStore[ch]; ok { // make sure Pc exist
					// clear the selection in local cache
					emu.selectionStore[ch] = ""
					set = true
				}
			}
			if set {
				// store the selection data, later it will be sent to terminal.
				emu.selectionData = fmt.Sprintf("\x1B]%d;%s;%s\x1B\\", cmd, Pc, Pd)
			}
		}
	}
}

// validate the Pc content is in the fix set
func osc52InRange(Pc string) (ret bool) {
	specSet := "cpqs01234567"
	for _, ch := range Pc {
		if !strings.Contains(specSet, string(ch)) {
			return false
		}
	}

	return true
}

/*
Ps = 4 ; c ; spec ⇒  Change Color Number c to the color specified by spec.

The spec can be a name or RGB specification as per
XParseColor.  Any number of c/spec pairs may be given.  The
color numbers correspond to the ANSI colors 0-7, their bright
versions 8-15, and if supported, the remainder of the 88-color
or 256-color table.

If a "?" is given rather than a name or RGB specification,
xterm replies with a control sequence of the same form which
can be used to set the corresponding color.  Because more than
one pair of color number and specification can be given in one
control sequence, xterm can make more than one reply.

string names for colors: https://tronche.com/gui/x/xlib/color/strings/
xterm 256 color protocol: https://unix.stackexchange.com/questions/105568/how-can-i-list-the-available-color-names
color and formatting: https://misc.flogisoft.com/bash/tip_colors_and_formatting
*/
func hdl_osc_4(emu *Emulator, cmd int, arg string) {
	count := strings.Count(arg, ";")
	if count > 0 && count%2 == 1 { // c/spec pair has 2n-1 ';'
		pairs := (count + 1) / 2 // count the c/spec pair
		if pairs >= 8 {          // limit the c/spec pair up to 8
			pairs = 8
		}

		args := strings.Split(arg, ";")
		idx := 0
		for i := 0; i < pairs; i++ {

			idx = i * 2
			c := args[idx]
			spec := args[idx+1]

			// we only support query for the time being.
			if spec == "?" {
				colorIdx, err := strconv.Atoi(c)
				if err != nil {
					// emu.logW.Printf("OSC 4: can't parse c parameter. %q\n", arg)
					util.Logger.Warn("OSC 4: can't parse c parameter", "arg", arg)
					return
				}
				color := PaletteColor(colorIdx)
				response := fmt.Sprintf("\x1B]%d;%d;%s\x1B\\", cmd, colorIdx, color)
				emu.writePty(response)
			} // TODO set the screen colors palette values to any RGB value.
		}
	} else {
		// emu.logW.Printf("OSC 4: malformed argument, missing ';'. %q\n", arg)
		util.Logger.Warn("OSC 4: malformed argument, missing ';'", "arg", arg)
		return
	}
}

// OSC of the form "\x1B]X;<title>\007" where X can be:
// * 0: set icon name and window title
// * 1: set icon name
// * 2: set window title
func hdl_osc_0_1_2(emu *Emulator, cmd int, arg string) {
	// set icon name / window title
	setIcon := cmd == 0 || cmd == 1
	setTitle := cmd == 0 || cmd == 2
	if setIcon || setTitle {
		emu.setTitleInitialized()

		if setIcon {
			emu.setIconLabel(arg)
		}

		if setTitle {
			emu.setWindowTitle(arg)
			// util.Log.Debug("OSC 0 set window title","title", emu.GetWindowTitle())
		}
	}
}

// OSC Ps ; Pt BEL
//
// OSC Ps ; Pt ST
//
//	The dynamic colors can also be reset to their default (resource) values:
//	  Ps = 1 1 2  ⇒  Reset text cursor color.
func hdl_osc_112(emu *Emulator, _ int, _ string) {
	emu.cf.cursor.color = ColorDefault
}

// A hyperlink is opened upon encountering an OSC 8 escape sequence with the target URI.
// The syntax is
//
// OSC 8 ; params ; URI ST
//
// Following this, all subsequent cells that are painted are hyperlinks to this target.
// A hyperlink is closed with the same escape sequence, omitting the parameters and the
// URI but keeping the separators:
//
// OSC 8 ; ; ST
//
// printf '\e]8;12;http://example.com\e\\This is a link\e]8;;\e\\\n'
func hdl_osc_8(_ *Emulator, cmd int, arg string) {
	util.Logger.Warn("OSC 8 is not implemented!", "cmd", cmd, "arg", arg)
}

// CSI Pm h  Set Mode (SM).
// *  Ps = 2  ⇒  Keyboard Action Mode (KAM).
// *  Ps = 4  ⇒  Insert Mode (IRM).
// *  Ps = 1 2  ⇒  Send/receive (SRM).
// *  Ps = 2 0  ⇒  Automatic Newline (LNM).
func hdl_csi_sm(emu *Emulator, params []int) {
	for _, param := range params {
		switch param {
		case 2:
			emu.keyboardLocked = true
		case 4:
			emu.insertMode = true
		case 12:
			emu.localEcho = false
		case 20:
			emu.autoNewlineMode = true
		default:
			// emu.logW.Printf("Ignored bogus set mode %d.\n", param)
			util.Logger.Warn("Ignored bogus set mode", "param", param)
		}
	}
}

// CSI Pm l  Reset Mode (RM).
// *  Ps = 2  ⇒  Keyboard Action Mode (KAM).
// *  Ps = 4  ⇒  Replace Mode (IRM).
// *  Ps = 1 2  ⇒  Send/receive (SRM).
// *  Ps = 2 0  ⇒  Normal Linefeed (LNM).
func hdl_csi_rm(emu *Emulator, params []int) {
	for _, param := range params {
		switch param {
		case 2:
			emu.keyboardLocked = false
		case 4:
			emu.insertMode = false
		case 12:
			emu.localEcho = true
		case 20:
			emu.autoNewlineMode = false
		default:
			// emu.logW.Printf("Ignored bogus reset mode %d.\n", param)
			util.Logger.Warn("Ignored bogus reset mode", "param", param)
		}
	}
}

// CSI ? Pm h
// DEC Private Mode Set (DECSET).
func hdl_csi_privSM(emu *Emulator, params []int) {
	for _, param := range params {
		switch param {
		case 1:
			// emu.framebuffer.DS.ApplicationModeCursorKeys = true // DECCKM Apllication zutty:cursorKeyMode
			emu.cursorKeyMode = CursorKeyMode_Application
		case 2:
			resetCharsetState(&emu.charsetState) // Designate USASCII for character sets G0-G3 (DECANM), VT100, and set VT100 mode.
			emu.setCompatLevel(CompatLevel_VT400)
			// util.Logger.Warn("DECANM", "changeTo", "Designate USASCII for character sets G0-G3 (DECANM), VT100, and set VT100 mode", "params", params)
		case 3:
			emu.switchColMode(ColMode_C132)
		case 4:
			// emu.logT.Println("DECSCLM: Set smooth scroll")
			util.Logger.Debug("DECSCLM: Set smooth scroll", "unimplement", "DECSET", "params", params)
		case 5:
			// emu.framebuffer.DS.ReverseVideo = true // DECSCNM Reverse
			emu.reverseVideo = true
		case 6:
			// emu.framebuffer.DS.OriginMode = true // DECOM ScrollingRegion zutty:originMode
			emu.originMode = OriginMode_ScrollingRegion
		case 7:
			// emu.framebuffer.DS.AutoWrapMode = true // DECAWM zutty:autoWrapMode
			emu.autoWrapMode = true
		case 8:
			// emu.logU.Println("DECARM: Set auto-repeat mode")
			util.Logger.Warn("DECARM: Set auto-repeat mode", "unimplement", "DECSET", "params", params)
		case 9:
			// emu.framebuffer.DS.mouseTrk.mode = MouseModeX10
			emu.mouseTrk.mode = MouseTrackingMode_X10_Compat
		case 12:
			hdl_csi_decscusr(emu, int(CursorStyle_BlinkBlock))
		case 13:
			// Start blinking cursor (set only via resource or menu)
			util.Logger.Warn("Start blinking cursor", "unimplement", "DECSET", "params", params)
		case 25:
			// emu.framebuffer.DS.CursorVisible = true // DECTCEM zutty:showCursorMode
			emu.showCursorMode = true
		case 47:
			emu.switchScreenBufferMode(true)
		case 67:
			// emu.framebuffer.DS.bkspSendsDel = false // Backarrow key sends backspace (DECBKM), VT340, VT420.
			emu.bkspSendsDel = false
		case 69:
			// emu.framebuffer.DS.horizMarginMode = true // DECLRMM: Set Left and Right Margins
			// emu.framebuffer.DS.hMargin = 0
			// emu.framebuffer.DS.nColsEff = emu.framebuffer.DS.width
			emu.horizMarginMode = true
			emu.hMargin = 0
			emu.nColsEff = emu.nCols
		case 1000:
			// emu.framebuffer.DS.mouseTrk.mode = MouseModeVT200
			emu.mouseTrk.mode = MouseTrackingMode_VT200
		case 1001:
			// emu.logU.Println("Set VT200 Highlight Mouse mode")
			emu.mouseTrk.mode = MouseTrackingMode_VT200_HighLight
		case 1002:
			// emu.framebuffer.DS.mouseTrk.mode = MouseModeButtonEvent
			emu.mouseTrk.mode = MouseTrackingMode_VT200_ButtonEvent
		case 1003:
			// emu.framebuffer.DS.mouseTrk.mode = MouseModeAnyEvent
			emu.mouseTrk.mode = MouseTrackingMode_VT200_AnyEvent
		case 1004:
			// TODO replace MouseFocusEvent with mouseTrk.focusEventMode
			// emu.framebuffer.DS.MouseFocusEvent = true // xterm zutty:mouseTrk.focusEventMode
			// emu.framebuffer.DS.mouseTrk.focusEventMode = true
			emu.mouseTrk.focusEventMode = true
		case 1005:
			// emu.framebuffer.DS.mouseTrk.enc = MouseEncUTF
			emu.mouseTrk.enc = MouseTrackingEnc_UTF8
		case 1006:
			// emu.framebuffer.DS.mouseTrk.enc = MouseEncSGR
			emu.mouseTrk.enc = MouseTrackingEnc_SGR
		case 1007:
			// emu.framebuffer.DS.MouseAlternateScroll = true // xterm zutty:altScrollMode
			emu.altScrollMode = true
		case 1015:
			// emu.framebuffer.DS.mouseTrk.enc = MouseEncURXVT
			emu.mouseTrk.enc = MouseTrackingEnc_URXVT
		case 1036, 1039:
			// emu.framebuffer.DS.altSendsEscape = true
			emu.altSendsEscape = true
		case 1047:
			emu.switchScreenBufferMode(true)
		case 1048:
			hdl_esc_decsc(emu)
		case 1049:
			hdl_esc_decsc(emu)
			emu.switchScreenBufferMode(true)
			emu.altScreen1049 = true
			// util.Log.Debug("privSM",
			// 	"altScreenBufferMode", emu.altScreenBufferMode,
			// 	"altScreen1049", emu.altScreen1049)
		case 2004:
			// emu.framebuffer.DS.BracketedPaste = true // xterm zutty:bracketedPasteMode
			emu.bracketedPasteMode = true
		default:
			// emu.logU.Printf("set priv mode %d\n", param)
			util.Logger.Warn("set priv mode", "unimplement", "DECSET", "params", param)
		}
	}
}

// TODO: Synchronized output is not supported
// https://gist.github.com/christianparpart/d8a62cc1ab659194337d73e399004036
func hdl_csi_decrqm(emu *Emulator, params []int) {
	resp := fmt.Sprintf("\x1B[?%d;%d$y", params[0], 0)
	util.Logger.Debug("Synchronized output is not supported", "resp", resp)
	emu.writePty(resp)
}

// TODO: implement it
// Detection of support for this protocol
// An application can query the terminal for support of this protocol by sending
// the escape code querying for the current progressive enhancement status followed
// by request for the primary device attributes. If an answer for the device
// attributes is received without getting back an answer for the progressive
// enhancement the terminal does not support this protocol.
func hdl_csi_u(_ *Emulator, _ []int) {
	util.Logger.Warn("CSI U is not supported")
}

// CSI ? Pm l
// DEC Private Mode Reset (DECRST).
func hdl_csi_privRM(emu *Emulator, params []int) {
	for _, param := range params {
		switch param {
		case 1:
			// emu.framebuffer.DS.ApplicationModeCursorKeys = false // ANSI
			emu.cursorKeyMode = CursorKeyMode_ANSI
		case 2:
			resetCharsetState(&emu.charsetState) // Designate VT52 mode (DECANM), VT100.
			emu.setCompatLevel(CompatLevel_VT52)
			// util.Logger.Warn("DECANM", "changeTo", "Designate VT52 mode", "params", params)
		case 3:
			emu.switchColMode(ColMode_C80)
		case 4:
			// emu.logT.Println("DECSCLM: Set jump scroll")
			util.Logger.Debug("DECSCLM: Set jump scroll", "unimplement", "DECRST", "params", params)
		case 5:
			// emu.framebuffer.DS.ReverseVideo = false // Normal
			emu.reverseVideo = false
		case 6:
			// emu.framebuffer.DS.OriginMode = false // Absolute
			emu.originMode = OriginMode_Absolute
		case 7:
			// emu.framebuffer.DS.AutoWrapMode = false
			emu.autoWrapMode = false
		case 8:
			// emu.logU.Println("DECARM: Reset auto-repeat mode")
			util.Logger.Warn("DECARM: Reset auto-repeat mode", "unimplement", "DECRST", "params", params)
		case 9, 1000, 1001, 1002, 1003:
			// emu.framebuffer.DS.mouseTrk.mode = MouseModeNone
			emu.mouseTrk.mode = MouseTrackingMode_Disable
		case 12:
			hdl_csi_decscusr(emu, int(CursorStyle_SteadyBlock))
		case 13:
			// Disable blinking cursor (reset only via resource or menu).
			util.Logger.Warn("Stop blinking cursor", "unimplement", "DECRST", "params", params)
		case 25:
			// emu.framebuffer.DS.CursorVisible = false
			emu.showCursorMode = false
		case 47:
			emu.switchScreenBufferMode(false)
		case 67:
			// emu.framebuffer.DS.bkspSendsDel = true // Backarrow key sends delete (DECBKM), VT340,
			emu.bkspSendsDel = true
		case 69:
			// emu.framebuffer.DS.horizMarginMode = false // DECLRMM: Set Left and Right Margins
			// emu.framebuffer.DS.hMargin = 0
			// emu.framebuffer.DS.nColsEff = emu.framebuffer.DS.width
			emu.horizMarginMode = false
			emu.hMargin = 0
			emu.nColsEff = emu.nCols
		// case 1001:
		// 	emu.logU.Println("Reset VT200 Highlight Mouse mode")
		case 1004:
			// TODO replace MouseFocusEvent with mouseTrk.focusEventMode
			// emu.framebuffer.DS.MouseFocusEvent = false
			// emu.framebuffer.DS.mouseTrk.focusEventMode = false
			emu.mouseTrk.focusEventMode = false
		case 1005, 1006, 1015:
			// emu.framebuffer.DS.mouseTrk.enc = MouseEncNone
			emu.mouseTrk.enc = MouseTrackingEnc_Default
		case 1007:
			// emu.framebuffer.DS.MouseAlternateScroll = false
			emu.altScrollMode = false
		case 1036, 1039:
			// emu.framebuffer.DS.altSendsEscape = false
			emu.altSendsEscape = false
		case 1047:
			emu.switchScreenBufferMode(false)
		case 1048:
			hdl_esc_decrc(emu)
		case 1049:
			emu.switchScreenBufferMode(false)
			hdl_esc_decrc(emu)
			emu.altScreen1049 = false
			// fmt.Printf("privRM:1559 swith screen buffer mode. altScreen1049=%t\n", emu.altScreen1049)
		case 2004:
			// emu.framebuffer.DS.BracketedPaste = false
			emu.bracketedPasteMode = false
		default:
			// emu.logU.Printf("reset priv mode %d\n", param)
			util.Logger.Warn("reset priv mode", "unimplement", "DECRST", "params", param)
		}
	}
}

// CSI Ps ; Ps r
// Set Scrolling Region [top;bottom] (default = full size of  window) (DECSTBM), VT100.
func hdl_csi_decstbm(emu *Emulator, params []int) {
	// only top is set to 0
	if len(params) == 0 {
		if emu.marginTop != 0 || emu.marginBottom != emu.nRows {
			emu.marginTop, emu.marginBottom = emu.cf.resetMargins()
		}
	} else if len(params) == 2 {
		newMarginTop := 0
		newMarginBottom := params[1]

		if params[0] > 0 {
			newMarginTop = params[0] - 1
		}

		if newMarginBottom < newMarginTop+2 || emu.nRows < newMarginBottom {
			// emu.logT.Printf("Illegal arguments to SetTopBottomMargins: top=%d, bottom=%d\n", params[0], params[1])
			util.Logger.Warn("Illegal arguments to SetTopBottomMargins",
				"top", params[0],
				"bottom", params[1])
		} else if newMarginTop != emu.marginTop || newMarginBottom != emu.marginBottom {
			emu.marginTop = newMarginTop
			emu.marginBottom = newMarginBottom
			if emu.marginTop == 0 && emu.marginBottom == emu.nRows {
				emu.marginTop, emu.marginBottom = emu.cf.resetMargins()
			} else {
				emu.cf.setMargins(emu.marginTop, emu.marginBottom)
			}
		}
	}

	if emu.originMode == OriginMode_Absolute {
		emu.posX = 0
		emu.posY = 0
	} else {
		emu.posX = emu.hMargin
		emu.posY = emu.marginTop
	}
	emu.lastCol = false
}

// CSI ! p   Soft terminal reset (DECSTR), VT220 and up.
func hdl_csi_decstr(emu *Emulator) {
	emu.resetScreen()
	emu.resetAttrs()
}

// DCS $ q Pt ST
//
//	Request Status String (DECRQSS), VT420 and up.
//	The string following the "q" is one of the following:
//	  m       ⇒  SGR
//	  " p     ⇒  DECSCL
//	  SP q    ⇒  DECSCUSR
//	  " q     ⇒  DECSCA
//	  r       ⇒  DECSTBM
//	  s       ⇒  DECSLRM
//	  t       ⇒  DECSLPP
//	  $ |     ⇒  DECSCPP
//	  $ }     ⇒  DECSASD
//	  $ ~     ⇒  DECSSDT
//	  * |     ⇒  DECSNLS
//	xterm responds with DCS 1 $ r Pt ST for valid requests,
//	replacing the Pt with the corresponding CSI string, or DCS 0 $
//	r Pt ST for invalid requests.
//
// DECRQSS—Request Selection or Setting
func hdl_dcs_decrqss(emu *Emulator, arg string) {
	// only response to DECSCL
	if arg == "$q\"p" {
		emu.setCompatLevel(CompatLevel_VT400)
		resp := fmt.Sprintf("\x1BP1$r%s\x1B\\", DEVICE_ID)
		emu.writePty(resp)
	} else {
		resp := fmt.Sprintf("\x1BP0$r%s\x1B\\", arg[2:])
		emu.writePty(resp)
	}
}

/*
DCS + q Pt ST

	Request Termcap/Terminfo String (XTGETTCAP), xterm.  The
	string following the "q" is a list of names encoded in
	hexadecimal (2 digits per character) separated by ; which
	correspond to termcap or terminfo key names.
	A few special features are also recognized, which are not key
	names:

	o   Co for termcap colors (or colors for terminfo colors), and

	o   TN for termcap name (or name for terminfo name).

	o   RGB for the ncurses direct-color extension.
	    Only a terminfo name is provided, since termcap
	    applications cannot use this information.

	xterm responds with
	DCS 1 + r Pt ST for valid requests, adding to Pt an = , and
	the value of the corresponding string that xterm would send,
	or
	DCS 0 + r ST for invalid requests.
	The strings are encoded in hexadecimal (2 digits per
	character).  If more than one name is given, xterm replies
	with each name/value pair in the same response.  An invalid
	name (one not found in xterm's tables) ends processing of the
	list of names.
*/
func hdl_dcs_xtgettcap(_ *Emulator, arg string) {
	name := strings.Split(arg, ";")
	n2 := []string{}
	for i := range name {
		dst := make([]byte, hex.DecodedLen(len(name[i])))
		n, err := hex.Decode(dst, []byte(name[i]))
		if err != nil {
			util.Logger.Warn("XTGETTCAP decode error", "i", i, "name[i]", name[i], "error", err)
		}
		// util.Logger.Warn("XTGETTCAP", "i", i, "name[i]", dst[:n])
		n2 = append(n2, string(dst[:n]))
		// TODO: lookup terminfo and return the result
	}
	util.Logger.Warn("XTGETTCAP is not implemented!", "arg", arg, "name", name)
	util.Logger.Warn("XTGETTCAP is not implemented!", "n2", n2)
}

// CSI Pl ; Pr s
//
//	Set left and right margins (DECSLRM), VT420 and up.  This is
//	available only when DECLRMM is enabled.
func hdl_csi_decslrm(emu *Emulator, params []int) {
	if len(params) == 0 {
		emu.hMargin = 0
		emu.nColsEff = emu.nCols
	} else if len(params) == 2 {
		newMarginLeft := params[0]
		newMarginRight := params[1]

		if newMarginLeft > 0 {
			newMarginLeft -= 1
		}

		if newMarginRight < newMarginLeft+2 || emu.nCols < newMarginRight {
			// emu.logT.Printf("Illegal arguments to SetLeftRightMargins: left=%d, right=%d\n", params[0], params[1])
			util.Logger.Warn("Illegal arguments to SetLeftRightMargins",
				"left", params[0],
				"right", params[1])
		} else if newMarginLeft != emu.hMargin || newMarginRight != emu.nColsEff {
			emu.hMargin = newMarginLeft
			emu.nColsEff = newMarginRight
		}
	}

	if emu.originMode == OriginMode_Absolute { // false means Absolute
		emu.posX = 0
		emu.posY = 0
	} else {
		emu.posX = emu.hMargin
		emu.posY = emu.marginTop
	}
	emu.lastCol = false
}

// CSI s     Save cursor, available only when DECLRMM is disabled (SCOSC, also ANSI.SYS).
func hdl_csi_scosc(emu *Emulator) {
	emu.savedCursor_SCO.posX = emu.posX
	emu.savedCursor_SCO.posY = emu.posY
	emu.savedCursor_SCO.isSet = true
}

// disambiguate SLRM and SCOSC based on horizMarginMode
func hdl_csi_slrm_scosc(emu *Emulator, params []int) {
	if emu.horizMarginMode {
		hdl_csi_decslrm(emu, params)
	} else {
		hdl_csi_scosc(emu)
	}
}

// CSI u     Restore cursor (SCORC, also ANSI.SYS).
func hdl_csi_scorc(emu *Emulator) {
	if !emu.savedCursor_SCO.isSet {
		// emu.logI.Println("Asked to restore cursor (SCORC) but it has not been saved.")
		util.Logger.Info("Asked to restore cursor (SCORC) but it has not been saved")
	} else {
		emu.posX = emu.savedCursor_SCO.posX
		emu.posY = emu.savedCursor_SCO.posY
		emu.normalizeCursorPos()
		emu.savedCursor_SCO.isSet = false
	}
}

// CSI Pl ; Pc " p
//
//	Set conformance level (DECSCL), VT220 and up.
//
//	The first parameter selects the conformance level.  Valid
//	values are:
//	  Pl = 6 1  ⇒  level 1, e.g., VT100.
//	  Pl = 6 2  ⇒  level 2, e.g., VT200.
//	  Pl = 6 3  ⇒  level 3, e.g., VT300.
//	  Pl = 6 4  ⇒  level 4, e.g., VT400.
//	  Pl = 6 5  ⇒  level 5, e.g., VT500.
//
//	The second parameter selects the C1 control transmission mode.
//	This is an optional parameter, ignored in conformance level 1.
//	Valid values are:
//	  Pc = 0  ⇒  8-bit controls.
//	  Pc = 1  ⇒  7-bit controls (DEC factory default).
//	  Pc = 2  ⇒  8-bit controls.
//
//	The 7-bit and 8-bit control modes can also be set by S7C1T and
//	S8C1T, but DECSCL is preferred.
func hdl_csi_decscl(emu *Emulator, params []int) {
	if len(params) > 0 {
		switch params[0] {
		case 61:
			// emu.setCompatLevel(CompatLevel_VT100)
			fallthrough
		case 62, 63, 64, 65:
			// emu.setCompatLevel(CompatLevel_VT400)
			emu.setCompatLevel(sclCompatLevel(params[0]))
		default:
			// emu.logU.Printf("compatibility mode: %d", params[0])
			util.Logger.Warn("compatibility mode",
				"unimplement", "DECSCL",
				"param", params[0])
		}
	}
	if len(params) > 1 {
		switch params[1] {
		case 0, 2:
			// emu.logT.Println("DECSCL: 8-bit controls")
			util.Logger.Debug("DECSCL: 8-bit controls")
		case 1:
			// emu.logT.Println("DECSCL: 7-bit controls")
			util.Logger.Debug("DECSCL: 7-bit controls")
		default:
			// emu.logU.Printf("DECSCL: C1 control transmission mode: %d", params[1])
			util.Logger.Warn("DECSCL: C1 control transmission mode",
				"unimplement", "DECSCL",
				"param", params[1])
		}
	}
}

// CSI Ps SP q
//
// Set cursor style (DECSCUSR), VT520.
//
//	Ps = 0  ⇒  blinking block.
//	Ps = 1  ⇒  blinking block (default).
//	Ps = 2  ⇒  steady block.
//	Ps = 3  ⇒  blinking underline.
//	Ps = 4  ⇒  steady underline.
//	Ps = 5  ⇒  blinking bar, xterm.
//	Ps = 6  ⇒  steady bar, xterm.
func hdl_csi_decscusr(emu *Emulator, arg int) {
	switch arg {
	case 0, 1:
		emu.cf.cursor.showStyle = CursorStyle_BlinkBlock
	case 2:
		emu.cf.cursor.showStyle = CursorStyle_SteadyBlock
	case 3:
		emu.cf.cursor.showStyle = CursorStyle_BlinkUnderline
	case 4:
		emu.cf.cursor.showStyle = CursorStyle_SteadyUnderline
	case 5:
		emu.cf.cursor.showStyle = CursorStyle_BlinkBar
	case 6:
		emu.cf.cursor.showStyle = CursorStyle_SteadyBar
	default:
		util.Logger.Warn("unexpected Ps parameter", "id", strHandlerID[CSI_DECSCUSR], "arg", arg)
	}
}

// Set conformance level based on Pl. treat VT200,VT300,VT500 as VT400
func sclCompatLevel(Pl int) (rcl CompatibilityLevel) {
	rcl = CompatLevel_Unused
	switch Pl {
	case 61:
		rcl = CompatLevel_VT100
	case 62, 63, 64, 65:
		rcl = CompatLevel_VT400
	}
	return
}

// CSI Ps ' }
//
//	Insert Ps Column(s) (default = 1) (DECIC), VT420 and up.
func hdl_csi_decic(emu *Emulator, num int) {
	if emu.isCursorInsideMargins() {
		num = min(num, emu.nColsEff-emu.posX)
		emu.insertCols(emu.posX, num)
	}
}

// CSI Ps ' ~
//
//	Delete Ps Column(s) (default = 1) (DECDC), VT420 and up.
func hdl_csi_decdc(emu *Emulator, num int) {
	if emu.isCursorInsideMargins() {
		num = min(num, emu.nColsEff-emu.posX)
		emu.deleteCols(emu.posX, num)
	}
}

// CSI Ps SP @
//
//	Shift left Ps columns(s) (default = 1) (SL), ECMA-48.
func hdl_csi_ecma48_SL(emu *Emulator, arg int) {
	arg = min(arg, emu.nColsEff-emu.hMargin)
	emu.deleteCols(emu.hMargin, arg)
}

// CSI Ps SP A
//
//	Shift right Ps columns(s) (default = 1) (SR), ECMA-48.
func hdl_csi_ecma48_SR(emu *Emulator, arg int) {
	arg = min(arg, emu.nColsEff-emu.hMargin)
	emu.insertCols(emu.hMargin, arg)
}

// CSI Ps E  Cursor Next Line Ps Times (default = 1) (CNL).
func hdl_csi_cnl(emu *Emulator, arg int) {
	hdl_csi_cud(emu, arg)
	hdl_c0_cr(emu)
}

// CSI Ps F  Cursor Preceding Line Ps Times (default = 1) (CPL).
func hdl_csi_cpl(emu *Emulator, arg int) {
	hdl_csi_cuu(emu, arg)
	hdl_c0_cr(emu)
}

// CSI > Pp ; Pv m
// CSI > Pp m
//
//	Set/reset key modifier options (XTMODKEYS), xterm.  Set or
//	reset resource-values used by xterm to decide whether to
//	construct escape sequences holding information about the
//	modifiers pressed with a given key.
//
//	The first parameter Pp identifies the resource to set/reset.
//	The second parameter Pv is the value to assign to the
//	resource.
//
//	If the second parameter is omitted, the resource is reset to
//	its initial value.  Values 3  and 5  are reserved for keypad-
//	keys and string-keys.
//
//	  Pp = 0  ⇒  modifyKeyboard.
//	  Pp = 1  ⇒  modifyCursorKeys.
//	  Pp = 2  ⇒  modifyFunctionKeys.
//	  Pp = 4  ⇒  modifyOtherKeys.
//
//	If no parameters are given, all resources are reset to their
//	initial values.
func hdl_csi_xtmodkeys(emu *Emulator, params []int) {
	switch len(params) {
	case 0:
		// Reset all options to initial values
		break
	case 1:
		params = append(params, 0)
		fallthrough
	case 2:
		switch params[0] { // TODO the meaning of second parameters.
		case 0:
			if params[1] != 0 {
				// emu.logU.Printf("XTMODKEYS: modifyKeyboard = %d\n", params[1])
				util.Logger.Warn("XTMODKEYS: modifyKeyboard",
					"unimplement", "XTMODKEYS",
					"params", params[1])
			}
		case 1:
			if params[1] != 2 {
				// emu.logU.Printf("XTMODKEYS: modifyCursorKeys = %d\n", params[1])
				util.Logger.Warn("XTMODKEYS: modifyCursorKeys",
					"unimplement", "XTMODKEYS",
					"params", params[1])
			}
		case 2:
			if params[1] != 2 {
				// emu.logU.Printf("XTMODKEYS: modifyFunctionKeys = %d\n", params[1])
				util.Logger.Warn("XTMODKEYS: modifyFunctionKeys",
					"unimplement", "XTMODKEYS",
					"params", params[1])
			}
		case 4:
			if params[1] <= 2 {
				emu.modifyOtherKeys = uint(params[1])
				util.Logger.Debug("XTMODKEYS: modifyOtherKeys set to",
					"modifyOtherKeys", emu.modifyOtherKeys)

			} else {
				// emu.logI.Printf("XTMODKEYS: illegal argument for modifyOtherKeys: %d\n", params[1])
				util.Logger.Warn("XTMODKEYS: illegal argument for modifyOtherKeys",
					"params", params[1])
			}
		}
	}
}

// CSI Ps ; Ps ; Ps t
//
//	Window manipulation (XTWINOPS), dtterm, extended by xterm.
//	These controls may be disabled using the allowWindowOps
//	resource.
//
//	xterm uses Extended Window Manager Hints (EWMH) to maximize
//	the window.  Some window managers have incomplete support for
//	EWMH.  For instance, fvwm, flwm and quartz-wm advertise
//	support for maximizing windows horizontally or vertically, but
//	in fact equate those to the maximize operation.
//	  Ps = 2 2 ; 0  ⇒  Save xterm icon and window title on stack.
//	  Ps = 2 2 ; 1  ⇒  Save xterm icon title on stack.
//	  Ps = 2 2 ; 2  ⇒  Save xterm window title on stack.
//	  Ps = 2 3 ; 0  ⇒  Restore xterm icon and window title from stack.
//	  Ps = 2 3 ; 1  ⇒  Restore xterm icon title from stack.
//	  Ps = 2 3 ; 2  ⇒  Restore xterm window title from stack.
func hdl_csi_xtwinops(emu *Emulator, params []int, sequence string) {
	if len(params) == 0 {
		util.Logger.Warn("unhandled operation", "seq", sequence,
			"params", params, "id", strHandlerID[CSI_XTWINOPS])
		return
	}
	switch params[0] {
	case 22:
		switch params[1] {
		case 0, 2:
			emu.saveWindowTitleOnStack()
		case 1:
			fallthrough
		default:
			util.Logger.Warn("unhandled operation", "seq", sequence,
				"params", params, "id", strHandlerID[CSI_XTWINOPS])
		}
	case 23:
		switch params[1] {
		case 0, 2:
			emu.restoreWindowTitleOnStack()
		case 1:
			fallthrough
		default:
			util.Logger.Warn("unhandled operation", "seq", sequence,
				"params", params, "id", strHandlerID[CSI_XTWINOPS])
		}
	default:
		util.Logger.Warn("unhandled operation", "seq", sequence,
			"params", params, "id", strHandlerID[CSI_XTWINOPS])
	}
}

// CSI Ps b  Repeat the preceding graphic character Ps times (REP).
func hdl_csi_rep(emu *Emulator, arg int, chs []rune) {
	for k := 0; k < arg; k++ {
		hdl_graphemes(emu, chs...)
	}
}
