/*

MIT License

Copyright (c) 2022 wangqi

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

package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/netip"
	"os"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
	// "golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

type Direction uint

const (
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
	diff = int(tsnew - tsold)
	if diff < 0 {
		diff += 65536
	}

	return uint16(diff)
}

type Packet struct {
	seq            uint64
	direction      Direction
	timestamp      uint16
	timestampReply uint16
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

func NewPacketFrom(m encrypt.Message) *Packet {
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

type ADDRESS_TYPE int

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

	MIN_RTO uint64 = 50   // ms
	MAX_RTO uint64 = 1000 // ms

	PORT_RANGE_LOW  = 60001
	PORT_RANGE_HIGH = 60999

	SERVER_ASSOCIATION_TIMEOUT = 40000
	PORT_HOP_INTERVAL          = 10000

	MAX_PORTS_OPEN     = 10
	MAX_OLD_SOCKET_AGE = 60000

	CONGESTION_TIMESTAMP_PENALTY = 500 // ms

	ADDRESS_IPV4       ADDRESS_TYPE = 0
	ADDRESS_IPV6       ADDRESS_TYPE = 1
	ADDRESS_IPV4_IN_V6 ADDRESS_TYPE = 2

	NETWORK = "udp4"
)

type Connection struct {
	socks         []net.PacketConn
	hasRemoteAddr bool
	// remoteAddr    net.UDPAddr
	remoteAddr net.Addr
	server     bool

	mtu int

	key     *encrypt.Base64Key
	session *encrypt.Session

	direction                Direction
	savedTimestamp           int16
	savedTimestampReceivedAt int64
	expectedReceiverSeq      uint64

	lastHeard            int64
	lastPortChoice       int64
	lastRoundTripSuccess int64 // transport layer needs to tell us this

	RTTHit bool
	SRTT   float64
	RTTVAR float64

	sendError string
	logW      *log.Logger
}

func NewConnection(desiredIp string, desiredPort string) *Connection { // server
	c := &Connection{}

	c.hasRemoteAddr = false
	c.server = true

	c.mtu = DEFAULT_SEND_MTU

	c.key = encrypt.NewBase64Key()
	c.session, _ = encrypt.NewSession(*c.key)

	c.direction = TO_CLIENT
	c.savedTimestamp = -1

	c.lastHeard = -1
	c.lastPortChoice = -1
	c.lastRoundTripSuccess = -1

	c.RTTHit = false
	c.SRTT = 1000
	c.RTTVAR = 500

	c.logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	c.socks = make([]net.PacketConn, 0)

	c.setup()

	/* The mosh wrapper always gives an IP request, in order
	   to deal with multihomed servers. The port is optional. */

	/* If an IP request is given, we try to bind to that IP, but we also
	   try INADDR_ANY. If a port request is given, we bind only to that port. */

	desiredPortHigh := -1
	desiredPortLow := -1
	ok := false

	if len(desiredPort) > 0 {
		if desiredPortLow, desiredPortHigh, ok = ParsePortRange(desiredPort, c.logW); !ok {
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

	c.socks = make([]net.PacketConn, 0)
	c.hasRemoteAddr = false
	c.server = false

	c.mtu = DEFAULT_SEND_MTU

	c.key = encrypt.NewBase64Key2(keyStr)
	c.session, _ = encrypt.NewSession(*c.key)

	c.direction = TO_SERVER
	c.savedTimestamp = -1

	c.lastHeard = -1
	c.lastPortChoice = -1
	c.lastRoundTripSuccess = -1

	c.RTTHit = false
	c.SRTT = 1000
	c.RTTVAR = 500

	c.logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

	c.setup()
	if !c.dialUDP(ip, port) {
		c.logW.Printf("#connection failed to dial %s:%s\n", ip, port)
		return nil
	}
	c.setMTU()

	return c
}

func parsePort(portStr string, hint string, logW *log.Logger) (port int, ok bool) {
	value, err := strconv.Atoi(portStr)
	if err != nil {
		logW.Printf("Invalid (%s) port number (%s)\n", hint, portStr)
		return
	}

	if value < 0 || value > 65535 {
		logW.Printf("(%s) port number %d outside valid range [0..65535]\n", hint, value)
		return
	}

	port = value
	ok = true
	return
}

// parse "port" or "portlow:porthigh"
func ParsePortRange(desiredPort string, logW *log.Logger) (desiredPortLow, desiredPortHigh int, ok bool) {
	var value int

	all := strings.Split(desiredPort, ":")
	if len(all) == 2 {
		// parse "portlow:porthigh"
		if value, ok = parsePort(all[0], "low", logW); !ok {
			return
		} else {
			desiredPortLow = value
		}

		if value, ok = parsePort(all[1], "high", logW); !ok {
			return
		} else {
			desiredPortHigh = value
		}

		if desiredPortLow > desiredPortHigh {
			logW.Printf("low port %d greater than high port %d\n", desiredPortLow, desiredPortHigh)
			ok = false
			return
		}
	} else {
		// parse solo port
		if value, ok = parsePort(all[0], "solo", logW); !ok {
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
func markECN(fd int) error {
	tosConf, err := unix.GetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_TOS)
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
	err = unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_TOS, tosConf)
	if err != nil {
		return err
	}
	return nil
}

// update lastPortChoice timestamp
func (c *Connection) setup() {
	c.lastPortChoice = time.Now().UnixMilli()
}

func (c *Connection) sock() net.PacketConn {
	return c.socks[len(c.socks)-1]
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
	lc := net.ListenConfig{
		Control: func(network, address string, raw syscall.RawConn) error {
			var opErr error
			if err := raw.Control(func(fd uintptr) { // TODO the following code only works on linux and for ipv4!
				// dsable path MTU discovery
				// IP_MTU_DISCOVER and IP_PMTUDISC_DONT is not defined on macOS
				// opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_MTU_DISCOVER, unix.IP_PMTUDISC_DONT)
				// if opErr != nil {
				// 	fmt.Printf("#ListenConfig %s\n", opErr.Error())
				// 	return
				// }

				if opErr = markECN(int(fd)); opErr != nil {
					c.logW.Printf("#tryBind %s\n", opErr)
					return
				}

				// request explicit congestion notification on received datagrams
				if opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_RECVTOS, 1); opErr != nil {
					c.logW.Printf("#tryBind %s\n", opErr)
					return
				}
				// https://groups.google.com/g/golang-nuts/c/TcHb_bXT18U

				// syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				// syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			}); err != nil {
				c.logW.Printf("#tryBind %s\n", err)
				return err
			}
			return opErr
		},
	}

	for i := searchLow; i <= searchHigh; i++ {
		address := net.JoinHostPort(desireIp, strconv.Itoa(i))

		ladd, err := net.ResolveUDPAddr(NETWORK, address)
		if err != nil {
			c.logW.Printf("#tryBind %s\n", err)
			return false
		}

		// fmt.Printf("#tryBind %d-%d, ladd=%q\n", i, searchHigh, ladd)
		conn, err := lc.ListenPacket(context.Background(), NETWORK, ladd.String())
		if err != nil {
			if i == searchHigh { // last port to search
				c.logW.Printf("#tryBind error=%q address=%q\n", err, address)
			}
		} else {
			c.socks = append(c.socks, conn)
			c.setMTU()
			return true
		}
	}
	return false
}

// build a packet based on sending and receiving timestamp and payload
func (c *Connection) newPacket(payload string) *Packet {
	var outgoingTimestampReply uint16

	outgoingTimestampReply = 0 // -1 for c++
	now := time.Now().UnixMilli()
	if now-c.savedTimestampReceivedAt < 1000 { // we have a recent received timestamp
		// send "corrected" timestamp advanced by how long we held it
		outgoingTimestampReply = uint16(c.savedTimestamp) + uint16(now-c.savedTimestampReceivedAt)
		c.savedTimestamp = -1
		c.savedTimestampReceivedAt = 0
	}
	return NewPacket(c.direction, timestamp16(), outgoingTimestampReply, []byte(payload))
}

func (c *Connection) dialUDP(ip, port string) bool {
	var d net.Dialer

	var opErr error
	d.Control = func(network, address string, raw syscall.RawConn) error {
		if err := raw.Control(func(fd uintptr) {
			if opErr = markECN(int(fd)); opErr != nil {
				c.logW.Printf("#dialUDP %s\n", opErr)
				return
			}
		}); err != nil {
			return err
		}
		return opErr
	}

	conn, err := d.Dial(NETWORK, net.JoinHostPort(ip, port))
	if err != nil {
		c.logW.Printf("#dialUDP %s\n", err)
		return false
	}

	c.remoteAddr = conn.RemoteAddr()
	c.socks = append(c.socks, conn.(net.PacketConn))
	c.hasRemoteAddr = true

	// c.logW.Printf("#dialUDP successfully connect to %q\n", net.JoinHostPort(ip, port))
	return true
}

// clear the old and over size sockets
func (c *Connection) pruneSockets() {
	// don't keep old sockets if the new socket has been working for long enough
	if len(c.socks) > 1 {
		now := time.Now().UnixMilli()
		if now-c.lastPortChoice > MAX_OLD_SOCKET_AGE {
			numToKill := len(c.socks) - 1

			// TODO race condition
			socks := make([]net.PacketConn, 1)
			copy(socks, c.socks[numToKill:])
			c.socks = socks
		}
	} else {
		return
	}

	// make sure we don't have too many receive sockets open
	if len(c.socks) > MAX_PORTS_OPEN {
		numToKill := len(c.socks) - MAX_PORTS_OPEN

		// TODO race condition
		socks := make([]net.PacketConn, MAX_PORTS_OPEN)
		copy(socks, c.socks[numToKill:])
		c.socks = socks
	}
}

// reconnect server with
func (c *Connection) hopPort() {
	c.setup()

	host, port, _ := net.SplitHostPort(c.remoteAddr.String())
	if !c.dialUDP(host, port) {
		c.logW.Printf("#hopPort failed to dial %s\n", c.remoteAddr)
		return
	}

	c.pruneSockets()
}

// check the current remote address type ipv4,ipv6 or ipv4_in_ipv6
func (c *Connection) remoteAddrType() ADDRESS_TYPE {
	// determine the ipv6/ipv4 network

	if addr, ok := c.remoteAddr.(*net.UDPAddr); ok {
		if addr, ok := netip.AddrFromSlice(addr.IP); ok {
			if addr.Is4In6() {
				return ADDRESS_IPV4_IN_V6
			} else if addr.Is6() {
				return ADDRESS_IPV6
			}
		}
	}
	return ADDRESS_IPV4
}

func (c *Connection) setMTU() {
	switch c.remoteAddrType() {
	case ADDRESS_IPV4:
		c.mtu = DEFAULT_IPV4_MTU - IPV4_HEADER_LEN
	case ADDRESS_IPV4_IN_V6:
		fallthrough
	case ADDRESS_IPV6:
		c.mtu = DEFAULT_IPV6_MTU - IPV6_HEADER_LEN
	}
}

// use the latest connection to send the message to remote
func (c *Connection) send(s string) {
	if !c.hasRemoteAddr {
		return
	}

	px := c.newPacket(s)
	p := c.session.Encrypt(px.toMessage())

	// write data to socket (latest socket)
	conn := c.sock().(*net.UDPConn)
	bytesSent, err := conn.Write(p)
	if err != nil {
		c.sendError = fmt.Sprintf("#send %s\n", err)
		return
	}

	if bytesSent != len(p) {
		// Make sendto() failure available to the frontend.
		c.sendError = fmt.Sprintf("#send size %s\n", err)

		// TODO in case EMSGSIZE err, adjust mtu to DEFAULT_SEND_MTU
	}

	now := time.Now().UnixMilli()
	if c.server {
		if now-c.lastHeard > SERVER_ASSOCIATION_TIMEOUT {
			c.hasRemoteAddr = false
			c.logW.Printf("#send server now detached from client. [%s]\n", c.remoteAddr)
		}
	} else {
		if now-c.lastPortChoice > PORT_HOP_INTERVAL && now-c.lastRoundTripSuccess > PORT_HOP_INTERVAL {
			c.hopPort()
		}
	}
	// fmt.Printf("#send %q from %q to %q\n", p, conn.LocalAddr(), conn.RemoteAddr())
}

// receive packet from remote
func (c *Connection) recv() string {
	for i := range c.socks {
		payload, err := c.recvOne(c.socks[i].(*net.UDPConn))
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) {
				// test the above method
				continue
			} else {
				c.logW.Printf("#recv %s\n", err)
				break
			}
		}

		c.pruneSockets()
		return payload
	}
	return ""
}

func (c *Connection) recvOne(conn *net.UDPConn) (string, error) {
	data := make([]byte, c.mtu)
	oob := make([]byte, 40)

	// read from the socket
	// in golang, the flags parameters for recvfrom system call is always 0.
	// that means it's always blocking read. which means unix.MSG_DONTWAIT can't be applied here.
	n, oobn, flags, raddr, err := conn.ReadMsgUDP(data, oob)
	if err != nil {
		return "", err
	}
	if n < 0 {
		return "", errors.New("#recvOne receive zero or negative length data.")
	}

	if flags&unix.MSG_TRUNC == unix.MSG_TRUNC {
		return "", errors.New("#recvOne received oversize datagram.")
	}

	// fmt.Printf("#recvOne flags=0x%x, MSG_TRUNC=0x%x, n=%d, oobn=%d, err=%p, raddr=%s\n", flags, unix.MSG_TRUNC, n, oobn, err, raddr)
	// parse the optional ancillary data
	ctrlMsgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return "", err
	}

	// receive ECN
	congestionExperienced := false
	for _, ctrlMsg := range ctrlMsgs {
		if ctrlMsg.Header.Level == unix.IPPROTO_IP &&
			(ctrlMsg.Header.Type == unix.IP_TOS || ctrlMsg.Header.Type == unix.IP_RECVTOS) {
			// fmt.Printf("#recvOne got %08b\n", ctrlMsg.Data)
			// CE: RFC 3168
			// https://www.ietf.org/rfc/rfc3168.html
			congestionExperienced = ctrlMsg.Data[0]&0x03 == 0x03
		}
	}
	// fmt.Printf("#recvOne congestionExperienced=%t\n", congestionExperienced)

	p := NewPacketFrom(*c.session.Decrypt(data[:n]))
	// prevent malicious playback to sender
	if c.server {
		if p.direction != TO_SERVER {
			return "", nil
		}
	} else {
		if p.direction != TO_CLIENT {
			return "", nil
		}
	}

	if p.seq >= c.expectedReceiverSeq { // don't use out-of-order packets for timestamp or targeting
		// this is security-sensitive because a replay attack could otherwise screw up the timestamp and targeting
		c.expectedReceiverSeq = p.seq + 1

		if p.timestamp != 0 {
			c.savedTimestamp = int16(p.timestamp)
			c.savedTimestampReceivedAt = time.Now().UnixMilli()

			if congestionExperienced {
				// signal counterparty to slow down
				// this will gradually slow the counterparty down to the minimum frame rate
				c.savedTimestamp -= CONGESTION_TIMESTAMP_PENALTY
				if c.server {
					c.logW.Println("#recvOne received explicit congestion notification.")
				}
			}
		}

		if p.timestampReply != 0 { // -1 in c++
			now16 := timestamp16()
			R := float64(timestampDiff(now16, p.timestampReply))

			if R < 5000 { // ignore large values, e.g. server was Ctrl-Zed
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
			if reflect.DeepEqual(raddr, c.remoteAddr) {
				c.remoteAddr = raddr
				c.logW.Printf("#recvOne server now attached to client at %s\n", c.remoteAddr)
			}
		}
	}

	return string(p.payload), nil // we do return out-of-order or duplicated packets to caller
}
