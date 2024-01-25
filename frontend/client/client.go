// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	"github.com/rivo/uniseg"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
	"log/slog"
)

const (
	_APRILSH_KEY          = "APRISH_KEY"
	_PREDICTION_DISPLAY   = "APRISH_PREDICTION_DISPLAY"
	_PREDICTION_OVERWRITE = "APRISH_PREDICTION_OVERWRITE"
	_VERBOSE_LOG_TMPFILE  = 2
)

var (
	usage = `Usage:
  ` + frontend.CommandClientName + ` [--version] [--help] [--colors]
  ` + frontend.CommandClientName + ` [--verbose] [--port PORT] [--pwd PASSWORD] user@server.domain
Options:
  -h, --help     print this message
  -v, --version  print version information
  -c, --colors   print the number of colors of terminal
  -p, --port     server port (default 60000)
      --verbose  verbose output mode
      --pwd      ssh password
`
	predictionValues = []string{"always", "never", "adaptive", "experimental"}
	signals          frontend.Signals
)

func printVersion() {
	fmt.Printf("%s\t\t: %s client, %s\n", frontend.AprilshPackageName,
		frontend.AprilshPackageName, frontend.CommandClientName)
	frontend.PrintVersion()
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
				fmt.Printf("Dynamic load terminfo failed. %s Install infocmp (ncurses package) first.\n", err)
			}
		} else {
			fmt.Println("The TERM is empty string.")
		}
	} else {
		fmt.Println("The TERM doesn't exist.")
	}
}

func printUsage(hint string, usage ...string) {
	if hint != "" {
		var header string
		if len(usage) != 0 {
			header = "Hints: "
		}
		fmt.Printf("%s%s\n", header, hint)
	}
	if len(usage) > 0 {
		fmt.Printf("%s", usage[0])
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

	flagSet.StringVar(&conf.pwd, "pwd", "", "ssh password")

	err = flagSet.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}

	// get the non-flag command-line arguments.
	conf.target = flagSet.Args()
	return &conf, buf.String(), nil
}

type Config struct {
	version          bool
	target           []string // raw parameter
	host             string   // target host/server
	user             string   // target user
	port             int      // target port
	verbose          int
	colors           bool
	key              string
	predictMode      string
	predictOverwrite string
	pwd              string // user password for ssh login
}

// read password from specified input source
func (c *Config) getPassword(in *os.File) (string, error) {
	fmt.Print("Password: ")
	bytepw, err := term.ReadPassword(int(in.Fd()))
	defer fmt.Printf("\n")

	if err != nil {
		return "", err
	}

	return string(bytepw), nil
}

// utilize ssh to fetch the key from remote server and start a server.
// return empty string if success, otherwise return error info.
//
// For alpine, ssh is provided by openssh package, nc and echo is provided by busybox.
// % ssh ide@localhost  "echo 'open aprilsh:' | nc localhost 6000 -u -w 1"
func (c *Config) fetchKey() error {

	// https://betterprogramming.pub/a-simple-cross-platform-ssh-client-in-100-lines-of-go-280644d8beea
	cc := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.pwd),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(1) * time.Second,
	}
	client, err := ssh.Dial("tcp", c.host+":22", cc)
	if err != nil {
		return err
	}
	defer client.Close()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b []byte
	// cmd := fmt.Sprintf("echo '%s' | nc localhost %d -u -w 1", _ASH_OPEN, c.port)
	cmd := fmt.Sprintf("~/.local/bin/apshd -b -t %s -target %s", os.Getenv("TERM"), c.target[0])
	// util.Log.With("cmd", cmd).Debug("execute command")

	if b, err = session.Output(cmd); err != nil {
		return err
	}
	out := strings.TrimSpace(string(b))

	/*
		args := []string{
			fmt.Sprintf("%s@%s", c.user, c.host),
			fmt.Sprintf("\"echo 'open aprilsh:' | nc localhost %d -u -w 1\"", c.port),
		}

		out, err := exec.Command("ssh", args...).Output()
		if err != nil {
			return err.Error()
		}
	*/

	// open aprilsh:60001,31kR3xgfmNxhDESXQ8VIQw==
	// util.Log.With("out", out).Debug("fetchKey")
	body := strings.Split(out, ":")
	if len(body) != 2 || !strings.HasPrefix(frontend.AprilshMsgOpen, body[0]) {
		resp := fmt.Sprintf("no response, please make sure the server is running.")
		return errors.New(resp)
	}

	// parse port and key
	idx := strings.Index(body[1], ",")
	if idx > 0 && idx+1 < len(body[1]) {
		p, e := strconv.Atoi(body[1][:idx])
		if e != nil {
			return errors.New("can't get port")
		}
		c.port = p

		idx++
		if encrypt.NewBase64Key2(body[1][idx:]) != nil {
			c.key = body[1][idx:]
		} else {
			return errors.New("can't get key")
		}
		// fmt.Printf("fetchKey port=%d, key=%s\n", c.port, c.key)
	} else {
		return errors.New(fmt.Sprintf("malform response : %s", body[1]))
	}

	return nil
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

	// check target is in the form of usr@host parameter
	idx := strings.Index(c.target[0], "@")
	if idx > 0 && idx < len(c.target[0])-1 {
		c.host = c.target[0][idx+1:]
		c.user = c.target[0][:idx]
	} else {
		return "target parameter should be in the form of User@Server", false

	}

	// Read key from environment
	// c.key = os.Getenv(_APRILSH_KEY)
	// if c.key == "" {
	// 	return _APRILSH_KEY + " environment variable not found.", false
	// }
	// os.Unsetenv(_APRILSH_KEY)

	// Read prediction preference, predictMode can be empty
	foundInScope := false
	c.predictMode = strings.ToLower(os.Getenv(_PREDICTION_DISPLAY))
	if c.predictMode != "" {
		// if predictMode is not empty string, it's must be one of predictionValues
		for i := range predictionValues {
			if predictionValues[i] == c.predictMode {
				foundInScope = true
			}
		}
		if !foundInScope {
			return _PREDICTION_DISPLAY + " unknown prediction mode.", false
		}
	}

	// Read prediction insertion preference. can be ""
	c.predictOverwrite = strings.ToLower(os.Getenv(_PREDICTION_OVERWRITE))

	return "", true
}

func main() {
	// cpuf, err := os.Create("cpu.profile")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// pprof.StartCPUProfile(cpuf)
	// defer pprof.StopCPUProfile()

	// For security, make sure we don't dump core
	encrypt.DisableDumpingCore()

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

	// setup client log file
	if conf.verbose > 0 {
		util.Log.SetLevel(slog.LevelDebug)
	} else {
		util.Log.SetLevel(slog.LevelInfo)
	}
	util.Log.SetOutput(os.Stderr)
	if conf.verbose == _VERBOSE_LOG_TMPFILE { //TODO consider remove this.
		logf, err := util.Log.CreateLogFile(frontend.CommandClientName)
		if err != nil {
			fmt.Printf("can't create log file %s.\n", logf.Name())
			return
		}
		util.Log.SetOutput(logf)
	}

	// get pwd
	if conf.pwd == "" {
		conf.pwd, err = conf.getPassword(os.Stdin)
		if err != nil {
			printUsage(err.Error())
			return
		}
	}

	// login to remote server and fetch the key
	if err = conf.fetchKey(); err != nil {
		printUsage(err.Error())
		return
	}

	// start client
	// the Stdin, Stderr, Stdout are all set to pts/N
	util.SetNativeLocale()
	client := newSTMClient(conf)
	if err := client.init(); err != nil {
		fmt.Printf("%s init error:%s\n", frontend.CommandClientName, err)
		return
	}
	client.main()
	client.shutdown()
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

func newSTMClient(config *Config) *STMClient {
	sc := STMClient{}

	sc.ip = config.host
	sc.port = config.port
	sc.key = config.key
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
	sc.verbose = config.verbose

	if config.predictMode != "" {
		switch config.predictMode {
		case predictionValues[0]: // always
			sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Always)
		case predictionValues[1]: // never
			sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Never)
		case predictionValues[2]: // adaptive
			sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Adaptive)
		case predictionValues[3]: // experimental
			sc.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Experimental)
		}
	}

	if config.predictOverwrite == "yes" {
		sc.overlays.GetPredictionEngine().SetPredictOverwrite(true)
	}
	return &sc
}

func (sc *STMClient) mainInit() error {
	// get initial window size
	col, row, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	util.Log.With("col", col).With("row", row).Debug("client window size")

	// local state
	savedLines := terminal.SaveLinesRowsOption
	sc.localFramebuffer = terminal.NewEmulator3(col, row, savedLines)
	sc.newState = terminal.NewEmulator3(col, row, savedLines)

	// initialize screen
	// init := sc.display.NewFrame(true, sc.localFramebuffer, sc.localFramebuffer)
	// CSI ? 1049l Use Normal Screen Buffer and restore cursor as in DECRC, xterm.
	// CSI ? 1l		Normal Cursor Keys (DECCKM)
	// CSI ? 1004l Disable FocusIn/FocusOut
	init := "\x1B[?1049l\x1B[?1l\x1B[?1004l"
	util.Log.With("init", init).Debug("mainInit")
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

func (sc *STMClient) processNetworkInput(s string) {
	// sc.network.Recv()
	if err := sc.network.ProcessPayload(s); err != nil {
		util.Log.With("error", err).Warn("ProcessPayload")
	}

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
	if sc.network.ShutdownInProgress() {
		return true
	}
	sc.overlays.GetPredictionEngine().SetLocalFrameSent(sc.network.GetSentStateLast())

	// Don't predict for bulk data.
	paste := len(buf) > 100
	if paste {
		sc.overlays.GetPredictionEngine().Reset()
	}

	util.Log.With("buf", buf).Debug("processUserInput")
	var input []rune
	graphemes := uniseg.NewGraphemes(buf)
	for graphemes.Next() {
		input = graphemes.Runes()
		theByte := input[0] // the first byte

		if !paste {
			sc.overlays.GetPredictionEngine().NewUserInput(sc.localFramebuffer, input)
		}

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

				if err := term.Restore(int(os.Stdin.Fd()), sc.savedTermios); err != nil {
					util.Log.With("error", err).Error("restore terminal failed")
					return false
				}

				fmt.Printf("\n\033[37;44m[%s is suspended.]\033[m\n", frontend.CommandClientName)

				// fflush(NULL)
				//
				/* actually suspend */
				// kill(0, SIGSTOP);
				// TODO check SIGSTOP

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
	// util.Log.SetLevel(slog.LevelInfo)
	// diff := sc.display.NewFrame(!sc.repaintRequested, sc.localFramebuffer, sc.newState)
	diff := state.GetState().GetDiff()
	// util.Log.SetLevel(slog.LevelDebug)
	os.Stdout.WriteString(diff)
	if diff != "" {
		util.Log.With("diff", diff).Debug("outputNewFrame")
	}

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

		fmt.Printf("%s needs a UTF-8 native locale to run.\n\n", frontend.CommandClientName)
		fmt.Printf("Unfortunately, the client's environment (%s) specifies\nthe character set %q.\n\n",
			nativeType, nativeCharset)
		return errors.New(frontend.CommandClientName + " requires UTF-8 environment.")
	}

	var err error
	// Verify terminal configuration
	sc.savedTermios, err = term.GetState(int(os.Stdin.Fd()))
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
	util.Log.With("seq", sc.display.Open()).Debug("open terminal")

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
	util.Log.With("seq", sc.display.Close()).Debug("close terminal")

	if err := term.Restore(int(os.Stdin.Fd()), sc.savedTermios); err != nil {
		util.Log.With("error", err).Warn("restore terminal failed")
		return err
	}

	if sc.stillConnecting() {
		fmt.Printf("%s did not make a successful connection to %s:%d.\n",
			frontend.CommandClientName, sc.ip, sc.port)
		fmt.Printf("Please verify that UDP port %d is not firewalled and can reach the server.\n\n",
			sc.port)
		fmt.Printf("By default, %s uses a UDP port between 60000 and 61000. The -p option\n%s\n",
			frontend.CommandClientName, "selects a initial UDP port number.")
	} else if sc.network != nil {
		if !sc.cleanShutdown {
			fmt.Printf("\n%s did not shut down cleanly.\n", frontend.CommandClientName)
			fmt.Printf("Please verify that UDP port %d is not firewalled and can reach the server.\n",
				sc.port)
		} else {
			fmt.Printf("Connection to %s:%d closed.\n", sc.ip, sc.port)
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
	fileDownChan := make(chan any, 1)
	networkDownChan := make(chan any, 1)

	eg := errgroup.Group{}
	// read from network
	eg.Go(func() error {
		frontend.ReadFromNetwork(1, networkChan, networkDownChan, sc.network.GetConnection())
		return nil
	})

	// read from pty master file
	eg.Go(func() error {
		frontend.ReadFromFile(10, fileChan, fileDownChan, os.Stdin)
		return nil
	})

	// intercept signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH, syscall.SIGTERM, syscall.SIGINT,
		syscall.SIGHUP, syscall.SIGPIPE, syscall.SIGCONT)
	// shutdownChan := make(chan bool)
	// eg.Go(func() error {
	// 	for {
	// 		select {
	// 		case s := <-sigChan:
	// 			util.Log.With("signal", s).Debug("got signal")
	// 			signals.Handler(s)
	// 		case <-shutdownChan:
	// 			return nil
	// 		}
	// 	}
	// })

mainLoop:
	for {
		sc.outputNewFrame()

		w0 := sc.network.WaitTime()
		w1 := sc.overlays.WaitTime()
		waitTime := min(w0, w1)
		// waitTime := terminal.Min(sc.network.WaitTime(), sc.overlays.WaitTime())

		// Handle startup "Connecting..." message
		if sc.stillConnecting() {
			waitTime = min(250, waitTime)
		}

		timer := time.NewTimer(time.Duration(waitTime) * time.Millisecond)
		util.Log.With("point", 100).With("network.WaitTime", w0).
			With("overlays.WaitTime", w1).With("timeout", waitTime).Debug("mainLoop")
		select {
		case <-timer.C:
			// util.Log.With("overlays", sc.overlays.WaitTime()).
			// 	With("network", sc.network.WaitTime()).
			// 	With("waitTime", waitTime).
			// 	Debug("mainLoop")
		case networkMsg := <-networkChan:

			// got data from server
			if networkMsg.Err != nil {

				// if read from server failed, retry after 0.2 second
				util.Log.With("error", networkMsg.Err).Warn("receive from network")
				if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNetworkError(networkMsg.Err.Error())
				}
				// TODO handle "use of closed network connection" error?
				time.Sleep(time.Duration(200) * time.Millisecond)
				continue mainLoop
			}
			// util.Log.With("data", networkMsg.Data).Info("got from network")
			sc.processNetworkInput(networkMsg.Data)

		case fileMsg := <-fileChan:

			// input from the user needs to be fed to the network
			if fileMsg.Err != nil || !sc.processUserInput(fileMsg.Data) {

				// if read from local pts terminal failed, quit
				if fileMsg.Err != nil {
					util.Log.With("error", fileMsg.Err).Warn("read from file")
				}
				if !sc.network.HasRemoteAddr() {
					break mainLoop
				} else if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNotificationString("Exiting...", true, true)
					sc.network.StartShutdown()
				}
			}
		case s := <-sigChan:
			util.Log.With("signal", s).Debug("got signal")
			signals.Handler(s)
		}

		if signals.GotSignal(syscall.SIGWINCH) {
			// resize
			if !sc.processResize() {
				return nil
			}
		}

		if signals.GotSignal(syscall.SIGCONT) {
			sc.resume()
		}

		if signals.GotSignal(syscall.SIGTERM) || signals.GotSignal(syscall.SIGINT) ||
			signals.GotSignal(syscall.SIGHUP) || signals.GotSignal(syscall.SIGPIPE) {
			// shutdown signal
			if !sc.network.HasRemoteAddr() {
				break
			} else if !sc.network.ShutdownInProgress() {
				util.Log.Debug("start shutting down.")
				sc.overlays.GetNotificationEngine().SetNotificationString(
					"Signal received, shutting down...", true, true)
				sc.network.StartShutdown()
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
		sinceLastResponse := now - remoteState.GetTimestamp()
		if sc.stillConnecting() && !sc.network.ShutdownInProgress() && sinceLastResponse > 250 {
			if sinceLastResponse > frontend.TimeoutIfNoConnect {
				if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNotificationString(
						"Timed out waiting for server...", true, true)
					// sc.network.StartShutdown()
					util.Log.With("seconds", frontend.TimeoutIfNoConnect/1000).Warn("No connection within x seconds")
					break
				}
			} else {
				sc.overlays.GetNotificationEngine().SetNotificationString(
					sc.connectingNotification, false, true)
			}
		} else if sc.network.GetRemoteStateNum() != 0 &&
			sc.overlays.GetNotificationEngine().GetNotificationString() == sc.connectingNotification {
			sc.overlays.GetNotificationEngine().SetNotificationString("", false, true)
		}

		// util.Log.With("before", "tick").Warn("mainLoop")
		err := sc.network.Tick()
		if err != nil {
			util.Log.With("error", err).Warn("tick send failed")
			sc.overlays.GetNotificationEngine().SetNetworkError(err.Error())
			// if errors.Is(err, syscall.ECONNREFUSED) {
			sc.network.StartShutdown()
			util.Log.Debug("start shutting down.")
		} else {
			sc.overlays.GetNotificationEngine().ClearNetworkError()
		}

		// if connected and no response over TimeoutIfNoResp
		if sc.network.GetRemoteStateNum() != 0 && sinceLastResponse > frontend.TimeoutIfNoResp {
			// if no awaken
			if !sc.network.Awaken(now) {
				util.Log.With("seconds", frontend.TimeoutIfNoResp).Warn("No server response over x seconds")
				break
			}
		}
	}

	// stop signal and network
	signal.Stop(sigChan)
	sc.network.Close()

	// shutdown the goroutines: file reader and network reader
	select {
	case fileDownChan <- "done":
	default:
	}
	select {
	case networkDownChan <- "done":
	default:
	}

	// consume last message to release reader if possible
	select {
	case <-fileChan:
	default:
	}
	select {
	case <-networkChan:
	default:
	}
	eg.Wait()

	return nil
}
