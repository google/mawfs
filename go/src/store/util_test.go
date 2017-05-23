
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