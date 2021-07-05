package client

import (
	"log"
	"net"
	"stun"
	"time"
)

type NatType uint8
const timeout = 2
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
		case OpenInternet: return "OpenInternet"
		case FirewallBlocksUdp: return "FirewallBlocksUdp"
		case FirewallAllowsUdp: return "FirewallAllowsUdp"
		case FullConeNat: return "FullConeNat"
		case SymmetricNat: return "SymmetricNat"
		case RestrictedConeNat: return "RestrictedConeNat"
		case RestrictedPortConeNat: return "RestrictedPortConeNat"
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
func Detect(address string) NatType {
	conn, err := net.Dial("udp", address)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := make(chan string)
	go handleResp(conn, c)
	// test1
	if res,mappedAddress := test1(conn,c);res {
		// test2
		if test2(conn,c) {
			return OpenInternet
		} else {
			return FirewallAllowsUdp
		}
	} else {
		if mappedAddress == "" {
			return FirewallBlocksUdp
		}
		// test2
		if test2(conn,c) {
			return FullConeNat
		}
		// test1
		if !test12(conn,c,mappedAddress) {
			return SymmetricNat
		}
		// test3
		if test3(conn,c) {
			return RestrictedConeNat
		} else {
			return RestrictedPortConeNat
		}
	}
}
func test1(conn net.Conn,ch chan string) (bool,string) {
	m, err := stun.NewBindRequest(nil, "", false, false)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	conn.Write(m.ToRaw())
	select {
	case mappedAddress := <-ch: return mappedAddress == conn.LocalAddr().String(),mappedAddress
	case <-time.After(time.Second * timeout): return false,""
	}
}
func test12(conn net.Conn,ch chan string,address string) bool {
	m, err := stun.NewBindRequest(nil, address, false, false)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	conn.Write(m.ToRaw())
	select {
	case mappedAddress := <-ch: return mappedAddress == address
	case <-time.After(time.Second * timeout): return false
	}
}
func test2(conn net.Conn,ch chan string) bool {
	m, err := stun.NewBindRequest(nil, "", true, true)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	conn.Write(m.ToRaw())
	select {
	case <-ch: return true
	case <-time.After(time.Second * timeout): return false
	}
}
func test3(conn net.Conn,ch chan string) bool {
	m, err := stun.NewBindRequest(nil, "", false, true)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	conn.Write(m.ToRaw())
	select {
	case <-ch: return true
	case <-time.After(time.Second * timeout): return false
	}
}

func handleResp(conn net.Conn, c chan string) {
	buf := make([]byte, 1500)
	for true {
		n, err := conn.Read(buf)
		if err != nil {
			log.Fatal(err)
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
	log.Printf("receive message from server,%v",msg.ToString())
	av := msg.GetAttribute(stun.AttrMappedAddress)
	mappedAddress := av.(string)
	return mappedAddress
}
