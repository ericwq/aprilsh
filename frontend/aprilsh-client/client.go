// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	_ "github.com/ericwq/terminfo/base"
	"github.com/rivo/uniseg"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	_PACKAGE_STRING     = "aprilsh"
	_COMMAND_NAME       = "aprilsh-client"
	_APRILSH_KEY        = "APRISH_KEY"
	_PREDICTION_DISPLAY = "APRISH_PREDICTION_DISPLAY"
)

var (
	logW         *log.Logger
	logI         *log.Logger
	BuildVersion = "0.1.0" // ready for ldflags

	usage = `Usage:
  ` + _COMMAND_NAME + ` [--version] [--help] [--colors]
  ` + _COMMAND_NAME + ` [--verbose] [--port PORT]  User@Server
Options:
  -h, --help     print this message
  -v, --version  print version information
  -c, --colors   print the number of colors of terminal
  -p, --port     server port (default 60000)
      --verbose  verbose output mode
`
	predictionValues = []string{"always", "never", "adaptive", "experimental"}
)

func init() {
	initLog()
}

func initLog() {
	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	logI = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func printVersion() {
	fmt.Printf("%s (%s) [build %s]\n\n", _COMMAND_NAME, _PACKAGE_STRING, BuildVersion)
	fmt.Printf(`Copyright (c) 2022~2023 wangqi ericwq057[AT]qq[dot]com
This is free software: you are free to change and redistribute it.
There is NO WARRANTY, to the extent permitted by law.

reborn mosh with aprilsh
`)
}

func printColors() {
	value, ok := os.LookupEnv("TERM")
	if ok {
		if value != "" {
			// ti, err := terminfo.LookupTerminfo(value)
			ti, err := terminal.LookupTerminfo(value)
			if err == nil {
				fmt.Printf("%s %d\n", value, ti.Colors)
			} else {
				// ti, _, err = dynamic.LoadTerminfo(value)
				// if err == nil {
				// 	fmt.Printf("%s %d (dynamic)\n", value, ti.Colors)
				// } else {
				fmt.Printf("Dynamic load terminfo failed. %s Install infocmp (ncurses package) first.\n", err)
				// }
			}
		} else {
			fmt.Println("The TERM is empty string.")
		}
	} else {
		fmt.Println("The TERM doesn't exist.")
	}
}

func printUsage(hint, usage string) {
	if hint != "" {
		fmt.Printf("Hints: %s\n%s", hint, usage)
	} else {
		fmt.Printf("%s", usage)
	}
}

func parseFlags(progname string, args []string) (config *Config, output string, err error) {
	// https://eli.thegreenplace.net/2020/testing-flag-parsing-in-go-programs/
	flagSet := flag.NewFlagSet(progname, flag.ContinueOnError)
	var buf bytes.Buffer
	flagSet.SetOutput(&buf)

	var conf Config

	flagSet.IntVar(&conf.verbose, "verbose", 0, "verbose output mode")

	flagSet.BoolVar(&conf.version, "version", false, "print version information")
	flagSet.BoolVar(&conf.version, "v", false, "print version information")

	flagSet.IntVar(&conf.port, "port", 60000, "server port")
	flagSet.IntVar(&conf.port, "p", 60000, "server port")

	flagSet.BoolVar(&conf.colors, "color", false, "terminal number of colors")
	flagSet.BoolVar(&conf.colors, "c", false, "terminal number of colors")

	err = flagSet.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}

	// get the non-flag command-line arguments.
	conf.target = flagSet.Args()
	return &conf, buf.String(), nil
}

type Config struct {
	version     bool
	target      []string // raw parameter
	host        string
	user        string
	port        int
	verbose     int
	colors      bool
	key         string
	predictMode string
}

func (c *Config) buildConfig() (string, bool) {
	// just need version info
	if c.version {
		return "", true
	}

	// just need terminal number of colors
	if c.colors {
		return "", true
	}

	if len(c.target) == 0 {
		return "target parameter (User@Server) is mandatory.", false
	}

	if len(c.target) != 1 {
		return "only one target parameter (User@Server) is allowed.", false
	}

	// validate server parameter
	idx := strings.Index(c.target[0], "@")
	if idx == -1 || idx < 1 || idx == len(c.target[0])-1 {
		return "target parameter should be in the form of User@Server", false
	}
	c.host = c.target[0][idx+1:]
	c.user = c.target[0][:idx]

	// Read key from environment
	c.key = os.Getenv(_APRILSH_KEY)
	if c.key == "" {
		return _APRILSH_KEY + " environment variable not found.", false
	}
	os.Unsetenv(_APRILSH_KEY)

	// Read prediction preference
	foundInScope := false
	c.predictMode = strings.ToLower(os.Getenv(_PREDICTION_DISPLAY))
	for i := range predictionValues {
		if predictionValues[i] == c.predictMode {
			foundInScope = true
		}
	}
	if !foundInScope {
		return _PREDICTION_DISPLAY + " unknown prediction mode.", false
	}

	return "", true
}

func main() {
	conf, _, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		printUsage("", usage)
		return
	} else if err != nil {
		printUsage(err.Error(), usage)
		return
	} else if hint, ok := conf.buildConfig(); !ok {
		printUsage(hint, usage)
		return
	}

	if conf.version {
		printVersion()
		return
	}

	if conf.colors {
		printColors()
		return
	}

	util.SetNativeLocale()
}

type STMClient struct {
	ip   string
	port int
	key  string

	escapeKey        int
	escapePassKey    int
	escapePassKey2   int
	escapeRequireslf bool
	escapeKeyHelp    string

	savedTermios *term.State // store the original termios, used for shutdown.
	rawTermios   *term.State // set IUTF8 flag, set raw terminal in raw mode, used for resume.
	windowSize   *unix.Winsize

	localFramebuffer *terminal.Emulator
	newState         *terminal.Emulator
	overlays         *frontend.OverlayManager
	network          *network.Transport[*statesync.UserStream, *statesync.Complete]
	display          *terminal.Display

	connectingNotification string
	repaintRequested       bool
	lfEntered              bool
	quitSequenceStarted    bool
	cleanShutdown          bool
	verbose                int
}

func newSTMClient(ip string, port int, key string, predictMode string, verbose int) *STMClient {
	sc := STMClient{}

	sc.ip = ip
	sc.port = port
	sc.key = key
	sc.escapeKey = 0x1E
	sc.escapePassKey = '^'
	sc.escapePassKey2 = '^'
	sc.escapeRequireslf = false
	sc.escapeKeyHelp = "?"
	sc.overlays = frontend.NewOverlayManager()

	var err error
	sc.display, err = terminal.NewDisplay(true)
	if err != nil {
		return nil
	}

	sc.repaintRequested = false
	sc.lfEntered = false
	sc.quitSequenceStarted = false
	sc.cleanShutdown = false
	sc.verbose = verbose

	switch predictMode {
	case predictionValues[0]: // always
		sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Always)
	case predictionValues[1]: // never
		sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Never)
	case predictionValues[2]: // adaptive
		sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Adaptive)
	case predictionValues[3]: // experimental
		sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Experimental)
	default:
		return nil
	}

	return &sc
}

func (sc *STMClient) mainInit() error {
	// get initial window size
	col, row, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	// local state
	savedLines := row * 5
	sc.localFramebuffer = terminal.NewEmulator3(col, row, savedLines)
	sc.newState = terminal.NewEmulator3(1, 1, savedLines)

	// initialize screen
	init := sc.display.NewFrame(false, sc.localFramebuffer, sc.localFramebuffer)
	os.Stdout.WriteString(init)

	// open network
	blank := &statesync.UserStream{}
	terminal, err := statesync.NewComplete(col, row, savedLines)
	sc.network = network.NewTransportClient(blank, terminal, sc.key, sc.ip, fmt.Sprintf("%d", sc.port))

	// minimal delay on outgoing keystrokes
	sc.network.SetSendDelay(1)

	// tell server the size of the terminal
	sc.network.GetCurrentState().PushBackResize(col, row)

	// be noisy as necessary
	sc.network.SetVerbose(uint(sc.verbose))

	return nil
}

func (sc *STMClient) processNetworkInput() {
	// sc.network.Recv()

	//  Now give hints to the overlays
	rs := sc.network.GetLatestRemoteState()
	sc.overlays.GetNotificationEngine().ServerHeard(rs.GetTimestamp())
	sc.overlays.GetNotificationEngine().ServerAcked(sc.network.GetSentStateAckedTimestamp())

	sc.overlays.GetPredictionEngine().SetLocalFrameAcked(sc.network.GetSentStateAcked())
	sc.overlays.GetPredictionEngine().SetSendInterval(sc.network.SentInterval())
	state := sc.network.GetLatestRemoteState()
	lateAcked := state.GetState().GetEchoAck()
	sc.overlays.GetPredictionEngine().SetLocalFrameLateAcked(lateAcked)
}

func (sc *STMClient) processUserInput(buf string) bool {
	if !sc.network.ShutdownInProgress() {
		sc.overlays.GetPredictionEngine().SetLocalFrameSent(sc.network.GetSentStateLast())

		var input []rune
		graphemes := uniseg.NewGraphemes(buf)
		for graphemes.Next() {
			input = graphemes.Runes()
			theByte := input[0] // the first byte

			sc.overlays.GetPredictionEngine().NewUserInput(sc.localFramebuffer, string(input))

			if sc.quitSequenceStarted {
				if theByte == '.' { // Quit sequence is Ctrl-^ .
					if sc.network.HasRemoteAddr() && !sc.network.ShutdownInProgress() {
						sc.overlays.GetNotificationEngine().SetNotificationString(
							"Exiting on user request...", true, true)
						sc.network.StartShutdown()
						return true
					} else {
						return false
					}
				} else if theByte == 0x1A { // Suspend sequence is escape_key Ctrl-Z
					// Restore terminal and terminal-driver state
					os.Stdout.WriteString(sc.display.Close())

					term.Restore(int(os.Stdin.Fd()), sc.savedTermios)

					fmt.Printf("\n\033[37;44m[%s is suspended.]\033[m\n", _PACKAGE_STRING)

					// fflush(NULL)
					//
					/* actually suspend */
					// kill(0, SIGSTOP);

					sc.resume()
				} else if theByte == rune(sc.escapePassKey) || theByte == rune(sc.escapePassKey2) {
					// Emulation sequence to type escape_key is escape_key +
					// escape_pass_key (that is escape key without Ctrl)
					sc.network.GetCurrentState().PushBack([]rune{rune(sc.escapeKey)})
				} else {
					// Escape key followed by anything other than . and ^ gets sent literally
					sc.network.GetCurrentState().PushBack([]rune{rune(sc.escapeKey), theByte})
				}

				sc.quitSequenceStarted = false

				if sc.overlays.GetNotificationEngine().GetNotificationString() == sc.escapeKeyHelp {
					sc.overlays.GetNotificationEngine().SetNotificationString("", false, true)
				}

				continue
			}

			sc.quitSequenceStarted = sc.escapeKey > 0 && theByte == rune(sc.escapeKey) &&
				(sc.lfEntered || !sc.escapeRequireslf)

			if sc.quitSequenceStarted {
				sc.lfEntered = false
				sc.overlays.GetNotificationEngine().SetNotificationString(sc.escapeKeyHelp, true, false)
				continue
			}

			sc.lfEntered = theByte == 0x0A || theByte == 0x0D // LineFeed, Ctrl-J, '\n' or CarriageReturn, Ctrl-M, '\r'

			if theByte == 0x0C { // Ctrl-L
				sc.repaintRequested = true
			}

			sc.network.GetCurrentState().PushBack(input)
		}

	}
	return true
}

func (sc *STMClient) processResize() bool {
	// get new size
	col, row, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return false
	}

	// newSize := terminal.Resize{Width: col, Height: row}
	// tell remote emulator
	if !sc.network.ShutdownInProgress() {
		sc.network.GetCurrentState().PushBackResize(col, row)
	}
	// note remote emulator will probably reply with its own Resize to adjust our state

	// tell prediction engine
	sc.overlays.GetPredictionEngine().Reset()
	return true
}

func (sc *STMClient) outputNewFrame() {
	// clean shutdown even when not initialized
	if sc.network == nil {
		return
	}

	// fetch target state
	state := sc.network.GetLatestRemoteState()
	sc.newState = state.GetState().GetEmulator()

	// apply local overlays
	sc.overlays.Apply(sc.newState)

	// calculate minimal difference from where we are
	diff := sc.display.NewFrame(!sc.repaintRequested, sc.localFramebuffer, sc.newState)
	os.Stdout.WriteString(diff)

	sc.repaintRequested = false
	sc.localFramebuffer = sc.newState
}

func (sc *STMClient) stillConnecting() bool {
	// Initially, network == nil
	return sc.network != nil && sc.network.GetRemoteStateNum() == 0
}

func (sc *STMClient) resume() {
	// Restore termios state
	if err := term.Restore(int(os.Stdin.Fd()), sc.rawTermios); err != nil {
		os.Exit(1)
	}

	// Put terminal in application-cursor-key mode
	os.Stdout.WriteString(sc.display.Open())

	// Flag that outer terminal state is unknown
	sc.repaintRequested = true
}

func (sc *STMClient) init() error {
	if !util.IsUtf8Locale() {
		nativeType := util.GetCtype()
		nativeCharset := util.LocaleCharset()

		fmt.Printf("%s needs a UTF-8 native locale to run.\n\n", _COMMAND_NAME)
		fmt.Printf("Unfortunately, the client's environment (%s) specifies\nthe character set %q.\n\n",
			nativeType, nativeCharset)
		return errors.New(_COMMAND_NAME + " requires UTF-8 environment.")
	}

	var err error
	// Verify terminal configuration
	sc.savedTermios, _ = term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	// set IUTF8 if available
	// term package doesn't allow us to access termios, we use util package to do that.
	if err = util.SetIUTF8(int(os.Stdin.Fd())); err != nil {
		return err
	}

	// Put terminal driver in raw mode
	// https://learnku.com/go/t/23460/bit-operation-of-go
	// &^ is used to clean the specified bit
	_, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	// save raw + IUTF8 termios to rawTermios
	sc.rawTermios, err = term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	// Put terminal in application-cursor-key mode
	os.Stdout.WriteString(sc.display.Open())

	// Add our name to window title
	prefix := os.Getenv("APRILSH_TITLE_PREFIX")
	if prefix != "" {
		sc.overlays.SetTitlePrefix(prefix)
	}

	// Set terminal escape key.
	escapeKeyEnv := os.Getenv("APRILSH_ESCAPE_KEY")
	if escapeKeyEnv != "" {
		if len(escapeKeyEnv) == 1 {
			sc.escapeKey = int(escapeKeyEnv[0])
			if sc.escapeKey > 0 && sc.escapeKey < 128 {
				if sc.escapeKey < 32 {
					// If escape is ctrl-something, pass it with repeating the key without ctrl.
					sc.escapePassKey = sc.escapeKey + '@'
				} else {
					// If escape is something else, pass it with repeating the key itself.
					sc.escapePassKey = sc.escapeKey
				}
				if sc.escapePassKey >= 'A' && sc.escapePassKey <= 'Z' {
					// If escape pass is an upper case character, define optional version as lower case of the same.
					sc.escapePassKey2 = sc.escapePassKey + 'a' - 'A'
				} else {
					sc.escapePassKey2 = sc.escapePassKey
				}
			} else {
				sc.escapeKey = 0x1E
				sc.escapePassKey = '^'
				sc.escapePassKey2 = '^'
			}
		} else if len(escapeKeyEnv) == 0 {
			sc.escapeKey = -1
		} else {
			sc.escapeKey = 0x1E
			sc.escapePassKey = '^'
			sc.escapePassKey2 = '^'
		}
	} else {
		sc.escapeKey = 0x1E
		sc.escapePassKey = '^'
		sc.escapePassKey2 = '^'
	}

	// There are so many better ways to shoot oneself into leg than
	// setting escape key to Ctrl-C, Ctrl-D, NewLine, Ctrl-L or CarriageReturn
	// that we just won't allow that.

	if sc.escapeKey == 0x03 || sc.escapeKey == 0x04 || sc.escapeKey == 0x0A ||
		sc.escapeKey == 0x0C || sc.escapeKey == 0x0D {
		sc.escapeKey = 0x1E
		sc.escapePassKey = '^'
		sc.escapePassKey2 = '^'
	}

	// Adjust escape help differently if escape is a control character.
	if sc.escapeKey > 0 {
		var b strings.Builder
		escapeKeyName := ""
		escapePassName := fmt.Sprintf("\"%c\"", sc.escapePassKey)
		if sc.escapeKey < 32 {
			escapeKeyName = fmt.Sprintf("Ctrl-%c", sc.escapePassKey)
			sc.escapeRequireslf = false
		} else {
			escapeKeyName = fmt.Sprintf("\"%c\"", sc.escapePassKey)
			sc.escapeRequireslf = true
		}

		sc.escapeKeyHelp = fmt.Sprintf("Commands: Ctrl-Z suspends, \".\" quits, " + escapePassName +
			" gives literal " + escapeKeyName)
		sc.overlays.GetNotificationEngine().SetEscapeKeyString(b.String())
	}
	sc.connectingNotification = fmt.Sprintf("Nothing received from server on UDP port %d.", sc.port)

	return nil
}

func (sc *STMClient) shutdown() error {
	// Restore screen state
	sc.overlays.GetNotificationEngine().SetNotificationString("", false, true)
	sc.overlays.GetNotificationEngine().ServerHeard(time.Now().UnixMilli())
	sc.overlays.SetTitlePrefix("")

	sc.outputNewFrame()

	// Restore terminal and terminal-driver state
	os.Stdout.WriteString(sc.display.Close())

	err := term.Restore(int(os.Stdin.Fd()), sc.savedTermios)
	if err != nil {
		return err
	}

	if sc.stillConnecting() {
		fmt.Fprintf(os.Stderr, "%s did not make a successful connection to %s:%d.\n",
			_PACKAGE_STRING, sc.ip, sc.port)
		fmt.Fprintf(os.Stderr, "Please verify that UDP port %d is not firewalled and can reach the server.\n\n",
			sc.port)
		fmt.Fprintf(os.Stderr, "By default, %s uses a UDP port between 60000 and 61000. The -p option\n%s",
			_PACKAGE_STRING, "selects a specific UDP port number.)")
	} else if sc.network != nil {
		if !sc.cleanShutdown {
			fmt.Fprintf(os.Stderr, "\n\n%s did not shut down cleanly. Please note that the\n%s",
				_PACKAGE_STRING, "aprilsh-server process may still be running on the server.\n")
		}
	}
	return nil
}

func (sc *STMClient) main() error {
	// initialize signal handling and structures
	sc.mainInit()

	// 	/* Drop unnecessary privileges */
	// #ifdef HAVE_PLEDGE
	// 	/* OpenBSD pledge() syscall */
	// 	if (pledge("stdio inet tty", NULL)) {
	// 		perror("pledge() failed");
	// 		exit(1);
	// 	}
	// #endif

	var networkChan chan frontend.Message
	var fileChan chan frontend.Message
	networkChan = make(chan frontend.Message, 1)
	fileChan = make(chan frontend.Message, 1)

	eg := errgroup.Group{}
	// read from network
	eg.Go(func() error {
		// if we have 2 client ip, 5 ms for each client
		frontend.ReadFromNetwork(5, networkChan, sc.network)
		return nil
	})
	// read from pty master file
	eg.Go(func() error {
		frontend.ReadFromFile(10, fileChan, os.Stdin)
		return nil
	})

	// intercept signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH, syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGHUP, syscall.SIGPIPE, syscall.SIGCONT)
	shutdownChan := make(chan bool)
	eg.Go(func() error {
		for {
			select {
			case s := <-sigChan:
				clientSignalHandler(s)
			case <-shutdownChan:
				return nil
			}
		}
	})

mainLoop:
	for {
		sc.outputNewFrame()

		select {
		case networkMsg := <-networkChan: // got data from socket
			if networkMsg.Err != nil { // error handling
				logW.Printf("#readFromSocket receive error:%s\n", networkMsg.Err)
				continue mainLoop
			}
			sc.processNetworkInput()
		case fileMsg := <-fileChan: // got data from file
			if fileMsg.Err != nil {
				logW.Println("#readFromMaster read error: ", fileMsg.Err)
				sc.network.StartShutdown()
			}

			// input from the user needs to be fed to the network
			if !sc.processUserInput(fileMsg.Data) {
				if !sc.network.HasRemoteAddr() {
					break
				} else if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNotificationString("Exiting...", true, true)
					sc.network.StartShutdown()
				}
			}
		default:
		}

		for i := range gotSignal {
			switch gotSignal[i].Load() {
			case int32(syscall.SIGWINCH):
				gotSignal[i].Store(0)
				// resize
				if !sc.processResize() {
					return nil
				}
			case int32(syscall.SIGCONT):
				gotSignal[i].Store(0)
				sc.resume()
			case int32(syscall.SIGTERM), int32(syscall.SIGINT), int32(syscall.SIGHUP), int32(syscall.SIGPIPE):
				gotSignal[i].Store(0)
				// shutdown signal
				if !sc.network.HasRemoteAddr() {
					break
				} else if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNotificationString(
						"Signal received, shutting down...", true, true)
					sc.network.StartShutdown()
				}
			}
		}

		// quit if our shutdown has been acknowledged
		if sc.network.ShutdownInProgress() && sc.network.ShutdownAcknowledged() {
			sc.cleanShutdown = true
			break
		}

		// quit after shutdown acknowledgement timeout
		if sc.network.ShutdownInProgress() && sc.network.ShutdownAckTimedout() {
			break
		}

		// quit if we received and acknowledged a shutdown request
		if sc.network.CounterpartyShutdownAckSent() {
			sc.cleanShutdown = true
			break
		}

		// write diagnostic message if can't reach server
		now := time.Now().UnixMilli()
		remoteState := sc.network.GetLatestRemoteState()
		if sc.stillConnecting() && !sc.network.ShutdownInProgress() && now-remoteState.GetTimestamp() > 250 {
			if now-remoteState.GetTimestamp() > 15000 {
				if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNotificationString(
						"Timed out waiting for server...", true, true)
					sc.network.StartShutdown()
				}
			} else {
				sc.overlays.GetNotificationEngine().SetNotificationString(
					sc.connectingNotification, false, true)
			}
		} else if sc.network.GetRemoteStateNum() != 0 &&
			sc.overlays.GetNotificationEngine().GetNotificationString() == sc.connectingNotification {
			sc.overlays.GetNotificationEngine().SetNotificationString("", false, true)
		}

		sc.network.Tick()
	}

	// shutdown the goroutine
	shutdownChan <- true
	fileChan <- frontend.Message{Err: nil, Data: "shutdown"}
	networkChan <- frontend.Message{Err: nil, Data: "shutdown"}
	eg.Wait()

	return nil
}

var gotSignal [frontend.MAX_SIGNAL_NUMBER]atomic.Int32

func clientSignalHandler(signal os.Signal) {
	// We assume writes to these ints are atomic, though we also try to mask out
	// concurrent signal handlers.

	switch signal {
	case syscall.SIGWINCH:
		gotSignal[syscall.SIGWINCH].Store(int32(syscall.SIGWINCH))
	case syscall.SIGTERM:
		gotSignal[syscall.SIGTERM].Store(int32(syscall.SIGTERM))
	case syscall.SIGINT:
		gotSignal[syscall.SIGINT].Store(int32(syscall.SIGINT))
	case syscall.SIGHUP:
		gotSignal[syscall.SIGHUP].Store(int32(syscall.SIGHUP))
	case syscall.SIGPIPE:
		gotSignal[syscall.SIGPIPE].Store(int32(syscall.SIGPIPE))
	case syscall.SIGCONT:
		gotSignal[syscall.SIGCONT].Store(int32(syscall.SIGCONT))
	default:
	}
}
