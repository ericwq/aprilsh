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
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"log/slog"
	"log/syslog"

	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	utmp "github.com/ericwq/goutmp"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

const (
	_PATH_BSHELL = "/bin/sh"

	_VERBOSE_OPEN_PTS_FAIL    = 99  // test purpose
	_VERBOSE_SKIP_START_SHELL = 100 // test purpose
	_VERBOSE_SKIP_READ_PIPE   = 101 // skip start shell message
	_VERBOSE_LOG_TO_SYSLOG    = 514 // log to syslog
)

var usage = `Usage:
  ` + frontend.CommandServerName + ` [-v] [-h] [--auto N]
  ` + frontend.CommandServerName + ` [-b] [-t TERM] [-destination user@server.domain]
  ` + frontend.CommandServerName + ` [-s] [-vv[v]] [-i LOCALADDR] [-p PORT[:PORT2]] [-l NAME=VALUE] [-- command...]
Options:
---------------------------------------------------------------------------------------------------
  -v,  --version     print version information
  -h,  --help        print this message
  -a,  --auto        auto stop the server after N seconds
---------------------------------------------------------------------------------------------------
  -b,  --begin       begin a client connection
  -t,  --term        client TERM (such as xterm-256color, or alacritty or xterm-kitty)
  -d,  --destination in the form of user@host[:port], here the port is ssh server port (default 22)
---------------------------------------------------------------------------------------------------
  -s,  --server      listen with SSH ip
  -i,  --ip          listen with this ip/host
  -p,  --port        listen base port (default 60000)
  -l,  --locale      key-value pairs (such as LANG=UTF-8, you can have multiple -l options)
  -vv, --verbose     verbose log output (debug level, default no verbose)
  -vvv               verbose log output (trace level)
       -- command    shell command and options (note the space before command)
---------------------------------------------------------------------------------------------------
`

var failToStartShell = errors.New("fail to start shell")

var (
	userCurrentTest = false
	getShellTest    = false
	buildConfigTest = false
)

var (
	utmpSupport   bool
	syslogSupport bool
	syslogWriter  *syslog.Writer
	signals       frontend.Signals
	maxPortLimit  = 100 // assume 10 concurrent users, each owns 10 connections
)

// https://www.antoniojgutierrez.com/posts/2021-05-14-short-and-long-options-in-go-flags-pkg/
type localeFlag map[string]string

func init() {
	utmpSupport = utmp.HasUtmpSupport()
}

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
	child       bool       // begin a child process
	destination string     // [user@]hostname, destination string
	host        string     // target host/server
	user        string     // target user
	addSource   bool       // add source file to log

	commandPath string   // shell command path (absolute path)
	commandArgv []string // the positional (non-flag) command-line arguments.
	withMotd    bool

	// the serve func
	serve func(*os.File, *os.File, *io.PipeWriter, *statesync.Complete, // chan bool,
		*network.Transport[*statesync.Complete, *statesync.UserStream], int64, int64, string) error
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
			fmt.Printf("%s needs a UTF-8 native locale to run.\n", frontend.CommandServerName)
			fmt.Printf("Unfortunately, the local environment %s specifies "+
				"the character set \"%s\",\n", nativeType, nativeCharset)
			fmt.Printf("The client-supplied environment %s specifies "+
				"the character set \"%s\".\n", clientType, clientCharset)

			return "UTF-8 locale fail.", false
		}
	}
	return "", true
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

	// flagSet.IntVar(&conf.verbose, "verbose", 0, "verbose output")
	var v1, v2 bool
	flagSet.BoolVar(&v1, "vv", false, "verbose log output debug level")
	flagSet.BoolVar(&v1, "verbose", false, "verbose log output debug levle")
	flagSet.BoolVar(&v2, "vvv", false, "verbose log output trace level")

	flagSet.BoolVar(&conf.addSource, "source", false, "add source info to log")

	flagSet.IntVar(&conf.autoStop, "auto", 0, "auto stop after N seconds")
	flagSet.IntVar(&conf.autoStop, "a", 0, "auto stop after N seconds")

	flagSet.BoolVar(&conf.version, "version", false, "print version information")
	flagSet.BoolVar(&conf.version, "v", false, "print version information")

	flagSet.BoolVar(&conf.begin, "begin", false, "begin a client connection")
	flagSet.BoolVar(&conf.begin, "b", false, "begin a client connection")

	flagSet.BoolVar(&conf.child, "child", false, "begin child process")
	flagSet.BoolVar(&conf.child, "c", false, "begin child process")

	flagSet.BoolVar(&conf.server, "server", false, "listen with SSH ip")
	flagSet.BoolVar(&conf.server, "s", false, "listen with SSH ip")

	flagSet.StringVar(&conf.desiredIP, "ip", "", "listen ip")
	flagSet.StringVar(&conf.desiredIP, "i", "", "listen ip")

	flagSet.StringVar(&conf.desiredPort, "port", strconv.Itoa(frontend.DefaultPort), "listen port range")
	flagSet.StringVar(&conf.desiredPort, "p", strconv.Itoa(frontend.DefaultPort), "listen port range")

	flagSet.StringVar(&conf.term, "term", "", "client TERM")
	flagSet.StringVar(&conf.term, "t", "", "client TERM")

	flagSet.StringVar(&conf.destination, "destination", "", "destination string")

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

	// detremine verbose level
	if v1 {
		conf.verbose = util.DebugLevel
	} else if v2 {
		conf.verbose = util.TraceLevel
	}

	return &conf, buf.String(), nil
}

func printVersion() {
	fmt.Printf("%s package : %s server, %s\n",
		frontend.AprilshPackageName, frontend.AprilshPackageName, frontend.CommandServerName)
	frontend.PrintVersion()
}

func printUsage(hint string, usage ...string) {
	if hint != "" {
		fmt.Printf("Hints: %s\n%s", hint, usage)
	} else {
		fmt.Printf("%s", usage)
	}
}

func beginClientConn(conf *Config) { //(port string, term string) {
	// Unlike Dial, ListenPacket creates a connection without any
	// association with peers.
	conn, _ := net.ListenPacket("udp", ":0")
	// conn, err := net.ListenPacket("udp", ":0")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	defer conn.Close()

	dest, _ := net.ResolveUDPAddr("udp", "localhost:"+conf.desiredPort)
	// dest, err := net.ResolveUDPAddr("udp", "localhost:"+port)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// request from server
	// open aprilsh:TERM,user@server.domain
	request := fmt.Sprintf("%s%s,%s", frontend.AprilshMsgOpen, conf.term, conf.destination)
	conn.SetDeadline(time.Now().Add(time.Millisecond * 20))
	conn.WriteTo([]byte(request), dest)
	// n, err := conn.WriteTo([]byte(request), dest)
	// if err != nil {
	// 	fmt.Println("write to udp: ", err)
	// 	return
	// } else if n != len(request) {
	// 	fmt.Println("can't send correct query.")
	// 	return
	// }

	// read the response
	response := make([]byte, 128)
	conn.SetDeadline(time.Now().Add(time.Millisecond * 200))
	m, _, err := conn.ReadFrom(response)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s", string(response[:m]))
}

type workhorse struct {
	shell *os.Process
	ptmx  *os.File
}

type mainSrv struct {
	workers    map[int]workhorse
	runWorker  func(*Config, chan string, chan workhorse) error // worker
	exChan     chan string                                      // worker done or passing key
	whChan     chan workhorse                                   // workhorse
	downChan   chan bool                                        // shutdown mainSrv
	uxdownChan chan bool                                        // ux shutdown mainSrv
	maxPort    int                                              // max worker port
	timeout    int                                              // read udp time out,
	port       int                                              // main listen port
	conn       *net.UDPConn                                     // mainSrv listen port
	wg         sync.WaitGroup
}

func newMainSrv(conf *Config, runWorker func(*Config, chan string, chan workhorse) error) *mainSrv {
	m := mainSrv{}
	m.runWorker = runWorker
	m.port, _ = strconv.Atoi(conf.desiredPort)
	m.maxPort = m.port + 1
	m.workers = make(map[int]workhorse)
	m.downChan = make(chan bool, 1)
	m.uxdownChan = make(chan bool, 1)
	m.exChan = make(chan string, 1)
	m.whChan = make(chan workhorse, 1)
	m.timeout = 20

	return &m
}

// start mainSrv, which listen on the main udp port.
// each new client send a shake hands message to mainSrv. mainSrv response
// with the session key and target udp port for the new client.
// mainSrv is shutdown by SIGTERM and all sessions must be done.
// otherwise mainSrv will wait for the live session.
func (m *mainSrv) start(conf *Config) {
	// listen the port
	if err := m.listen(conf); err != nil {
		util.Log.Warn("listen failed", "error", err)
		return
	}

	// start main server waiting for open/close message.
	m.wg.Add(1)
	go func() {
		m.run(conf)
		m.wg.Done()
	}()

	// shutdown if the auto stop flag is set
	if conf.autoStop > 0 {
		time.AfterFunc(time.Duration(conf.autoStop)*time.Second, func() {
			m.downChan <- true
		})
	}
}

// to support multiple clients, mainServer listen on the specified port.
// start udp server for each new client.
func (m *mainSrv) listen(conf *Config) error {
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

// two kind of cmd: 60002 or 60002:shutdown.
// the latter is used to stop the specified shell.
// the former is used to clean the worker list.
func (m *mainSrv) cleanWorkers(cmd string) {
	ps := strings.Split(cmd, ":")
	if len(ps) == 1 {
		p, err := strconv.Atoi(cmd)
		if err != nil {
			util.Log.Debug("cleanWorkers receive wrong portStr", "portStr", cmd, "err", err)
		}

		// clean worker list
		delete(m.workers, p)
		// util.Log.Warn("#run clean worker","worker", ps[0])
	} else if ps[1] == "shutdown" {
		idx, err := strconv.Atoi(ps[0])
		if err != nil {
			util.Log.Warn("#run receive malform message", "portStr", cmd)
		} else if _, ok := m.workers[idx]; ok {
			// stop the specified shell
			m.workers[idx].shell.Kill()
			// util.Log.Debug("#run kill shell","shell", idx)
		}
	}
}

/*
upon receive frontend.AprilshMsgOpen message, run() stat a new worker
to serve the client, response to the client with choosen port number
and session key.

sample request  : open aprilsh:TERM,user@server.domain

sample response : open aprilsh:60001,31kR3xgfmNxhDESXQ8VIQw==

upon receive frontend.AprishMsgClose message, run() stop the worker
specified by port number.

sample request  : close aprilsh:60001

sample response : close aprilsh:done

when shutdown message is received (via SIGTERM or SIGINT), run() will send
sutdown message to all workers and wait for the workers to finish. when
-auto flag is set, run() will shutdown after specified seconds.
*/
func (m *mainSrv) run(conf *Config) {
	if m.conn == nil {
		return
	}
	// prepare to receive the signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	// clean up
	defer func() {
		signal.Stop(sig)
		if syslogSupport {
			syslogWriter.Info(fmt.Sprintf("stop listening on %s.", m.conn.LocalAddr()))
		}
		m.conn.Close()
		util.Log.Info("stop listening on", "port", m.port)
	}()

	buf := make([]byte, 128)
	shutdown := false

	if syslogSupport {
		syslogWriter.Info(fmt.Sprintf("start listening on %s.", m.conn.LocalAddr()))
	}

	printWelcome(os.Getpid(), m.port, nil)
	for {
		select {
		case portStr := <-m.exChan:
			m.cleanWorkers(portStr)
			// util.Log.Info("run some worker is done","port", portStr)
		case ss := <-sig:
			switch ss {
			case syscall.SIGHUP: // TODO:reload the config?
				util.Log.Info("got signal: SIGHUP")
			case syscall.SIGTERM, syscall.SIGINT:
				util.Log.Info("got signal: SIGTERM or SIGINT")
				shutdown = true
			}
		case <-m.downChan:
			// another way to shutdown besides signal
			shutdown = true
		default:
		}

		if shutdown {
			// util.Log.Debug("run","shutdown", shutdown)
			if len(m.workers) == 0 {
				return
			} else {
				// send kill message to the workers
				for i := range m.workers {
					m.workers[i].shell.Kill()
					// util.Log.Debug("stop shell","port", i)
				}
				// wait for workers to finish, set time out to prevent dead lock
				timeout := time.NewTimer(time.Duration(200) * time.Millisecond)
				for len(m.workers) > 0 {
					select {
					case portStr := <-m.exChan: // some worker is done
						m.cleanWorkers(portStr)
					case t := <-timeout.C:
						util.Log.Warn("run quit with timeout", "timeout", t)
						return
					default:
					}
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

		req := strings.TrimSpace(string(buf[0:n]))
		if strings.HasPrefix(req, frontend.AprilshMsgOpen) { // 'open aprilsh:'
			if len(m.workers) >= maxPortLimit {
				resp := m.writeRespTo(addr, frontend.AprishMsgClose, "over max port limit")
				util.Log.Warn("over max port limit", "request", req, "response", resp)
				continue
			}
			// prepare next port
			p := m.getAvailabePort()

			// open aprilsh:TERM,user@server.domain
			// prepare configuration
			conf2 := *conf
			conf2.desiredPort = fmt.Sprintf("%d", p)
			body := strings.Split(req, ":")
			content := strings.Split(body[1], ",")
			if len(content) != 2 {
				resp := m.writeRespTo(addr, frontend.AprilshMsgOpen, "malform request")
				util.Log.Warn("malform request", "request", req, "response", resp)
				continue
			}
			conf2.term = content[0]
			conf2.destination = content[1]

			// parse user and host from destination
			idx := strings.Index(content[1], "@")
			if idx > 0 && idx < len(content[1])-1 {
				conf2.host = content[1][idx+1:]
				conf2.user = content[1][:idx]
			} else {
				// return "target parameter should be in the form of User@Server", false
				resp := m.writeRespTo(addr, frontend.AprilshMsgOpen, "malform destination")
				util.Log.Warn("malform destination", "destination", content[1], "response", resp)

				continue
			}

			// we don't need to check if user exist, ssh already done that before

			// For security, make sure we don't dump core
			encrypt.DisableDumpingCore()

			// start the worker
			m.wg.Add(1)
			go func(conf *Config, exChan chan string, whChan chan workhorse) {
				m.runWorker(conf, exChan, whChan)
				m.wg.Done()
			}(&conf2, m.exChan, m.whChan)

			// blocking read the key from worker
			key := <-m.exChan

			// response session key and udp port to client
			msg := fmt.Sprintf("%d,%s", p, key)
			m.writeRespTo(addr, frontend.AprilshMsgOpen, msg)

			// blocking read the workhorse from runWorker
			wh := <-m.whChan
			if wh.shell != nil {
				m.workers[p] = wh
			}
		} else if strings.HasPrefix(req, frontend.AprishMsgClose) { // 'close aprilsh:[port]'
			pstr := strings.TrimPrefix(req, frontend.AprishMsgClose)
			port, err := strconv.Atoi(pstr)
			if err == nil {
				// find workhorse
				if wh, ok := m.workers[port]; ok {
					// kill the process, TODO SIGKILL or SIGTERM?
					wh.shell.Kill()

					m.writeRespTo(addr, frontend.AprishMsgClose, "done")
				} else {
					resp := m.writeRespTo(addr, frontend.AprishMsgClose, "port does not exist")
					util.Log.Warn("port does not exist", "request", req, "response", resp)
				}
			} else {
				resp := m.writeRespTo(addr, frontend.AprishMsgClose, "wrong port number")
				util.Log.Warn("wrong port number", "request", req, "response", resp)
			}
		} else {
			resp := m.writeRespTo(addr, frontend.AprishMsgClose, "unknow request")
			util.Log.Warn("unknow request", "request", req, "response", resp)
		}
	}
	/*
	   just for test purpose:

	   in aprilsh: we can use nc client to get the key and send it back to client.
	   we don't print it to the stdout as mosh did.

	   send udp request and read reply
	   % echo "open aprilsh:" | nc localhost 6000 -u -w 1
	   % echo "close aprilsh:6001" | nc localhost 6000 -u -w 1

	   send udp request to remote host
	   % ssh ide@localhost  "echo 'open aprilsh:' | nc localhost 6000 -u -w 1"
	*/
}

// return the minimal available port and increase the maxWorkerPort if necessary.
// shrink the max port number if possible
// https://coolaj86.com/articles/how-to-test-if-a-port-is-available-in-go/
// https://github.com/devlights/go-unix-domain-socket-example
func (m *mainSrv) getAvailabePort() (port int) {
	port = m.port
	if len(m.workers) > 0 {
		// sort the current ports
		ports := make([]int, 0, len(m.workers))
		for k := range m.workers {
			ports = append(ports, k)
		}
		sort.Ints(ports)
		// shrink max if possible
		m.maxPort = ports[len(ports)-1] + 1

		// util.Log.Info("getAvailabePort",
		// 	"ports", ports, "port", port, "maxPort", m.maxPort, "workers", len(m.workers))

		// check minimal available port
		for i := 0; i < m.maxPort-m.port; i++ {
			if i < len(ports) && port+i+1 < ports[i] {
				port = port + i + 1
				break
			}
		}

		// right most case
		if port == m.port {
			port = m.maxPort
			m.maxPort++
		}
	} else if len(m.workers) == 0 {
		// first port
		port = m.port + 1
		m.maxPort = port + 1
	}

	// util.Log.Info("getAvailabePort","port", port,"maxPort", m.maxPort,"workers", len(m.workers))
	return port
}

// write header and message to addr
func (m *mainSrv) writeRespTo(addr *net.UDPAddr, header, msg string) (resp string) {
	resp = fmt.Sprintf("%s%s\n", header, msg)
	// util.Log.Debug("writeRespTo","resp", resp)
	m.conn.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(m.timeout)))
	m.conn.WriteToUDP([]byte(resp), addr)
	return
}

func (m *mainSrv) wait() {
	m.wg.Wait()
	util.Log.Info("quit " + frontend.CommandServerName)
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
	util.Log.Info("start listening on", "port", port, "gitTag", frontend.GitTag, "pid", pid)
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

// TODO can't get current user.
func getCurrentUser() string {
	user, err := user.Current()
	if err != nil || userCurrentTest {
		// logW.Printf("#getCurrentUser report: %s\n", err)
		util.Log.Warn("Get current user", "error", err)
		return ""
	}

	return user.Username
}

// check unattached session and print warning message if there is any
// ignore current session
func warnUnattached(w io.Writer, ignoreHost string) {
	userName := getCurrentUser()

	// check unattached sessions
	unatttached := util.CheckUnattachedUtmpx(userName, ignoreHost, frontend.CommandServerName)

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
func startShell(pts *os.File, pr *io.PipeReader, utmpHost string, conf *Config) (*os.Process, error) {
	if conf.verbose == _VERBOSE_SKIP_START_SHELL {
		return nil, failToStartShell
	}
	// set IUTF8 if available
	if err := util.SetIUTF8(int(pts.Fd())); err != nil {
		return nil, err
	}

	var env []string

	// set TERM based on client TERM
	if conf.term != "" {
		env = append(env, "TERM="+conf.term)
	} else {
		env = append(env, "TERM=xterm-256color")
	}

	// clear STY environment variable so GNU screen regards us as top level
	// os.Unsetenv("STY")

	// get login user info, we already checked the user exist when ssh perform authentication.
	// users := strings.Split(conf.destination, "@")
	var changeUser bool
	if conf.user != getCurrentUser() {
		changeUser = true
	}
	u, _ := user.Lookup(conf.user)
	var uid int64
	var gid int64
	if changeUser {
		uid, _ = strconv.ParseInt(u.Uid, 10, 32)
		gid, _ = strconv.ParseInt(u.Gid, 10, 32)
	}

	// set base env
	// TODO should we put LOGNAME, MAIL into env?
	env = append(env, "PWD="+u.HomeDir)
	env = append(env, "HOME="+u.HomeDir) // it's important for shell to source .profile
	env = append(env, "USER="+conf.user)
	env = append(env, "SHELL="+conf.commandPath)
	env = append(env, fmt.Sprintf("TZ=%s", os.Getenv("TZ")))

	// TODO should we set ssh env ?
	env = append(env, fmt.Sprintf("SSH_CLIENT=%s", os.Getenv("SSH_CLIENT")))
	env = append(env, fmt.Sprintf("SSH_CONNECTION=%s", os.Getenv("SSH_CONNECTION")))

	// ask ncurses to send UTF-8 instead of ISO 2022 for line-drawing chars
	env = append(env, "NCURSES_NO_UTF8_ACS=1")

	util.Log.Debug("start shell check user", "user", u.Username, "gid", u.Gid, "HOME", u.HomeDir)
	util.Log.Info("start shell check env", "env", env)
	util.Log.Info("start shell check command",
		"commandPath", conf.commandPath, "commandArgv", conf.commandArgv)

	sysProcAttr := &syscall.SysProcAttr{}
	sysProcAttr.Setsid = true  // start a new session
	sysProcAttr.Setctty = true // set controlling terminal
	if changeUser {
		sysProcAttr.Credential = &syscall.Credential{ // change user
			Uid: uint32(uid),
			Gid: uint32(gid),
		}
	}

	procAttr := os.ProcAttr{
		Files: []*os.File{pts, pts, pts}, // use pts as stdin, stdout, stderr
		Dir:   u.HomeDir,
		Sys:   sysProcAttr,
		Env:   env,
	}

	// https://stackoverflow.com/questions/21705950/running-external-commands-through-os-exec-under-another-user
	//
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

	// set new title
	fmt.Fprintf(pts, "\x1B]0;%s %s:%s\a", frontend.CommandClientName, conf.destination, conf.desiredPort)

	encrypt.ReenableDumpingCore()

	/*
		additional logic for pty.StartWithAttrs() end
	*/

	// wait for serve() to release us
	if pr != nil && conf.verbose != _VERBOSE_SKIP_READ_PIPE {
		ch := make(chan bool, 0)
		timer := time.NewTimer(time.Duration(frontend.TimeoutIfNoConnect) * time.Millisecond)

		// util.Log.Debug("start shell message", "action", "wait", "port", conf.desiredPort)
		// add timeout for pipe read
		go func(pr *io.PipeReader, ch chan bool) {
			buf := make([]byte, 81)

			_, err := pr.Read(buf)
			if err != nil && errors.Is(err, io.EOF) {
				ch <- true
				// util.Log.Debug("start shell message", "action", "received", "port", conf.desiredPort)
			} else {
				// util.Log.Debug("start shell message", "action", "readFailed", "port", conf.desiredPort)
			}
		}(pr, ch)

		// waiting for time out or get the pipe reader send message
		select {
		case <-ch:
		case <-timer.C:
			pr.Close() // close pipe will stop the Read operation
			// util.Log.Debug("start shell message", "action", "timeout", "port", conf.desiredPort)
			return nil, fmt.Errorf("pipe read: %w", os.ErrDeadlineExceeded)
		}
		timer.Stop()

		pr.Close()
		util.Log.Info("start shell at", "pty", pts.Name())
	}

	proc, err := os.StartProcess(conf.commandPath, conf.commandArgv, &procAttr)
	if err != nil {
		return nil, err
	}
	return proc, nil
}

// worker started by mainSrv.run(). worker will listen on specified port and
// forward user input to shell (started by runWorker. the output is forward
// to the network.
func runWorker(conf *Config, exChan chan string, whChan chan workhorse) (err error) {
	defer func() {
		// notify this worker is done
		exChan <- conf.desiredPort
	}()

	/*
		If this variable is set to a positive integer number, it specifies how
		long (in seconds) apshd will wait to receive an update from the
		client before exiting.  Since aprilsh is very useful for mobile
		clients with intermittent operation and connectivity, we suggest
		setting this variable to a high value, such as 604800 (one week) or
		2592000 (30 days).  Otherwise, apshd will wait indefinitely for a
		client to reappear.  This variable is somewhat similar to the TMOUT
		variable found in many Bourne shells. However, it is not a login-session
		inactivity timeout; it only applies to network connectivity.
	*/
	networkTimeout := getTimeFrom("APRILSH_SERVER_NETWORK_TMOUT", 0)

	/*
		If this variable is set to a positive integer number, it specifies how
		long (in seconds) apshd will ignore SIGUSR1 while waiting to receive
		an update from the client.  Otherwise, SIGUSR1 will always terminate
		apshd. Users and administrators may implement scripts to clean up
		disconnected aprilsh sessions. With this variable set, a user or
		administrator can issue

		$ pkill -SIGUSR1 aprilsh-server

		to kill disconnected sessions without killing connected login
		sessions.
	*/
	networkSignaledTimeout := getTimeFrom("APRILSH_SERVER_SIGNAL_TMOUT", 0)

	// util.Log.Debug("runWorker", "networkTimeout", networkTimeout,
	// 	"networkSignaledTimeout", networkSignaledTimeout)

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
	// util.Log.Debug("init terminal size", "cols", windowSize.Col, "rows", windowSize.Row)

	// open parser and terminal
	savedLines := terminal.SaveLinesRowsOption
	terminal, err := statesync.NewComplete(int(windowSize.Col), int(windowSize.Row), savedLines)

	// open network
	blank := &statesync.UserStream{}
	server := network.NewTransportServer(terminal, blank, conf.desiredIP, conf.desiredPort)
	server.SetVerbose(uint(conf.verbose))
	// defer server.Close()

	/*
		// If server is run on a pty, then typeahead may echo and break mosh.pl's
		// detection of the CONNECT message.  Print it on a new line to bodge
		// around that.

		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Printf("\r\n")
		}
	*/

	exChan <- server.GetKey() // send the key to run()

	// in mosh: the parent print this to stderr.
	// fmt.Printf("#runWorker %s CONNECT %s %s\n", COMMAND_NAME, network.Port(), network.GetKey())
	// printWelcome(os.Stdout, os.Getpid(), os.Stdin)

	// prepare for openPTS fail
	if conf.verbose == _VERBOSE_OPEN_PTS_FAIL {
		windowSize = nil
	}

	ptmx, pts, err := openPTS(windowSize)
	if err != nil {
		util.Log.Warn("openPTS fail", "error", err)
		whChan <- workhorse{}
		return err
	}
	defer func() {
		ptmx.Close()
		// pts.Close()
	}() // Best effort.
	// fmt.Printf("#runWorker openPTS successfully.\n")

	// use pipe to signal when to start shell
	// pw and pr is close inside serve() and startShell()
	pr, pw := io.Pipe()

	// prepare host field for utmp record
	utmpHost := fmt.Sprintf("%s:%s", frontend.CommandServerName, server.GetServerPort())

	// add utmp entry
	if utmpSupport {
		ok := util.AddUtmpx(pts, utmpHost)
		if !ok {
			utmpSupport = false
			util.Log.Warn("#runWorker can't update utmp")
		}
	}

	// start the udp server, serve the udp request
	var wg sync.WaitGroup
	wg.Add(1)
	// waitChan := make(chan bool)
	// go conf.serve(ptmx, pw, terminal, waitChan, network, networkTimeout, networkSignaledTimeout)
	go func() {
		conf.serve(ptmx, pts, pw, terminal, server, networkTimeout, networkSignaledTimeout, conf.user)
		exChan <- fmt.Sprintf("%s:shutdown", conf.desiredPort)
		wg.Done()
	}()

	// TODO update last log ?
	// util.UpdateLastLog(ptmxName, getCurrentUser(), utmpHost)

	defer func() { // clear utmp entry
		if utmpSupport {
			util.ClearUtmpx(pts)
		}
	}()

	util.Log.Info("start listening on", "port", conf.desiredPort, "clientTERM", conf.term)

	// start the shell with pts
	shell, err := startShell(pts, pr, utmpHost, conf)
	pts.Close() // it's copied by shell process, it's safe to close it here.
	if err != nil {
		util.Log.Warn("startShell fail", "error", err)
		whChan <- workhorse{}
	} else {

		whChan <- workhorse{shell, ptmx}
		// wait for the shell to finish.
		var state *os.ProcessState
		state, err = shell.Wait()
		if err != nil || state.Exited() {
			if err != nil {
				util.Log.Warn("shell.Wait fail", "error", err, "state", state)
				// } else {
				// util.Log.Debug("shell.Wait quit", "state.exited", state.Exited())
			}
		}
	}

	// wait serve to finish
	wg.Wait()
	util.Log.Info("stop listening on", "port", conf.desiredPort)

	// fmt.Printf("[%s is exiting.]\n", frontend.COMMAND_SERVER_NAME)
	// https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/
	// https://www.prakharsrivastav.com/posts/golang-context-and-cancellation/

	// util.Log.Debug("runWorker quit", "port", conf.desiredPort)
	return err
}

func serve(ptmx *os.File, pts *os.File, pw *io.PipeWriter, complete *statesync.Complete, // waitChan chan bool,
	server *network.Transport[*statesync.Complete, *statesync.UserStream],
	networkTimeout int64, networkSignaledTimeout int64, user string) error {
	// scale timeouts
	networkTimeoutMs := networkTimeout * 1000
	networkSignaledTimeoutMs := networkSignaledTimeout * 1000

	lastRemoteNum := server.GetRemoteStateNum()
	var connectedUtmp bool
	var forceConnectionChangEvt bool
	var savedAddr net.Addr

	if syslogSupport {
		util.Log.Info("user session begin", "user", user)
	}

	var terminalToHost strings.Builder
	var timeSinceRemoteState int64

	// var networkChan chan frontend.Message
	networkChan := make(chan frontend.Message, 1)
	fileChan := make(chan frontend.Message, 1)
	fileDownChan := make(chan any, 1)
	networkDownChan := make(chan any, 1)

	eg := errgroup.Group{}
	// read from socket
	eg.Go(func() error {
		frontend.ReadFromNetwork(1, networkChan, networkDownChan, server.GetConnection())
		return nil
	})

	// read from pty master file
	// the following doesn't work for terminal, when the shell start, the file
	// is reset back to blocking IO mode.
	// syscall.SetNonblock(int(ptmx.Fd()), true)
	eg.Go(func() error {
		frontend.ReadFromFile(10, fileChan, fileDownChan, ptmx)
		return nil
	})

	// intercept signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGINT, syscall.SIGTERM)

	childReleased := false
	largeFeed := make(chan string, 1)

mainLoop:
	for {
		timeout := math.MaxInt16
		now := time.Now().UnixMilli()

		timeout = min(timeout, server.WaitTime()) // network.WaitTime cost time
		w0 := timeout
		w1 := complete.WaitTime(now)
		timeout = min(timeout, w1)
		// timeout = terminal.Min(timeout, complete.WaitTime(now))

		if server.GetRemoteStateNum() > 0 || server.ShutdownInProgress() {
			timeout = min(timeout, 5000)
		}

		// The server goes completely asleep if it has no remote peer.
		// We may want to wake up sooner.
		var networkSleep int64
		if networkTimeoutMs > 0 {
			rs := server.GetLatestRemoteState()
			networkSleep = networkTimeoutMs - (now - rs.GetTimestamp())
			if networkSleep < 0 {
				networkSleep = 0
			} else if networkSleep > math.MaxInt16 {
				networkSleep = math.MaxInt16
			}
			timeout = min(timeout, int(networkSleep))
		}

		now = time.Now().UnixMilli()
		p := server.GetLatestRemoteState()
		timeSinceRemoteState = now - p.GetTimestamp()
		terminalToHost.Reset()

		util.Log.Debug("mainLoop", "port", server.GetServerPort(),
			"network.WaitTime", w0, "complete.WaitTime", w1, "timeout", timeout)
		timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
		select {
		case <-timer.C:
			util.Log.Debug("mainLoop", "complete", complete.WaitTime(now),
				"networkSleep", networkSleep, "timeout", timeout)
		case s := <-sigChan:
			signals.Handler(s)
		case socketMsg := <-networkChan: // packet received from the network
			if socketMsg.Err != nil {
				// TODO handle "use of closed network connection" error?
				util.Log.Warn("read from network", "error", socketMsg.Err)
				continue mainLoop
			}
			server.ProcessPayload(socketMsg.Data)
			p = server.GetLatestRemoteState()
			timeSinceRemoteState = now - p.GetTimestamp()

			// is new user input available for the terminal?
			if server.GetRemoteStateNum() != lastRemoteNum {
				lastRemoteNum = server.GetRemoteStateNum()

				us := &statesync.UserStream{}
				us.ApplyString(server.GetRemoteDiff())

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
							server.StartShutdown()
						}
						winSize.Col = uint16(res.Width)
						winSize.Row = uint16(res.Height)
						if err = unix.IoctlSetWinsize(int(ptmx.Fd()), unix.TIOCSWINSZ, winSize); err != nil {
							fmt.Printf("#serve ioctl TIOCSWINSZ %s", err)
							server.StartShutdown()
						}
						// util.Log.Debug("input from remote", "col", winSize.Col, "row", winSize.Row)
						if !childReleased {
							// only do once
							server.InitSize(res.Width, res.Height)
						}
					}
					terminalToHost.WriteString(complete.ActOne(action))
				}

				if terminalToHost.Len() > 0 {
					util.Log.Debug("input from remote", "arise", "socket", "data", terminalToHost.String())
				}

				if !us.Empty() {
					// register input frame number for future echo ack
					complete.RegisterInputFrame(lastRemoteNum, now)
				}

				// update client with new state of terminal
				if !server.ShutdownInProgress() {
					server.SetCurrentState(complete)
				}

				if utmpSupport || syslogSupport {
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
					// HAVE_SYSLOG - log connect to syslog
					//
					// update utmp entry if we have become "connected"
					if forceConnectionChangEvt || !reflect.DeepEqual(savedAddr, socketMsg.RAddr) {
						savedAddr = socketMsg.RAddr
						host := savedAddr.(*net.UDPAddr).IP.String() // default host name is ip string
						// convert savedAddr to host name
						// hostList, e := net.LookupAddr(host)
						// if e == nil {
						// 	host = hostList[0] // got the host name, use the first one
						// }

						if utmpSupport {
							util.ClearUtmpx(pts)
							utmpHost := fmt.Sprintf("%s via %s:%s", host, frontend.CommandServerName, server.GetServerPort())
							util.AddUtmpx(pts, utmpHost)
							connectedUtmp = true
						}
						if syslogSupport {
							util.Log.Info("connected from remote host", "user", user, "host", host)
							syslogWriter.Info(fmt.Sprintf("user %s connected from host: %s -> port %s",
								user, server.GetRemoteAddr(), server.GetServerPort()))
						}
					}
				}

				// upon receive network message, perform the following one time action,
				// release startShell() to start login session
				if !childReleased {
					if err := pw.Close(); err != nil {
						util.Log.Error("send start shell message failed", "error", err)
					}
					// util.Log.Debug("start shell message", "action", "send")
					childReleased = true
				}
			}
		case remains := <-largeFeed:
			if !server.ShutdownInProgress() {
				out := complete.ActLarge(remains, largeFeed)
				terminalToHost.WriteString(out)

				util.Log.Debug("ouput from host", "arise", "remains", "input", out)

				// update client with new state of terminal
				server.SetCurrentState(complete)
			}
		case masterMsg := <-fileChan:
			// input from the host needs to be fed to the terminal
			if !server.ShutdownInProgress() {

				// If the pty slave is closed, reading from the master can fail with
				// EIO (see #264).  So we treat errors on read() like EOF.
				if masterMsg.Err != nil {
					if len(masterMsg.Data) > 0 {
						util.Log.Warn("read from master", "error", masterMsg.Err)
					}
					if !signals.AnySignal() { // avoid conflict with signal
						util.Log.Debug("shutdown", "from", "read file failed", "port", server.GetServerPort())
						// &fs.PathError{Op:"read", Path:"/dev/ptmx", Err:0x5}
						server.StartShutdown()
					}
				} else {
					out := complete.ActLarge(masterMsg.Data, largeFeed)
					terminalToHost.WriteString(out)

					util.Log.Debug("output from host", "arise", "master", "ouput", masterMsg.Data, "input", out)

					// update client with new state of terminal
					server.SetCurrentState(complete)
				}
			}
		}

		// write user input and terminal writeback to the host
		if terminalToHost.Len() > 0 {
			_, err := ptmx.WriteString(terminalToHost.String())
			if err != nil && !signals.AnySignal() { // avoid conflict with signal
				server.StartShutdown()
			}

			util.Log.Debug("input to host", "arise", "merge-", "data", terminalToHost.String())
		}

		idleShutdown := false
		if networkTimeoutMs > 0 && networkTimeoutMs <= timeSinceRemoteState {
			// if network timeout is set and over networkTimeoutMs quit this session.
			idleShutdown = true
			// fmt.Printf("Network idle for %d seconds.\n", timeSinceRemoteState/1000)
			util.Log.Info("Network idle for x seconds", "seconds", timeSinceRemoteState/1000)
		}

		if signals.GotSignal(syscall.SIGUSR1) {
			if networkSignaledTimeoutMs == 0 || networkSignaledTimeoutMs <= timeSinceRemoteState {
				idleShutdown = true
				// fmt.Printf("Network idle for %d seconds when SIGUSR1 received.\n", timeSinceRemoteState/1000)
				util.Log.Info("Network idle for x seconds when SIGUSR1 received", "seconds",
					timeSinceRemoteState/1000)
			}
		}

		if signals.AnySignal() || idleShutdown {
			util.Log.Debug("got signal: start shutdown",
				"HasRemoteAddr", server.HasRemoteAddr(),
				"ShutdownInProgress", server.ShutdownInProgress())
			signals.Clear()
			// shutdown signal
			if server.HasRemoteAddr() && !server.ShutdownInProgress() {
				server.StartShutdown()
			} else {
				util.Log.Debug("got signal: break loop",
					"HasRemoteAddr", server.HasRemoteAddr(),
					"ShutdownInProgress", server.ShutdownInProgress())
				break
			}
		}

		// quit if our shutdown has been acknowledged
		if server.ShutdownInProgress() && server.ShutdownAcknowledged() {
			util.Log.Debug("shutdown", "from", "acked", "port", server.GetServerPort())
			break
		}

		// quit after shutdown acknowledgement timeout
		if server.ShutdownInProgress() && server.ShutdownAckTimedout() {
			util.Log.Warn("shutdown", "from", "act timeout", "port", server.GetServerPort())
			break
		}

		// quit if we received and acknowledged a shutdown request
		if server.CounterpartyShutdownAckSent() {
			util.Log.Warn("shutdown", "from", "peer acked", "port", server.GetServerPort())
			break
		}

		// update utmp if has been more than 30 seconds since heard from client
		if utmpSupport && connectedUtmp && timeSinceRemoteState > 30000 {
			if !server.Awaken(now) {
				util.ClearUtmpx(pts)
				utmpHost := fmt.Sprintf("%s:%s", frontend.CommandServerName, server.GetServerPort())
				util.AddUtmpx(pts, utmpHost)
				connectedUtmp = false
				// util.Log.Info("serve doesn't heard from client over 16 minutes.")
			}
		}

		if complete.SetEchoAck(now) && !server.ShutdownInProgress() {
			// update client with new echo ack
			server.SetCurrentState(complete)
		}

		// util.Log.Debug("mainLoop","point", 500)
		err := server.Tick()
		if err != nil {
			util.Log.Warn("#serve send failed", "error", err)
		}
		// util.Log.Debug("mainLoop","point", "d")

		now = time.Now().UnixMilli()
		if server.GetRemoteStateNum() == 0 && server.ShutdownInProgress() {
			// abort if no connection over TimeoutIfNoConnect seconds

			util.Log.Warn("No connection within x seconds", "seconds", frontend.TimeoutIfNoConnect/1000,
				"timeout", "shutdown", "port", server.GetServerPort())
			break
		} else if server.GetRemoteStateNum() != 0 && timeSinceRemoteState >= frontend.TimeoutIfNoResp {
			// if no response from client over TimeoutIfNoResp seconds
			// if now-server.GetSentStateLastTimestamp() >= frontend.TimeoutIfNoResp-network.SERVER_ASSOCIATION_TIMEOUT {
			if !server.Awaken(now) {
				// abort if no request send over TimeoutIfNoResp seconds
				util.Log.Warn("Time out for no client request", "seconds", frontend.TimeoutIfNoResp/1000,
					"port", server.GetServerPort(), "timeSinceRemoteState", timeSinceRemoteState)
				break
			}
			// }
		}
	}

	// stop signal and network
	signal.Stop(sigChan)
	server.Close()

	// shutdown the goroutines: file reader and network reader
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

	if syslogSupport {
		util.Log.Info("user session end", "user", user)
		syslogWriter.Info(fmt.Sprintf("user %s disconnected from host: %s -> port %s",
			user, server.GetRemoteAddr(), server.GetServerPort()))
	}

	return nil
}

const (
	unixsockNetwork = "unixgram"
)

// "/tmp/aprilsh.sock"
var unixsockAddr string = filepath.Join(os.TempDir(), "aprilsh.sock")

type uxClient struct {
	connection net.Conn
}

func newUxClient() (client *uxClient, err error) {
	client = &uxClient{}
	client.connection, err = net.Dial(unixsockNetwork, unixsockAddr)
	return
}

func (uc *uxClient) send(msg string) (err error) {
	_, err = uc.connection.Write([]byte(msg))
	util.Log.Debug("uxClient send", "message", msg)
	return
}

func (uc *uxClient) close() (err error) {
	return uc.connection.Close()
}

func uxCleanup() (err error) {
	if _, err = os.Stat(unixsockAddr); err == nil {
		if err = os.RemoveAll(unixsockAddr); err != nil {
			return err
		}
	}
	err = nil // doesn't exist
	return
}

func (m *mainSrv) uxListen() (conn *net.UnixConn, err error) {
	if err = uxCleanup(); err != nil {
		return
	}

	addr, _ := net.ResolveUnixAddr(unixsockNetwork, unixsockAddr)
	conn, err = net.ListenUnixgram("unixgram", addr)
	os.Chmod(unixsockAddr, 0666)

	if err != nil {
		return nil, err
	}
	return
}

// get a message from unix docket and forward it to exChan
func (m *mainSrv) uxServe(conn *net.UnixConn, timeout int) {
	// prepare to receive the signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	// clean up
	defer func() {
		conn.Close()
		uxCleanup()
		// util.Log.Info("uxServe stopped")
	}()

	// util.Log.Info("uxServe started")
	var buf [1024]byte
	shutdown := false
	for {
		select {
		case ss := <-sig:
			switch ss {
			case syscall.SIGHUP: // TODO:reload the config?
				util.Log.Info("got signal: SIGHUP", "receiver", "uxServe")
			case syscall.SIGTERM, syscall.SIGINT:
				util.Log.Info("got signal: SIGTERM or SIGINT", "receiver", "uxServe")
				shutdown = true
			}
		case <-m.uxdownChan:
			shutdown = true
		default:
		}

		if shutdown {
			return
		}

		conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
		n, err := conn.Read(buf[:])
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
		}
		resp := string(buf[:n])

		util.Log.Debug("uxServe forward message to exChan", "resp", resp)
		m.exChan <- resp
	}
}

func (m *mainSrv) start2(conf *Config) {
	// listen the port
	if err := m.listen(conf); err != nil {
		util.Log.Warn("listen failed", "error", err)
		return
	}

	uxConn, err := m.uxListen()
	if err != nil {
		util.Log.Warn("listen unix domain socket failed", "error", err)
		return
	}

	// start main server waiting for open/close message.
	m.wg.Add(1)
	go func() {
		m.run2(conf)
		m.wg.Done()
	}()

	// start unix domain socket (datagram)
	m.wg.Add(1)
	go func() {
		m.uxServe(uxConn, 2)
		m.wg.Done()
	}()

	// shutdown if the auto stop flag is set
	if conf.autoStop > 0 {
		time.AfterFunc(time.Duration(conf.autoStop)*time.Second, func() {
			m.downChan <- true
		})
	}
}

func (m *mainSrv) run2(conf *Config) {
	if m.conn == nil {
		return
	}
	// prepare to receive the signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	// clean up
	defer func() {
		signal.Stop(sig)
		if syslogSupport {
			syslogWriter.Info(fmt.Sprintf("stop listening on %s.", m.conn.LocalAddr()))
		}
		m.conn.Close()
		util.Log.Info("stop listening on", "port", m.port)
	}()

	buf := make([]byte, 128)
	shutdown := false

	if syslogSupport {
		syslogWriter.Info(fmt.Sprintf("start listening on %s.", m.conn.LocalAddr()))
	}

	printWelcome(os.Getpid(), m.port, nil)
	for {
		select {
		case portStr := <-m.exChan:
			m.cleanWorkers(portStr)
			// util.Log.Info("run some worker is done","port", portStr)
		case ss := <-sig:
			switch ss {
			case syscall.SIGHUP: // TODO:reload the config?
				util.Log.Info("got signal: SIGHUP", "receiver", "run2")
			case syscall.SIGTERM, syscall.SIGINT:
				util.Log.Info("got signal: SIGTERM or SIGINT", "receiver", "run2")
				shutdown = true
			}
		case <-m.downChan:
			// another way to shutdown besides signal
			shutdown = true
		default:
		}

		if shutdown {
			// shutdown uxServe
			m.uxdownChan <- true

			// util.Log.Info("run2", "shutdown", shutdown)
			if len(m.workers) == 0 {
				return
			} else {
				// send kill message to the workers
				for i := range m.workers {
					m.workers[i].shell.Kill()
					// util.Log.Debug("stop shell","port", i)
				}
				// wait for workers to finish, set time out to prevent dead lock
				timeout := time.NewTimer(time.Duration(200) * time.Millisecond)
				for len(m.workers) > 0 {
					select {
					case portStr := <-m.exChan: // some worker is done
						m.cleanWorkers(portStr)
					case t := <-timeout.C:
						util.Log.Warn("run quit with timeout", "timeout", t)
						return
					default:
					}
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

		req := strings.TrimSpace(string(buf[0:n]))
		if strings.HasPrefix(req, frontend.AprilshMsgOpen) { // 'open aprilsh:'
			if len(m.workers) >= maxPortLimit {
				resp := m.writeRespTo(addr, frontend.AprishMsgClose, "over max port limit")
				util.Log.Warn("over max port limit", "request", req, "response", resp)
				continue
			}
			// prepare next port
			p := m.getAvailabePort()

			// open aprilsh:TERM,user@server.domain
			// prepare configuration
			conf2 := *conf
			conf2.desiredPort = fmt.Sprintf("%d", p)
			body := strings.Split(req, ":")
			content := strings.Split(body[1], ",")
			if len(content) != 2 {
				resp := m.writeRespTo(addr, frontend.AprilshMsgOpen, "malform request")
				util.Log.Warn("malform request", "request", req, "response", resp)
				continue
			}
			conf2.term = content[0]
			conf2.destination = content[1]

			// parse user and host from destination
			idx := strings.Index(content[1], "@")
			if idx > 0 && idx < len(content[1])-1 {
				conf2.host = content[1][idx+1:]
				conf2.user = content[1][:idx]
			} else {
				// return "target parameter should be in the form of User@Server", false
				resp := m.writeRespTo(addr, frontend.AprilshMsgOpen, "malform destination")
				util.Log.Warn("malform destination", "destination", content[1], "response", resp)

				continue
			}

			// we don't need to check if user exist, ssh already done that before

			shell, err := startChild(&conf2)
			if err != nil {
				if errors.Is(err, syscall.EPERM) {
					util.Log.Warn("operation not permitted")
				} else {
					util.Log.Warn("can't start child", "error", err)
					fmt.Printf("can't start child, error=%#v\n", err)
				}
				continue
			}
			util.Log.Debug("start child successfully, wait for the key.")

			m.wg.Add(1)
			go func() { // avatar is looking after the child process
				ps, err := shell.Wait()
				if err != nil {
					util.Log.Warn("start child return", "error", err, "ProcessState", ps)
				}
				m.wg.Done()
			}()

			// // start the worker
			// m.wg.Add(1)
			// go func(conf *Config, exChan chan string, whChan chan workhorse) {
			// 	m.runWorker(conf, exChan, whChan)
			// 	m.wg.Done()
			// }(&conf2, m.exChan, m.whChan)

			// blocking read the key from worker
			timer := time.NewTimer(time.Duration(20) * time.Millisecond)
			select {
			case <-timer.C:
				resp := m.writeRespTo(addr, frontend.AprilshMsgOpen, "get key timeout")
				util.Log.Warn("get key timeout", "request", req, "response", resp)
				continue
			case key := <-m.exChan:
				util.Log.Debug("run2 got key")
				// response session key and udp port to client
				msg := fmt.Sprintf("%d,%s", p, key)
				m.writeRespTo(addr, frontend.AprilshMsgOpen, msg)

				wh := workhorse{shell: shell}
				if wh.shell != nil {
					m.workers[p] = wh
				}
			}
		} else if strings.HasPrefix(req, frontend.AprishMsgClose) { // 'close aprilsh:[port]'
			pstr := strings.TrimPrefix(req, frontend.AprishMsgClose)
			port, err := strconv.Atoi(pstr)
			if err == nil {
				// find workhorse
				if wh, ok := m.workers[port]; ok {
					// kill the process, TODO SIGKILL or SIGTERM?
					wh.shell.Kill()

					m.writeRespTo(addr, frontend.AprishMsgClose, "done")
				} else {
					resp := m.writeRespTo(addr, frontend.AprishMsgClose, "port does not exist")
					util.Log.Warn("port does not exist", "request", req, "response", resp)
				}
			} else {
				resp := m.writeRespTo(addr, frontend.AprishMsgClose, "wrong port number")
				util.Log.Warn("wrong port number", "request", req, "response", resp)
			}
		} else {
			resp := m.writeRespTo(addr, frontend.AprishMsgClose, "unknow request")
			util.Log.Warn("unknow request", "request", req, "response", resp)
		}
	}
}

func startChild(conf *Config) (*os.Process, error) {
	// conf{term,user,desiredPort,destination}

	// use the root's SHELL as replacement for user SHELL
	shell := os.Getenv("SHELL")
	if shell == "" {
		err := errors.New("can't get shell from SHELL")
		util.Log.Warn("startChild", "error", err)
		return nil, err
	}

	util.Log.Debug("startChild", "user", conf.user, "term", conf.term,
		"desiredPort", conf.desiredPort, "destination", conf.destination)

	// specify child process
	commandPath := "/usr/bin/apshd"
	commandArgv := []string{
		commandPath, "-child", "-port", conf.desiredPort, "-destination", conf.destination,
		"-term", conf.term,
	}

	// inherit vervoce and source options form parent
	if conf.verbose == util.DebugLevel {
		commandArgv = append(commandArgv, "-vv")
	} else if conf.verbose == util.TraceLevel {
		commandArgv = append(commandArgv, "-vvv")
	}
	if conf.addSource {
		commandArgv = append(commandArgv, "-source")
	}

	// var pts *os.File
	// var pr *io.PipeReader
	// var utmpHost string

	// if conf.verbose == _VERBOSE_SKIP_START_SHELL {
	// 	return nil, failToStartShell
	// }
	// set IUTF8 if available
	// if err := util.SetIUTF8(int(pts.Fd())); err != nil {
	// 	return nil, err
	// }

	var env []string

	// set TERM based on client TERM
	if conf.term != "" {
		env = append(env, "TERM="+conf.term)
	} else {
		env = append(env, "TERM=xterm-256color")
	}

	// clear STY environment variable so GNU screen regards us as top level
	// os.Unsetenv("STY")

	// get login user info, we already checked the user exist when ssh perform authentication.
	u, _ := user.Lookup(conf.user)
	// uid, _ := strconv.ParseInt(u.Uid, 10, 32)
	// gid, _ := strconv.ParseInt(u.Gid, 10, 32)
	util.Log.Debug("startChild check user", "user", u.Username, "gid", u.Gid, "HOME", u.HomeDir)

	// set base env
	// TODO should we put LOGNAME, MAIL into env?
	env = append(env, "PWD="+u.HomeDir)
	env = append(env, "HOME="+u.HomeDir) // it's important for shell to source .profile
	env = append(env, "USER="+conf.user)
	env = append(env, "SHELL="+shell)
	env = append(env, fmt.Sprintf("TZ=%s", os.Getenv("TZ")))

	// TODO should we set ssh env ?
	env = append(env, fmt.Sprintf("SSH_CLIENT=%s", os.Getenv("SSH_CLIENT")))
	env = append(env, fmt.Sprintf("SSH_CONNECTION=%s", os.Getenv("SSH_CONNECTION")))

	// ask ncurses to send UTF-8 instead of ISO 2022 for line-drawing chars
	env = append(env, "NCURSES_NO_UTF8_ACS=1")

	util.Log.Debug("startChild env:", "env", env)
	util.Log.Debug("startChild command:", "commandPath", commandPath, "commandArgv", commandArgv)

	sysProcAttr := &syscall.SysProcAttr{}
	sysProcAttr.Setsid = true // start a new session
	// sysProcAttr.Setctty = true                    // set controlling terminal
	// sysProcAttr.Credential = &syscall.Credential{ // change user
	// 	Uid: uint32(uid),
	// 	Gid: uint32(gid),
	// }

	procAttr := os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}, // use pts as stdin, stdout, stderr
		Dir:   u.HomeDir,
		Sys:   sysProcAttr,
		Env:   env,
	}

	// https://stackoverflow.com/questions/21705950/running-external-commands-through-os-exec-under-another-user
	//
	// if conf.withMotd && !motdHushed() {
	// 	// For Ubuntu, try and print one of {,/var}/run/motd.dynamic.
	// 	// This file is only updated when pam_motd is run, but when
	// 	// mosh-server is run in the usual way with ssh via the script,
	// 	// this always happens.
	// 	// XXX Hackish knowledge of Ubuntu PAM configuration.
	// 	// But this seems less awful than build-time detection with autoconf.
	// 	if !printMotd(pts, "/run/motd.dynamic") {
	// 		printMotd(pts, "/var/run/motd.dynamic")
	// 	}
	// 	// Always print traditional /etc/motd.
	// 	printMotd(pts, "/etc/motd")
	//
	// 	warnUnattached(pts, utmpHost)
	// }
	//
	// // set new title
	// fmt.Fprintf(pts, "\x1B]0;%s %s:%s\a", frontend.CommandClientName, conf.destination, conf.desiredPort)
	//
	// encrypt.ReenableDumpingCore()

	/*
		additional logic for pty.StartWithAttrs() end
	*/

	// wait for serve() to release us
	// if pr != nil && conf.verbose != _VERBOSE_SKIP_READ_PIPE {
	// 	ch := make(chan bool, 0)
	// 	timer := time.NewTimer(time.Duration(frontend.TimeoutIfNoConnect) * time.Millisecond)
	//
	// 	// util.Log.Debug("start shell message", "action", "wait", "port", conf.desiredPort)
	// 	// add timeout for pipe read
	// 	go func(pr *io.PipeReader, ch chan bool) {
	// 		buf := make([]byte, 81)
	//
	// 		_, err := pr.Read(buf)
	// 		if err != nil && errors.Is(err, io.EOF) {
	// 			ch <- true
	// 			// util.Log.Debug("start shell message", "action", "received", "port", conf.desiredPort)
	// 		} else {
	// 			// util.Log.Debug("start shell message", "action", "readFailed", "port", conf.desiredPort)
	// 		}
	// 	}(pr, ch)
	//
	// 	// waiting for time out or get the pipe reader send message
	// 	select {
	// 	case <-ch:
	// 	case <-timer.C:
	// 		pr.Close() // close pipe will stop the Read operation
	// 		// util.Log.Debug("start shell message", "action", "timeout", "port", conf.desiredPort)
	// 		return nil, fmt.Errorf("pipe read: %w", os.ErrDeadlineExceeded)
	// 	}
	// 	timer.Stop()
	//
	// 	pr.Close()
	// 	util.Log.Info("start shell at", "pty", pts.Name())
	// }

	proc, err := os.StartProcess(commandPath, commandArgv, &procAttr)
	if err != nil {
		return nil, err
	}
	return proc, nil
}

func runChild(conf *Config) (err error) {
	// name := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%d.%s", frontend.CommandServerName, os.Getpid(), "log"))
	// file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	// defer file.Close()
	//
	// if err != nil {
	// 	fmt.Printf("error %#v\n", err)
	// 	return
	// }
	// os.Stderr = file
	// util.Log.SetLevel(slog.LevelDebug)
	// util.Log.AddSource(true)
	// util.Log.SetOutput(os.Stderr)
	// fmt.Println("log is ready", file)

	// prepare unix socket client (datagram)
	uxClient, err := newUxClient()
	if err != nil {
		fmt.Printf("error %#v\n", err)
		return
	}

	defer func() {
		// notify this child is done
		// exChan <- conf.desiredPort
		uxClient.send(conf.desiredPort)
		uxClient.close()
	}()

	// parse destination
	first := strings.Split(conf.destination, "@")
	if len(first) == 2 {
		conf.user = first[0]
		// second := strings.Split(first[1], ":")
		conf.host = ""
	}
	util.Log.Debug("runChild", "user", conf.user, "host", conf.host, "term", conf.term,
		"desiredPort", conf.desiredPort, "destination", conf.destination)
	/*
		If this variable is set to a positive integer number, it specifies how
		long (in seconds) apshd will wait to receive an update from the
		client before exiting.  Since aprilsh is very useful for mobile
		clients with intermittent operation and connectivity, we suggest
		setting this variable to a high value, such as 604800 (one week) or
		2592000 (30 days).  Otherwise, apshd will wait indefinitely for a
		client to reappear.  This variable is somewhat similar to the TMOUT
		variable found in many Bourne shells. However, it is not a login-session
		inactivity timeout; it only applies to network connectivity.
	*/
	networkTimeout := getTimeFrom("APRILSH_SERVER_NETWORK_TMOUT", 0)

	/*
		If this variable is set to a positive integer number, it specifies how
		long (in seconds) apshd will ignore SIGUSR1 while waiting to receive
		an update from the client.  Otherwise, SIGUSR1 will always terminate
		apshd. Users and administrators may implement scripts to clean up
		disconnected aprilsh sessions. With this variable set, a user or
		administrator can issue

		$ pkill -SIGUSR1 aprilsh-server

		to kill disconnected sessions without killing connected login
		sessions.
	*/
	networkSignaledTimeout := getTimeFrom("APRILSH_SERVER_SIGNAL_TMOUT", 0)

	// util.Log.Debug("runWorker", "networkTimeout", networkTimeout,
	// 	"networkSignaledTimeout", networkSignaledTimeout)

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
	// util.Log.Debug("init terminal size", "cols", windowSize.Col, "rows", windowSize.Row)

	// open parser and terminal
	savedLines := terminal.SaveLinesRowsOption
	terminal, err := statesync.NewComplete(int(windowSize.Col), int(windowSize.Row), savedLines)

	// open network
	blank := &statesync.UserStream{}
	server := network.NewTransportServer(terminal, blank, conf.desiredIP, conf.desiredPort)
	server.SetVerbose(uint(conf.verbose))
	// defer server.Close()

	/*
		// If server is run on a pty, then typeahead may echo and break mosh.pl's
		// detection of the CONNECT message.  Print it on a new line to bodge
		// around that.

		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Printf("\r\n")
		}
	*/

	// send the key to run()
	uxClient.send(server.GetKey())
	// exChan <- server.GetKey()

	// in mosh: the parent print this to stderr.
	// fmt.Printf("#runWorker %s CONNECT %s %s\n", COMMAND_NAME, network.Port(), network.GetKey())
	// printWelcome(os.Stdout, os.Getpid(), os.Stdin)

	// prepare for openPTS fail
	if conf.verbose == _VERBOSE_OPEN_PTS_FAIL {
		windowSize = nil
	}

	ptmx, pts, err := openPTS(windowSize)
	if err != nil {
		util.Log.Warn("openPTS fail", "error", err)
		return err
	}
	defer func() {
		ptmx.Close()
		// pts.Close()
	}() // Best effort.
	// fmt.Printf("#runWorker openPTS successfully.\n")

	// use pipe to signal when to start shell
	// pw and pr is close inside serve() and startShell()
	pr, pw := io.Pipe()

	// prepare host field for utmp record
	utmpHost := fmt.Sprintf("%s:%s", frontend.CommandServerName, server.GetServerPort())

	// add utmp entry
	if utmpSupport {
		ok := util.AddUtmpx(pts, utmpHost)
		if !ok {
			utmpSupport = false
			util.Log.Warn("runChild can't update utmp")
		}
	}

	// start the udp server, serve the udp request
	var wg sync.WaitGroup
	wg.Add(1)
	// waitChan := make(chan bool)
	// go conf.serve(ptmx, pw, terminal, waitChan, network, networkTimeout, networkSignaledTimeout)
	go func() {
		conf.serve(ptmx, pts, pw, terminal, server, networkTimeout, networkSignaledTimeout, conf.user)
		// exChan <- fmt.Sprintf("%s:shutdown", conf.desiredPort)
		uxClient.send(fmt.Sprintf("%s:shutdown", conf.desiredPort))
		wg.Done()
	}()

	// TODO update last log ?
	// util.UpdateLastLog(ptmxName, getCurrentUser(), utmpHost)

	defer func() { // clear utmp entry
		if utmpSupport {
			util.ClearUtmpx(pts)
		}
	}()

	util.Log.Info("start listening on", "port", conf.desiredPort, "clientTERM", conf.term)

	// start the shell with pts
	shell, err := startShell(pts, pr, utmpHost, conf)
	pts.Close() // it's copied by shell process, it's safe to close it here.
	if err != nil {
		util.Log.Warn("startShell fail", "error", err)
		// whChan <- workhorse{}
	} else {

		// whChan <- workhorse{shell, ptmx}
		// wait for the shell to finish.
		var state *os.ProcessState
		state, err = shell.Wait()
		if err != nil || state.Exited() {
			if err != nil {
				util.Log.Warn("shell.Wait fail", "error", err, "state", state)
				// } else {
				// util.Log.Debug("shell.Wait quit", "state.exited", state.Exited())
			}
		}
	}

	// wait serve to finish
	wg.Wait()
	util.Log.Info("stop listening on", "port", conf.desiredPort)

	// fmt.Printf("[%s is exiting.]\n", frontend.COMMAND_SERVER_NAME)
	// https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/
	// https://www.prakharsrivastav.com/posts/golang-context-and-cancellation/

	// util.Log.Debug("runWorker quit", "port", conf.desiredPort)
	return err
}

// parse the flag first, print help or version based on flag
// then run the main listening server
// aprilsh-server should be installed under $HOME/.local/bin
func main() {
	fmt.Fprintf(os.Stderr, "main process %d args=%s\n", os.Getpid(), os.Args)

	conf, _, err := parseFlags(os.Args[0], os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		printUsage("", usage)
		return
	} else if err != nil {
		printUsage(err.Error())
		return
	} else if hint, ok := conf.buildConfig(); !ok {
		printUsage(hint)
		return
	}

	if conf.version {
		printVersion()
		return
	}

	// For security, make sure we don't dump core
	encrypt.DisableDumpingCore()

	if conf.begin {
		beginClientConn(conf)
		return
	}

	// setup client log file
	switch conf.verbose {
	case util.DebugLevel:
		util.Log.SetLevel(slog.LevelDebug)
	case util.TraceLevel:
		util.Log.SetLevel(util.LevelTrace)
	default:
		util.Log.SetLevel(slog.LevelInfo)
	}
	util.Log.AddSource(conf.addSource)
	util.Log.SetOutput(os.Stderr)

	// setup syslog
	syslogWriter, err = syslog.New(syslog.LOG_WARNING|syslog.LOG_LOCAL7, frontend.CommandServerName)
	if err != nil {
		util.Log.Warn("can't find syslog service on this server.")
		syslogSupport = false
	} else {
		syslogSupport = true
	}
	defer syslogWriter.Close()

	// https://jvns.ca/blog/2017/09/24/profiling-go-with-pprof/
	//
	// cpuf, err := os.Create("cpu.profile")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// pprof.StartCPUProfile(cpuf)
	// defer pprof.StopCPUProfile()

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

	// run child process
	if conf.child {
		runChild(conf)
		return
	}

	// start mainSrv
	srv := newMainSrv(conf, runWorker)
	srv.start2(conf)
	srv.wait()
}
