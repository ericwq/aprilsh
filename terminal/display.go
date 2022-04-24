package terminal

/* questions

 do we need to package the the terminfo DB into application?
    yes, mosh-server depends on ncurses-terminfo-base and  ncurses-libs
 how to read terminfo DB? through ncurses lib or directly?
 how to operate terminal? through direct escape sequence or through terminfo DB?
 how to replace the following functions? setupterm(), tigetnum(), tigetstr(), tigetflag()

 */
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
