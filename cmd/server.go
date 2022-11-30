/*

MIT License

Copyright (c) 2022~2023 wangqi

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

const (
	PACKAGE_STRING = "aprilsh"
	COMMAND_NAME   = "aprilsh-server"
	BUILD_VERSION  = "0.1.0"
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

// [-s] [-v] [-i LOCALADDR] [-p PORT[:PORT2]] [-c COLORS] [-l NAME=VALUE] [-- COMMAND...]
var usage = `Usage:
  ` + COMMAND_NAME + ` [--version] [--help]
  ` + COMMAND_NAME + ` [--server] [--verbose] [--ip ADDR] [--port PORT[:PORT2]] [--command] [command arguments]
Options:
  -h, --help     print this message
  -v, --version  print version information
  -s, --server   listen with SSH ip
  -i, --ip       listen ip
  -p, --port     listen port range
  -c, --command  server shell
      --verbose  verbose output
`

func main() {
	var version, help, server, verbose bool
	var desiredIP, desiredPort, command string

	flag.BoolVar(&verbose, "verbose", false, "verbose output")

	flag.BoolVar(&version, "version", false, "print version information")
	flag.BoolVar(&version, "v", false, "print version information")

	flag.BoolVar(&help, "help", false, "print this message")
	flag.BoolVar(&help, "h", false, "print this message")

	flag.BoolVar(&server, "server", false, "listen with SSH ip")
	flag.BoolVar(&server, "s", false, "listen with SSH ip")

	flag.StringVar(&desiredIP, "ip", "", "listen ip")
	flag.StringVar(&desiredIP, "i", "", "listen ip")

	flag.StringVar(&desiredPort, "port", "", "listen port range")
	flag.StringVar(&desiredPort, "p", "", "listen port range")

	flag.StringVar(&command, "command", "", "server shell")
	flag.StringVar(&command, "c", "", "server shell")

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
}
