package client

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"stun"
	"stun/transform"
	"time"
)

type NatType uint8

const timeout = 10
const (
	OpenInternet          NatType = 1
	FirewallBlocksUdp     NatType = 2
	FirewallAllowsUdp     NatType = 3
	FullConeNat           NatType = 4
	SymmetricNat          NatType = 5
	RestrictedConeNat     NatType = 6
	RestrictedPortConeNat NatType = 7
)

func NatTypeName(t NatType) string {
	switch t {
	case OpenInternet:
		return "OpenInternet"
	case FirewallBlocksUdp:
		return "FirewallBlocksUdp"
	case FirewallAllowsUdp:
		return "FirewallAllowsUdp"
	case FullConeNat:
		return "FullConeNat"
	case SymmetricNat:
		return "SymmetricNat"
	case RestrictedConeNat:
		return "RestrictedConeNat"
	case RestrictedPortConeNat:
		return "RestrictedPortConeNat"
	}
	return ""
}

/** 测试nat类型

                       +--------+
                       |  Test  |
                       |   I    |
                       +--------+
                            |
                            |
                            V
                           /\              /\
                        N /  \ Y          /  \ Y             +--------+
         UDP     <-------/Resp\--------->/ IP \------------->|  Test  |
         Blocked         \ ?  /          \Same/              |   II   |
                          \  /            \? /               +--------+
                           \/              \/                    |
                                            | N                  |
                                            |                    V
                                            V                    /\
                                        +--------+  Sym.      N /  \
                                        |  Test  |  UDP    <---/Resp\
                                        |   II   |  Firewall   \ ?  /
                                        +--------+              \  /
                                            |                    \/
                                            V                     |Y
                 /\                         /\                    |
  Symmetric  N  /  \       +--------+   N  /  \                   V
     NAT  <--- / IP \<-----|  Test  |<--- /Resp\               Open
               \Same/      |   I    |     \ ?  /               Internet
                \? /       +--------+      \  /
                 \/                         \/
                 |                           |Y
                 |                           |
                 |                           V
                 |                           Full
                 |                           Cone
                 V              /\
             +--------+        /  \ Y
             |  Test  |------>/Resp\---->Restricted
             |   III  |       \ ?  /
             +--------+        \  /
                                \/
                                 |N
                                 |       Port
                                 +------>Restricted
*/
func Detect(lAddress, rAddress string) NatType {
	lAddr, err := net.ResolveUDPAddr("udp", lAddress)
	if err != nil {
		log.Fatal(err)
	}
	rAddr, err := net.ResolveUDPAddr("udp", rAddress)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := make(chan string)
	defer close(c)
	go handleResp(conn, c)
	// test1
	if res, mappedAddress := test1(conn, rAddr, c); res {
		// test2
		if test2(conn, rAddr, c) {
			return OpenInternet
		} else {
			return FirewallAllowsUdp
		}
	} else {
		if mappedAddress == "" {
			return FirewallBlocksUdp
		}
		// test2
		if test2(conn, rAddr, c) {
			return FullConeNat
		}
		// test1
		if !test12(conn, rAddr, c, mappedAddress) {
			return SymmetricNat
		}
		// test3
		if test3(conn, rAddr, c) {
			return RestrictedConeNat
		} else {
			return RestrictedPortConeNat
		}
	}
}
func test1(conn *net.UDPConn, rAddr *net.UDPAddr, ch chan string) (bool, string) {
	m, err := stun.NewBindRequest(nil, "", false, false)
	if err != nil {
		log.Fatal(err)
	}
	conn.WriteToUDP(m.ToRaw(), rAddr)
	select {
	case mappedAddress := <-ch:
		return mappedAddress == conn.LocalAddr().String(), mappedAddress
	case <-time.After(time.Second * timeout):
		return false, ""
	}
}
func test12(conn *net.UDPConn, rAddr *net.UDPAddr, ch chan string, address string) bool {
	m, err := stun.NewBindRequest(nil, address, false, false)
	if err != nil {
		log.Fatal(err)
	}
	conn.WriteToUDP(m.ToRaw(), rAddr)
	select {
	case mappedAddress := <-ch:
		return mappedAddress == address
	case <-time.After(time.Second * timeout):
		return false
	}
}
func test2(conn *net.UDPConn, rAddr *net.UDPAddr, ch chan string) bool {
	m, err := stun.NewBindRequest(nil, "", true, true)
	if err != nil {
		log.Fatal(err)
	}
	conn.WriteToUDP(m.ToRaw(), rAddr)
	select {
	case <-ch:
		return true
	case <-time.After(time.Second * timeout):
		return false
	}
}
func test3(conn *net.UDPConn, rAddr *net.UDPAddr, ch chan string) bool {
	m, err := stun.NewBindRequest(nil, "", false, true)
	if err != nil {
		log.Fatal(err)
	}
	conn.WriteToUDP(m.ToRaw(), rAddr)
	select {
	case <-ch:
		return true
	case <-time.After(time.Second * timeout):
		return false
	}
}

func handleResp(conn *net.UDPConn, c chan string) {
	buf := make([]byte, 1500)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("%v", err.Error())
			return
		}
		if stun.IsMessage(buf[:n]) {
			m, err := stun.ToMessage(buf[:n])
			if err != nil {
				log.Fatal(err)
			} else {
				switch m.MessageType() {
				case stun.BindResp:
					c <- handleBindResp(m)
				}
			}
		}
	}

}

func handleBindResp(msg stun.OutMessage) string {
	log.Printf("receive message from server,%v", msg.ToString())
	av := msg.GetAttribute(stun.AttrMappedAddress)
	mappedAddress := av.(string)
	return mappedAddress
}

func ListenEcho(laddr, saddr string) {
	lAddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		log.Fatal(err)
	}
	sAddr, err := net.ResolveUDPAddr("udp", saddr)
	if err != nil {
		log.Fatal(err)
	}

	p2p, err := transform.ListenP2p(lAddr, sAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listen echo on %s", p2p.NatAddr().String())
	go echoOut(p2p)
	go echoIn(p2p)
	<-make(chan struct{})

}
func Echo(laddr, raddr, saddr string) {
	lAddr, err := net.ResolveUDPAddr("udp", laddr)
	if err != nil {
		log.Fatal(err)
	}
	rAddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		log.Fatal(err)
	}
	sAddr, err := net.ResolveUDPAddr("udp", saddr)
	if err != nil {
		log.Fatal(err)
	}

	p2p, err := transform.DialP2p(lAddr, rAddr, sAddr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("echo to %s from %s \n", raddr, p2p.NatAddr().String())
	go echoOut(p2p)
	go echoIn(p2p)
	<-make(chan struct{})

}
func echoOut(reader io.Reader) {
	for {
		buf := make([]byte, 1500)
		n, err := reader.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		//log.Println("write a word to stdout")
		fmt.Println(string(buf[:n]))
	}
}
func echoIn(writer io.Writer) {
	in := bufio.NewReader(os.Stdin)
	for {
		// todo 优化自旋（EOF）
		str, _, err := in.ReadLine()
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		//log.Println("read a word from stdin")
		_, err = writer.Write(str)
		if err != nil {
			log.Fatal(err)
		}
	}
}
