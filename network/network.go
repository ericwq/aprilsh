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
	"encoding/binary"
	"time"

	"github.com/ericwq/aprilsh/encrypt"
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
	// p.payload = payload
	copy(p.payload, payload)

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
	// p.payload = m.GetPayload()
	copy(p.payload, m.GetPayload())

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
