// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/network"
)

const (
	PACKAGE_STRING = "aprilsh"
	COMMAND_NAME   = "aprilsh-server"
	BUILD_VERSION  = "0.1.0"

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

func main() {
	// For security, make sure we don't dump core
	if err := encrypt.DisableDumpingCore(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	// verbose : don't close stdin/stdout/stderr
	var version, help, server, verbose bool
	var desiredIP, desiredPort string
	var locales localeFlag = make(localeFlag)
	var colors int

	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.BoolVar(&verbose, "v", false, "verbose output")

	flag.BoolVar(&version, "version", false, "print version information")

	flag.BoolVar(&help, "help", false, "print this message")
	flag.BoolVar(&help, "h", false, "print this message")

	flag.BoolVar(&server, "server", false, "listen with SSH ip")
	flag.BoolVar(&server, "s", false, "listen with SSH ip")

	flag.StringVar(&desiredIP, "ip", "", "listen ip")
	flag.StringVar(&desiredIP, "i", "", "listen ip")

	flag.StringVar(&desiredPort, "port", "", "listen port range")
	flag.StringVar(&desiredPort, "p", "", "listen port range")

	flag.Var(&locales, "locale", "locale list")
	flag.Var(&locales, "l", "locale list")

	flag.IntVar(&colors, "color", 0, "xterm color")
	flag.IntVar(&colors, "c", 0, "xterm color")

	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	if help {
		printUsage(os.Stdout, usage)
		return
	}
	if version {
		printVersion(os.Stdout)
		return
	}
	if server {
		if sshIP := getSSHip(); len(sshIP) != 0 {
			desiredIP = sshIP
			// fmt.Printf("#main sshIP=%s\n", desiredIP)
		}
	}

	if len(desiredPort) > 0 {
		// Sanity-check arguments

		// fmt.Printf("#main desiredPort=%s\n", desiredPort)
		_, _, ok := network.ParsePortRange(desiredPort, logW)
		if !ok {
			logW.Printf("%s: Bad UDP port range (%s)", COMMAND_NAME, desiredPort)
			return
		}
	}

	// get the non-flag command-line arguments.
	commandArgv := flag.Args()

	commandPath := ""
	withMotd := false

	// fmt.Printf("#main before get shell commandArgv=%q\n", commandArgv)
	// Get shell
	if len(commandArgv) == 0 {
		shell := os.Getenv("SHELL")
		if len(shell) == 0 {
			shell, _ = getShell() // another way to get shell path
		}

		shellPath := shell
		if len(shellPath) == 0 { // empty shell means Bourne shell
			shellPath = _PATH_BSHELL
		}

		commandPath = shellPath

		shellName := getShellNameFrom(shellPath)

		commandArgv = []string{shellName}

		withMotd = true
	}

	if len(commandPath) == 0 {
		// TODO the commandArgv is different from the default value,
		// consider to update commandArgv to be the same.
		commandPath = commandArgv[0]
	}

	// Adopt implementation locale
	setNativeLocale()
	if !isUtf8Locale() {
		nativeType := getCtype()
		nativeCharset := localeCharset()

		// apply locale-related environment variables from client
		clearLocaleVariables()
		for _, l := range locales {
			kv := strings.Split(l, "=")
			if len(kv) == 2 {
				os.Setenv(kv[0], kv[1])
			}
		}

		// check again
		setNativeLocale()
		if !isUtf8Locale() {
			clientType := getCtype()
			clientCharset := localeCharset()
			fmt.Fprintf(os.Stderr, "mosh-server needs a UTF-8 native locale to run.\n\n")
			fmt.Fprintf(os.Stderr, "Unfortunately, the local environment (%s) specifies\n"+
				"the character set \"%s\",\n\n", nativeType, nativeCharset)
			fmt.Fprintf(os.Stderr, "The client-supplied environment (%s) specifies\n"+
				"the character set \"%s\".\n\n", clientType, clientCharset)
		}
		return
	}
	fmt.Printf("#main commandPath=%q\n", commandPath)
	fmt.Printf("#main commandArgv=%q\n", commandArgv)
	fmt.Printf("#main withMotd=%t\n", withMotd)
	fmt.Printf("#main locales=%s\n", locales)
	fmt.Printf("#main colors=%d\n", colors)

	runServer(desiredIP, desiredPort, commandPath, commandArgv, colors, verbose, withMotd)
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

func runServer(desiredIP string, desiredPort string, commandPath string, commandArgv []string,
	colors int, verbose bool, withMotd bool) {
}
