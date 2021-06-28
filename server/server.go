package server

import (
	"log"
	"net"
	"strconv"
	"stun"
)


func Serve(address string) {
	udpAddr,err := net.ResolveUDPAddr("udp",address)
	if err != nil {
		log.Fatal(err)
		return
	}
	udpConn,err := net.ListenUDP("udp",udpAddr)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer udpConn.Close()
	buf := make([]byte,1500)
	for {
		n,rUdpAddr,err := udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		if stun.IsMessage(buf[:n]) {
			m,err := stun.ToMessage(buf[:n])
			if err != nil {
				log.Fatal(err)
			} else {
				log.Printf("receive a message from client,%v",m.ToString())
				switch m.MessageType() {
				case stun.BindReq : handleBindReq(udpConn,rUdpAddr,m)
				case stun.ShareSecretReq : handleShareSecretReq(udpConn,m)
				}
			}
		}
	}

}
func handleBindReq(udpConn *net.UDPConn,rUdpAddr *net.UDPAddr,msg stun.OutMessage) error {
	traId := msg.TransactionId()
	resp, err := stun.NewBindResponse(traId[:],rUdpAddr.String(),udpConn.LocalAddr().String(),rUdpAddr.String())
	if err != nil {
		return err
	}
	av := msg.GetAttribute(stun.AttrChangeRequest)
	if av == nil {
		av = [2]bool{false,false}
	}
	cip := av.([2]bool)

	if !cip[0] && !cip[1] {
		udpConn.WriteToUDP(resp.ToRaw(),rUdpAddr)
	} else {
		addr := udpConn.LocalAddr().(*net.UDPAddr)
		port := addr.Port
		ip:= addr.IP.String()
		//if cip[0] {
		//	ips := strings.Split(ip,".")
		//	ips3, err := strconv.Atoi(ips[3])
		//	if err != nil {
		//		return err
		//	}
		//	if ips3 > 128 {
		//		ips3 -= 1
		//	} else {
		//		ips3 += 1
		//	}
		//	ips[3] = strconv.Itoa(ips3)
		//	ip = strings.Join(ips,".")
		//}
		if cip[1] {
			if port < 65500 {
				port += 1
			} else {
				port -= 1
			}
		}
		udpAddr,err := net.ResolveUDPAddr("udp",ip+":"+strconv.Itoa(port))
		if err != nil {
			return err
		}
		udpConn,err := net.DialUDP("udp",udpAddr,rUdpAddr)
		if err != nil {
			return err
		}
		defer udpConn.Close()
		udpConn.WriteToUDP(resp.ToRaw(),rUdpAddr)
	}

	return nil
}
func handleShareSecretReq(udpConn *net.UDPConn,msg stun.OutMessage) error {
	return nil
}