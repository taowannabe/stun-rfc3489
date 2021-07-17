package stun

import (
	"fmt"
	"log"
	"testing"
)

func TestToMessage(t *testing.T) {
	raw := make([]byte, 44)
	_, err := fmt.Sscanf("000100189566c74d10037c4d7bbb0407d1e2c649000200080001303c2418eec2000300080000", "%x", &raw)
	if err != nil {
		log.Fatalln(err)
	}
	ToMessage(raw)
}
