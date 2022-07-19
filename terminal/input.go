package terminal

import "fmt"

const (
	USER_INPUT_GROUND = iota
	USER_INPUT_ESC
	USER_INPUT_SS3
)

// the default state is USER_INPUT_GROUND = 0
type UserInput struct {
	state int
}

type UserByte struct {
	c rune
}

type Resize struct {
	width  int
	height int
}

func (u UserByte) handle(emu Emulator) {
	ret := emu.user.parse(u, emu.cf.DS.ApplicationModeCursorKeys)
	emu.dispatcher.terminalToHost.WriteString(ret)
}

func (r Resize) handle(emu Emulator) {
	emu.resize(r.width, r.height)
}

// The user will always be in application mode. If client is not in
// application mode, convert user's cursor control function to an
// ANSI cursor control sequence */
func (u *UserInput) parse(act UserByte, applicationModeCursorKeys bool) string {
	// We need to look ahead one byte in the SS3 state to see if
	// the next byte will be A, B, C, or D (cursor control keys).

	switch u.state {
	case USER_INPUT_GROUND:
		if act.c == '\x1B' {
			u.state = USER_INPUT_ESC
		}
		return string(act.c)

	case USER_INPUT_ESC:
		if act.c == 'O' { // ESC O = 7-bit SS3
			u.state = USER_INPUT_SS3
			return ""
		} else {
			u.state = USER_INPUT_GROUND
			return string(act.c)
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
		if !applicationModeCursorKeys && 'A' <= act.c && act.c <= 'D' {
			return fmt.Sprintf("[%c", act.c) // CSI
		} else {
			return fmt.Sprintf("O%c", act.c) // SS3
		}
	}

	// This doesn't handle the 8-bit SS3 C1 control, which would be
	// two octets in UTF-8. Fortunately nobody seems to send this. */
	return ""
}
