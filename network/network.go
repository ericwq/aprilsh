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
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
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

func NewPacket2(m encrypt.Message) *Packet {
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

	MIN_RTO uint64 = 50   // ms
	MAX_RTO uint64 = 1000 // ms

	PORT_RANGE_LOW  = 60001
	PORT_RANGE_HIGH = 60999

	SERVER_ASSOCIATION_TIMEOUT = 40000
	PORT_HOP_INTERVAL          = 10000

	MAX_PORTS_OPEN     = 10
	MAX_OLD_SOCKET_AGE = 60000

	CONGESTION_TIMESTAMP_PENALTY = 500 // ms
)

type Connection struct {
	socks         []net.PacketConn
	hasRemoteAddr bool
	remoteAddr    net.Addr
	server        bool

	mtu int

	key     *encrypt.Base64Key
	session *encrypt.Session

	direction                Direction
	savedTimestamp           int16
	savedTimestampReceivedAt uint64
	expectedReceiverSeq      uint64

	lastHeard            int64
	lastPortChoice       int64
	lastRoundTripSuccess int64 // transport layer needs to tell us this

	RTTHit bool
	SRTT   float32
	RTTVAR float32

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
			c.logW.Printf("Invalid port range. %q\n", desiredPort)
			return nil
		}
	}

	// fmt.Printf("#connection (%d:%d)\n", desiredPortLow, desiredPortHigh)
	if !c.tryBind(desiredIp, desiredPortLow, desiredPortHigh) {
		// fmt.Printf("#connection failed\n\n")
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

	radd, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, port))
	if err != nil {
		c.logW.Printf("#connection %s\n", err)
		return nil
	}
	conn, err := net.DialUDP("udp", nil, radd)
	if err != nil {
		c.logW.Printf("#connection %s\n", err)
		return nil
	}

	c.socks = append(c.socks, conn)
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

func (c *Connection) setup() {
	c.lastPortChoice = time.Now().UnixMilli()
}

func (c *Connection) sock() net.PacketConn {
	if len(c.socks) == 0 {
		return nil
	}
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
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				// isable path MTU discovery
				// opErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_MTU_DISCOVER, syscall.IP_PMTUDISC_DONT)
				// if opErr != nil {
				// 	fmt.Printf("#ListenConfig %s\n", opErr)
				// 	return
				// }

				// int dscp = 0x92; /* OS X does not have IPTOS_DSCP_AF42 constant */
				// opErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, unix.IP_TOS, 0x02)
				// if opErr != nil {
				// 	fmt.Printf("#ListenConfig %s\n", opErr)
				// 	return
				// }

				// request explicit congestion notification on received datagrams
				// opErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_RECVTOS, 1)
				// if opErr != nil {
				// 	fmt.Printf("#ListenConfig %s\n", opErr)
				// 	return
				// }

				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
			}); err != nil {
				return err
			}
			return opErr
		},
	}

	for i := searchLow; i <= searchHigh; i++ {
		address := net.JoinHostPort(desireIp, strconv.Itoa(i))
		// fmt.Printf("#tryBind %d-%d, address=%q\n", i, searchHigh, address)
		l, err := lc.ListenPacket(context.Background(), "udp", address)
		if err != nil {
			if i == searchHigh {
				// last port to search
				c.logW.Printf("#tryBind error=%q\n", err)
			} else {
				c.logW.Printf("#tryBind error=%q\n", err)
			}
		} else {
			c.socks = append(c.socks, l)
			// TODO setMTU()
			return true
		}
	}
	return false
}
