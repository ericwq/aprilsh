package encrypt

import (
	"fmt"
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

func TestAESbase(t *testing.T) {
	s := "Hello"
	key := "zb0SLh88rdSHswjcgcC6949ZUuopGXTt"

	ciphertext, _ := AesGCMEncrypt(key, s)
	fmt.Println(ciphertext)

	plaintext, _ := AesGCMDecrypt(key, ciphertext)
	fmt.Printf("Decrypt:: %s\n", plaintext)
}
