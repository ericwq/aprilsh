// Copyright 2022 wangqi. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"sync/atomic"
	"syscall"

	"github.com/ericwq/aprilsh/util"
)

const (
	NONCE_LEN = 12

	RECEIVE_MTU = 2048
	ADDED_BYTES = 16 /* final OCB block */
)

// var logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)

// use sys call to generate random number
func PrngFill(size int) (dst []byte) {
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
		dst := PrngFill(1)
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
		// logW.Printf("#randomNonce. %s\n", err)
		util.Logger.Warn("#randomNonce", "error", err)
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
	b.key = PrngFill(16)
	return b
}

func NewBase64Key2(printableKey string) *Base64Key {
	key, err := base64.StdEncoding.DecodeString(printableKey)
	if err != nil {
		// logW.Printf("#Base64Key Key must be well-formed base64. %s\n", err)
		util.Logger.Warn("key must be well-formed base64", "error", err)
		return nil
	}

	if len(key) != 16 {
		// logW.Println("#Base64Key Key must represent 16 octets.")
		util.Logger.Warn("key must represent 16 octets.", "key", key)
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
	return atomic.LoadUint64(&counter)
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
	aead      cipher.AEAD
	base64Key Base64Key
	// sync.Mutex
}

func NewSession(key Base64Key) (*Session, error) {
	s := &Session{base64Key: key}
	block, err := aes.NewCipher([]byte(s.base64Key.key))
	if err != nil {
		// logW.Printf("#session %s\n", err)
		util.Logger.Warn("create session from key", "error", err)
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
	// s.Lock()
	// defer s.Unlock()
	nonce := plainText.nonce

	cipherText := s.aead.Seal(nonce, nonce, plainText.text, nil)
	return cipherText
}

// Decrypt with AES-128 GCM
func (s *Session) Decrypt(text []byte) (*Message, error) {
	// s.Lock()
	// defer s.Unlock()
	ns := s.aead.NonceSize()
	nonce, cipherText := text[:ns], text[ns:]
	// fmt.Printf("#decrypt ciphertext=% x, %p\n", cipherText, cipherText)

	plainText, err := s.aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		// logW.Printf("#decrypt %s\n", err)
		return nil, err
	}

	m := Message{}
	m.nonce = nonce
	m.text = plainText
	// fmt.Printf("#decrypt nonce=% x, plaintext=% x\n", nonce, plainText)

	return &m, nil
}

var savedCoreLimit uint64

// Disable dumping core, as a precaution to avoid saving sensitive data to disk.
func DisableDumpingCore() error {
	// the value argument is provided by last parameter of accessRlimit
	f := func(rlim *syscall.Rlimit, value uint64) {
		savedCoreLimit = rlim.Cur
		rlim.Cur = value
	}
	return accessRlimit(syscall.RLIMIT_CORE, f, 0)
}

// restore the dumping core to saved value
func ReenableDumpingCore() error {
	f := func(rlim *syscall.Rlimit, _ uint64) {
		rlim.Cur = savedCoreLimit
	}

	// we don't use value parameter, so it's not important the specific value
	return accessRlimit(syscall.RLIMIT_CORE, f, 0)
}

// get specified resource, then do some action defined by f, finally set the specififed resource
func accessRlimit(resource int, f func(rlim *syscall.Rlimit, value uint64), value uint64) error {
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(resource, &rlim); err != nil {
		return fmt.Errorf("Getrlimit() reports %s", err.Error())
	}

	f(&rlim, value)

	if err := syscall.Setrlimit(resource, &rlim); err != nil {
		return fmt.Errorf("Setrlimit() reports %s", err.Error())
	}
	return nil
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
