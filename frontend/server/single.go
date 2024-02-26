// Copyright 2022~2024 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/frontend"
	"github.com/ericwq/aprilsh/network"
	"github.com/ericwq/aprilsh/statesync"
	"github.com/ericwq/aprilsh/terminal"
	"github.com/ericwq/aprilsh/util"
	"golang.org/x/sys/unix"
)

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

		whChan <- workhorse{shell, 0}
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
					m.workers[i].child.Kill()
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
			if wh.child != nil {
				m.workers[p] = &wh
			}
		} else if strings.HasPrefix(req, frontend.AprishMsgClose) { // 'close aprilsh:[port]'
			pstr := strings.TrimPrefix(req, frontend.AprishMsgClose)
			port, err := strconv.Atoi(pstr)
			if err == nil {
				// find workhorse
				if wh, ok := m.workers[port]; ok {
					// kill the process, TODO SIGKILL or SIGTERM?
					wh.child.Kill()

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
