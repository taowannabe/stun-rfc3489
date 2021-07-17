package main

import (
	"flag"
	"fmt"
	"stun/client"
	"stun/server"
)

const (
	serverMode       = "server"
	clientModeEchoOn = "client-echo-on"
	clientModeEchoTo = "client-echo-to"
)

func main() {
	m := flag.String("m", "server", "server or client")
	s := flag.String("s", "127.0.0.1:3478", "server host")
	l := flag.String("l", "127.0.0.1:12345", "local host")
	r := flag.String("r", "127.0.0.1:12345", "endpoint host")
	flag.Parse()

	if serverMode == *m {
		server.Serve(*s)
	} else if clientModeEchoOn == *m {
		client.ListenEcho(*l, *s)
	} else if clientModeEchoTo == *m {
		client.Echo(*l, *r, *s)
	} else {
		fmt.Printf("%s", "参数不合法")
	}
}
