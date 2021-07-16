package server

import (
	"log"
	"stun/transform"
	"stun/util"
	"syscall"
	"testing"
)

func TestServe(t *testing.T) {
	Serve("0.0.0.0:3478")
}

func TestRawSocket(t *testing.T) {

	srcIp, dstIp := util.Ip2l([]byte{127, 0, 0, 1}[:]), util.Ip2l([]byte{127, 0, 0, 1}[:])
	udpPkg, err := transform.NewUdpPackage(srcIp, dstIp, uint16(1234), uint16(12345), []byte{1, 2, 3}[:])
	if err != nil {
		log.Fatal(err)
	}
	ipPkg, err := transform.NewIpPackage(srcIp, dstIp, udpPkg.ToRaw())
	if err != nil {
		log.Fatal(err)
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_IPV4)
	if err != nil {
		log.Fatal(err)
	}
	_, opErr := syscall.Write(fd, ipPkg.ToRaw())
	if opErr == syscall.EAGAIN {
		log.Print(opErr)
	}

}
