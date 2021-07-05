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
		ip:= make([]byte,len(addr.IP))
		copy(ip,addr.IP)
		if cip[0] {
			len := len(ip)
			ip[len - 1] = ((ip[len - 1] + 1) % 254) + 1
		}
		if cip[1] {
			port = (port + 1) % math.MaxInt8
		}
		//udpAddr := net.UDPAddr{ip,port,""}

		rawConn,err := udpConn.SyscallConn()
		if err != nil {
			log.Fatal(err)
			return err
		}
		srcIp,dstIp := util.Ip2l(ip),util.Ip2l(addr.IP)
		udpPkg,err :=  transform.NewUdpPackage(srcIp,dstIp,uint16(port),uint16(addr.Port),resp.ToRaw())
		if err != nil {
			log.Fatal(err)
			return err
		}
		ipPkg,err := transform.NewIpPackage(srcIp,dstIp,udpPkg.ToRaw())
		if err != nil {
			log.Fatal(err)
			return err
		}

		err = rawConn.Write(func(s uintptr) bool {
			// todo sendto
			//_, opErr := syscall.Sendto(int(s), ipPkg.ToRaw())
			//if opErr == syscall.EAGAIN {
			//	return false
			//}
			return true
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
func handleShareSecretReq(udpConn *net.UDPConn,msg stun.OutMessage) error {
	return nil
}