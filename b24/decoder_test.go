package b24

import (
	"log"
	"os"
	"testing"
)

func TestDecodeString(t *testing.T) {
	log.SetOutput(os.Stderr)
	testVector := []byte{27, 36, 59, 15, 122, 107, 27, 36, 57, 15, 50, 62, 76, 76, 27, 124, 233, 164, 192, 249, 234, 208, 164, 185, 33, 33, 66, 104, 14, 49, 15, 79, 67, 251, 50, 72, 66, 50, 14, 33, 15, 55, 64, 76, 115, 14, 33, 15, 48, 45, 75, 98, 27, 125, 181, 181, 228, 175, 14, 33, 252, 27, 36, 59, 15, 122, 88, 122, 86}
	if len(testVector) != 69 {
		t.Fatal(len(testVector))
	}
	decoded, err := DecodeString(testVector)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != "[新]仮面ライダーリバイス　第1話「家族!契約!悪魔ささやく!」[デ][字]" {
		t.Fatalf("result is not what expected: %s", decoded)
	}
}
