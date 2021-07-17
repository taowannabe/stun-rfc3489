package server

import (
	"log"
	"math"
	"net"
	"stun"
	"stun/transform"
	"stun/util"
	"syscall"
)

func Serve(address string) {
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Fatal(err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer udpConn.Close()
	log.Printf("%s%s", "listen on ", address)
	buf := make([]byte, 1500)
	for {
		n, rUdpAddr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Fatal(err)
		}
		if stun.IsMessage(buf[:n]) {
			m, err := stun.ToMessage(buf[:n])
			if err != nil {
				log.Fatal(err)
			} else {
				log.Printf("receive a message from client,%v", m.ToString())
				switch m.MessageType() {
				case stun.BindReq:
					handleBindReq(udpConn, rUdpAddr, m)
				case stun.ShareSecretReq:
					handleShareSecretReq(udpConn, m)
				}
			}
		}
	}

}
func handleBindReq(udpConn *net.UDPConn, rUdpAddr *net.UDPAddr, msg stun.OutMessage) error {
	traId := msg.TransactionId()
	resp, err := stun.NewBindResponse(traId[:], rUdpAddr.String(), udpConn.LocalAddr().String(), rUdpAddr.String())
	if err != nil {
		log.Fatal(err)
	}
	av := msg.GetAttribute(stun.AttrChangeRequest)
	if av == nil {
		av = [2]bool{false, false}
	}
	cip := av.([2]bool)

	if !cip[0] && !cip[1] {
		log.Printf("send a message to client,%v", resp.ToString())
		udpConn.WriteToUDP(resp.ToRaw(), rUdpAddr)
	} else {
		sAddr := udpConn.LocalAddr().(*net.UDPAddr)
		port := sAddr.Port
		sIp := make([]byte, len(sAddr.IP))
		copy(sIp, sAddr.IP)
		if cip[0] {
			len := len(sIp)
			sIp[len-1] = ((sIp[len-1] + 1) % 254) + 1
		}
		if cip[1] {
			port = (port + 1) % math.MaxInt8
		}
		srcIp, dstIp := util.Ip2l(sIp), util.Ip2l(rUdpAddr.IP)
		//log.Printf("srcIp:%v,dstIp:%v,sport:%v,dport:%v", sAddr.IP, rUdpAddr.IP, port, rUdpAddr.Port)
		udpPkg, err := transform.NewUdpPackage(srcIp, dstIp, uint16(port), uint16(rUdpAddr.Port), resp.ToRaw())
		if err != nil {
			log.Fatal(err)
		}
		ipPkg, err := transform.NewIpPackage(srcIp, dstIp, udpPkg.ToRaw())
		if err != nil {
			log.Fatal(err)
		}

		fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
		if err != nil {
			log.Fatal(err)
		}
		defer syscall.Shutdown(fd, syscall.SHUT_RDWR)
		var dst syscall.SockaddrInet4
		dst.Addr[0], dst.Addr[1], dst.Addr[2], dst.Addr[3] = rUdpAddr.IP[0], rUdpAddr.IP[1], rUdpAddr.IP[2], rUdpAddr.IP[3]
		log.Printf("send a message to client,%v", resp.ToString())
		err = syscall.Sendto(fd, ipPkg.ToRaw(), 0, &dst)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}
func handleShareSecretReq(udpConn *net.UDPConn, msg stun.OutMessage) error {
	return nil
}
