package client

import (
	"fmt"
	"testing"
)

func TestClient(t *testing.T) {
	natType := Detect("192.168.1.8:12345", "192.168.1.8:3478")
	//natType := Detect("127.0.0.1:3478")
	fmt.Println(NatTypeName(natType))
}
