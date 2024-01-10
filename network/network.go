// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"net/netip"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
	"github.com/ericwq/aprilsh/util"

	// "golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

type Direction uint

const (
	APRILSH_PROTOCOL_VERSION = 3 // bumped for echo-ack

	TO_SERVER Direction = iota
	TO_CLIENT
)

const (
	DIRECTION_MASK uint64 = uint64(1) << 63
	SEQUENCE_MASK  uint64 = ^DIRECTION_MASK
)

func timestamp16() uint16 {
	ts := time.Now().UnixMilli() % 65535
	return uint16(ts)
}

func timestampDiff(tsnew, tsold uint16) uint16 {
	var diff int
	diff = int(tsnew) - int(tsold)
	if diff < 0 {
		diff += 65536
	}

	return uint16(diff)
}

// Packet is used for RTT calculation
type Packet struct {
	seq            uint64
	direction      Direction // packet direciton
	timestamp      uint16    // current packet send time
	timestampReply uint16    // last packet received time
	payload        []byte
}

func NewPacket(direction Direction, timestamp uint16, timestampReply uint16, payload []byte) *Packet {
	p := &Packet{}

	p.seq = encrypt.Unique()
	p.direction = direction
	p.timestamp = timestamp
	p.timestampReply = timestampReply
	p.payload = payload
	// copy(p.payload, payload)

	return p
}

func NewPacketFrom(m *encrypt.Message) *Packet {
	p := &Packet{}

	p.seq = m.NonceVal() & SEQUENCE_MASK
	if m.NonceVal()&DIRECTION_MASK > 0 {
		p.direction = TO_CLIENT
	} else {
		p.direction = TO_SERVER
	}
	p.timestamp = m.GetTimestamp()
	p.timestampReply = m.GetTimestampReply()

	p.payload = m.GetPayload()

	return p
}

// seqNonce = seq | direction
// payload = timestamp+timestampReply+content
func (p *Packet) toMessage() *encrypt.Message {
	// combine seq and direction together
	var seqNonce uint64
	if p.direction == TO_CLIENT {
		// client is 1, server is 0
		seqNonce = DIRECTION_MASK | (p.seq & SEQUENCE_MASK)
	} else {
		seqNonce = p.seq & SEQUENCE_MASK
	}

	// combine time stamp and payload together
	tsb := p.timestampBytes()
	payload := append(tsb, p.payload...)

	return encrypt.NewMessage(seqNonce, payload)
}

// build a byte slice with timestamp and timestampReply fields
func (p *Packet) timestampBytes() []byte {
	t := struct {
		timestamp      uint16
		timestampReply uint16
	}{p.timestamp, p.timestampReply}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, t)
	return buf.Bytes()
}

const (
	/*
	 * For IPv4, guess the typical (minimum) header length;
	 * fragmentation is not dangerous, just inefficient.
	 */
	IPV4_HEADER_LEN = 20 + 8 // base IP header + UDP

	/*
	 * For IPv6, we don't want to ever have MTU issues, so make a
	 * conservative guess about header size.
	 */
	IPV6_HEADER_LEN = 40 + 16 + 8 // base IPv6 header + 2 minimum-sized extension headers + UDP */

	/* Application datagram MTU. For constructors and fallback. */
	DEFAULT_SEND_MTU = 500

	/*
	 * IPv4 MTU. Don't use full Ethernet-derived MTU,
	 * mobile networks have high tunneling overhead.
	 *
	 * As of July 2016, VPN traffic over Amtrak Acela wifi seems to be
	 * dropped if tunnelled packets are 1320 bytes or larger.  Use a
	 * 1280-byte IPv4 MTU for now.
	 *
	 * We may have to implement ICMP-less PMTUD (RFC 4821) eventually.
	 */
	DEFAULT_IPV4_MTU = 1280

	/* IPv6 MTU. Use the guaranteed minimum to avoid fragmentation. */
	DEFAULT_IPV6_MTU = 1280

	MIN_RTO int64 = 50   // ms
	MAX_RTO int64 = 1000 // ms

	PORT_RANGE_LOW  = 60001
	PORT_RANGE_HIGH = 60999

	SERVER_ASSOCIATION_TIMEOUT = 40000
	PORT_HOP_INTERVAL          = 10000

	MAX_PORTS_OPEN     = 10
	MAX_OLD_SOCKET_AGE = 60000

	CONGESTION_TIMESTAMP_PENALTY = 500 // ms

	NETWORK = "udp" // IPv6 is only supported on Docker daemons running on Linux hosts.

	// Network transport overhead.
	ADDED_BYTES = 8 /* seqno/nonce */ + 4 /* timestamps */
)

var (
	// set socket options
	controlFunc = func(network, address string, raw syscall.RawConn) (err error) {
		// err got value from different positions, they are not conflict with each other
		err = raw.Control(func(fd uintptr) {
			err = markECN(int(fd), unix.GetsockoptInt, unix.SetsockoptInt)
		})
		return
	}

	// validate congestion experienced
	congestionFunc = func(in byte) bool {
		return in&0x03 == 0x03
	}
)

var (
	ErrRecvLength    = errors.New("#recvOne receive zero or negative length data.")
	ErrRecvOversize  = errors.New("#recvOne received oversize datagram.")
	ErrRecvDirection = errors.New("#recvOne direction is wrong.")
)

// internal conneciton for testability.
type udpConn interface {
	WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error)
	ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error)
	SetReadDeadline(t time.Time) error
	Close() error
}

// internal cipher session for testability.
type cipherSession interface {
	Encrypt(plainText *encrypt.Message) []byte
	Decrypt(text []byte) (*encrypt.Message, error)
}

type Connection struct {
	socks         []udpConn // server has only one socket, client has several socket.
	hasRemoteAddr bool
	remoteAddr    net.Addr
	server        bool

	mtu int

	key     *encrypt.Base64Key
	session cipherSession
	// session *encrypt.Session

	direction                Direction
	savedTimestamp           int16 // the timestamp when the packet is created
	savedTimestampReceivedAt int64 // the timestamp when the last packet is received
	expectedReceiverSeq      uint64

	lastHeard            int64 // last packet receive time
	lastPortChoice       int64 // last change port time
	lastRoundtripSuccess int64 // last acked send state timestamp

	RTTHit bool
	SRTT   float64 // smoothed round-trip time
	RTTVAR float64 // round-trip time variation

	sync.RWMutex
}

func NewConnection(desiredIp string, desiredPort string) *Connection { // server
	c := &Connection{}

	c.hasRemoteAddr = false
	c.server = true

	c.mtu = DEFAULT_SEND_MTU

	c.key = encrypt.NewBase64Key()
	c.session, _ = encrypt.NewSession(*c.key)

	c.direction = TO_CLIENT
	c.savedTimestamp = math.MaxInt16

	c.lastHeard = math.MaxInt64
	c.lastPortChoice = math.MaxInt64
	c.lastRoundtripSuccess = math.MaxInt64

	c.RTTHit = false
	c.SRTT = 1000
	c.RTTVAR = 500

	// c.logW = logFunc
	c.socks = make([]udpConn, 0)

	c.setup()

	/* The mosh wrapper always gives an IP request, in order
	   to deal with multihomed servers. The port is optional. */

	/* If an IP request is given, we try to bind to that IP, but we also
	   try INADDR_ANY. If a port request is given, we bind only to that port. */

	desiredPortHigh := -1
	desiredPortLow := -1
	ok := false

	if len(desiredPort) > 0 {
		if desiredPortLow, desiredPortHigh, ok = ParsePortRange(desiredPort); !ok {
			// c.logW.Printf("Invalid port value. %q\n", desiredPort)
			return nil
		}
	}

	// fmt.Printf("#connection (%d:%d)\n", desiredPortLow, desiredPortHigh)
	if !c.tryBind(desiredIp, desiredPortLow, desiredPortHigh) {
		return nil
	}
	// fmt.Printf("#connection finished\n\n")

	return c
}

func NewConnectionClient(keyStr string, ip, port string) *Connection { // client
	c := &Connection{}

	c.socks = make([]udpConn, 0)
	c.hasRemoteAddr = false
	c.server = false

	c.mtu = DEFAULT_SEND_MTU

	c.key = encrypt.NewBase64Key2(keyStr)
	// var err error
	if c.key == nil {
		util.Log.With("keyStr", keyStr).
			With("error", nil).Warn("#NeNewConnectionClient build key failed")
		return nil

	}
	c.session, _ = encrypt.NewSession(*c.key) // TODO error handling
	// if err != nil {
	// 	// fmt.Printf("#NewConnectionClient :%s\n", err)
	// 	util.Log.With("keyStr", keyStr).
	// 		With("error").Warn("#NeNewConnectionClient create session failed")
	// 	return nil
	// }

	c.direction = TO_SERVER
	c.savedTimestamp = math.MaxInt16

	c.lastHeard = math.MaxInt64
	c.lastPortChoice = math.MaxInt64
	c.lastRoundtripSuccess = math.MaxInt64

	c.RTTHit = false
	c.SRTT = 1000
	c.RTTVAR = 500

	// c.logW = logFunc

	c.setup()
	if !c.dialUDP(ip, port) {
		return nil
	}
	c.setMTU(c.remoteAddr)

	return c
}

func parsePort(portStr string, hint string) (port int, ok bool) {
	value, err := strconv.Atoi(portStr)
	if err != nil {
		// logW.Printf("#parsePort invalid (%s) port number (%s)\n", hint, portStr)
		util.Log.With("error", err).With("hint", hint).
			With("port", portStr).Warn("#parsePort invalid port number")
		return
	}

	if value < 0 || value > 65535 {
		// logW.Printf("#parsePort (%s) port number %d outside valid range [0..65535]\n", hint, value)
		util.Log.With("error", err).With("hint", hint).
			With("port number", value).Warn("#parsePort port number is outside of valid range [0..65535]")
		return
	}

	port = value
	ok = true
	return
}

// parse "port" or "portlow:porthigh"
func ParsePortRange(desiredPort string) (desiredPortLow, desiredPortHigh int, ok bool) {
	var value int

	all := strings.Split(desiredPort, ":")
	if len(all) == 2 {
		// parse "portlow:porthigh"
		if value, ok = parsePort(all[0], "low"); !ok {
			return
		} else {
			desiredPortLow = value
		}

		if value, ok = parsePort(all[1], "high"); !ok {
			return
		} else {
			desiredPortHigh = value
		}

		if desiredPortLow > desiredPortHigh {
			// logW.Printf("#ParsePortRange low port %d greater than high port %d\n", desiredPortLow, desiredPortHigh)
			util.Log.With("low port", desiredPortLow).
				With("high port", desiredPortHigh).Warn("#ParsePortRange low port is greater than high port")
			ok = false
			return
		}
	} else {
		// parse solo port
		if value, ok = parsePort(all[0], "solo"); !ok {
			return
		} else {
			desiredPortLow = value
			desiredPortHigh = desiredPortLow
		}
	}

	ok = true
	return
}

// mark the ECN bit for socket options (IP_TOS ), currently only EC0 is marked
func markECN(fd int,
	getSocketOpt func(fd, level, opt int) (value int, err error),
	setSocketOpt func(fd, level, opt int, value int) (err error),
) error {
	// dsable path MTU discovery
	// IP_MTU_DISCOVER and IP_PMTUDISC_DONT is not defined on macOS
	// opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_MTU_DISCOVER, unix.IP_PMTUDISC_DONT)
	// if opErr != nil {
	// 	fmt.Printf("#ListenConfig %s\n", opErr.Error())
	// 	return
	// }

	// TODO the following code only works on linux and for ipv4!
	tosConf, err := getSocketOpt(fd, unix.IPPROTO_IP, unix.IP_TOS)
	if err != nil {
		return err
	}
	tosConf |= 0x02 // ECN-capable transport only, ECT(0), https://www.rfc-editor.org/rfc/rfc3168

	/*
	   Differentiated Service Code Point - TCP/IP Illustrated V1, Page 188
	   Explicit Congestion Notification - TCP/IP Illustrated V1, Page 783
	   Random Early Detection algorithm - check it for packet drop tail.
	   RFC 1349: Type of Service in the Internet Protocol Suite
	   RFC 2474: Definition of the Differentiated Services Field (DS Field) in the IPv4 and IPv6 Headers
	   RFC 3168: The Addition of Explicit Congestion Notification (ECN) to IP
	   https://dl.acm.org/doi/10.1145/2815675.2815716
	*/
	// int tosConf = 0x92; // OS X does not have IPTOS_DSCP_AF42 constant
	// dscp := 0x02 // ECN-capable transport only
	err = setSocketOpt(fd, unix.IPPROTO_IP, unix.IP_TOS, tosConf)
	if err != nil {
		return err
	}

	// request explicit congestion notification on received datagrams
	err = setSocketOpt(fd, unix.IPPROTO_IP, unix.IP_RECVTOS, 1)
	if err != nil {
		return err
	}
	// https://groups.google.com/g/golang-nuts/c/TcHb_bXT18U

	// syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	// syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	return nil
}

func (c *Connection) dialUDP(ip, port string) bool {
	var d net.Dialer
	d.Control = controlFunc

	conn, err := d.Dial(NETWORK, net.JoinHostPort(ip, port))
	if err != nil {
		util.Log.With("error", err).Warn("#dialUDP dial fail")
		return false
	}

	c.remoteAddr = conn.RemoteAddr()
	c.socks = append(c.socks, conn.(udpConn))
	c.hasRemoteAddr = true

	util.Log.With("ip", ip).With("port", port).
		With("localAddr", conn.LocalAddr()).
		With("remoteAddr", conn.RemoteAddr()).Debug("dialUDP")
	return true
}

func (c *Connection) tryBind(desireIp string, portLow, portHigh int) bool {
	searchLow := PORT_RANGE_LOW
	searchHigh := PORT_RANGE_HIGH

	if portLow != -1 { // low port preference
		searchLow = portLow
	}
	if portHigh != -1 { // high port preference
		searchHigh = portHigh
	}

	// prepare for additional socket options
	var lc net.ListenConfig
	lc.Control = controlFunc

	for i := searchLow; i <= searchHigh; i++ {
		address := net.JoinHostPort(desireIp, strconv.Itoa(i))

		localAddr, err := net.ResolveUDPAddr(NETWORK, address)
		if err != nil {
			// c.logW.Printf("#tryBind %s\n", err)
			util.Log.With("error", err).With("address", address).Warn("#tryBind resolve")
			return false
		}

		// fmt.Printf("#tryBind %d-%d, ladd=%q\n", i, searchHigh, ladd)
		conn, err := lc.ListenPacket(context.Background(), NETWORK, localAddr.String())
		if err != nil {
			if i == searchHigh { // last port to search
				util.Log.With("address", c.remoteAddr).With("error", err).Warn("#tryBind listen")
			}
		} else {
			c.socks = append(c.socks, conn.(udpConn))
			c.setMTU(localAddr)
			return true
		}
	}
	return false
}

// update lastPortChoice timestamp
func (c *Connection) setup() {
	c.lastPortChoice = time.Now().UnixMilli()
}

// build a packet based on sending and receiving timestamp and payload
func (c *Connection) newPacket(payload string) *Packet {
	var outgoingTimestampReply uint16

	outgoingTimestampReply = 0 // -1 for c++
	now := time.Now().UnixMilli()
	if now-c.savedTimestampReceivedAt < 1000 { // we have a recent received timestamp
		// send "corrected" timestamp advanced by how long we held it
		outgoingTimestampReply = uint16(c.savedTimestamp) + uint16(now-c.savedTimestampReceivedAt)
		c.savedTimestamp = math.MaxInt16
		c.savedTimestampReceivedAt = 0
	}
	return NewPacket(c.direction, timestamp16(), outgoingTimestampReply, []byte(payload))
}

// reconnect server with new local address
func (c *Connection) hopPort() {
	c.printSocks("before hopPort")
	c.setup()

	host, port, _ := net.SplitHostPort(c.remoteAddr.String())
	if !c.dialUDP(host, port) {
		// c.logW.Printf("#hopPort failed to dial %s\n", c.remoteAddr)
		util.Log.With("remote addr", c.remoteAddr).Warn("#hopPort failed to dial")
		return
	}

	c.pruneSockets()
	c.printSocks("after hopPort")
}

func (c *Connection) printSocks(postfix string) {
	// print socks

	str := fmt.Sprintf("printSocks %s", postfix)
	for i := range c.socks {
		var x net.Conn

		x = c.socks[i].(net.Conn)
		util.Log.With("i", i).With("localAddr", x.LocalAddr()).
			With("remoteAddr", x.RemoteAddr()).Debug(str)
	}
}

// return the last udp connection
func (c *Connection) sock() udpConn {
	return c.socks[len(c.socks)-1]
}

func (c *Connection) cleanSocks(numToKill int) {
	// util.Log.With("socks length", len(c.socks)).With("numToKill", numToKill).Debug("cleanSocks")
	for i := 0; i < numToKill; i++ {
		c.socks[i].Close()
	}
	c.socks = c.socks[numToKill:]
	// util.Log.With("socks length", len(c.socks)).With("numToKill", numToKill).Debug("cleanSocks")
}

// clear the old and over size sockets
func (c *Connection) pruneSockets() {
	// don't keep old sockets if the new socket has been working for long enough
	if len(c.socks) > 1 {
		now := time.Now().UnixMilli()
		if now-c.lastPortChoice > MAX_OLD_SOCKET_AGE {
			numToKill := len(c.socks) - 1
			// c.socks = c.socks[numToKill:]
			c.cleanSocks(numToKill)
		}
	} else {
		// c.printSocks()
		return
	}

	// make sure we don't have too many receive sockets open
	if len(c.socks) > MAX_PORTS_OPEN {
		// c.socks = c.socks[MAX_PORTS_OPEN:]
		c.cleanSocks(MAX_PORTS_OPEN)
	}
}

func (c *Connection) recvOne(conn udpConn) (string, error) {
	data := make([]byte, c.mtu)
	oob := make([]byte, 40)

	// read from the socket,TODO does this method support read deadline?
	// in golang, the flags parameters for recvfrom system call is always 0.
	// that means it's always blocking read. which means unix.MSG_DONTWAIT can't be applied here.
	n, oobn, flags, raddr, err := conn.ReadMsgUDP(data, oob)
	if err != nil {
		return "", err
	}
	if n < 0 {
		return "", fmt.Errorf("length=%d: %w", n, ErrRecvLength)
	}

	if flags&unix.MSG_TRUNC == unix.MSG_TRUNC {
		return "", fmt.Errorf("datagram size=%d: %w", n, ErrRecvOversize)
	}

	// fmt.Printf("#recvOne flags=0x%x, MSG_TRUNC=0x%x, n=%d, oobn=%d, err=%p, raddr=%s\n", flags, unix.MSG_TRUNC, n, oobn, err, raddr)
	// parse the optional ancillary data
	ctrlMsgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return "", err
	}

	// fmt.Printf("#recvOne ctrlMsgs=%v\n", ctrlMsgs)
	// receive ECN
	congestionExperienced := false
	for _, ctrlMsg := range ctrlMsgs {
		if ctrlMsg.Header.Level == unix.IPPROTO_IP &&
			(ctrlMsg.Header.Type == unix.IP_TOS || ctrlMsg.Header.Type == unix.IP_RECVTOS) {
			// fmt.Printf("#recvOne got %08b\n", ctrlMsg.Data)
			// CE: RFC 3168
			// https://www.ietf.org/rfc/rfc3168.html
			congestionExperienced = congestionFunc(ctrlMsg.Data[0])
		}
	}
	// fmt.Printf("#recvOne congestionExperienced=%t\n", congestionExperienced)

	// decrypt the message and build the packet.
	msg, err := c.session.Decrypt(data[:n])
	// fmt.Printf("#recvOne msg=%p\n", msg)
	if err != nil || msg == nil {
		return "", err
	}
	p := NewPacketFrom(msg)

	// prevent malicious playback to sender
	if c.server {
		if p.direction != TO_SERVER {
			// return "", errors.New("#recvOne server direction is wrong.")
			return "", fmt.Errorf("to server: %w", ErrRecvDirection)
		}
	} else {
		if p.direction != TO_CLIENT {
			// return "", errors.New("#recvOne client direction is wrong.")
			return "", fmt.Errorf("to client: %w", ErrRecvDirection)
		}
	}

	if p.seq < c.expectedReceiverSeq {
		// don't use (but do return) out-of-order packets for timestamp or targeting
		// c.hasRemoteAddr = true
		util.Log.With("expectedReceiverSeq", c.expectedReceiverSeq).With("got", p.seq).
			Warn("#recvOne received explicit out-of-order packets")
		return string(p.payload), nil
	}
	// this is security-sensitive because a replay attack could otherwise screw up the timestamp and targeting
	c.expectedReceiverSeq = p.seq + 1

	// fmt.Printf("#recvOne seq=%d, timestamp=%d, timestampReply=%d\n", p.seq, p.timestamp, p.timestampReply)
	if p.timestamp != math.MaxUint16 {
		c.savedTimestamp = int16(p.timestamp)
		c.savedTimestampReceivedAt = time.Now().UnixMilli()

		// fmt.Printf("#recvOne savedTimestamp=%d, savedTimestampReceivedAt=%d\n", c.savedTimestamp, c.savedTimestampReceivedAt)
		if congestionExperienced {
			// signal counterparty to slow down
			// this will gradually slow the counterparty down to the minimum frame rate
			c.savedTimestamp -= CONGESTION_TIMESTAMP_PENALTY
			if c.server {
				// c.logW.Println("#recvOne received explicit congestion notification.")
				util.Log.Warn("#recvOne received explicit congestion notification")
			}
		}
	}

	if p.timestampReply != math.MaxUint16 { // -1 in c++
		now16 := timestamp16()
		R := float64(timestampDiff(now16, p.timestampReply))

		if R < 5000 { // ignore large values, e.g. server was Ctrl-Zed
			// see https://datatracker.ietf.org/doc/html/rfc2988 for the algorithm
			if !c.RTTHit { // first measurement
				c.SRTT = R
				c.RTTVAR = R / 2
				c.RTTHit = true
			} else {
				alpha := 1.0 / 8.0
				beta := 1.0 / 4.0

				c.RTTVAR = (1-beta)*c.RTTVAR + (beta * math.Abs(c.SRTT-R))
				c.SRTT = (1-alpha)*c.SRTT + (alpha * R)
			}
		}
	}

	// auto-adjust to remote host
	c.hasRemoteAddr = true
	c.lastHeard = time.Now().UnixMilli()

	if c.server { // only client can roam
		if !reflect.DeepEqual(raddr, c.remoteAddr) {
			c.remoteAddr = raddr
			util.Log.With("localAddr", conn.(net.Conn).LocalAddr()).
				With("remoteAddr", c.remoteAddr).With("hasRemoteAddr", c.hasRemoteAddr).
				Info("#recvOne server now attached to client")
		}
	}

	// util.Log.With("localAddr", conn.(net.Conn).LocalAddr()).
	// 	With("remoteAddr", c.remoteAddr).Debug("recvOne")

	return string(p.payload), nil // we do return out-of-order or duplicated packets to caller
}

func (c *Connection) setMTU(addr net.Addr) {
	if addr, ok := addr.(*net.UDPAddr); ok {
		if addr, ok := netip.AddrFromSlice(addr.IP); ok {
			if addr.Is6() {
				c.mtu = DEFAULT_IPV6_MTU - IPV6_HEADER_LEN
				return
			}
		}
	}
	c.mtu = DEFAULT_IPV4_MTU - IPV4_HEADER_LEN
}

// use the latest connection to send the message to remote
func (c *Connection) send(s string) (sendError error) {
	c.Lock()
	defer c.Unlock()

	conn := c.sock()
	// check hibernate case
	if !c.hasRemoteAddr {
		util.Log.With("localAddr", conn.(net.Conn).LocalAddr()).
			With("remoteAddr", c.remoteAddr).With("hasRemoteAddr", c.hasRemoteAddr).Debug("send message")
		return
	}

	px := c.newPacket(s)
	p := c.session.Encrypt(px.toMessage())

	// write data back to remote
	var (
		bytesSent int
		err       error
	)

	if c.server {
		bytesSent, _, err = conn.WriteMsgUDP(p, nil, c.remoteAddr.(*net.UDPAddr)) // server
		util.Log.With("localAddr", conn.(net.Conn).LocalAddr()).
			With("remoteAddr", c.remoteAddr).Debug("send message")
	} else {
		bytesSent, _, err = conn.WriteMsgUDP(p, nil, nil) // client connection is connected
		util.Log.With("localAddr", conn.(net.Conn).LocalAddr()).
			With("remoteAddr", conn.(net.Conn).RemoteAddr()).Debug("send message")
	}

	if err != nil {
		return fmt.Errorf("#send write: %w", err)
	}
	if bytesSent != len(p) {
		// Make sendto() failure available to the frontend.
		// consider change the sendError to error type
		sendError = fmt.Errorf("#send data sent (%d) doesn't match expected data (%d)", bytesSent, len(p))

		// with conn.Write() method, there is no chance of EMSGSIZE
		// payload MTU of last resort
		c.mtu = DEFAULT_SEND_MTU
	}

	now := time.Now().UnixMilli()
	if c.server {
		if now-c.lastHeard > SERVER_ASSOCIATION_TIMEOUT {
			c.hasRemoteAddr = false
			util.Log.With("localAddr", conn.(net.Conn).LocalAddr()).
				With("remoteAddr", c.remoteAddr).Warn("#send server now detached from client")
		}
	} else {
		if now-c.lastPortChoice > PORT_HOP_INTERVAL && now-c.lastRoundtripSuccess > PORT_HOP_INTERVAL {
			c.hopPort()
		}
	}
	return
}

func (c *Connection) Close() {
	for i := range c.socks {
		c.socks[i].Close()
	}
}

// receive packet from remote side, for client, there might be sevral connections
// to the server, Recv() will iterate every connection in order and read from the
// connection with the specified timeout (millisecond) value.
func (c *Connection) Recv(timeout int) (payload string, remoteAddr net.Addr, err error) {
	c.Lock()
	defer c.Unlock()

	// util.Log.With("remoteAddr", c.remoteAddr).
	// 	With("hasRemoteAddr", c.hasRemoteAddr).Debug("got message")
	for i := range c.socks {
		c.socks[i].SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))

		payload, err = c.recvOne(c.socks[i])
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// util.Log.With("i", i).With("error", err).Warn("#recv")
				continue
			} else if errors.Is(err, unix.EWOULDBLOCK) {
				// EAGAIN is processed by go netpoll
				continue
			} else {
				break
			}
		}

		remoteAddr = c.remoteAddr
		util.Log.With("i", i).With("localAddr", c.socks[i].(net.Conn).LocalAddr()).
			With("remoteAddr", c.remoteAddr).With("payload", len(payload)).
			With("hasRemoteAddr", c.hasRemoteAddr)
		c.pruneSockets()
		return
	}

	// return timeout if it's the case
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return "", nil, os.ErrDeadlineExceeded
	}
	return
}

func (c *Connection) getMTU() int {
	c.RLock()
	defer c.RUnlock()
	return c.mtu
}

// func (c *Connection) port() string {
// 	// TODO need implementation
// 	return ""
// }

func (c *Connection) getKey() string {
	return c.key.String()
}

func (c *Connection) getHasRemoteAddr() bool {
	c.RLock()
	defer c.RUnlock()
	return c.hasRemoteAddr
}

// calculate and restrict the RTO (retransmission timeout) between 50-1000 ms.
func (c *Connection) timeout() int64 {
	c.RLock()
	defer c.RUnlock()

	// uint64_t RTO = lrint(ceil(SRTT + 4 * RTTVAR))
	RTO := (int64)(math.Round(math.Ceil(c.SRTT + 4*c.RTTVAR)))
	if RTO < MIN_RTO {
		RTO = MIN_RTO
	} else if RTO > MAX_RTO {
		RTO = MAX_RTO
	}
	return RTO
}

func (c *Connection) getSRTT() float64 {
	c.RLock()
	defer c.RUnlock()
	return c.SRTT
}

func (c *Connection) getRemoteAddr() net.Addr {
	c.RLock()
	defer c.RUnlock()
	return c.remoteAddr
}

func (c *Connection) setLastRoundtripSuccess(success int64) {
	c.Lock()
	defer c.Unlock()
	c.lastRoundtripSuccess = success
}

// func (c *Connection) SetReadDeadline(t time.Time) (err error) {
// 	c.Lock()
// 	defer c.Unlock()
//
// 	for i := range c.socks {
// 		err = c.socks[i].SetReadDeadline(t)
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	return nil
// }
