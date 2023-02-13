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
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

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

	if err := doConfig(conf); err != nil {
		logW.Printf("%s: %s\n", COMMAND_NAME, err.Error())
		return
	}

	runServer(conf)
}

func doConfig(conf *Config) error {
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

func runServer(conf *Config) {
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

	printWelcome(os.Stderr, os.Getpid(), os.Stdin)
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

func runShell(sz *pty.Winsize, conf *Config) (*os.File, error) {
	cmd := exec.Command(conf.commandPath, conf.commandArgv...)

	// open pts master and slave
	ptmx, pts, err := pty.Open() // open pty master and slave
	if err != nil {
		return nil, err
	}
	defer func() { _ = pts.Close() }() // Best effort.

	if sz != nil { // set terminal size
		if err := pty.Setsize(ptmx, sz); err != nil {
			_ = ptmx.Close() // Best effort.
			return nil, err
		}
	}

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

	// set IUTF8 if available
	if err := setIUTF8(int(pts.Fd())); err != nil {
		return nil, err
	}

	// set TERM
	// TODO we should set the TERM based on user client TERM

	// clear STY environment variable so GNU screen regards us as top level
	os.Unsetenv("STY")

	chdirHomedir("")

	// ask ncurses to send UTF-8 instead of ISO 2022 for line-drawing chars
	ncursesEnv := "NCURSES_NO_UTF8_ACS=1"
	newEnv := append(os.Environ(), ncursesEnv)
	cmd.Env = newEnv

	// set working directory
	cmd.Dir = getHomeDir()

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Setsid = true  // start a new session
	cmd.SysProcAttr.Setctty = true // set controlling terminal

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

	if err := cmd.Start(); err != nil {
		_ = ptmx.Close() // Best effort.
		return nil, err
	}
	return ptmx, err
}
