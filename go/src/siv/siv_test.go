
package siv

import (
    "bytes"
    "github.com/jacobsa/crypto/siv"
    crtest "github.com/jacobsa/crypto/testing"
    "encoding/hex"
    "testing"
    "fmt"
)

// Verify that encryption/decryption in Go works for the example in the RFC,
// suggesting that it is completely compatible with Crack's SIV-AES.
func TestEncryptDecrypt(t *testing.T) {

    k := crtest.FromRfcHex("fffefdfc fbfaf9f8 f7f6f5f4" +
                           "f3f2f1f0 f0f1f2f3 f4f5f6f7" +
        						   "f8f9fafb fcfdfeff")

    ad := crtest.FromRfcHex("10111213 14151617 18191a1b 1c1d1e1f" +
        							"20212223 24252627")

    plain := crtest.FromRfcHex("11223344 55667788 99aabbcc ddee")
    finalCMAC := crtest.FromRfcHex("85632d07 c6e8f37f 950acd32 0a2ecc93")
    finalCipherText := crtest.FromRfcHex("40c02b96 90c4dc04 daef7f6a fe5c")
    macAndCipher := bytes.Join([][]byte{finalCMAC, finalCipherText}, nil)

    data := [][]byte{[]byte(ad)};
    ciphertext, err := siv.Encrypt(nil, k, []byte(plain), data)
    if err != nil {
        t.Error("error during encryption")
		t.Fail()
	}

	if bytes.Compare(ciphertext, macAndCipher) != 0 {
	   t.Errorf("Didn't get expected cipher text, got %s",
             hex.EncodeToString(ciphertext))
	   t.Fail()
	}

	decryptedPlaintext, err := siv.Decrypt(k, ciphertext, data)
	if err != nil {
	   t.Error("Error during decryption")
	   t.Fail()
	}

	if bytes.Compare(decryptedPlaintext, plain) != 0 {
	   t.Error("Decryption failed")
	   t.Fail()
	}
}
