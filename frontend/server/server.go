// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"os/signal"
	"os/user"
	"reflect"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	utmp "github.com/ericwq/goutmp"
	"golang.org/x/exp/slog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

var (
	BuildVersion    = "0.1.0" // ready for ldflags
	userCurrentTest = false
	getShellTest    = false
	buildConfigTest = false
)

var (
	utmpSupport   bool
	syslogSupport bool
	signals       frontend.Signals
)

const (
	_PACKAGE_STRING = "aprilsh"
	_COMMAND_NAME   = "aprilsh-server"
	_PATH_BSHELL    = "/bin/sh"

	_ASH_OPEN  = "open aprilsh:"
	_ASH_CLOSE = "close aprilsh:"

	_VERBOSE_OPEN_PTS    = 99  // test purpose
	_VERBOSE_START_SHELL = 100 // test purpose
	_VERBOSE_LOG_SYSLOG  = 514 // log to syslog
)

func init() {
	utmpSupport = utmp.HasUtmpSupport()
}

func printVersion() {
	fmt.Printf("%s (%s) [build %s]\n\n", _COMMAND_NAME, _PACKAGE_STRING, BuildVersion)
	fmt.Println("Copyright (c) 2022~2023 wangqi ericwq057@qq.com")
	fmt.Println("This is free software: you are free to change and redistribute it.")
	fmt.Printf("There is NO WARRANTY, to the extent permitted by law.\n\n")
	fmt.Println("reborn mosh with aprilsh")
}

// [-s] [-v] [-i LOCALADDR] [-p PORT[:PORT2]] [-c COLORS] [-l NAME=VALUE] [-- COMMAND...]
var usage = `Usage:
  ` + _COMMAND_NAME + ` [-v] [-h] [--auto N]
  ` + _COMMAND_NAME + ` [-b]
  ` + _COMMAND_NAME + ` [-s] [--verbose V] [-i LOCALADDR] [-p PORT[:PORT2]] [-l NAME=VALUE] [-t TERM] [-- command...]
Options:
  -h, --help     print this message
  -v, --version  print version information
  -b, --begin    begin a client connection
  -a, --auto     auto stop the server after N seconds
  -s, --server   listen with SSH ip
  -i, --ip       listen with this ip/host
  -p, --port     listen port range (default port 60000)
  -l, --locale   key-value pairs (such as LANG=UTF-8, you can have multiple -l options)
  -t, --term     client TERM (such as xterm-256color, or alacritty or xterm-kitty)
      --verbose  verbose output (such as 1)
     -- command  shell command and options (note the space before command)
`

func printUsage(hint, usage string) {
	if hint != "" {
		fmt.Printf("Hints: %s\n%s", hint, usage)
	} else {
		fmt.Printf("%s", usage)
	}
}

// Print the motd from a given file, if available
func printMotd(w io.Writer, filename string) bool {
	// fmt.Printf("#printMotd print %q\n", filename)
	// https://zetcode.com/golang/readfile/

	motd, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer motd.Close()

	// read line by line, print each line to writer
	scanner := bufio.NewScanner(motd)
	for scanner.Scan() {
		fmt.Fprintf(w, "%s\n", scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return false
	}

	return true
}

func chdirHomedir(home string) bool {
	var err error
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return false
		}
	}

	err = os.Chdir(home)
	if err != nil {
		return false
	}
	os.Setenv("PWD", home)

	return true
}

// get current user home directory
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return home
}

func motdHushed() bool {
	// must be in home directory already
	_, err := os.Lstat(".hushlogin")
	if err != nil {
		return false
	}

	return true
}

// extract server ip addresss from SSH_CONNECTION
func getSSHip() (string, bool) {
	env := os.Getenv("SSH_CONNECTION")
	if len(env) == 0 { // Older sshds don't set this
		return fmt.Sprintf("Warning: SSH_CONNECTION not found; binding to any interface."), false
	}

	// SSH_CONNECTION' Identifies the client and server ends of the connection.
	// The variable contains four space-separated values: client IP address,
	// client port number, server IP address, and server port number.
	//
	// ipv4 sample: SSH_CONNECTION=172.17.0.1 58774 172.17.0.2 22
	sshConn := strings.Split(env, " ")
	if len(sshConn) != 4 {
		return fmt.Sprintf("Warning: Could not parse SSH_CONNECTION; binding to any interface."), false
	}

	localInterfaceIP := strings.ToLower(sshConn[2])
	prefixIPv6 := "::ffff:"

	// fmt.Printf("#getSSHip localInterfaceIP=%q, prefixIPv6=%q\n", localInterfaceIP, prefixIPv6)
	if len(localInterfaceIP) > len(prefixIPv6) && strings.HasPrefix(localInterfaceIP, prefixIPv6) {
		return localInterfaceIP[len(prefixIPv6):], true
	}

	return localInterfaceIP, true
}

// https://www.antoniojgutierrez.com/posts/2021-05-14-short-and-long-options-in-go-flags-pkg/
type localeFlag map[string]string

func (lv *localeFlag) String() string {
	return fmt.Sprint(*lv)
}

func (lv *localeFlag) Set(value string) error {
	kv := strings.Split(value, "=")
	if len(kv) != 2 {
		return errors.New("malform locale parameter: " + value)
	}

	(*lv)[kv[0]] = kv[1]
	return nil
}

func (lv *localeFlag) IsBoolFlag() bool {
	return false
}

// parseFlags parses the command-line arguments provided to the program.
// Typically os.Args[0] is provided as 'progname' and os.args[1:] as 'args'.
// Returns the Config in case parsing succeeded, or an error. In any case, the
// output of the flag.Parse is returned in output.
// A special case is usage requests with -h or -help: then the error
// flag.ErrHelp is returned and output will contain the usage message.
func parseFlags(progname string, args []string) (config *Config, output string, err error) {
	// https://eli.thegreenplace.net/2020/testing-flag-parsing-in-go-programs/
	flagSet := flag.NewFlagSet(progname, flag.ContinueOnError)
	var buf bytes.Buffer
	flagSet.SetOutput(&buf)

	var conf Config
	conf.locales = make(localeFlag)
	conf.commandArgv = []string{}

	flagSet.IntVar(&conf.verbose, "verbose", 0, "verbose output")

	flagSet.IntVar(&conf.autoStop, "auto", 0, "auto stop after N seconds")
	flagSet.IntVar(&conf.autoStop, "a", 0, "auto stop after N seconds")

	flagSet.BoolVar(&conf.version, "version", false, "print version information")
	flagSet.BoolVar(&conf.version, "v", false, "print version information")

	flagSet.BoolVar(&conf.begin, "begin", false, "begin a client connection")
	flagSet.BoolVar(&conf.begin, "b", false, "begin a client connection")

	flagSet.BoolVar(&conf.server, "server", false, "listen with SSH ip")
	flagSet.BoolVar(&conf.server, "s", false, "listen with SSH ip")

	flagSet.StringVar(&conf.desiredIP, "ip", "", "listen ip")
	flagSet.StringVar(&conf.desiredIP, "i", "", "listen ip")

	flagSet.StringVar(&conf.desiredPort, "port", "60000", "listen port range")
	flagSet.StringVar(&conf.desiredPort, "p", "60000", "listen port range")

	flagSet.StringVar(&conf.term, "term", "", "client TERM")
	flagSet.StringVar(&conf.term, "t", "", "client TERM")

	flagSet.Var(&conf.locales, "locale", "locale list, key=value pair")
	flagSet.Var(&conf.locales, "l", "locale list, key=value pair")

	err = flagSet.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}

	// check the format of desiredPort
	// _, err = strconv.Atoi(conf.desiredPort)
	// if err != nil {
	// 	return nil, buf.String(), err
	// }

	// get the non-flag command-line arguments.
	conf.commandArgv = flagSet.Args()
	return &conf, buf.String(), nil
}

type Config struct {
	version     bool       // print version information
	server      bool       // use SSH ip
	verbose     int        // verbose output
	desiredIP   string     // server ip/host
	desiredPort string     // server port
	locales     localeFlag // localse environment variables
	term        string     // client TERM
	autoStop    int        // auto stop after N seconds
	begin       bool       // begin a client connection

	commandPath string   // shell command path (absolute path)
	commandArgv []string // the positional (non-flag) command-line arguments.
	withMotd    bool

	// the serve func
	serve func(*os.File, *statesync.Complete, chan bool,
		*network.Transport[*statesync.Complete, *statesync.UserStream], int64, int64) error
}

// build the config instance and check the utf-8 locale. return error if the terminal
// can't support utf-8 locale.
func (conf *Config) buildConfig() (string, bool) {
	// just need version info
	if conf.version {
		return "", true
	}

	if conf.server {
		if sshIP, ok := getSSHip(); ok {
			conf.desiredIP = sshIP
		} else {
			msg := sshIP
			return msg, false
		}
	}

	if len(conf.desiredPort) > 0 {
		// Sanity-check arguments

		// fmt.Printf("#main desiredPort=%s\n", conf.desiredPort)
		_, _, ok := network.ParsePortRange(conf.desiredPort)
		if !ok {
			return fmt.Sprintf("Bad UDP port (%s)", conf.desiredPort), false
		}
	}

	conf.commandPath = ""
	conf.withMotd = false
	conf.serve = serve

	// Get shell
	if len(conf.commandArgv) == 0 {
		shell := os.Getenv("SHELL")
		if len(shell) == 0 {
			shell, _ = util.GetShell() // another way to get shell path
		}

		shellPath := shell
		if len(shellPath) == 0 || getShellTest { // empty shell means Bourne shell
			shellPath = _PATH_BSHELL
		}

		conf.commandPath = shellPath

		shellName := getShellNameFrom(shellPath)

		conf.commandArgv = []string{shellName}

		conf.withMotd = true
	}

	if len(conf.commandPath) == 0 {
		conf.commandPath = conf.commandArgv[0]

		if len(conf.commandArgv) == 1 {
			shellName := getShellNameFrom(conf.commandPath)
			conf.commandArgv = []string{shellName}
		} else {
			conf.commandArgv = conf.commandArgv[1:]
		}
	}

	// Adopt implementation locale
	util.SetNativeLocale()
	if !util.IsUtf8Locale() || buildConfigTest {
		nativeType := util.GetCtype()
		nativeCharset := util.LocaleCharset()

		// apply locale-related environment variables from client
		util.ClearLocaleVariables()
		for k, v := range conf.locales {
			// fmt.Printf("#buildConfig setenv %s=%s\n", k, v)
			os.Setenv(k, v)
		}

		// check again
		util.SetNativeLocale()
		if !util.IsUtf8Locale() || buildConfigTest {
			clientType := util.GetCtype()
			clientCharset := util.LocaleCharset()
			fmt.Printf("%s needs a UTF-8 native locale to run.\n", _COMMAND_NAME)
			fmt.Printf("Unfortunately, the local environment %s specifies "+
				"the character set \"%s\",\n", nativeType, nativeCharset)
			fmt.Printf("The client-supplied environment %s specifies "+
				"the character set \"%s\".\n", clientType, clientCharset)

			return "UTF-8 locale fail.", false
		}
	}
	return "", true
}

// parse the flag first, print help or version based on flag
// then run the main listening server
// aprilsh-server should be installed under $HOME/.local/bin
func main() {
	// https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/
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

	if conf.begin {
		beginClientConn(conf.desiredPort)
		return
	}

	// setup server log file
	if conf.verbose > 0 {
		util.Log.SetLevel(slog.LevelDebug)
	} else {
		util.Log.SetLevel(slog.LevelInfo)
	}
	util.Log.SetOutput(os.Stderr)
	syslogSupport = false
	if conf.verbose == _VERBOSE_LOG_SYSLOG {
		if util.Log.SetupSyslog("udp", "localhost:514") == nil {
			syslogSupport = true
		}
	}

	cpuf, err := os.Create("cpu.profile")
	if err != nil {
		fmt.Println(err)
		return
	}
	pprof.StartCPUProfile(cpuf)
	defer pprof.StopCPUProfile()

	// f, err := os.Create("mem.profile")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// pprof.WriteHeapProfile(f)
	// defer f.Close()

	// we need a webserver to get the pprof webserver
	// go func() {
	// 	fmt.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	// start server
	srv := newMainSrv(conf, runWorker)
	srv.start(conf)
	srv.wait()
}

// extract shell name from shellPath and prepend '-' to the returned shell name
func getShellNameFrom(shellPath string) (shellName string) {
	shellSplash := strings.LastIndex(shellPath, "/")
	if shellSplash == -1 {
		shellName = shellPath
	} else {
		shellName = shellPath[shellSplash+1:]
	}

	// prepend '-' to make login shell
	shellName = "-" + shellName

	return
}

func beginClientConn(port string) {
	// Unlike Dial, ListenPacket creates a connection without any
	// association with peers.
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()

	dest, err := net.ResolveUDPAddr("udp", "localhost:"+port)
	if err != nil {
		fmt.Println(err)
		return
	}

	// request from server
	request := fmt.Sprintf("%s", _ASH_OPEN)
	conn.SetDeadline(time.Now().Add(time.Millisecond * 20))
	n, err := conn.WriteTo([]byte(request), dest)
	if err != nil {
		fmt.Println("write to udp: ", err)
		return
	} else if n != len(request) {
		fmt.Println("can't send correct query.")
		return
	}

	// read the response
	response := make([]byte, 128)
	conn.SetDeadline(time.Now().Add(time.Millisecond * 90))
	m, _, err := conn.ReadFrom(response)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s", string(response[:m]))
}

// worker started by mainSrv.run(). worker will listen on specified port and
// forward user input to shell (started by runWorker. the output is forward
// to the network.
func runWorker(conf *Config, exChan chan string, whChan chan *workhorse) (err error) {
	defer func() {
		// notify this worker is done
		exChan <- conf.desiredPort
	}()

	/*
		If this variable is set to a positive integer number, it specifies how
		long (in seconds) aprilsh-server will wait to receive an update from the
		client before exiting.  Since aprilsh is very useful for mobile
		clients with intermittent operation and connectivity, we suggest
		setting this variable to a high value, such as 604800 (one week) or
		2592000 (30 days).  Otherwise, aprilsh-server will wait
		indefinitely for a client to reappear.  This variable is somewhat
		similar to the TMOUT variable found in many Bourne shells.
		However, it is not a login-session inactivity timeout; it only applies
		to network connectivity.

	*/
	networkTimeout := getTimeFrom("APRILSH_SERVER_NETWORK_TMOUT", 0)

	/*
		If this variable is set to a positive integer number, it specifies how
		long (in seconds) aprilsh-server will ignore SIGUSR1 while waiting
		to receive an update from the client.  Otherwise, SIGUSR1 will
		always terminate aprilsh-server.  Users and administrators may
		implement scripts to clean up disconnected aprilsh sessions.  With this
		variable set, a user or administrator can issue

		$ pkill -SIGUSR1 aprilsh-server

		to kill disconnected sessions without killing connected login
		sessions.
	*/
	networkSignaledTimeout := getTimeFrom("APRILSH_SERVER_SIGNAL_TMOUT", 0)

	// fmt.Printf("#runWorker networkTimeout=%d, networkSignaledTimeout=%d\n", networkTimeout, networkSignaledTimeout)

	// get initial window size
	var windowSize *unix.Winsize
	windowSize, err = unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	// windowSize, err := pty.GetsizeFull(os.Stdin)
	if err != nil || windowSize.Col == 0 || windowSize.Row == 0 {
		// Fill in sensible defaults. */
		// They will be overwritten by client on first connection.
		windowSize.Col = 80
		windowSize.Row = 24
	}

	// open parser and terminal
	savedLines := int(windowSize.Row) * terminal.SaveLinesRowsRatio
	terminal, err := statesync.NewComplete(int(windowSize.Col), int(windowSize.Row), savedLines)

	// open network
	blank := &statesync.UserStream{}
	network := network.NewTransportServer(terminal, blank, conf.desiredIP, conf.desiredPort)
	network.SetVerbose(uint(conf.verbose))
	defer network.Close()

	/*
		// If server is run on a pty, then typeahead may echo and break mosh.pl's
		// detection of the CONNECT message.  Print it on a new line to bodge
		// around that.

		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Printf("\r\n")
		}
	*/

	exChan <- network.GetKey() // send the key to run()

	// in mosh: the parent print this to stderr.
	// fmt.Printf("#runWorker %s CONNECT %s %s\n", COMMAND_NAME, network.Port(), network.GetKey())
	// printWelcome(os.Stdout, os.Getpid(), os.Stdin)

	// prepare for openPTS fail
	if conf.verbose == _VERBOSE_OPEN_PTS {
		windowSize = nil
	}

	ptmx, pts, err := openPTS(windowSize)
	if err != nil {
		// logW.Printf("#runWorker openPTS fail: %s\n", err)
		util.Log.With("error", err).Warn("openPTS fail")
		whChan <- &workhorse{}
		return err
	}
	defer func() {
		ptmx.Close()
		// pts.Close()
	}() // Best effort.
	// fmt.Printf("#runWorker openPTS successfully.\n")

	// prepare host field for utmp record
	utmpHost := fmt.Sprintf("%s [%d]", _PACKAGE_STRING, os.Getpid())

	// start the udp server, serve the udp request
	waitChan := make(chan bool)
	go conf.serve(ptmx, terminal, waitChan, network, networkTimeout, networkSignaledTimeout)
	util.Log.With("desiredPort", conf.desiredPort).Info("start listening on")

	// start the shell with pts
	shell, err := startShell(pts, utmpHost, conf)
	pts.Close() // it's copied by shell process, it's safe to close it here.
	if err != nil {
		// logW.Printf("#runWorker startShell fail: %s\n", err)
		util.Log.With("error", err).Warn("startShell fail")
		whChan <- &workhorse{}
	} else {
		// add utmp entry
		ptmxName := ptmx.Name() // TODO remove it?
		if utmpSupport {
			util.AddUtmpx(ptmx, utmpHost)
		}

		// update last log
		util.UpdateLastLog(ptmxName, getCurrentUser(), utmpHost) // TODO use pts.Name() or ptmx name?

		whChan <- &workhorse{shell, ptmx}

		// wait for the shell to finish.
		if state, err := shell.Wait(); err != nil || state.Exited() {
			// logW.Printf("#runWorker shell.Wait fail: %s, state: %s\n", err, state)
			if err != nil {
				util.Log.With("error", err).With("state", state).Warn("shell.Wait fail")
			}
		}

		// wait serve to finish
		util.Log.With("ptmx", ptmx).Debug("wait serve to finish")
		<-waitChan
		// logI.Printf("#runWorker stop listening on :%s\n", conf.desiredPort)
		util.Log.With("desiredPort", conf.desiredPort).Info("stop listening on")

		// clear utmp entry
		if utmpSupport {
			util.ClearUtmpx(ptmx)
		}
	}

	// fmt.Printf("[%s is exiting.]\n\n", _COMMAND_NAME)
	// https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/
	// https://www.prakharsrivastav.com/posts/golang-context-and-cancellation/

	return err
}

func getCurrentUser() string {
	user, err := user.Current()
	if err != nil || userCurrentTest {
		// logW.Printf("#getCurrentUser report: %s\n", err)
		util.Log.With("error", err).Warn("Get current user")
		return ""
	}

	return user.Username
}

func serve(ptmx *os.File, complete *statesync.Complete, waitChan chan bool,
	network *network.Transport[*statesync.Complete, *statesync.UserStream],
	networkTimeout int64, networkSignaledTimeout int64) error {
	// TODO consider timeout according to mosh 1.4

	// scale timeouts
	networkTimeoutMs := networkTimeout * 1000
	networkSignaledTimeoutMs := networkSignaledTimeout * 1000

	lastRemoteNum := network.GetRemoteStateNum()
	var connectedUtmp bool
	var forceConnectionChangEvt bool
	var savedAddr net.Addr

	var terminalToHost strings.Builder
	var timeSinceRemoteState int64

	// var networkChan chan frontend.Message
	var fileChan chan frontend.Message
	networkChan := make(chan frontend.Message, 1)
	fileChan = make(chan frontend.Message, 1)
	fileDownChan := make(chan any, 1)
	networkDownChan := make(chan any, 1)

	eg := errgroup.Group{}
	// read from socket
	eg.Go(func() error {
		frontend.ReadFromNetwork(1, networkChan, networkDownChan, network.GetConnection())
		return nil
	})

	// read from pty master file
	eg.Go(func() error {
		frontend.ReadFromFile(1, fileChan, fileDownChan, ptmx)
		return nil
	})

	// intercept signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGINT, syscall.SIGTERM)
	shutdownChan := make(chan bool)
	eg.Go(func() error { // TODO how to handle signal for goroutine?
		for {
			select {
			case s := <-sigChan:
				signals.Handler(s)
			case <-shutdownChan:
				return nil
			}
		}
	})

	var timeoutIfNoClient int64 = 60000
	childReleased := false

mainLoop:
	for {
		// util.Log.With("point", "a").Debug("mainLoop")

		timeout := math.MaxInt16
		now := time.Now().UnixMilli()

		// util.Log.With("point", "a1").Debug("mainLoop")
		timeout = terminal.Min(timeout, network.WaitTime()) // network.WaitTime cost time
		// util.Log.With("point", "a2").Debug("mainLoop")
		timeout = terminal.Min(timeout, complete.WaitTime(now))

		if network.GetRemoteStateNum() > 0 || network.ShutdownInProgress() {
			timeout = terminal.Min(timeout, 5000)
		}

		// util.Log.With("point", "a3").Debug("mainLoop")
		// The server goes completely asleep if it has no remote peer.
		// We may want to wake up sooner.
		var networkSleep int64
		if networkTimeoutMs > 0 {
			rs := network.GetLatestRemoteState()
			networkSleep = networkTimeoutMs - (now - rs.GetTimestamp())
			if networkSleep < 0 {
				networkSleep = 0
			} else if networkSleep > math.MaxInt16 {
				networkSleep = math.MaxInt16
			}
			timeout = terminal.Min(timeout, int(networkSleep))
		}

		// util.Log.With("point", "a4").Debug("mainLoop")

		p := network.GetLatestRemoteState()
		timeSinceRemoteState = now - p.GetTimestamp()
		terminalToHost.Reset()

		// payload, err := frontend.NetworkRead(5, network.GetConnection())
		// if err != nil {
		// 	util.Log.With("error", err).Warn("read from network")
		// 	continue mainLoop
		// } else if payload != "" {
		// }

		timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
		// util.Log.With("point", "b").Debug("mainLoop")
		select {
		case <-timer.C:
			// util.Log.With("complete", complete.WaitTime(now)).
			// 	// With("network", network.WaitTime()).
			// 	With("networkSleep", networkSleep).
			// 	With("timeout", timeout).
			// 	Debug("mainLoop")
		case socketMsg := <-networkChan: // packet received from the network
			if socketMsg.Err != nil {
				// fmt.Printf("#readFromSocket receive error:%s\n", socketMsg.Err)
				util.Log.With("error", socketMsg.Err).Warn("read from network")
				continue mainLoop
			}
			network.ProcessPayload(socketMsg.Data)

			// is new user input available for the terminal?
			if network.GetRemoteStateNum() != lastRemoteNum {
				lastRemoteNum = network.GetRemoteStateNum()

				us := &statesync.UserStream{}
				us.ApplyString(network.GetRemoteDiff())
				// rd := network.GetRemoteDiff()
				// us.ApplyString(rd)
				// util.Log.With("remoteDiff", rd).Debug("got from client")

				// apply userstream to terminal
				for i := 0; i < us.Size(); i++ {
					action := us.GetAction(i)
					if res, ok := action.(terminal.Resize); ok {
						//  apply only the last consecutive Resize action
						if i < us.Size()-1 {
							if _, ok = us.GetAction(i + 1).(terminal.Resize); ok {
								continue
							}
						}
						// resize master
						winSize, err := unix.IoctlGetWinsize(int(ptmx.Fd()), unix.TIOCGWINSZ)
						if err != nil {
							fmt.Printf("#serve ioctl TIOCGWINSZ %s", err)
							network.StartShutdown()
						}
						winSize.Col = uint16(res.Width)
						winSize.Row = uint16(res.Height)
						if err = unix.IoctlSetWinsize(int(ptmx.Fd()), unix.TIOCSWINSZ, winSize); err != nil {
							fmt.Printf("#serve ioctl TIOCSWINSZ %s", err)
							network.StartShutdown()
						}
						util.Log.With("col", winSize.Col).With("row", winSize.Row).Debug("input from host")
					}
					terminalToHost.WriteString(complete.ActOne(action))
				}

				util.Log.With("arise", "socket").With("data", terminalToHost.String()).Debug("input from host")

				if !us.Empty() {
					// register input frame number for future echo ack
					complete.RegisterInputFrame(lastRemoteNum, now)
				}

				// update client with new state of terminal
				if !network.ShutdownInProgress() {
					network.SetCurrentState(complete)
				}

				if utmpSupport {
					if !connectedUtmp {
						forceConnectionChangEvt = true
					} else {
						forceConnectionChangEvt = false
					}
				} else {
					forceConnectionChangEvt = false
				}

				// HAVE_UTEMPTER - update utmp entry if we have become "connected"
				// HAVE_SYSLOG - log connection information to syslog
				//
				// update utmp entry if we have become "connected"
				if forceConnectionChangEvt || !reflect.DeepEqual(savedAddr, network.GetRemoteAddr()) {

					util.ClearUtmpx(ptmx)

					// convert savedAddr to host name
					savedAddr = network.GetRemoteAddr()
					host := savedAddr.String() // default host name is ip string
					hostList, e := net.LookupAddr(host)
					if e == nil {
						host = hostList[0] // got the host name, use the first one
					}
					newHost := fmt.Sprintf("%s via %s [%d]", host, _PACKAGE_STRING, os.Getpid())

					util.AddUtmpx(ptmx, newHost)

					connectedUtmp = true
				}

				// TODO syslog?

				// upon receive network message, perform the following one time action,
				// release startShell() to start login session
				if !childReleased {
					_, err := ptmx.WriteString("\n")
					if err != nil {
						util.Log.With("error", err).Error("send release shell message failed")
					}
					util.Log.With("action", "send").Debug("release shell message")
					childReleased = true
				}
			}
		case masterMsg := <-fileChan:
			// input from the host needs to be fed to the terminal
			if !network.ShutdownInProgress() {

				// If the pty slave is closed, reading from the master can fail with
				// EIO (see #264).  So we treat errors on read() like EOF.
				if masterMsg.Err != nil {
					// fmt.Println("#readFromMaster report error: ", masterMsg.Err)
					util.Log.With("error", masterMsg.Err).Warn("read from master")
					network.StartShutdown()
				} else {
					out := complete.Act(masterMsg.Data)
					terminalToHost.WriteString(out)

					util.Log.With("arise", "master").
						With("data", masterMsg.Data).
						With("out", out).Debug("input from host")

					// update client with new state of terminal
					network.SetCurrentState(complete)
				}
			}
		}

		// write user input and terminal writeback to the host
		if terminalToHost.Len() > 0 {
			_, err := ptmx.WriteString(terminalToHost.String())
			if err != nil {
				network.StartShutdown()
			}

			util.Log.With("arise", "merge-").With("data", terminalToHost.String()).Debug("input from host")
		}

		idleShutdown := false
		if networkTimeoutMs > 0 && networkTimeoutMs <= timeSinceRemoteState {
			// if network timeout is set and over networkTimeoutMs quit this session.
			idleShutdown = true
			fmt.Printf("Network idle for %d seconds.\n", timeSinceRemoteState/1000)
		}

		if signals.GotSignal(syscall.SIGUSR1) {
			if networkSignaledTimeoutMs == 0 || networkSignaledTimeoutMs <= timeSinceRemoteState {
				idleShutdown = true
				fmt.Printf("Network idle for %d seconds when SIGUSR1 received.\n", timeSinceRemoteState/1000)
			}
		}

		if signals.AnySignal() || idleShutdown {
			// shutdown signal
			if network.HasRemoteAddr() && !network.ShutdownInProgress() {
				network.StartShutdown()
			} else {
				break
			}
		}

		// quit if our shutdown has been acknowledged
		if network.ShutdownInProgress() && network.ShutdownAcknowledged() {
			break
		}

		// quit after shutdown acknowledgement timeout
		if network.ShutdownInProgress() && network.ShutdownAckTimedout() {
			break
		}

		// quit if we received and acknowledged a shutdown request
		if network.CounterpartyShutdownAckSent() {
			break
		}

		// update utmp if has been more than 30 seconds since heard from client
		if utmpSupport && connectedUtmp && timeSinceRemoteState > 30000 {
			util.ClearUtmpx(ptmx)

			newHost := fmt.Sprintf("%s [%d]", _PACKAGE_STRING, os.Getpid())
			util.AddUtmpx(ptmx, newHost)

			connectedUtmp = false
		}

		if complete.SetEchoAck(now) && !network.ShutdownInProgress() {
			// update client with new echo ack
			network.SetCurrentState(complete)
		}

		// abort if no connection over 60 seconds
		if network.GetRemoteStateNum() == 0 && timeSinceRemoteState >= timeoutIfNoClient {
			util.Log.With("seconds", timeoutIfNoClient/1000).Warn("No connection within %d seconds")
			break
		}

		// util.Log.With("point", "c").Debug("mainLoop")

		err := network.Tick()
		if err != nil {
			util.Log.With("error", err).Warn("tick send failed")
		}
		// util.Log.With("point", "d").Debug("mainLoop")
	}

	// shutdown the goroutine
	shutdownChan <- true
	select {
	case fileDownChan <- "done":
	default:
	}
	select {
	case networkDownChan <- "done":
	default:
	}

	// consume last message to free reader if possible
	select {
	case <-fileChan:
	default:
	}
	select {
	case <-networkChan:
	default:
	}
	eg.Wait()

	// notify the runWorker
	waitChan <- true

	return nil
}

func getTimeFrom(env string, def int64) (ret int64) {
	ret = def

	v, exist := os.LookupEnv(env)
	if exist {
		var err error
		ret, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stdout, "%s not a valid integer, ignoring\n", env)
		} else if ret < 0 {
			fmt.Fprintf(os.Stdout, "%s is negative, ignoring\n", env)
			ret = 0
		}
	}
	return
}

func printWelcome(pid int, port int, tty *os.File) {
	// fmt.Printf("%s start listening on :%d. build version %s [pid=%d] \n", _COMMAND_NAME, port, BuildVersion, pid)
	util.Log.With("port", port).With("buildVersion", BuildVersion).With("pid", pid).
		Info(_COMMAND_NAME + " start listening on")
	// fmt.Printf("Copyright 2022~2023 wangqi.\n")
	// fmt.Printf("%s%s", "Use of this source code is governed by a MIT-style",
	// 	"license that can be found in the LICENSE file.\n")
	// logI.Printf("[%s detached, pid=%d]\n", COMMAND_NAME, pid)

	if tty != nil {
		inputUTF8, err := util.CheckIUTF8(int(tty.Fd()))
		if err != nil {
			// fmt.Printf("Warning: %s\n", err)
			util.Log.Warn(err.Error())
		}

		if !inputUTF8 {
			// Input is UTF-8 (since Linux 2.6.4)
			// fmt.Printf("%s %s %s", "Warning: termios IUTF8 flag not defined.",
			// 	"Character-erase of multibyte character sequence",
			// 	"probably does not work properly on this platform.\n")

			msg := fmt.Sprintf("%s %s %s", "Warning: termios IUTF8 flag not defined.",
				"Character-erase of multibyte character sequence",
				"probably does not work properly on this platform.")
			util.Log.Warn(msg)
		}
	}
}

// open pts master and slave, set terminal size according to window size.
func openPTS(wsize *unix.Winsize) (ptmx *os.File, pts *os.File, err error) {
	// open pts master and slave
	ptmx, pts, err = pty.Open()
	if wsize == nil {
		err = errors.New("invalid parameter")
	}
	if err == nil {
		sz := util.ConvertWinsize(wsize)
		// fmt.Printf("#openPTS sz=%v\n", sz)

		err = pty.Setsize(ptmx, sz) // set terminal size
	}
	return
}

// set IUTF8 flag for pts file. start shell process according to Config.
func startShell(pts *os.File, utmpHost string, conf *Config) (*os.Process, error) {
	if conf.verbose == _VERBOSE_START_SHELL {
		return nil, errors.New("fail to start shell")
	}
	// set IUTF8 if available
	if err := util.SetIUTF8(int(pts.Fd())); err != nil {
		return nil, err
	}

	// set TERM based on client TERM
	if conf.term != "" {
		os.Setenv("TERM", conf.term)
	} else {
		os.Setenv("TERM", "xterm-256color") // default TERM
	}

	// clear STY environment variable so GNU screen regards us as top level
	os.Unsetenv("STY")

	// the following function will set PWD environment variable
	// chdirHomedir("")

	// ask ncurses to send UTF-8 instead of ISO 2022 for line-drawing chars
	ncursesEnv := "NCURSES_NO_UTF8_ACS=1"
	// should be the last statement related to environment variable
	env := append(os.Environ(), ncursesEnv)

	// set working directory
	// cmd.Dir = getHomeDir()

	sysProcAttr := &syscall.SysProcAttr{}
	sysProcAttr.Setsid = true  // start a new session
	sysProcAttr.Setctty = true // set controlling terminal

	procAttr := os.ProcAttr{
		Files: []*os.File{pts, pts, pts}, // use pts as stdin, stdout, stderr
		Dir:   getHomeDir(),
		Sys:   sysProcAttr,
		Env:   env,
	}

	if conf.withMotd && !motdHushed() {
		// For Ubuntu, try and print one of {,/var}/run/motd.dynamic.
		// This file is only updated when pam_motd is run, but when
		// mosh-server is run in the usual way with ssh via the script,
		// this always happens.
		// XXX Hackish knowledge of Ubuntu PAM configuration.
		// But this seems less awful than build-time detection with autoconf.
		if !printMotd(pts, "/run/motd.dynamic") {
			printMotd(pts, "/var/run/motd.dynamic")
		}
		// Always print traditional /etc/motd.
		printMotd(pts, "/etc/motd")

		warnUnattached(pts, utmpHost)
	}

	encrypt.ReenableDumpingCore()

	/*
		additional logic for pty.StartWithAttrs() end
	*/

	// wait for serve() to release us
	buf := make([]byte, 81)
	var n int
	var err error

	util.Log.With("action", "wait").With("buf", buf[:n]).Debug("release shell message")
	if n, err = pts.Read(buf); err != nil {
		util.Log.With("error", err).With("n", n).Error("wait for release shell failed")
		return nil, err
	}
	util.Log.With("action", "receive").With("buf", buf[:n]).Debug("release shell message")

	proc, err := os.StartProcess(conf.commandPath, conf.commandArgv, &procAttr)
	if err != nil {
		return nil, err
	}
	return proc, nil
}

// check unattached session and print warning message if there is any
// ignore current session
func warnUnattached(w io.Writer, ignoreHost string) {
	userName := getCurrentUser()

	// check unattached sessions
	unatttached := util.CheckUnattachedUtmpx(userName, ignoreHost, _PACKAGE_STRING)

	if unatttached == nil || len(unatttached) == 0 {
		return
	} else if len(unatttached) == 1 {
		fmt.Fprintf(w, "\033[37;44mAprilsh: You have a detached session on this server (%s).\033[m\n\n",
			unatttached[0])
	} else {
		var sb strings.Builder
		for _, v := range unatttached {
			fmt.Fprintf(&sb, "- %s\n", v)
		}

		fmt.Fprintf(w, "\033[37;44mAprilsh: You have %d detached sessions on this server, with PIDs:\n%s\033[m\n",
			len(unatttached), sb.String())
	}
}

type mainSrv struct {
	workers   map[int]*workhorse
	runWorker func(*Config, chan string, chan *workhorse) error // worker
	exChan    chan string                                       // worker done or passing key
	whChan    chan *workhorse                                   // workhorse
	downChan  chan bool                                         // shutdown mainSrv
	maxPort   int                                               // max worker port
	timeout   int                                               // read udp time out,
	port      int                                               // main listen port
	conn      *net.UDPConn                                      // mainSrv listen port
	wg        sync.WaitGroup
	eg        errgroup.Group
}

type workhorse struct {
	shell *os.Process
	ptmx  *os.File
}

func newMainSrv(conf *Config, runWorker func(*Config, chan string, chan *workhorse) error) *mainSrv {
	m := mainSrv{}
	m.runWorker = runWorker
	m.port, _ = strconv.Atoi(conf.desiredPort)
	m.maxPort = m.port + 1
	m.workers = make(map[int]*workhorse)
	m.downChan = make(chan bool, 1)
	m.exChan = make(chan string, 1)
	m.whChan = make(chan *workhorse, 1)
	m.timeout = 20
	m.eg = errgroup.Group{}

	return &m
}

// start mainSrv, which listen on the main udp port.
// each new client send a shake hands message to mainSrv. mainSrv response
// with the session key and target udp port for the new client.
// mainSrv is shutdown by SIGTERM and all sessions must be done.
// otherwise mainSrv will wait for the live session.
func (m *mainSrv) start(conf *Config) {
	// init udp server

	// handle signal: SIGTERM, SIGHUP
	go m.handler()

	// start udp server upon receive the shake hands message.
	if err := m.listen(conf); err != nil {
		// logW.Printf("%s: %s\n", _COMMAND_NAME, err.Error())
		util.Log.With("error", err).Warn("listen failed")
		return
	}

	// fmt.Printf("#start listening on %s, next port is %d\n", conf.desiredPort, m.nextWorkerPort+1)
	m.wg.Add(1)
	go m.run(conf)

	if conf.autoStop > 0 {
		time.AfterFunc(time.Duration(5)*time.Second, func() {
			m.downChan <- true
		})
	}
}

func (m *mainSrv) handler() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sig)

	for s := range sig {
		switch s {
		case syscall.SIGHUP: // TODO:reload the config?
			// logI.Println("got message SIGHUP.")
			util.Log.Info("got message SIGHUP")
		case syscall.SIGTERM, syscall.SIGINT:
			// logI.Println("got message SIGTERM.")
			m.downChan <- true
			return
		}
	}
}

// to support multiple clients, mainServer listen on the specified port.
// start udp server for each new client.
func (m *mainSrv) listen(conf *Config) error {
	// fmt.Println("#start ResolveUDPAddr.")
	local_addr, err := net.ResolveUDPAddr("udp", ":"+conf.desiredPort)
	if err != nil {
		return err
	}

	m.conn, err = net.ListenUDP("udp", local_addr)
	if err != nil {
		return err
	}

	return nil
}

/*
in aprilsh: we can use nc client to get the key and send it back to client.
we don't print it to the stdout as mosh did.

send udp request and read reply
% echo "open aprilsh:" | nc localhost 6000 -u -w 1
% echo "close aprilsh:6001" | nc localhost 6000 -u -w 1

send udp request to remote host
% ssh ide@localhost  "echo 'open aprilsh:' | nc localhost 6000 -u -w 1"
*/
func (m *mainSrv) run(conf *Config) {
	if m.conn == nil {
		return
	}

	defer func() {
		m.conn.Close()
		m.wg.Done()
		// fmt.Printf("%s  stop listening on :%d.\n", _COMMAND_NAME, m.port)
		util.Log.With("port", m.port).Info(_COMMAND_NAME + " stop listening")
	}()

	buf := make([]byte, 128)
	shutdown := false

	printWelcome(os.Getpid(), m.port, nil)
	for {
		select {
		case portStr := <-m.exChan: // some worker is done
			p, err := strconv.Atoi(portStr)
			if err != nil {
				// fmt.Printf("#run got %s from workDone channel. error: %s\n", portStr, err)
				break
			}
			// util.Log.With("port", p).With("maxPort", m.maxPort).Debug("worker is done")
			// clear worker list
			delete(m.workers, p)
		case sd := <-m.downChan: // ready to shutdown mainSrv
			// fmt.Printf("#run got shutdown message %t\n", sd)
			shutdown = sd
		default:
		}

		if shutdown {
			if len(m.workers) == 0 {
				return
			} else { // kill the workers
				for i := range m.workers {
					m.workers[i].shell.Kill()
					// fmt.Printf("kill %d\n", i)
				}
				return
			}
		}

		// set read time out: 200ms
		m.conn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(m.timeout)))
		n, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// fmt.Printf("#run read time out, workers=%d, shutdown=%t, err=%s\n", len(m.workers), shutdown, err)
				continue
			} else {
				// take a break in case reading error
				time.Sleep(time.Duration(5) * time.Millisecond)
				// fmt.Println("#run read error: ", err)
				continue
			}
		}
		// fmt.Printf("#run receive %q from %s\n", strings.TrimSpace(string(buf[0:n])), addr)

		req := strings.TrimSpace(string(buf[0:n]))
		// 'open aprilsh:' to start the server
		if strings.HasPrefix(req, _ASH_OPEN) {
			// prepare next port
			p := m.getAvailabePort() // TODO set limit for port?

			if !m.isPortExist(p) { // check exist port
				// start the worker
				conf2 := *conf
				conf2.desiredPort = fmt.Sprintf("%d", p)

				// For security, make sure we don't dump core
				encrypt.DisableDumpingCore()

				m.eg.Go(func() error {
					return m.runWorker(&conf2, m.exChan, m.whChan)
				})
				// fmt.Printf("#run start a worker at %s\n", conf2.desiredPort)

				// blocking read the key from runWorker
				key := <-m.exChan
				// fmt.Printf("#run got key %q\n", key)

				// response session key and udp port to client
				msg := fmt.Sprintf("%d,%s", p, key)
				m.writeRespTo(addr, _ASH_OPEN, msg)

				// blocking read the workhorse from runWorker
				wh := <-m.whChan
				// logI.Printf("#run got workhorse %p %v\n", wh.shell, wh.shell)
				if wh.shell != nil {
					m.workers[p] = wh
				}
			} else {
				resp := m.writeRespTo(addr, _ASH_OPEN, "duplicate request")
				util.Log.With("request", req).With("response", resp).Warn("duplicate request")
			}
		} else if strings.HasPrefix(req, _ASH_CLOSE) {
			// fmt.Printf("#mainSrv run() receive request %q\n", req)
			// 'close aprish:[port]' to stop the server
			pstr := strings.TrimPrefix(req, _ASH_CLOSE)
			port, err := strconv.Atoi(pstr)
			if err == nil {
				// fmt.Printf("#run got request to stop %d\n", port)
				// find workhorse
				if wh, ok := m.workers[port]; ok {
					// kill the process, TODO SIGKILL or SIGTERM?
					wh.shell.Kill()

					m.writeRespTo(addr, _ASH_CLOSE, "done")
					// fmt.Printf("#mainSrv run() send %q to client\n", resp)
				} else {
					resp := m.writeRespTo(addr, _ASH_CLOSE, "port does not exist")
					util.Log.With("request", req).With("response", resp).Warn("port does not exit")
				}
			} else {
				resp := m.writeRespTo(addr, _ASH_CLOSE, "wrong port number")
				util.Log.With("request", req).With("response", resp).Warn("wrong port number")
			}
		} else {
			resp := m.writeRespTo(addr, _ASH_CLOSE, "unknow request")
			util.Log.With("request", req).With("response", resp).Warn("unknow request")
		}
	}
}

// return the minimal available port and increase the maxWorkerPort if necessary.
func (m *mainSrv) getAvailabePort() (port int) {
	port = m.port
	if len(m.workers) > 0 {
		// sort the current ports
		ports := make([]int, 0, len(m.workers))
		for k := range m.workers {
			ports = append(ports, k)
		}
		sort.Ints(ports)
		// fmt.Printf("#getAvailabePort got ports=%v\n", ports)

		// check minimal available port
		for i := 0; i < m.maxPort-m.port-1; i++ {
			// fmt.Printf("#getAvailabePort check port+k=%d, ports[i]=%d\n", port+i+1, ports[i])
			if port+i+1 < ports[i] {
				port = port + i + 1
				break
			}
		}

		// right most case
		if port == m.port {
			port = m.maxPort
			m.maxPort++
		}
		// fmt.Printf("#getAvailabePort search port=%d\n", port)
	} else if len(m.workers) == 0 {
		port = m.port + 1
	}

	// fmt.Printf("#getAvailabePort got port=%d\n", port)
	// util.Log.With("port", port).With("maxPort", m.maxPort).
	// 	With("workers", len(m.workers)).Debug("getAvailabePort")
	return port
}

func (m *mainSrv) isPortExist(port int) bool {
	for p := range m.workers {
		if port == p {
			return true
		}
	}
	return false
}

// write header and message to addr
func (m *mainSrv) writeRespTo(addr *net.UDPAddr, header, msg string) (resp string) {
	resp = fmt.Sprintf("%s%s\n", header, msg)
	// util.Log.With("resp", resp).Debug("writeRespTo")
	m.conn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(m.timeout)))
	m.conn.WriteToUDP([]byte(resp), addr)
	return
}

func (m *mainSrv) wait() {
	m.wg.Wait()
	if err := m.eg.Wait(); err != nil {
		// logW.Printf("#mainSrv wait() reports %s\n", err.Error())
		util.Log.With("error", err).Warn("wait failed")
	}
}
