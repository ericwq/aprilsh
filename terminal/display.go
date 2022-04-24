package terminal

type Display struct {
	hasECH   bool
	hasBCE   bool
	hasTitle bool
	smcup    string
	rmcup    string
}

// https://github.com/nsf/termbox-go the first replace ncurses?
// https://github.com/gdamore/tcell the successor of termbox-go
// https://cs.opensource.google/go/x/term/+/master:README.md 
// apk add mandoc man-pages
// apk add ncurses-terminfo
// apk add ncurses-terminfo-base 
// apk add ncurses-doc
// https://ishuah.com/2021/03/10/build-a-terminal-emulator-in-100-lines-of-go/
