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
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	term "github.com/ericwq/aprilsh/terminal"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
	// "golang.org/x/term"
)

var (
	BuildVersion    = "0.1.0" // ready for ldflags
	userCurrentTest = false
	execCmdTest     = false
	buildConfigTest = false
)

var utmpSupport bool

var (
	logW *log.Logger
	logI *log.Logger
)

const (
	_PACKAGE_STRING = "aprilsh"
	_COMMAND_NAME   = "aprilsh-server"
	_PATH_BSHELL    = "/bin/sh"

	_ASH_OPEN  = "open aprilsh:"
	_ASH_CLOSE = "close aprilsh:"

	_VERBOSE_OPEN_PTS    = 99
	_VERBOSE_START_SHELL = 100
)

func init() {
	initLog()
}

func initLog() {
	logW = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	logI = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func printVersion() {
	logI.Printf("%s (%s) [build %s]\n", _COMMAND_NAME, _PACKAGE_STRING, BuildVersion)
	logI.Printf("Copyright (c) 2022~2023 wangqi ericwq057[AT]qq[dot]com\n")
	logI.Printf("reborn mosh with aprilsh\n")
}

func printUsage(usage string) {
	logI.Printf("%s", usage)
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
func getSSHip() string {
	env := os.Getenv("SSH_CONNECTION")
	if len(env) == 0 { // Older sshds don't set this
		logW.Printf("Warning: SSH_CONNECTION not found; binding to any interface.\n")
		return ""
	}

	// SSH_CONNECTION' Identifies the client and server ends of the connection.
	// The variable contains four space-separated values: client IP address,
	// client port number, server IP address, and server port number.
	//
	// ipv4 sample: SSH_CONNECTION=172.17.0.1 58774 172.17.0.2 22
	sshConn := strings.Split(env, " ")
	if len(sshConn) != 4 {
		logW.Printf("Warning: Could not parse SSH_CONNECTION; binding to any interface.\n")
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
  ` + _COMMAND_NAME + ` [--version] [--help]
  ` + _COMMAND_NAME + ` [--server] [--verbose] [--ip ADDR] [--port PORT[:PORT2]] [--color COLORS]` +
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

type Config struct {
	version     bool // verbose : don't close stdin/stdout/stderr
	server      bool
	verbose     int
	desiredIP   string
	desiredPort string
	locales     localeFlag
	color       int
	term        string // client TERM

	commandPath string
	commandArgv []string // the positional (non-flag) command-line arguments.
	withMotd    bool

	// the serve func
	serve func(*os.File, *os.File, *statesync.Complete,
		*network.Transport[*statesync.Complete, *statesync.UserStream], int64, int64) error
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
	flagSet.IntVar(&conf.verbose, "v", 0, "verbose output")

	flagSet.BoolVar(&conf.version, "version", false, "print version information")

	flagSet.BoolVar(&conf.server, "server", false, "listen with SSH ip")
	flagSet.BoolVar(&conf.server, "s", false, "listen with SSH ip")

	flagSet.StringVar(&conf.desiredIP, "ip", "", "listen ip")
	flagSet.StringVar(&conf.desiredIP, "i", "", "listen ip")

	flagSet.StringVar(&conf.desiredPort, "port", "6000", "listen port range")
	flagSet.StringVar(&conf.desiredPort, "p", "6000", "listen port range")

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
	conf, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
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
	if conf.server {
		if sshIP := getSSHip(); len(sshIP) != 0 {
			conf.desiredIP = sshIP
			// fmt.Printf("#main sshIP=%s\n", desiredIP)
		}
	}

	if len(conf.desiredPort) > 0 {
		// Sanity-check arguments

		// fmt.Printf("#main desiredPort=%s\n", conf.desiredPort)
		_, _, ok := network.ParsePortRange(conf.desiredPort, logW)
		if !ok {
			logW.Printf("#main ParsePortRange failed: Bad UDP port range (%s)", conf.desiredPort)
			return
		}
	}

	if err := buildConfig(conf); err != nil {
		logW.Printf("#main buildConfig faileds: %s\n", err.Error())
		return
	}

	srv := newMainSrv(conf, runWorker)
	srv.start(conf)
	srv.wait()
}

// build the config instance and check the utf-8 locale. return error if the terminal
// can't support utf-8 locale.
func buildConfig(conf *Config) error {
	conf.commandPath = ""
	conf.withMotd = false
	conf.serve = serve

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
		conf.commandPath = conf.commandArgv[0]

		if len(conf.commandArgv) == 1 {
			shellName := getShellNameFrom(conf.commandPath)
			conf.commandArgv = []string{shellName}
		} else {
			conf.commandArgv = conf.commandArgv[1:]
		}
	}

	// Adopt implementation locale
	setNativeLocale()
	if !isUtf8Locale() || buildConfigTest {
		nativeType := getCtype()
		nativeCharset := localeCharset()

		// apply locale-related environment variables from client
		clearLocaleVariables()
		for k, v := range conf.locales {
			// fmt.Printf("#buildConfig setenv %s=%s\n", k, v)
			os.Setenv(k, v)
		}

		// check again
		setNativeLocale()
		if !isUtf8Locale() || buildConfigTest {
			clientType := getCtype()
			clientCharset := localeCharset()
			logW.Printf("%s needs a UTF-8 native locale to run.\n", _COMMAND_NAME)
			logW.Printf("Unfortunately, the local environment (%s) specifies "+
				"the character set \"%s\",\n", nativeType, nativeCharset)
			logW.Printf("The client-supplied environment (%s) specifies "+
				"the character set \"%s\".\n", clientType, clientCharset)

			// fmt.Fprintf(os.Stdout, "%s needs a UTF-8 native locale to run.\n\n", COMMAND_NAME)
			// fmt.Fprintf(os.Stdout, "Unfortunately, the local environment (%s) specifies\n"+
			// 	"the character set \"%s\",\n\n", nativeType, nativeCharset)
			// fmt.Fprintf(os.Stdout, "The client-supplied environment (%s) specifies\n"+
			// 	"the character set \"%s\".\n\n", clientType, clientCharset)
			return errors.New("UTF-8 locale fail.")
		}
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

// worker started by mainSrv.run(). worker will listen on specified port and
// forward user input to shell (started by runWorker. the output is forward
// to the network.
func runWorker(conf *Config, exChan chan string, whChan chan *workhorse) (err error) {
	defer func() {
		// notify this worker is done
		exChan <- conf.desiredPort
	}()

	networkTimeout := getTimeFrom("APRILSH_SERVER_NETWORK_TMOUT", 0)
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
	terminal, err := statesync.NewComplete(int(windowSize.Col), int(windowSize.Row), 0)

	// open network
	blank := &statesync.UserStream{}
	network := network.NewTransportServer(terminal, blank, conf.desiredIP, conf.desiredPort)
	if conf.verbose == 1 {
		network.SetVerbose(uint(conf.verbose))
	}
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
		logW.Printf("#runWorker openPTS fail: %s\n", err)
		whChan <- &workhorse{}
		return err
	}
	defer func() { _ = ptmx.Close() }() // Best effort.
	// fmt.Printf("#runWorker openPTS successfully.\n")

	// prepare host field for utmp record
	utmpHost := fmt.Sprintf("%s [%d]", _PACKAGE_STRING, os.Getpid())

	// start the shell with pts
	shell, err := startShell(pts, utmpHost, conf)
	pts.Close() // it's copied by shell process, it's safe to close it here.
	if err != nil {
		logW.Printf("#runWorker startShell fail: %s\n", err)
		whChan <- &workhorse{}
	} else {
		// add utmp entry
		ptmxName := ptmx.Name()
		if utmpSupport && !addUtmpEntry(pts, utmpHost) {
			logW.Printf("#runWorker add utmp entry failed\n")
		}

		// update last log
		updateLastLog(ptmxName)

		// start the udp server, serve the udp request
		go conf.serve(ptmx, pts, terminal, network, networkTimeout, networkSignaledTimeout)
		whChan <- &workhorse{shell, ptmx}
		logI.Printf("#runWorker start listening on :%s\n", conf.desiredPort)

		// wait for the shell to finish.
		// fmt.Printf("#runWorker shell.Wait() %p %v\n", shell, shell)
		if state, err := shell.Wait(); err != nil || state.Exited() {
			logW.Printf("#runWorker shell.Wait fail: %s, state: %s\n", err, state)
		}
		logI.Printf("#runWorker stop listening on :%s\n", conf.desiredPort)

		// clear utmp entry
		if utmpSupport && !clearUtmpEntry(pts) {
			logW.Printf("#runWorker clear utmp entry failed\n")
		}
	}

	// fmt.Printf("#runWorker [%s is exiting.]\n\n", COMMAND_NAME)
	// https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/
	// https://www.prakharsrivastav.com/posts/golang-context-and-cancellation/

	return err
}

func getCurrentUser() string {
	user, err := user.Current()
	if err != nil || userCurrentTest {
		logW.Printf("#getCurrentUser report: %s\n", err)
		return ""
	}

	return user.Username
}

// read data from udp socket and send the result to socketChan
func readFromSocket(timeout int, socketChan chan msg,
	network *network.Transport[*statesync.Complete, *statesync.UserStream],
) {
	network.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
	for {
		// packet received from the network
		err := network.Recv()
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// network read timeout
			} else {
				socketChan <- msg{err, ""}
			}
		}
		socketChan <- msg{nil, ""}
	}
}

// read data from pts master and send the result to masterChan
func readFromMaster(timeout int, masterChan chan msg, ptmx *os.File) {
	var buf [16384]byte

	// set read time out
	ptmx.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))

	for {
		// fill buffer if possible
		bytesRead, err := ptmx.Read(buf[:])
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// file read timeout
			} else {
				masterChan <- msg{err, ""}
				// If the pty slave is closed, reading from the master can fail with
				// EIO (see #264).  So we treat errors on read() like EOF.
				break
			}
		}
		masterChan <- msg{err: nil, data: string(buf[:bytesRead])}
	}
}

type msg struct {
	err  error
	data string
}

func serve(ptmx *os.File, pts *os.File, terminal *statesync.Complete,
	network *network.Transport[*statesync.Complete, *statesync.UserStream],
	networkTimeout int64, networkSignaledTimeout int64,
) error {
	// scale timeouts
	networkTimeoutMs := networkTimeout * 1000
	// networkSignaledTimeoutMs := uint64(networkSignaledTimeout) * 1000
	lastRemoteNum := network.GetRemoteStateNum()
	var connectedUtmp bool
	var savedAddr net.Addr

	var terminalToHost strings.Builder
	var timeSinceRemoteState int64
	var socketChan chan msg
	var masterChan chan msg

	socketChan = make(chan msg, 1)
	masterChan = make(chan msg, 1)

	go readFromSocket(10, socketChan, network)
	go readFromMaster(10, masterChan, ptmx)

mainLoop:
	for {
		now := time.Now().UnixMilli()
		p := network.GetLatestRemoteState()
		timeSinceRemoteState = now - p.GetTimestamp()
		terminalToHost.Reset()

		select {
		case socketMsg := <-socketChan: // got data from socket
			if socketMsg.err != nil { // error handling
				logW.Printf("#readFromSocket receive error:%s\n", socketMsg.err)
				continue mainLoop
			}

			// is new user input available for the terminal?
			if network.GetRemoteStateNum() != lastRemoteNum {
				lastRemoteNum = network.GetRemoteStateNum()

				us := &statesync.UserStream{}
				us.ApplyString(network.GetRemoteDiff())
				// apply userstream to terminal
				for i := 0; i < us.Size(); i++ {
					action := us.GetAction(i)
					//  apply only the last consecutive Resize action
					if res, ok := action.(term.Resize); ok {
						for i < us.Size()-1 {
							if res, ok = us.GetAction(i + 1).(term.Resize); ok {
								i++
							}
						}
						// resize master
						winSize, err := unix.IoctlGetWinsize(int(ptmx.Fd()), unix.TIOCGWINSZ)
						if err != nil {
							logW.Printf("#serve ioctl TIOCGWINSZ %s", err)
							network.ShartShutdown()
						}
						winSize.Col = uint16(res.Width)
						winSize.Row = uint16(res.Height)
						if err = unix.IoctlSetWinsize(int(ptmx.Fd()), unix.TIOCSWINSZ, winSize); err != nil {
							logW.Printf("#serve ioctl TIOCSWINSZ %s", err)
							network.ShartShutdown()
						}
					}
					terminalToHost.WriteString(terminal.ActOne(action))
				}

				if !us.Empty() {
					// register input frame number for future echo ack
					terminal.RegisterInputFrame(lastRemoteNum, now)
				}

				// update client with new state of terminal
				if !network.ShutdownInProgress() {
					network.SetCurrentState(terminal)
				}

				// update utmp entry if we have become "connected"
				if utmpSupport && (!connectedUtmp || !reflect.DeepEqual(savedAddr, network.GetRemoteAddr())) {
					if !clearUtmpEntry(pts) {
						logW.Printf("#serve clear utmp entry failed\n")
					}
					savedAddr = network.GetRemoteAddr()

					// convert savedAddr to host name
					host := savedAddr.String() // default host name is ip string
					hostList, e := net.LookupAddr(host)
					if e == nil {
						host = hostList[0] // got the host name, use the first one
					}

					newHost := fmt.Sprintf("%s via %s [%d]", host, _PACKAGE_STRING, os.Getpid())
					if !addUtmpEntry(pts, newHost) {
						logW.Printf("#runWorker add utmp entry failed\n")
					}

					connectedUtmp = true
				}
			}
		case masterMsg := <-masterChan:
			// input from the host needs to be fed to the terminal
			if !network.ShutdownInProgress() {
				if masterMsg.err != nil {
					logW.Println("#readFromMaster read error: ", masterMsg.err)
					network.ShartShutdown()
				} else {
					r := terminal.Act(masterMsg.data)
					terminalToHost.WriteString(r)

					// update client with new state of terminal
					network.SetCurrentState(terminal)
				}
			}
		default:
		}

		// write user input and terminal writeback to the host
		if terminalToHost.Len() > 0 {
			_, err := ptmx.WriteString(terminalToHost.String())
			if err != nil {
				network.ShartShutdown()
			}
		}

		idelShutdown := false
		if networkTimeoutMs > 0 && networkTimeoutMs <= timeSinceRemoteState {
			idelShutdown = true
			logW.Printf("Network idle for %d seconds.\n", timeSinceRemoteState/1000)
		}

		if idelShutdown { // TODO how to process sel.any_signal()?
			// shutdown signal
			if network.HasRemoteAddr() && !network.ShutdownInProgress() {
				network.ShartShutdown()
			} else {
				break mainLoop
			}
		}

		// quit if our shutdown has been acknowledged
		if network.ShutdownInProgress() && network.ShutdownAcknowledged() {
			break mainLoop
		}

		// quit after shutdown acknowledgement timeout
		if network.ShutdownInProgress() && network.ShutdownAckTimedout() {
			break mainLoop
		}

		// quit if we received and acknowledged a shutdown request
		if network.CounterpartyShutdownAckSent() {
			break mainLoop
		}

		network.Tick()
	}
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
	logI.Printf("%s start listening on :%d. build version %s [pid=%d] \n", _COMMAND_NAME, port, BuildVersion, pid)
	logI.Printf("Copyright 2022 wangqi.\n")
	logI.Printf("%s%s", "Use of this source code is governed by a MIT-style",
		"license that can be found in the LICENSE file.\n")
	// logI.Printf("[%s detached, pid=%d]\n", COMMAND_NAME, pid)

	if tty != nil {
		inputUTF8, err := checkIUTF8(int(tty.Fd()))
		if err != nil {
			logW.Printf("Warning: %s\n", err)
		}

		if !inputUTF8 {
			// Input is UTF-8 (since Linux 2.6.4)
			logW.Printf("%s %s %s", "Warning: termios IUTF8 flag not defined.",
				"Character-erase of multibyte character sequence",
				"probably does not work properly on this platform.\n")
		}
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
	unix.IoctlSetTermios(fd, setTermios, termios)

	return nil
}

func convertWinsize(windowSize *unix.Winsize) *pty.Winsize {
	if windowSize == nil {
		return nil
	}
	var sz pty.Winsize
	sz.Cols = windowSize.Col
	sz.Rows = windowSize.Row
	sz.X = windowSize.Xpixel
	sz.Y = windowSize.Ypixel

	return &sz
}

// open pts master and slave, set terminal size according to window size.
func openPTS(wsize *unix.Winsize) (ptmx *os.File, pts *os.File, err error) {
	// open pts master and slave
	ptmx, pts, err = pty.Open()
	if wsize == nil {
		err = errors.New("invalid parameter")
	}
	if err == nil {
		sz := convertWinsize(wsize)
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
	if err := setIUTF8(int(pts.Fd())); err != nil {
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

	// Wait for parent to release us.
	// var buf string
	// if _, err := fmt.Fscanf(cmd.Stdin, "%s", &buf); err != nil {
	// 	return cmd, err
	// }

	encrypt.ReenableDumpingCore()

	/*
		additional logic for pty.StartWithAttrs() end
	*/

	proc, err := os.StartProcess(conf.commandPath, conf.commandArgv, &procAttr)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("#startShell before cmd.Start(), %q\n", cmd.Path)
	// if err := cmd.Start(); err != nil {
	// 	return cmd, err
	// }
	//
	// return cmd, nil
	return proc, nil
}

func deviceExists(line string) bool {
	deviceName := fmt.Sprintf("/dev/%s", line)
	_, err := os.Lstat(deviceName)
	if err != nil {
		return false
	}

	return true
}

// check unattached session and print warning message if there is any
// ignore current session
func warnUnattached(w io.Writer, ignoreHost string) {
	userName := getCurrentUser()

	// check unattached sessions
	unatttached := checkUnattachedRecord(userName, ignoreHost)

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
	m.timeout = 200
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
		logW.Printf("%s: %s\n", _COMMAND_NAME, err.Error())
		return
	}

	// fmt.Printf("#start listening on %s, next port is %d\n", conf.desiredPort, m.nextWorkerPort+1)
	m.wg.Add(1)
	go m.run(conf)
}

func (m *mainSrv) handler() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGTERM)
	defer signal.Stop(sig)

	for s := range sig {
		switch s {
		case syscall.SIGHUP: // TODO:reload the config?
			logI.Println("got message SIGHUP.")
		case syscall.SIGTERM:
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
		logI.Printf("%s stop listening on :%d.", _COMMAND_NAME, m.port)
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
			// fmt.Printf("#run got workDone message from %s\n", portStr)
			// clear worker list
			delete(m.workers, p)
		case sd := <-m.downChan: // ready to shutdown mainSrv
			// fmt.Printf("#run got shutdown message %t\n", sd)
			shutdown = sd
		default:
		}

		if len(m.workers) == 0 && shutdown {
			return
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

			// blocking read the workhorse from runWorker
			wh := <-m.whChan
			// logI.Printf("#run got workhorse %p %v\n", wh.shell, wh.shell)
			if wh.shell != nil {
				m.workers[p] = wh
			}

			// response session key and udp port to client
			msg := fmt.Sprintf("%d,%s", p, key)
			m.writeRespTo(addr, _ASH_OPEN, msg)
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
					logW.Printf("#mainSrv run() request %q got %q\n", req, resp)
				}
			} else {
				resp := m.writeRespTo(addr, _ASH_CLOSE, "wrong port number")
				logW.Printf("#mainSrv run() request %q got %q\n", req, resp)
			}
		} else {
			resp := m.writeRespTo(addr, _ASH_CLOSE, "unknow request")
			logW.Printf("#mainSrv run() request %q got %q\n", req, resp)
		}
	}
}

// return the minimal available port and increase the maxWorkerPort if necessary.
func (m *mainSrv) getAvailabePort() (port int) {
	port = m.port
	if m.maxPort-m.port > 1 {
		// sort the current ports
		ports := make([]int, 0, len(m.workers))
		for k := range m.workers {
			ports = append(ports, k)
		}
		sort.Ints(ports)
		// fmt.Printf("#getAvailabePort got ports=%v\n", ports)

		// check minimal available port
		k := 0
		for i := 0; i < m.maxPort-m.port-1; i++ {
			k = i + 1
			// fmt.Printf("#getAvailabePort check port+k=%d, ports[i]=%d\n", port+i+1, ports[i])
			if (port+k > m.port) && (port+k < ports[i]) {
				port = port + k
				break
			}
		}
		// fmt.Printf("#getAvailabePort search port=%d\n", port)
	}
	if port == m.port {
		port = m.maxPort
		m.maxPort++
	}
	// fmt.Printf("#getAvailabePort got port=%d\n", port)
	return port
}

// write header and message to addr
func (m *mainSrv) writeRespTo(addr *net.UDPAddr, header, msg string) (resp string) {
	resp = fmt.Sprintf("%s%s\n", header, msg)
	m.conn.SetDeadline(time.Now().Add(time.Millisecond * 200))
	m.conn.WriteToUDP([]byte(resp), addr)
	return
}

func (m *mainSrv) wait() {
	m.wg.Wait()
	if err := m.eg.Wait(); err != nil {
		logW.Printf("#mainSrv wait() reports %s\n", err.Error())
	}
}
