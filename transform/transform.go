package transform

import (
	"encoding/binary"
	"log"
	"math"
	"math/rand"
	"net"
	"syscall"
)

var bin = binary.BigEndian

const (
	fixedIpHeaderLength  = 20
	udpCheckHeaderLength = 12
	udpHeaderLength      = 8
	protocolUdp          = 17
	ttl                  = 54
)

type IpPackage struct {
	version      byte
	headerLength byte
	ds           byte
	totalLength  uint16
	id           uint16
	mf           bool
	df           bool
	sliceOffset  uint16
	ttl          byte
	protocol     byte
	checkSum     uint16
	srcAddr      uint32
	dstAddr      uint32
	data         []byte
}

func NewIpPackage(srcAddr uint32, dstAddr uint32, data []byte) (*IpPackage, error) {
	p := IpPackage{
		version:      4,
		headerLength: fixedIpHeaderLength,
		ds:           0,
		totalLength:  uint16(fixedIpHeaderLength + len(data)),
		id:           uint16(rand.Uint32()),
		mf:           false,
		df:           false,
		sliceOffset:  0,
		ttl:          ttl,
		protocol:     protocolUdp,
		srcAddr:      srcAddr,
		dstAddr:      dstAddr,
		data:         data,
	}
	return &p, nil
}
func (p *IpPackage) ToRaw() []byte {
	raw := make([]byte, fixedIpHeaderLength, len(p.data)+fixedIpHeaderLength)
	raw[0] = p.version + p.headerLength<<2
	raw[1] = p.ds
	bin.PutUint16(raw[2:], p.totalLength)
	bin.PutUint16(raw[4:], p.id)
	flag := 0x00
	if p.mf {
		flag += 0x01
	}
	if p.df {
		flag += 0x02
	}
	bin.PutUint16(raw[6:], (uint16(flag)<<13)+p.sliceOffset)
	raw[8] = p.ttl
	raw[9] = p.protocol
	bin.PutUint16(raw[10:], 0)
	bin.PutUint32(raw[12:], p.srcAddr)
	bin.PutUint32(raw[16:], p.dstAddr)

	// checkSum
	bin.PutUint16(raw[10:], ipCheckSum(raw))

	raw = append(raw, p.data...)
	return raw
}
func ipCheckSum(head []byte) uint16 {
	sum := uint32(0)
	for i := 0; i < len(head); i += 2 {
		sum += uint32(bin.Uint16(head[i : i+2]))
		sum = (sum >> 16) + (sum & math.MaxUint16)
	}
	return uint16(^sum)
}

type UdpPackage struct {
	srcAddr   uint32
	dstAddr   uint32
	udpLength uint16
	srcPort   uint16
	dstPort   uint16
	data      []byte
}

func (u *UdpPackage) ToRaw() []byte {
	headerLength := udpCheckHeaderLength + udpHeaderLength
	raw := make([]byte, headerLength, headerLength+len(u.data))

	bin.PutUint32(raw, u.srcAddr)
	bin.PutUint32(raw[4:], u.dstAddr)
	raw[8] = 0
	raw[9] = protocolUdp
	bin.PutUint16(raw[10:], u.udpLength)
	bin.PutUint16(raw[12:], u.srcPort)
	bin.PutUint16(raw[14:], u.dstPort)
	bin.PutUint16(raw[16:], u.udpLength)
	bin.PutUint16(raw[18:], 0)
	raw = append(raw, u.data...)

	checkCode := uint32(0)
	rawLength := len(raw)
	for i := 0; i < rawLength; i += 2 {
		code := uint32(0)
		if i + 1 < rawLength {
			code = uint32(bin.Uint16(raw[i:i+2]))
		} else {
			code =  uint32(raw[i]) << 8
		}
		checkCode += code
		checkCode = (checkCode >> 16) + (checkCode & math.MaxUint16)
	}
	bin.PutUint16(raw[18:],uint16(^checkCode))
	return raw[udpCheckHeaderLength:udpCheckHeaderLength + u.udpLength]
}
func NewUdpPackage(srcAddr, dstAddr uint32, srcPort, dstPort uint16, data []byte) (*UdpPackage, error) {
	u := UdpPackage{
		srcAddr:   srcAddr,
		dstAddr:   dstAddr,
		udpLength: uint16(udpHeaderLength+len(data)),
		srcPort:   srcPort,
		dstPort:   dstPort,
		data:      data,
	}
	return &u, nil
}

type CamouflagedUdpConn struct {
	laddr   net.Addr
	raddr   net.Addr
	udpConn net.UDPConn
	rawFd   int
}

func DialCamouflagedUdp(laddr, raddr net.Addr) (*CamouflagedUdpConn, error) {
	conn := &CamouflagedUdpConn{laddr: laddr, raddr: raddr}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, 0)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	conn.rawFd = fd
	udpAddr, err := net.ResolveUDPAddr("udp", laddr.String())
	if err != nil {
		log.Fatal(err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	conn.udpConn = *udpConn
	return conn, nil
}
func (c *CamouflagedUdpConn) ok() bool { return c != nil && c.rawFd != 0 }

// Read implements the Conn Read method.
//func (c *CamouflagedUdpConn) Read(b []byte) (int, error) {
//	if !c.ok() {
//		return 0, syscall.EINVAL
//	}
//	n, err :=
//		syscall.read
//	return n, err
//}
//
//// Write implements the Conn Write method.
//func (c *CamouflagedUdpConn) Write(b []byte) (int, error) {
//	if !c.ok() {
//		return 0, syscall.EINVAL
//	}
//	n, err := c.Write(b)
//	if err != nil {
//		err = &OpError{Op: "write", Net: c.net, Source: c.laddr, Addr: c.raddr, Err: err}
//	}
//	return n, err
//}
//
//// Close closes the connection.
//func (c *CamouflagedUdpConn) Close() error {
//	if !c.ok() {
//		return syscall.EINVAL
//	}
//	err := c.Close()
//	if err != nil {
//		err = &OpError{Op: "close", Net: c.net, Source: c.laddr, Addr: c.raddr, Err: err}
//	}
//	return err
//}
//
//// LocalAddr returns the local network address.
//// The Addr returned is shared by all invocations of LocalAddr, so
//// do not modify it.
//func (c *CamouflagedUdpConn) LocalAddr() Addr {
//	if !c.ok() {
//		return nil
//	}
//	return c.laddr
//}
//
//// RemoteAddr returns the remote network address.
//// The Addr returned is shared by all invocations of RemoteAddr, so
//// do not modify it.
//func (c *CamouflagedUdpConn) RemoteAddr() Addr {
//	if !c.ok() {
//		return nil
//	}
//	return c.raddr
//}
//// SetDeadline implements the Conn SetDeadline method.
//func (c *CamouflagedUdpConn) SetDeadline(t time.Time) error {
//	if !c.ok() {
//		return syscall.EINVAL
//	}
//	if err := c.SetDeadline(t); err != nil {
//		return &OpError{Op: "set", Net: c.net, Source: nil, Addr: c.laddr, Err: err}
//	}
//	return nil
//}
//
//// SetReadDeadline implements the Conn SetReadDeadline method.
//func (c *CamouflagedUdpConn) SetReadDeadline(t time.Time) error {
//	if !c.ok() {
//		return syscall.EINVAL
//	}
//	if err := c.SetReadDeadline(t); err != nil {
//		return &OpError{Op: "set", Net: c.net, Source: nil, Addr: c.laddr, Err: err}
//	}
//	return nil
//}
//
//// SetWriteDeadline implements the Conn SetWriteDeadline method.
//func (c *CamouflagedUdpConn) SetWriteDeadline(t time.Time) error {
//	if !c.ok() {
//		return syscall.EINVAL
//	}
//	if err := c.SetWriteDeadline(t); err != nil {
//		return &OpError{Op: "set", Net: c.net, Source: nil, Addr: c.laddr, Err: err}
//	}
//	return nil
//}
