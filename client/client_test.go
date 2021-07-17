package client

import (
	"bufio"
	"log"
	"os"
	"testing"
)

func TestClient(t *testing.T) {
	//natType := Detect("0.0.0.0:12345", "120.92.164.196:3478")
	//natType := Detect("127.0.0.1:3478")
	//fmt.Println(NatTypeName(natType))

	in := bufio.NewReader(os.Stdin)
	bytes := make([]byte, 1024)
	for {

		n, err := in.Read(bytes)
		if err != nil {
			log.Fatal(err)
		}
		log.Print(bytes[:n])
	}

}
