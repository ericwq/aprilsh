package encrypt

import (
	"errors"
	"io"
	"log"
	"os"
	"reflect"
	"testing"
)

func TestPrng(t *testing.T) {
	tc := []int{0, 1, 2, 4, 8, 16, 32}

	for _, v := range tc {
		got := prngFill(v)
		if v != len(got) {
			t.Errorf("prngFill got %#v\n", got)
		}
	}

	for i := 0; i < 8; i++ {
		got := PrngUint8()
		if got == 0 {
			t.Errorf("prngUint8 got %#v\n", got)
		}
	}
}

func TestBase64Key(t *testing.T) {
	defer func() {
		logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	}()
	logW.SetOutput(io.Discard)

	// normal key
	normalKey := NewBase64Key()
	printKey := normalKey.printableKey()
	gotNormal := NewBase64Key2(printKey)
	if !reflect.DeepEqual(normalKey.data(), gotNormal.data()) {
		t.Errorf("two keys should be the same. got key1=\n%v, key2=\n%v\n", normalKey, gotNormal)
	}

	// malform key
	malformBase64 := "/msvMB1KwXL+ygJHdJwwQ=="
	malformKey := NewBase64Key2(malformBase64)
	if malformKey != nil {
		t.Error("malform key should be nil.")
	}

	// key length is short
	shortLengthKey := &Base64Key{key: prngFill(8)}
	key4 := NewBase64Key2(shortLengthKey.printableKey())
	if key4 != nil {
		t.Error("key length is short.")
		t.Errorf("key length is short. %q\n", shortLengthKey.printableKey())
	}
}

func TestUnique(t *testing.T) {
	for i := 0; i < 10; i++ {
		v := Unique()
		expect := i + 1
		if v != uint64(i+1) {
			t.Errorf("Unique expect %d, got %d\n", expect, v)
		}
	}
}

func TestSession(t *testing.T) {
	tc := []struct {
		name      string
		plainText string
	}{
		{"english plain text", "Datagrams are encrypted and authenticated using AES-128 in the Offset Codebook mode [1]"},
		{"chinese plain text", "原子操作是比其它同步技术更基础的操作。原子操作是无锁的，常常直接通过CPU指令直接实现。"},
	}

	s, _ := NewSession(*NewBase64Key())
	for _, v := range tc {
		nonce, _ := randomNonce()
		message := Message{nonce: nonce, text: []byte(v.plainText)}

		// fmt.Printf("#before message nonce=% x, nonce=%p\n", message.nonce, message.nonce)
		cipherText := s.encrypt(&message)
		// fmt.Printf("#after cipherText=% x\n", cipherText)

		message2 := s.decrypt(cipherText)
		gotNonce := message2.nonce
		gotText := message2.text

		if !reflect.DeepEqual(nonce, gotNonce) {
			t.Errorf("%q expect nonce %v, got %v\n", v.name, nonce, gotNonce)
		}
		if v.plainText != string(gotText) {
			t.Errorf("%q expect plain text \n%q, got \n%q\n", v.name, v.plainText, gotText)
		}
	}
}

func TestSessionError(t *testing.T) {
	defer func() {
		logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	}()
	logW.SetOutput(io.Discard)

	b := Base64Key{}
	b.key = prngFill(9)

	if _, err := NewSession(b); err == nil {
		t.Errorf("expect wrong key size error, got %s\n", err)
	}

	b.key = prngFill(32)
	s, _ := NewSession(b)
	nilMessage := s.decrypt([]byte("zb0SLh88rdSHswjcgcC6949ZUuopGXTt"))
	if nilMessage != nil {
		t.Errorf("expect nil message returned from decrypt(), got %v\n", nilMessage)
	}
}

func fakeRand(io.Reader, []byte) (int, error) {
	return -2, errors.New("design this error on purpose.")
}

func TestRandomNonce(t *testing.T) {
	defer func() {
		logW = log.New(os.Stderr, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	}()
	logW.SetOutput(io.Discard)

	nonce, err := _randomNonce(fakeRand)

	if nonce != nil {
		t.Errorf("expect nil nonce, got %v\n %s\n", nonce, err)
	}
}
