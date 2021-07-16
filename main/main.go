package main

import (
	"flag"
	"fmt"
	"stun/client"
	"stun/server"
)

const (
	serverMode = "server"
	clientMode = "client"
)

func main() {
	m := flag.String("m", "server", "server or client")
	h := flag.String("h", "127.0.0.1:3478", "host")
	l := flag.String("l", "127.0.0.1:12345", "local host")
	flag.Parse()

	if serverMode == *m {
		server.Serve(*h)
	} else if clientMode == *m {
		fmt.Printf("%s", client.NatTypeName(client.Detect(*l, *h)))
	} else {
		fmt.Printf("%s", "参数不合法")
	}
}
