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
	"strings"

	"github.com/ericwq/aprilsh/cmd"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	_ "github.com/ericwq/terminfo/base"
	"golang.org/x/sys/unix"
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
  -p, --port     server port (default 6000)
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

	flagSet.IntVar(&conf.port, "port", 6000, "server port")
	flagSet.IntVar(&conf.port, "p", 6000, "server port")

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

	cmd.SetNativeLocale()
}

type STMClient struct {
	ip   string
	port int
	key  string

	escapeKey        int
	escapePassKey    int
	escapePassKey2   int
	escapeRequiresIf bool
	escapeKeyHelp    string

	savedTermios *unix.Termios
	rawTermios   *unix.Termios
	windowSize   *unix.Winsize

	localFramebuffer terminal.Framebuffer
	newState         terminal.Framebuffer
	overlays         *frontend.OverlayManager
	network          *network.Transport[*statesync.UserStream, *statesync.Complete]
	display          *terminal.Display

	connectingNotification string
	repaintRequested       bool
	ifEntered              bool
	quitSequenceStarted    bool
	cleanShutdown          bool
	verbose                int
}

func newSTMClient(ip string, port int, key string, predictMode string, verbose int) *STMClient {
	c := STMClient{}

	c.ip = ip
	c.port = port
	c.key = key
	c.escapeKey = 0x1E
	c.escapePassKey = '^'
	c.escapePassKey2 = '^'
	c.escapeRequiresIf = false
	c.escapeKeyHelp = "?"
	c.localFramebuffer = terminal.NewFramebuffer2(1, 1)
	c.newState = terminal.NewFramebuffer2(1, 1)
	c.overlays = frontend.NewOverlayManager()

	var err error
	c.display, err = terminal.NewDisplay(true)
	if err != nil {
		return nil
	}

	c.repaintRequested = false
	c.ifEntered = false
	c.quitSequenceStarted = false
	c.cleanShutdown = false
	c.verbose = verbose

	switch predictMode {
	case predictionValues[0]: // always
		c.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Always)
	case predictionValues[1]: // never
		c.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Never)
	case predictionValues[2]: // adaptive
		c.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Adaptive)
	case predictionValues[3]: // experimental
		c.overlays.GetPredictionEngine().SetDisplayPreference(frontend.Experimental)
	default:
		return nil
	}

	return &c
}
