package terminal

import "fmt"

type UserByte struct {
	Chs []rune
}

type Resize struct {
	Width  int
	Height int
}

type ActOn interface {
	Handle(emu Emulator)
}

func (u UserByte) Handle(emu Emulator) {
	// TODO it seams that Parser can't handle Application mode?
	ret := emu.user.parse(u, emu.cursorKeyMode)
	emu.writePty(ret)
}

func (r Resize) Handle(emu Emulator) {
	emu.resize(r.Width, r.Height)
}

const (
	USER_INPUT_GROUND = iota
	USER_INPUT_ESC
	USER_INPUT_SS3
)

// the default state is USER_INPUT_GROUND = 0
type UserInput struct {
	state int
}

// The user will always be in application mode. If client is not in
// application mode, convert user's cursor control function to an
// ANSI cursor control sequence */
func (u *UserInput) parse(x UserByte, cursorKeyMode CursorKeyMode) string {
	// We need to look ahead one byte in the SS3 state to see if
	// the next byte will be A, B, C, or D (cursor control keys).

	if len(x.Chs) > 1 {
		return ""
	}
	var r rune = x.Chs[0]

	switch u.state {
	case USER_INPUT_GROUND:
		if r == '\x1B' {
			u.state = USER_INPUT_ESC
		}
		return string(r)

	case USER_INPUT_ESC:
		if r == 'O' { // ESC O = 7-bit SS3
			u.state = USER_INPUT_SS3
			return ""
		} else {
			u.state = USER_INPUT_GROUND
			return string(r)
		}

		// The cursor keys transmit the following escape sequences depending on the
		// mode specified via the DECCKM escape sequence.
		//
		//                   Key            Normal     Application
		//                   -------------+----------+-------------
		//                   Cursor Up    | CSI A    | SS3 A
		//                   Cursor Down  | CSI B    | SS3 B
		//                   Cursor Right | CSI C    | SS3 C
		//                   Cursor Left  | CSI D    | SS3 D
		//                   -------------+----------+-------------
	case USER_INPUT_SS3:
		u.state = USER_INPUT_GROUND
		if cursorKeyMode == CursorKeyMode_ANSI && 'A' <= r && r <= 'D' {
			return fmt.Sprintf("[%c", r) // CSI
		} else {
			return fmt.Sprintf("O%c", r) // SS3
		}
	}

	// This doesn't handle the 8-bit SS3 C1 control, which would be
	// two octets in UTF-8. Fortunately nobody seems to send this. */
	return ""
}
