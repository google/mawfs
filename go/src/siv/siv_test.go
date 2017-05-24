// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Test to verify that jacobsa AES-SIV package produces the results that we
// expect.

package siv

import (
	"bytes"
	"encoding/hex"
	"github.com/jacobsa/crypto/siv"
	crtest "github.com/jacobsa/crypto/testing"
	"testing"
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

	data := [][]byte{[]byte(ad)}
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
