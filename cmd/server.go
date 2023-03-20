// Copyright 2022 wangqi. All rights reserved.
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
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	utmp "blitter.com/go/goutmp"
	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	PACKAGE_STRING = "aprilsh"
	COMMAND_NAME   = "aprilsh-server"
	BUILD_VERSION  = "0.1.0"
	DOCONFIG_TEST  = PACKAGE_STRING + "_doconfig_test_only"

	_PATH_BSHELL = "/bin/sh"
)

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "%s (%s) [build %s]\n", COMMAND_NAME, PACKAGE_STRING, BUILD_VERSION)
	fmt.Fprintf(w, "Copyright (c) 2022~2023 wangqi ericwq057[AT]qq[dot]com\n")
	// TODO add a slogans here.
}

func printUsage(w io.Writer, usage string) {
	fmt.Fprintf(w, "%s", usage)
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

	// fmt.Printf("#chdirHomedir home=%q\n", home)
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
func getSSHip() string {
	env := os.Getenv("SSH_CONNECTION")
	if len(env) == 0 { // Older sshds don't set this
		fmt.Fprintf(os.Stderr, "Warning: SSH_CONNECTION not found; binding to any interface.\n")
		return ""
	}

	// SSH_CONNECTION' Identifies the client and server ends of the connection.
	// The variable contains four space-separated values: client IP address,
	// client port number, server IP address, and server port number.
	//
	// ipv4 sample: SSH_CONNECTION=172.17.0.1 58774 172.17.0.2 22
	sshConn := strings.Split(env, " ")
	if len(sshConn) != 4 {
		fmt.Fprintf(os.Stderr, "Warning: Could not parse SSH_CONNECTION; binding to any interface.\n")
		// fmt.Printf("#getSSHip env=%q, size=%d\n", sshConn, len(sshConn))
		return ""
	}

	localInterfaceIP := strings.ToLower(sshConn[2])
	prefixIPv6 := "::ffff:"

	// fmt.Printf("#getSSHip localInterfaceIP=%q, prefixIPv6=%q\n", localInterfaceIP, prefixIPv6)
	if len(localInterfaceIP) > len(prefixIPv6) && strings.HasPrefix(localInterfaceIP, prefixIPv6) {
		return localInterfaceIP[len(prefixIPv6):]
	}

	return localInterfaceIP
}

// [-s] [-v] [-i LOCALADDR] [-p PORT[:PORT2]] [-c COLORS] [-l NAME=VALUE] [-- COMMAND...]
var usage = `Usage:
  ` + COMMAND_NAME + ` [--version] [--help]
  ` + COMMAND_NAME + ` [--server] [--verbose] [--ip ADDR] [--port PORT[:PORT2]] [--color COLORS]` +
	` [--locale NAME=VALUE] [-- command...]
Options:
  -h, --help     print this message
      --version  print version information
  -v, --verbose  verbose output
  -s, --server   listen with SSH ip
  -i, --ip       listen ip
  -p, --port     listen port range
  -l, --locale   key-value pairs
  -c, --color    xterm color
  -t, --term     client TERM
`
var logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

type Config struct {
	version     bool // verbose : don't close stdin/stdout/stderr
	server      bool
	verbose     bool
	desiredIP   string
	desiredPort string
	locales     localeFlag
	color       int
	term        string // client TERM

	commandPath string
	commandArgv []string // the positional (non-flag) command-line arguments.
	withMotd    bool
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

	flagSet.BoolVar(&conf.verbose, "verbose", false, "verbose output")
	flagSet.BoolVar(&conf.verbose, "v", false, "verbose output")

	flagSet.BoolVar(&conf.version, "version", false, "print version information")

	flagSet.BoolVar(&conf.server, "server", false, "listen with SSH ip")
	flagSet.BoolVar(&conf.server, "s", false, "listen with SSH ip")

	flagSet.StringVar(&conf.desiredIP, "ip", "", "listen ip")
	flagSet.StringVar(&conf.desiredIP, "i", "", "listen ip")

	flagSet.StringVar(&conf.desiredPort, "port", "", "listen port range")
	flagSet.StringVar(&conf.desiredPort, "p", "", "listen port range")

	flagSet.StringVar(&conf.term, "term", "", "client TERM")
	flagSet.StringVar(&conf.term, "t", "", "client TERM")

	flagSet.Var(&conf.locales, "locale", "locale list, key=value pair")
	flagSet.Var(&conf.locales, "l", "locale list, key=value pair")

	flagSet.IntVar(&conf.color, "color", 0, "xterm color")
	flagSet.IntVar(&conf.color, "c", 0, "xterm color")

	err = flagSet.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}

	// get the non-flag command-line arguments.
	conf.commandArgv = flagSet.Args()
	return &conf, buf.String(), nil
}

// parse the flag first, print help or version based on flag
// then run the main listening server
func main() {
	// For security, make sure we don't dump core
	if err := encrypt.DisableDumpingCore(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	conf, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		printUsage(os.Stdout, usage)
		return
	} else if err != nil {
		// fmt.Println("got error:", err)
		fmt.Println("output:\n", output)
		return
	}

	if conf.version {
		printVersion(os.Stdout)
		return
	}
	if conf.server {
		if sshIP := getSSHip(); len(sshIP) != 0 {
			conf.desiredIP = sshIP
			// fmt.Printf("#main sshIP=%s\n", desiredIP)
		}
	}

	if len(conf.desiredPort) > 0 {
		// Sanity-check arguments

		// fmt.Printf("#main desiredPort=%s\n", desiredPort)
		_, _, ok := network.ParsePortRange(conf.desiredPort, logW)
		if !ok {
			logW.Printf("%s: Bad UDP port range (%s)", COMMAND_NAME, conf.desiredPort)
			return
		}
	}

	if err := buildConfig(conf); err != nil {
		logW.Printf("%s: %s\n", COMMAND_NAME, err.Error())
		return
	}

	// init udp server
	m := mainServer{port: conf.desiredPort}
	m.done = make(chan bool)
	m.nextPort, _ = strconv.Atoi(m.port)
	m.clients = make(map[int]bool)
	m.zeroWorker = true
	m.workerFinish = make(chan string)

	// handle signal: SIGTERM, SIGHUP
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM)
		for s := range sig {
			switch s {
			case syscall.SIGHUP: // TODO:what we need to do?
				logW.Println("got message SIGHUP.")
			case syscall.SIGTERM:
				logW.Println("got message SIGQUIT.")
				m.done <- true
				return
			}
		}
	}()

	// start udp server
	m.wg.Add(1)
	if err := m.start(conf); err != nil {
		m.wg.Done()
		logW.Printf("%s: %s\n", COMMAND_NAME, err.Error())
		return
	}

	m.wg.Wait()
}

// build the config instance and check the utf-8 locale. return error if the terminal
// can't support utf-8 locale.
func buildConfig(conf *Config) error {
	conf.commandPath = ""
	conf.withMotd = false

	// Get shell
	if len(conf.commandArgv) == 0 {
		shell := os.Getenv("SHELL")
		if len(shell) == 0 {
			shell, _ = getShell() // another way to get shell path
		}

		shellPath := shell
		if len(shellPath) == 0 { // empty shell means Bourne shell
			shellPath = _PATH_BSHELL
		}

		conf.commandPath = shellPath

		shellName := getShellNameFrom(shellPath)

		conf.commandArgv = []string{shellName}

		conf.withMotd = true
	}

	if len(conf.commandPath) == 0 {
		// TODO logic flaw?
		// if commandArgv is empty, the previous block asssign '-sh' to commandArgv[0]
		// if commandArgv is not empty, commandArgv[0] is in the form of '/bin/sh'
		// commandPath got the same value '/bin/sh', while commandArgv[0] got '-sh' or '/bin/sh'.
		conf.commandPath = conf.commandArgv[0]
	}

	// Adopt implementation locale
	setNativeLocale()
	if !isUtf8Locale() {
		nativeType := getCtype()
		nativeCharset := localeCharset()

		// apply locale-related environment variables from client
		clearLocaleVariables()
		for k, v := range conf.locales {
			// fmt.Printf("#doConfig setenv %s=%s\n", k, v)
			os.Setenv(k, v)
		}

		// check again
		setNativeLocale()
		if !isUtf8Locale() {
			clientType := getCtype()
			clientCharset := localeCharset()
			fmt.Fprintf(os.Stderr, "%s needs a UTF-8 native locale to run.\n\n", COMMAND_NAME)
			fmt.Fprintf(os.Stderr, "Unfortunately, the local environment (%s) specifies\n"+
				"the character set \"%s\",\n\n", nativeType, nativeCharset)
			fmt.Fprintf(os.Stderr, "The client-supplied environment (%s) specifies\n"+
				"the character set \"%s\".\n\n", clientType, clientCharset)
			return errors.New("UTF-8 locale fail.")
		}
	}

	if _, ok := os.LookupEnv(DOCONFIG_TEST); ok {
		return errors.New(DOCONFIG_TEST + " is set.")
	}
	return nil
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

func runServer(conf *Config, keyChan chan string, worker chan string) {
	networkTimeout := getTimeFrom("APRILSH_SERVER_NETWORK_TMOUT", 0)
	networkSignaledTimeout := getTimeFrom("APRILSH_SERVER_SIGNAL_TMOUT", 0)

	fmt.Printf("#runServer networkTimeout=%d, networkSignaledTimeout=%d\n", networkTimeout, networkSignaledTimeout)

	// get initial window size
	windowSize, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	// windowSize, err := pty.GetsizeFull(os.Stdin)
	if err != nil || windowSize.Col == 0 || windowSize.Row == 0 {
		// Fill in sensible defaults. */
		// They will be overwritten by client on first connection.
		windowSize.Col = 80
		windowSize.Row = 24
	}

	// open parser and terminal
	terminal, err := statesync.NewComplete(int(windowSize.Col), int(windowSize.Row), 0)

	// open network
	blank := &statesync.UserStream{}
	network := network.NewTransportServer(terminal, blank, conf.desiredIP, conf.desiredPort)
	if conf.verbose {
		network.SetVerbose(1)
	}

	// If server is run on a pty, then typeahead may echo and break mosh.pl's
	// detection of the CONNECT message.  Print it on a new line to bodge
	// around that.
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("\r\n")
	}
	fmt.Printf("%s CONNECT %s %s\n", COMMAND_NAME, network.Port(), network.GetKey())
	keyChan <- network.GetKey()

	printWelcome(os.Stderr, os.Getpid(), os.Stdin)

	ptmx, pts, err := openPTS(windowSize, conf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "#runServer prepare pts error: %s\n", err)
	}

	// add utmp entry
	ptsName := ptmx.Name()
	host := fmt.Sprintf("aprilsh [%d]", os.Getpid())
	usr := getCurrentUser()
	utmpEntry := utmp.Put_utmp(usr, ptsName, host)

	// update last log
	utmp.Put_lastlog_entry(COMMAND_NAME, usr, ptsName, host)

	// start the udp server, serve the udp request
	go serve(ptmx, terminal, network, networkTimeout, networkSignaledTimeout)

	// clear utmp entry
	utmp.Unput_utmp(utmpEntry)

	// start the shell
	shell, err := startShell(ptmx, pts, conf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "#runServer report: %s\n", err)
	}

	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	// wait for the shell to finish.
	if err2 := shell.Wait(); err2 != nil {
		fmt.Fprintf(os.Stderr, "#runServer start shell error: %s\n", err2)
	}

	// kill the shell when the server done
	defer func() { shell.Cancel() }()

	// notify the main server
	worker <- conf.desiredPort
	fmt.Printf("\n[%s is exiting.]\n", COMMAND_NAME)
	// https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/
	// https://www.prakharsrivastav.com/posts/golang-context-and-cancellation/
}

func getCurrentUser() string {
	user, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "#getCurrentUser report: %s\n", err)
		return ""
	}

	return user.Username
}

func serve(ptmx *os.File, terminal *statesync.Complete,
	network *network.Transport[*statesync.Complete, *statesync.UserStream],
	networkTimeout int64, networkSignaledTimeout int64,
) error {
	return nil
}

func getTimeFrom(env string, def int64) (ret int64) {
	ret = def

	v, exist := os.LookupEnv(env)
	if exist {
		var err error
		ret, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s not a valid integer, ignoring\n", env)
		} else if ret < 0 {
			fmt.Fprintf(os.Stderr, "%s is negative, ignoring\n", env)
			ret = 0
		}
	}
	return
}

func printWelcome(w io.Writer, pid int, tty *os.File) {
	fmt.Fprintf(w, "%s [build %s]\n", COMMAND_NAME, BUILD_VERSION)
	fmt.Fprintf(w, "Copyright 2022 wangqi.\n")
	fmt.Fprintf(w, "Use of this source code is governed by a MIT-style\n")
	fmt.Fprintf(w, "license that can be found in the LICENSE file.\n\n")
	fmt.Fprintf(w, "[%s detached, pid= %d]\n", COMMAND_NAME, pid)

	inputUTF8, err := checkIUTF8(int(tty.Fd()))
	if err != nil {
		fmt.Fprintf(w, "\nWarning: %s\n", err)
	}

	if !inputUTF8 {
		// Input is UTF-8 (since Linux 2.6.4)
		fmt.Fprintf(w, "%s%s%s",
			"\nWarning: termios IUTF8 flag not defined.\n",
			"Character-erase of multibyte character sequence\n",
			"probably does not work properly on this platform.\n")
	}
}

func checkIUTF8(fd int) (bool, error) {
	termios, err := unix.IoctlGetTermios(fd, getTermios)
	if err != nil {
		return false, err
	}

	// Input is UTF-8 (since Linux 2.6.4)
	return (termios.Iflag & unix.IUTF8) != 0, nil
}

func setIUTF8(fd int) error {
	termios, err := unix.IoctlGetTermios(fd, getTermios)
	if err != nil {
		return err
	}

	termios.Iflag |= unix.IUTF8

	if err := unix.IoctlSetTermios(fd, setTermios, termios); err != nil {
		return err
	}
	return nil
}

func convertWinsize(windowSize *unix.Winsize) *pty.Winsize {
	var sz pty.Winsize
	sz.Cols = windowSize.Col
	sz.Rows = windowSize.Row
	sz.X = windowSize.Xpixel
	sz.Y = windowSize.Ypixel

	return &sz
}

func openPTS(windowSize *unix.Winsize, conf *Config) (ptmx *os.File, pts *os.File, err error) {
	// open pts master and slave
	ptmx, pts, err = pty.Open() // open pty master and slave
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = pts.Close() }() // Best effort.

	sz := convertWinsize(windowSize)
	if sz != nil { // set terminal size
		if err := pty.Setsize(ptmx, sz); err != nil {
			_ = ptmx.Close() // Best effort.
			return nil, nil, err
		}
	}

	return ptmx, pts, nil
}

func startShell(ptmx *os.File, pts *os.File, conf *Config) (*exec.Cmd, error) {
	cmd := exec.Command(conf.commandPath, conf.commandArgv...)

	// copy from pty.StartWithSize()
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Setsid = true  // start a new session
	cmd.SysProcAttr.Setctty = true // set controlling terminal

	/*
		copy from pty.StartWithAttrs()
		need to add some logic inside pty.StartWithAttrs()
	*/

	/*
		// open pts master and slave
		ptmx, pts, err := pty.Open() // open pty master and slave
		if err != nil {
			return cmd, nil, err
		}
		defer func() { _ = pts.Close() }() // Best effort.

		sz := convertWinsize(windowSize)
		if sz != nil { // set terminal size
			if err := pty.Setsize(ptmx, sz); err != nil {
				_ = ptmx.Close() // Best effort.
				return cmd, nil, err
			}
		}
	*/

	// set stdin, stdout, stderr for pty slave
	if cmd.Stdout == nil {
		cmd.Stdout = pts
	}
	if cmd.Stderr == nil {
		cmd.Stderr = pts
	}
	if cmd.Stdin == nil {
		cmd.Stdin = pts
	}

	/*
		additional logic for pty.StartWithAttrs() begin
	*/

	// reenable signals for cmd.Start
	signal.Reset(syscall.SIGHUP) // reset means using default signal handler
	signal.Reset(syscall.SIGPIPE)

	// set IUTF8 if available
	if err := setIUTF8(int(pts.Fd())); err != nil {
		return cmd, err
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
	chdirHomedir("")

	// ask ncurses to send UTF-8 instead of ISO 2022 for line-drawing chars
	ncursesEnv := "NCURSES_NO_UTF8_ACS=1"
	// should be the last statement related to environment variable
	newEnv := append(os.Environ(), ncursesEnv)
	cmd.Env = newEnv

	// set working directory
	cmd.Dir = getHomeDir()

	if conf.withMotd && !motdHushed() {
		// For Ubuntu, try and print one of {,/var}/run/motd.dynamic.
		// This file is only updated when pam_motd is run, but when
		// mosh-server is run in the usual way with ssh via the script,
		// this always happens.
		// XXX Hackish knowledge of Ubuntu PAM configuration.
		// But this seems less awful than build-time detection with autoconf.
		if !printMotd(cmd.Stdout, "/run/motd.dynamic") {
			printMotd(cmd.Stdout, "/var/run/motd.dynamic")
		}
		// Always print traditional /etc/motd.
		printMotd(cmd.Stdout, "/etc/motd")
	}

	// Wait for parent to release us.
	var buf string
	if _, err := fmt.Fscanf(cmd.Stdin, "%s", &buf); err != nil {
		return cmd, err
	}

	encrypt.ReenableDumpingCore()

	/*
		additional logic for pty.StartWithAttrs() end
	*/

	if err := cmd.Start(); err != nil {
		_ = ptmx.Close() // Best effort.
		return cmd, err
	}

	return cmd, nil
}

type mainServer struct {
	port         string
	clients      map[int]bool
	nextPort     int
	done         chan bool
	workerFinish chan string
	zeroWorker   bool
	wg           sync.WaitGroup
}

// to support multiple clients, mainServe listen on the specified port.
// start udp server for each new client.
func (m *mainServer) start(conf *Config) error {
	local_addr, err := net.ResolveUDPAddr("udp", m.port)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", local_addr)
	if err != nil {
		return err
	}

	buf := make([]byte, 128)

	fmt.Printf("#start listening on %s, next port is %d\n", m.port, m.nextPort)
	go func() {
		defer func() {
			conn.Close()
			m.wg.Done()
		}()

		for {
			select {
			case portStr := <-m.workerFinish:
				// got finish messsage from finish
				p, err := strconv.Atoi(portStr)
				if err != nil {
					fmt.Printf("#start got %s from finish channel. error: %s\n", portStr, err)
					break
				}
				delete(m.clients, p)
			case <-m.done:
				if m.zeroWorker {
					return
				}
			default:
			}

			m.checkZeroWorker()

			// set read time out 2 seconds
			conn.SetDeadline(time.Now().Add(time.Second * 2))
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					// read time out
					continue
				} else {
					fmt.Println("#start Error: ", err) // read error to log?
					continue
				}
			}
			// fmt.Printf("Received %q from %s\n", strings.TrimSpace(string(buf[0:n])), addr)

			// only response to request start with 'open aprilsh'
			req := strings.TrimSpace(string(buf[0:n]))
			if strings.HasPrefix(req, "open aprilsh") {
				// prepare next port
				m.nextPort++

				// start the worker server
				conf2 := *conf
				conf2.desiredPort = fmt.Sprintf("%d", m.nextPort)
				keyChan := make(chan string)
				go runServer(&conf2, keyChan, m.workerFinish)

				// blocking read the key from runServer
				key := <-keyChan

				// mark the worker server
				m.clients[m.nextPort] = true
				m.zeroWorker = false

				// response session key and udp port to client
				resp := fmt.Sprintf("%d,%s\n", m.nextPort, key)
				conn.WriteToUDP([]byte(resp), addr)
			}
		}
	}()

	return nil
}

func (m *mainServer) checkZeroWorker() bool {
	if len(m.clients) == 0 {
		m.zeroWorker = true
		return true
	}
	m.zeroWorker = false
	return false
}
