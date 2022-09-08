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

package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"io"
	"log"
	"os"
	"sync/atomic"
)

const (
	NONCE_LEN = 12
)

var logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

// use sys call to generate random number
func prngFill(size int) (dst []byte) {
	dst = make([]byte, size)
	if size == 0 {
		return dst
	}

	rand.Read(dst)
	// _, err := rand.Read(dst)
	// if err != nil {
	// 	panic(fmt.Sprintf("Could not read random number. %s", err))
	// }
	return
}

func PrngUint8() uint8 {
	var u8 uint8
	for u8 == 0 {
		dst := prngFill(1)
		u8 = dst[0]
	}
	return u8
}

func randomNonce() ([]byte, error) {
	return _randomNonce()
}

// don't use this function directly, it's for internal purpose only
type _randFunc func(io.Reader, []byte) (int, error)

// don't use this function directly, it's for internal purpose only
func _randomNonce(r ..._randFunc) ([]byte, error) {
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	var f _randFunc
	if len(r) > 0 {
		f = r[0]
	} else {
		f = io.ReadFull
	}

	nonce := make([]byte, NONCE_LEN)
	if _, err := f(rand.Reader, nonce); err != nil {
		logW.Printf("#randomNonce. %s\n", err)
		return nil, err
	}

	return nonce, nil
}

type Base64Key struct {
	key []uint8
}

// random key 128bit
func NewBase64Key() *Base64Key {
	b := &Base64Key{}
	b.key = prngFill(16)
	return b
}

func NewBase64Key2(printableKey string) *Base64Key {
	key, err := base64.StdEncoding.DecodeString(printableKey)
	if err != nil {
		logW.Printf("#Base64Key Key must be well-formed base64. %s\n", err)
		return nil
	}

	if len(key) != 16 {
		logW.Println("#Base64Key Key must represent 16 octets.")
		return nil
	}

	b := &Base64Key{}
	b.key = key
	// // to catch changes after the first 128 bits
	// if printableKey != b.printableKey() {
	// 	panic("Base64 key was not encoded 128-bit key.")
	// }

	return b
}

func (b *Base64Key) printableKey() string {
	return base64.StdEncoding.EncodeToString(b.key)
}

func (b *Base64Key) data() []uint8 {
	return b.key
}

func (b *Base64Key) String() string {
	return b.printableKey()
}

var counter uint64

func Unique() uint64 {
	atomic.AddUint64(&counter, 1)
	return counter
}

type Message struct {
	nonce []byte
	text  []byte
}

func NewMessage(seqNonce uint64, payload []byte) (m *Message) {
	m = &Message{}
	// fmt.Printf("#Message seqNonce=% x\n", seqNonce)

	b := make([]byte, NONCE_LEN)
	binary.BigEndian.PutUint32(b[0:], 0)
	binary.BigEndian.PutUint64(b[4:], seqNonce)

	m.nonce = b
	m.text = payload

	// fmt.Printf("#Message head=% x, nonce=% x\n", m.nonce[:4], m.nonce[4:])
	// fmt.Printf("#Message text=% x\n", m.text)
	return m
}

func (m *Message) NonceVal() uint64 {
	seqNonce := binary.BigEndian.Uint64(m.nonce[4:])

	// fmt.Printf("#NonceVal seqNonce=%x\n", seqNonce)
	return seqNonce
}

// the first two bytes is timestamp in text field
func (m *Message) GetTimestamp() uint16 {
	// var ts uint16
	// buf := bytes.NewReader(m.text[:2])
	// err := binary.Read(buf, hostEndian, &ts)
	// if err != nil {
	// 	fmt.Printf("#GetTimestamp failed. %s\n", err)
	// }
	//
	// return ts
	return binary.BigEndian.Uint16(m.text[:2])
}

// the [2:4] bytes is timestampReply in text field
func (m *Message) GetTimestampReply() uint16 {
	// var tsr uint16
	// buf := bytes.NewReader(m.text[2:4])
	// err := binary.Read(buf, hostEndian, &tsr)
	// if err != nil {
	// 	fmt.Printf("#GetTimestampReply failed. %s\n", err)
	// }
	//
	// return tsr
	return binary.BigEndian.Uint16(m.text[2:4])
}

func (m *Message) GetPayload() (payload []byte) {
	return m.text[4:]
}

type Session struct {
	base64Key Base64Key
	aead      cipher.AEAD
}

func NewSession(key Base64Key) (*Session, error) {
	s := &Session{base64Key: key}
	block, err := aes.NewCipher([]byte(s.base64Key.key))
	if err != nil {
		logW.Printf("#session %s\n", err)
		return nil, err
	}

	aesgcm, _ := cipher.NewGCM(block)
	// aesgcm, err := cipher.NewGCM(block)
	// if err != nil {
	// 	return nil, err
	// }

	s.aead = aesgcm
	return s, nil
}

// https://stackoverflow.com/questions/1220751/how-to-choose-an-aes-encryption-mode-cbc-ecb-ctr-ocb-cfb
// https://installmd.com/c/276/go/encrypt-a-string-using-aes-gcm

// Encrypt with AES-128 GCM
func (s *Session) Encrypt(plainText *Message) []byte {
	nonce := plainText.nonce

	cipherText := s.aead.Seal(nonce, nonce, plainText.text, nil)
	return cipherText
}

// Decrypt with AES-128 GCM
func (s *Session) Decrypt(text []byte) *Message {
	ns := s.aead.NonceSize()
	nonce, cipherText := text[:ns], text[ns:]
	// fmt.Printf("#decrypt ciphertext=% x, %p\n", cipherText, cipherText)

	plainText, err := s.aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		logW.Printf("#decrypt %s\n", err)
		return nil
	}

	m := Message{}
	m.nonce = nonce
	m.text = plainText
	// fmt.Printf("#decrypt nonce=% x, plaintext=% x\n", nonce, plainText)

	return &m
}

// https://commandcenter.blogspot.com/2012/04/byte-order-fallacy.html
//
// var hostEndian binary.ByteOrder
//
// func init() {
// 	buf := [2]byte{}
// 	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)
//
// 	switch buf {
// 	case [2]byte{0xCD, 0xAB}:
// 		hostEndian = binary.LittleEndian
// 	case [2]byte{0xAB, 0xCD}:
// 		hostEndian = binary.BigEndian
// 	default:
// 		panic("Could not determine native endianness.")
// 	}
// }