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
	"fmt"
	"io"
	"sync/atomic"
)

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

	dst := prngFill(1)
	u8 = dst[0]
	return u8
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
	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()

	key, err := base64.StdEncoding.DecodeString(printableKey)
	if err != nil {
		panic(fmt.Sprintf("Key must be well-formed base64. %s\n", err))
	}

	if len(key) != 16 {
		panic("Key must represent 16 octets.")
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

var counter uint64

func Unique() uint64 {
	atomic.AddUint64(&counter, 1)
	return counter
}

type Message struct {
	nonce []byte
	text  []byte
}

type Session struct {
	base64Key Base64Key
	aead      cipher.AEAD
}

func NewSession(key Base64Key) (*Session, error) {
	s := &Session{base64Key: key}
	block, err := aes.NewCipher([]byte(s.base64Key.key))
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	s.aead = aesgcm
	return s, nil
}

// https://stackoverflow.com/questions/1220751/how-to-choose-an-aes-encryption-mode-cbc-ecb-ctr-ocb-cfb
// https://installmd.com/c/276/go/encrypt-a-string-using-aes-gcm

// encrypt with AES-128 GCM
func (s *Session) encrypt(plainText *Message) []byte {
	nonce := plainText.nonce

	cipherText := s.aead.Seal(nonce, nonce, plainText.text, nil)
	// fmt.Printf("#encrypt cipherText=% x, %p\n\n", cipherText, cipherText)
	return cipherText
}

// decrypt with AES-128 GCM
func (s *Session) decrypt(text []byte) *Message {
	ns := s.aead.NonceSize()
	nonce, cipherText := text[:ns], text[ns:]

	// fmt.Printf("#decrypt ciphertext=% x, %p\n", cipherText, cipherText)

	plainText, err := s.aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		panic(fmt.Sprintf("error = %s\n", err))
	}

	m := Message{}
	m.nonce = nonce
	m.text = plainText
	// fmt.Printf("#decrypt nonce=% x, plaintext=% x\n", nonce, plainText)

	return &m
}

func randomNonce() ([]byte, error) {
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return nonce, nil
}

// const (
// 	NONCE_LEN = 12
// )
//
// type Nonce struct {
// 	bytes [NONCE_LEN]byte
// }

/*
func AesGCMIv() ([]byte, error) {
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return nonce, nil
}

func AesGCMEncrypt(key string, text string) (string, error) {
	// When decoded the key should be 16 bytes (AES-128) or 32 (AES-256).
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	plaintext := []byte(text)

	nonce, err := AesGCMIv()
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	fmt.Printf("#AesGCMEncrypt encrypt nonce=% x\n", nonce)
	return fmt.Sprintf("%x", ciphertext), nil
}

func AesGCMDecrypt(key string, text string) (string, error) {
	// When decoded the key should be 16 bytes (AES-128) or 32 (AES-256).
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	in, _ := hex.DecodeString(text)
	ns := aesgcm.NonceSize()
	nonce, ciphertext := in[:ns], in[ns:]

	fmt.Printf("#AesGCMDecrypt decrypt nonce=% x\n", nonce)
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext[:]), nil
}
*/
