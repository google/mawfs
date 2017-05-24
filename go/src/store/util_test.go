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

package blockstore

import (
	"bytes"
	"fmt"
	"testing"
)

func TestAltEncoding(t *testing.T) {
	for _, val := range []string{"1", "12", "123", "12345",
		"this is a longer test of encoding."} {
		fmt.Printf("Encoded: %s\n", altEncode([]byte(val)))
		result, err := altDecode(altEncode([]byte(val)))
		if err != nil {
			t.Error("Error decoding: ", err)
			t.Fail()
		}
		if bytes.Compare(result, []byte(val)) != 0 {
			t.Errorf("Failed encode/decode of %s", val)
			t.Fail()
		}
	}
}
