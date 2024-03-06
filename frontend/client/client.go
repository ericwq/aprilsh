// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"log/slog"

	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	"github.com/rivo/uniseg"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	xknownhosts "golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	_APRILSH_KEY          = "APRISH_KEY"
	_PREDICTION_DISPLAY   = "APRISH_PREDICTION_DISPLAY"
	_PREDICTION_OVERWRITE = "APRISH_PREDICTION_OVERWRITE"
	// _VERBOSE_LOG_TMPFILE  = 2
)

var (
	usage = `Usage:
  ` + frontend.CommandClientName + ` [--version] [--help] [--colors]
  ` + frontend.CommandClientName + ` [-v[v]] [--port PORT] [-i identity_file] destination
Options:
---------------------------------------------------------------------------------------------------
  -h,  --help        print this message
  -c,  --colors      print the number of colors of terminal
       --version     print version information
---------------------------------------------------------------------------------------------------
  -p,  --port        apshd server port (default 8100)
  destination        in the form of user@host[:port], here the port is ssh server port (default 22)
  -i                 ssh client identity (private key) (default $HOME/.ssh/id_rsa)
  -v,  --verbose     verbose log output (debug level, default info level)
  -vv                verbose log output (trace level)
---------------------------------------------------------------------------------------------------
`
	predictionValues   = []string{"always", "never", "adaptive", "experimental"}
	defaultSSHClientID = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	signals            frontend.Signals
)

func printVersion() {
	fmt.Printf("%s package : %s client, %s\n",
		frontend.AprilshPackageName, frontend.AprilshPackageName, frontend.CommandClientName)
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

func parseFlags(progname string, args []string) (config *Config, output string, err error) {
	// https://eli.thegreenplace.net/2020/testing-flag-parsing-in-go-programs/
	flagSet := flag.NewFlagSet(progname, flag.ContinueOnError)
	var buf bytes.Buffer
	flagSet.SetOutput(&buf)

	var conf Config

	var v1, v2 bool
	flagSet.BoolVar(&v1, "v", false, "verbose log output debug level")
	flagSet.BoolVar(&v1, "verbose", false, "verbose log output debug levle")
	flagSet.BoolVar(&v2, "vv", false, "verbose log output trace level")

	flagSet.BoolVar(&conf.version, "version", false, "print version information")

	flagSet.BoolVar(&conf.addSource, "source", false, "add source info to log")

	flagSet.IntVar(&conf.port, "port", frontend.DefaultPort, frontend.CommandServerName+" server port")
	flagSet.IntVar(&conf.port, "p", frontend.DefaultPort, frontend.CommandServerName+" server port")

	flagSet.BoolVar(&conf.colors, "color", false, "terminal colors number")
	flagSet.BoolVar(&conf.colors, "c", false, "terminal colors number")

	flagSet.StringVar(&conf.sshClientID, "i", defaultSSHClientID, "ssh client identity file")

	err = flagSet.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}

	// get the non-flag command-line arguments.
	conf.destination = flagSet.Args()

	// detremine verbose level
	if v1 {
		conf.verbose = util.DebugLevel
	} else if v2 {
		conf.verbose = util.TraceLevel
	}
	return &conf, buf.String(), nil
}

type Config struct {
	version          bool
	destination      []string // raw parameter
	host             string   // target host/server
	user             string   // target user
	port             int      // target port
	verbose          int
	colors           bool
	key              string
	predictMode      string
	predictOverwrite string
	sshClientID      string // ssh client identity, for SSH public key authentication
	sshPort          string // ssh port, default 22
	addSource        bool   // add source file to log
}

var errNoResponse = errors.New("no response, please make sure the server is running.")

type hostkeyChangeError struct {
	hostname string
}

func (e *hostkeyChangeError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return "REMOTE HOST IDENTIFICATION HAS CHANGED for host '" +
		e.hostname + "' ! This may indicate a MITM attack."
}

func (e *hostkeyChangeError) Hostname() string { return e.hostname }

type responseError struct {
	Err error
	Msg string
}

func (e *responseError) Error() string { return e.Msg + ", " + e.Err.Error() }

// utilize ssh to fetch the key from remote server and start a server.
// return empty string if success, otherwise return error info.
//
// For alpine, ssh is provided by openssh package, nc and echo is provided by busybox.
// % ssh ide@localhost  "echo 'open aprilsh:' | nc localhost 6000 -u -w 1"
//
// ssh-keygen -t ed25519
// ssh-copy-id -i ~/.ssh/id_ed25519.pub root@localhost
// ssh-copy-id -i ~/.ssh/id_ed25519.pub ide@localhost
// ssh-add ~/.ssh/id_ed25519
func (c *Config) fetchKey() error {
	// var hostKey ssh.PublicKey
	var auth []ssh.AuthMethod
	var am ssh.AuthMethod

	auth = make([]ssh.AuthMethod, 0)
	if am = sshAgent(); am != nil {
		auth = append(auth, am) // ssh agent, for ssh key-based authentication
	}
	if am = publicKeyFile(c.sshClientID); am != nil {
		// fmt.Printf("auth.length=%d, defaultSSHClientID=%s, sshClientID=%s\n",
		// 	len(auth), defaultSSHClientID, c.sshClientID)
		if c.sshClientID != defaultSSHClientID {
			if len(auth) == 0 {
				// ssh client identification is the only available method,
				auth = append(auth, am)
				// fmt.Printf("auth=[pub]\n")
			} else {
				// ssh client identification is the first method,
				// agent is the second mehtod
				a2 := make([]ssh.AuthMethod, 0)
				a2 = append(a2, am)
				a2 = append(a2, auth...)
				auth = a2
				// fmt.Printf("auth=[pub,agent]\n")
			}
		} else {
			// agent is the first method,
			// ssh client identification is the second available method
			auth = append(auth, am)
			// fmt.Printf("auth=[agent, pub]\n")
		}
	}
	if len(auth) == 0 {
		// get password if we don't have any authenticate method
		pwd, err := getPassword("password", os.Stdin)
		if err != nil {
			return err
		}

		// password authentication is the last resort
		if am = ssh.Password(pwd); am != nil {
			auth = append(auth, am)
		}
	}

	sshHost := net.JoinHostPort(c.host, c.sshPort)
	khPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	kh, err := knownhosts.New(khPath)
	if err != nil {
		return err
	}

	// https://github.com/skeema/knownhosts
	// https://github.com/golang/go/issues/29286
	// Create a custom permissive hostkey callback which still errors on hosts
	// with changed keys, but allows unknown hosts and adds them to known_hosts
	cb := ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) (err error) {
		err = kh(hostname, remote, key)
		if knownhosts.IsHostKeyChanged(err) {
			return &hostkeyChangeError{hostname: hostname}
		} else if knownhosts.IsHostUnknown(err) {

			hint := "The authenticity of host '%s (%s)' can't be established.\n" +
				"%s key fingerprint is %s.\n" +
				"This key is not known by any other names\n" +
				"Are you sure you want to continue connecting (yes/no/[fingerprint])?"
			fmt.Printf(hint, hostname, remote, strings.ToUpper(key.Type()), ssh.FingerprintSHA256(key))

			var answer string
			fmt.Scanln(&answer)
			switch answer {
			case "yes", "y":
				f, ferr := os.OpenFile(khPath, os.O_APPEND|os.O_WRONLY, 0600)
				if ferr == nil {
					defer f.Close()
					ferr = knownhosts.WriteKnownHost(f, hostname, remote, key)
				}
				if ferr == nil {
					fmt.Printf("Warning: Permanently added '%s' (%s) to the list of known hosts.\n",
						hostname, strings.ToUpper(key.Type()))
					err = nil // permit previously-unknown hosts (warning: may be insecure)
				} else {
					fmt.Printf("Failed to add host %s to known_hosts: %v\n", hostname, ferr)
					err = ferr
				}
			case "no", "n":
				fallthrough
			default:
				fmt.Println("Host key verification failed.")
			}
		}
		return
	})

	// https://betterprogramming.pub/a-simple-cross-platform-ssh-client-in-100-lines-of-go-280644d8beea
	// https://blog.ralch.com/articles/golang-ssh-connection/
	// https://www.ssh.com/blog/what-are-ssh-host-keys
	clientConfig := &ssh.ClientConfig{
		User:              c.user,
		Auth:              auth,
		HostKeyCallback:   cb,
		HostKeyAlgorithms: kh.HostKeyAlgorithms(sshHost),
		Timeout:           time.Duration(3) * time.Second,
	}

	// TODO understand ssh login session, is that possible to replace the sshd depdends?
	client, err := ssh.Dial("tcp", sshHost, clientConfig)
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
	cmd := fmt.Sprintf("/usr/bin/apshd -b -t %s -destination %s", os.Getenv("TERM"), c.destination[0])
	// fmt.Printf("cmd=%s\n", cmd)

	if b, err = session.Output(cmd); err != nil {
		return err
	}
	out := strings.TrimSpace(string(b))

	// open aprilsh:60001,31kR3xgfmNxhDESXQ8VIQw==
	body := strings.Split(out, ":")
	if len(body) != 2 || body[0] != frontend.AprilshMsgOpen[:12] { // [:12]remove the last ':'
		return errors.New(fmt.Sprintf("response: %s", out))
	}

	// parse port and key
	content := strings.Split(body[1], ",")
	if len(content) == 2 {
		p, e := strconv.Atoi(content[0])
		if e != nil {
			return errors.New("can't get port")
		}
		// incase port mapping for docker
		c.port += (p - frontend.DefaultPort)

		if encrypt.NewBase64Key2(content[1]) != nil {
			c.key = content[1]
		} else {
			return errors.New("can't get key")
		}
		// fmt.Printf("fetchKey port=%d, key=%s\n", c.port, c.key)
	} else {
		return errors.New(fmt.Sprintf("response: %s", body[1]))
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

	if len(c.destination) == 0 {
		return "destination (user@host[:port]) is mandatory.", false
	}

	if len(c.destination) != 1 {
		return "only one destination (user@host[:port]) is allowed.", false
	}

	// check destination
	first := strings.Split(c.destination[0], "@")
	if len(first) == 2 {
		c.user = first[0]
		second := strings.Split(first[1], ":")
		c.host = second[0]
		if len(second) == 1 {
			c.sshPort = "22" // default ssh port
		} else {
			if _, err := strconv.Atoi(second[1]); err != nil {
				return "please check destination, illegal port number.", false
			}
			c.sshPort = second[1]
		}
	} else {
		return "destination should be in the form of user@host[:port]", false
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

// read password from specified input source
func getPassword(prompt string, in *os.File) (string, error) {
	fmt.Printf("%s: ", prompt)
	bytepw, err := term.ReadPassword(int(in.Fd()))
	defer fmt.Printf("\n")

	if err != nil {
		return "", err
	}

	return string(bytepw), nil
}

func sshAgent() ssh.AuthMethod {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		fmt.Printf("Failed to connect ssh agent. %s\n", err)
		return nil
	}
	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
}

func publicKeyFile(file string) ssh.AuthMethod {
	key, err := os.ReadFile(file)
	if err != nil {
		// fmt.Printf("Unable to read private key: %s\n", err)
		return nil
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if strings.Contains(err.Error(), "private key is passphrase protected") {
			passphrase, err2 := getPassword("passphrase", os.Stdin)
			if err2 != nil {
				fmt.Printf("Failed to get passphrase. %s\n", err2)
				return nil // read passphrase error
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
			if err != nil {
				fmt.Printf("Failed to parse private key. %s\n", err)
				return nil
			}
		} else {
			fmt.Printf("Unable to parse private key: %s\n", err)
			return nil
		}
	}
	return ssh.PublicKeys(signer) // Use the PublicKeys method for remote authentication.
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
	util.Logger.Debug("client window size", "col", col, "row", row)

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
	os.Stdout.WriteString(init)
	util.Logger.Debug("mainInit", "init", init)

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
		util.Logger.Warn("ProcessPayload", "error", err)
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

	util.Logger.Debug("processUserInput", "buf", buf)
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
					util.Logger.Error("restore terminal failed", "error", err)
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
		util.Logger.Debug("outputNewFrame", "diff", diff)
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
	util.Logger.Info("open terminal", "seq", sc.display.Open())

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
	util.Logger.Info("close terminal", "seq", sc.display.Close())

	if err := term.Restore(int(os.Stdin.Fd()), sc.savedTermios); err != nil {
		util.Logger.Warn("restore terminal failed", "error", err)
		return err
	}

	if sc.stillConnecting() {
		fmt.Printf("%s did not make a successful connection to '%s:%d'.\n",
			frontend.CommandClientName, sc.ip, sc.port)
		fmt.Printf("Please verify that UDP port is not firewalled and %s can reach the server.\n",
			frontend.CommandClientName)
		fmt.Printf("By default, %s uses UDP port begin with %d, The -p option specifies base %s port.\n",
			frontend.CommandClientName, frontend.DefaultPort+1, frontend.CommandServerName)
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
	// 			util.Log.Debug("got signal","signal", s)
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
		util.Logger.Debug("mainLoop", "point", 100,
			"network.WaitTime", w0, "overlays.WaitTime", w1, "timeout", waitTime)
		select {
		case <-timer.C:
			// util.Log.Debug("mainLoop", "overlays", sc.overlays.WaitTime(),
			// 	"network", sc.network.WaitTime(), "waitTime", waitTime)
		case networkMsg := <-networkChan:

			// got data from server
			if networkMsg.Err != nil {
				// quit asap for refused connection
				if errors.Is(networkMsg.Err, syscall.ECONNREFUSED) {
					break mainLoop
				}
				// if read from server failed, retry after 0.2 second
				util.Logger.Warn("receive from network", "error", networkMsg.Err)
				if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNetworkError(networkMsg.Err.Error())
				}
				// TODO handle "use of closed network connection" error?
				time.Sleep(time.Duration(200) * time.Millisecond)
				continue mainLoop
			}
			// util.Log.Info("got from network", "data", networkMsg.Data)
			sc.processNetworkInput(networkMsg.Data)

		case fileMsg := <-fileChan:

			// input from the user needs to be fed to the network
			if fileMsg.Err != nil || !sc.processUserInput(fileMsg.Data) {

				// if read from local pts terminal failed, quit
				if fileMsg.Err != nil {
					util.Logger.Warn("read from file", "error", fileMsg.Err)
				}
				if !sc.network.HasRemoteAddr() {
					break mainLoop
				} else if !sc.network.ShutdownInProgress() {
					sc.overlays.GetNotificationEngine().SetNotificationString("Exiting...", true, true)
					sc.network.StartShutdown()
				}
			}
		case s := <-sigChan:
			util.Logger.Debug("got signal", "signal", s)
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
				util.Logger.Debug("start shutting down.")
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
					util.Logger.Warn("No connection within x seconds", "seconds", frontend.TimeoutIfNoConnect/1000)
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

		// util.Log.Warn("mainLoop", "before", "tick")
		err := sc.network.Tick()
		if err != nil {
			util.Logger.Warn("tick send failed", "error", err)
			sc.overlays.GetNotificationEngine().SetNetworkError(err.Error())
			// if errors.Is(err, syscall.ECONNREFUSED) {
			sc.network.StartShutdown()
			util.Logger.Debug("start shutting down.")
		} else {
			sc.overlays.GetNotificationEngine().ClearNetworkError()
		}

		// if connected and no response over TimeoutIfNoResp
		if sc.network.GetRemoteStateNum() != 0 && sinceLastResponse > frontend.TimeoutIfNoResp {
			// if no awaken
			if !sc.network.Awaken(now) {
				util.Logger.Warn("No server response over x seconds", "seconds", frontend.TimeoutIfNoResp)
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
	if errors.Is(err, flag.ErrHelp) {
		frontend.PrintUsage("", usage)
		return
	} else if err != nil {
		frontend.PrintUsage(err.Error())
		return
	} else if hint, ok := conf.buildConfig(); !ok {
		frontend.PrintUsage(hint)
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

	var logWriter io.Writer
	logWriter = os.Stderr

	// https://rderik.com/blog/identify-if-output-goes-to-the-terminal-or-is-being-redirected-in-golang/
	//
	// if stderr outputs to terminal, we redirect it to /dev/null.
	f2, _ := os.Stderr.Stat()
	if (f2.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		os.Stderr = os.NewFile(uintptr(syscall.Stderr), os.DevNull)
		logWriter = io.Discard
	}

	// setup client log file
	switch conf.verbose {
	case util.DebugLevel:
		util.Logger.CreateLogger(logWriter, conf.addSource, slog.LevelDebug)
	case util.TraceLevel:
		util.Logger.CreateLogger(logWriter, conf.addSource, util.LevelTrace)
	default:
		util.Logger.CreateLogger(logWriter, conf.addSource, slog.LevelInfo)
	}
	// util.Log.AddSource(conf.addSource)
	// util.Log.SetOutput(os.Stderr)

	// https://earthly.dev/blog/golang-errors/
	// https://gosamples.dev/check-error-type/
	// https://www.digitalocean.com/community/tutorials/how-to-add-extra-information-to-errors-in-go
	//
	// ssh login to remote server and fetch the seesion key
	if err = conf.fetchKey(); err != nil {
		var dnsError *net.DNSError
		var opError *net.OpError
		var keyError *xknownhosts.KeyError
		var exitError *ssh.ExitError
		var hostkeyChangeError *hostkeyChangeError

		if errors.As(err, &dnsError) {
			frontend.PrintUsage(fmt.Sprintf("No such host: %q", dnsError.Name))
		} else if errors.As(err, &opError) && opError.Op == "dial" {
			frontend.PrintUsage(fmt.Sprintf("Failed to connect to: %s", opError.Addr))
		} else if strings.Contains(err.Error(), "unable to authenticate") {
			// the error returned by ssh.NewClientConn() doen't naming error,
			// we have to check the error message directly.

			// enable 'PubkeyAuthentication yes' line in sshd_config
			frontend.PrintUsage(fmt.Sprintf("Failed to authenticate user %q", conf.user))
		} else if errors.As(err, &keyError) {
			// } else if strings.Contains(err.Error(), "key is unknown") {
			// we already handle it
		} else if errors.Is(err, errNoResponse) {
			frontend.PrintUsage(err.Error())
		} else if errors.As(err, &exitError) && exitError.Waitmsg.ExitStatus() == 127 {
			frontend.PrintUsage("Plase check aprilsh is installed on server.")
		} else if errors.As(err, &hostkeyChangeError) {
			frontend.PrintUsage(hostkeyChangeError.Error())
		} else {
			// printUsage(fmt.Sprintf("%#v", err))
			frontend.PrintUsage(err.Error())
		}
		return
	}

	// start client
	util.SetNativeLocale()
	client := newSTMClient(conf)
	if err := client.init(); err != nil {
		fmt.Printf("%s init error:%s\n", frontend.CommandClientName, err)
		return
	}
	client.main()
	client.shutdown()
}
