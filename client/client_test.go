package client

import (
	"fmt"
	"testing"
)

func TestClient(t *testing.T) {
	natType := Detect("127.0.0.1:1234")
	fmt.Print(NatTypeName(natType))
}

