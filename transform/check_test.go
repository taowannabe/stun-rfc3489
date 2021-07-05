package transform

import (
	"fmt"
	. "net"
	"stun/util"
	"testing"
)

func TestNewUdpPackage(t *testing.T) {
	srcIp := IPv4(byte(153), byte(19), byte(8), byte(104))
	dstIp := IPv4(byte(171), byte(3), byte(14), byte(11))
	udpPck,_:=NewUdpPackage(util.Ip2l(srcIp),util.Ip2l(dstIp),uint16(1087),uint16(13),[]byte{0x54,0x45,0x53,0x54,0x49,0x4e,0x47})
	raw := udpPck.ToRaw()
	fmt.Printf("% 32b",raw)
}