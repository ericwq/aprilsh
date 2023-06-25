// Copyright 2022~2023 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	_PACKAGE_STRING = "aprilsh"
	_COMMAND_NAME   = "aprilsh-client"
)

var (
	logW         *log.Logger
	logI         *log.Logger
	BuildVersion = "0.1.0" // ready for ldflags

	usage = `Usage:
  ` + _COMMAND_NAME + ` [--version] [--help]
  ` + _COMMAND_NAME + ` [--verbose] [--ip ADDR] [--port PORT] [--color COLORS]` +
		`]
Options:
  -h, --help     print this message
  -v, --version  print version information
      --verbose  verbose output
  -i, --ip       server ip
  -p, --port     server port
  -c, --color    xterm color
`
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

func printUsage(usage string) {
	fmt.Printf("%s", usage)
}

func parseFlags(progname string, args []string) (config *Config, output string, err error) {
	// https://eli.thegreenplace.net/2020/testing-flag-parsing-in-go-programs/
	flagSet := flag.NewFlagSet(progname, flag.ContinueOnError)
	var buf bytes.Buffer
	flagSet.SetOutput(&buf)

	var conf Config

	flagSet.IntVar(&conf.verbose, "verbose", 0, "verbose output")

	flagSet.BoolVar(&conf.version, "version", false, "print version information")
	flagSet.BoolVar(&conf.version, "v", false, "print version information")

	flagSet.StringVar(&conf.desiredIP, "ip", "", "server ip")
	flagSet.StringVar(&conf.desiredIP, "i", "", "server ip")

	flagSet.StringVar(&conf.desiredPort, "port", "6000", "server port")
	flagSet.StringVar(&conf.desiredPort, "p", "6000", "server port")

	flagSet.IntVar(&conf.color, "color", 0, "xterm color")
	flagSet.IntVar(&conf.color, "c", 0, "xterm color")

	err = flagSet.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}

	return &conf, buf.String(), nil
}

type Config struct {
	version     bool
	desiredIP   string
	desiredPort string
	verbose     int
	color       int
}

func (c *Config) isValid() bool {
	// just need version info
	if c.version {
		return true
	}

	// no required IP and Port info
	if c.desiredIP == "" || c.desiredPort == "" {
		return false
	}

	return true
}

func main() {
	conf, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp || !conf.isValid() {
		printUsage(usage)
		return
	} else if err != nil {
		logW.Printf("#main parseFlags failed: %s\n", output)
		return
	}

	if conf.version {
		printVersion()
		return
	}
}
