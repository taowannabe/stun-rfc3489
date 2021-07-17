package transform

import (
	"encoding/binary"
	"errors"
	"log"
	"math"
	"net"
	"stun"
	"stun/util"
	"syscall"
	"time"
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
		id:           uint16(12345),
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
	raw[0] = (p.version << 4) + (p.headerLength / 4)
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
		if i+1 < rawLength {
			code = uint32(bin.Uint16(raw[i : i+2]))
		} else {
			code = uint32(raw[i]) << 8
		}
		checkCode += code
		checkCode = (checkCode >> 16) + (checkCode & math.MaxUint16)
	}
	bin.PutUint16(raw[18:], uint16(^checkCode))
	return raw[udpCheckHeaderLength : udpCheckHeaderLength+u.udpLength]
}
func NewUdpPackage(srcAddr, dstAddr uint32, srcPort, dstPort uint16, data []byte) (*UdpPackage, error) {
	u := UdpPackage{
		srcAddr:   srcAddr,
		dstAddr:   dstAddr,
		udpLength: uint16(udpHeaderLength + len(data)),
		srcPort:   srcPort,
		dstPort:   dstPort,
		data:      data,
	}
	return &u, nil
}

type P2pConn struct {
	lAddr   net.UDPAddr
	rAddr   net.UDPAddr
	sAddr   net.UDPAddr
	nAddr   net.UDPAddr
	udpConn *net.UDPConn
	fd      int
}

func hole(lAddr, rAddr *net.UDPAddr) (string, error) {
	conn, err := net.DialUDP("udp", lAddr, rAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	id := stun.NewTransactionID()
	request, err := stun.NewBindRequest(id[:], lAddr.String(), false, false)
	if err != nil {
		return "", err
	}
	conn.Write(request.ToRaw())

	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("%v", err.Error())
		return "", err
	}
	if stun.IsMessage(buf[:n]) {
		m, err := stun.ToMessage(buf[:n])
		if err != nil {
			return "", err
		} else {
			switch m.MessageType() {
			case stun.BindResp:
				log.Printf("receive message for hole from server,%v", m.ToString())
				av := m.GetAttribute(stun.AttrMappedAddress)
				return av.(string), nil
			default:
				return "", errors.New("hole failed")
			}
		}
	}
	return "", errors.New("hole failed")
}
func DialP2p(laddr, raddr, saddr net.Addr) (p2pConn *P2pConn, err error) {
	lAddr, err := net.ResolveUDPAddr("udp", laddr.String())
	if err != nil {
		return nil, err
	}
	rAddr, err := net.ResolveUDPAddr("udp", raddr.String())
	if err != nil {
		return nil, err
	}
	sAddr, err := net.ResolveUDPAddr("udp", saddr.String())
	if err != nil {
		return nil, err
	}
	naddr, err := hole(lAddr, sAddr)
	if err != nil {
		return nil, err
	}
	nAddr, err := net.ResolveUDPAddr("udp", naddr)
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			syscall.Shutdown(fd, syscall.SHUT_RDWR)
		}
	}()
	udpConn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			udpConn.Close()
		}
	}()

	if err != nil {
		return nil, err
	}
	p2pConn = &P2pConn{
		lAddr:   *lAddr,
		rAddr:   *rAddr,
		sAddr:   *sAddr,
		nAddr:   *nAddr,
		udpConn: udpConn,
		fd:      fd,
	}
	// 测试对端
	id := stun.NewTransactionID()
	request, err := stun.NewBindRequest(id[:], naddr, true, false)
	if err != nil {
		return nil, err
	}
	raw := request.ToRaw()
	log.Printf("message: %s", request.ToString())
	log.Printf("raw: %x", raw)
	_, err = p2pConn.Write(raw)
	if err != nil {
		return nil, err
	}
	return p2pConn, nil
}
func ListenP2p(laddr, saddr net.Addr) (*P2pConn, error) {
	lAddr, err := net.ResolveUDPAddr("udp", laddr.String())
	if err != nil {
		return nil, err
	}
	sAddr, err := net.ResolveUDPAddr("udp", saddr.String())
	if err != nil {
		return nil, err
	}
	naddr, err := hole(lAddr, sAddr)
	if err != nil {
		return nil, err
	}
	nAddr, err := net.ResolveUDPAddr("udp", naddr)
	if err != nil {
		return nil, err
	}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return nil, err
	}
	udpConn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		return nil, err
	}
	conn := &P2pConn{
		lAddr:   *lAddr,
		sAddr:   *sAddr,
		nAddr:   *nAddr,
		udpConn: udpConn,
		fd:      fd,
	}
	return conn, nil
}
func (c *P2pConn) ok() bool {
	return c != nil && c.udpConn != nil && c.fd != 0 && c.rAddr.IP != nil
}
func (c *P2pConn) readOk() bool {
	return c != nil && c.udpConn != nil && c.fd != 0
}
func (c *P2pConn) NatAddr() *net.UDPAddr {
	return &c.nAddr
}

//Read implements the Conn Read method.
func (c *P2pConn) Read(b []byte) (int, error) {
	if !c.readOk() {
		return 0, syscall.EINVAL
	}
	n, err := c.udpConn.Read(b)
	if c.rAddr.IP == nil {
		nb := make([]byte, n)
		copy(nb, b)
		err = c.setRAddr(nb)
		if err != nil {
			return 0, err
		}
	}
	return n, err
}
func (c *P2pConn) setRAddr(byte []byte) error {
	log.Printf("raw: %x", byte)
	m, err := stun.ToMessage(byte)
	if err != nil {
		return err
	}
	if stun.BindReq == m.MessageType() {
		log.Printf("receive message for hole from endpoint,%v", m.ToString())
		av := m.GetAttribute(stun.AttrResponseAddress)
		addr, err := net.ResolveUDPAddr("udp", av.(string))
		if err != nil {
			return err
		}
		c.rAddr = *addr
		return nil
	}
	return errors.New("hole to endpoint failed")

}

// Write implements the Conn Write method.
func (c *P2pConn) Write(b []byte) (int, error) {
	if !c.ok() {
		return 0, syscall.EINVAL
	}
	srcIp, dstIp := util.Ip2l(c.sAddr.IP), util.Ip2l(c.rAddr.IP)
	//log.Printf("srcIp:%v,dstIp:%v,sport:%v,dport:%v", c.sAddr.IP, c.rAddr.IP, c.sAddr.Port, c.rAddr.Port)
	udpPkg, err := NewUdpPackage(srcIp, dstIp, uint16(c.sAddr.Port), uint16(c.rAddr.Port), b)
	if err != nil {
		return 0, err
	}
	ipPkg, err := NewIpPackage(srcIp, dstIp, udpPkg.ToRaw())
	if err != nil {
		return 0, nil
	}
	sa := syscall.SockaddrInet4{}
	sa.Addr[0], sa.Addr[1], sa.Addr[2], sa.Addr[3] = c.rAddr.IP[0], c.rAddr.IP[1], c.rAddr.IP[2], c.rAddr.IP[3]

	err = syscall.Sendto(c.fd, ipPkg.ToRaw(), 0, &sa)
	return len(b), err
}

// Close closes the connection.
func (c *P2pConn) Close() error {
	if !c.ok() {
		return syscall.EINVAL
	}
	err := c.udpConn.Close()
	if err != nil {
		return err
	}
	err = syscall.Shutdown(c.fd, syscall.SHUT_RDWR)
	return err
}

// LocalAddr returns the local network address.
// The Addr returned is shared by all invocations of LocalAddr, so
// do not modify it.
func (c *P2pConn) LocalAddr() net.Addr {
	if !c.ok() {
		return nil
	}
	return &c.lAddr
}

// RemoteAddr returns the remote network address.
// The Addr returned is shared by all invocations of RemoteAddr, so
// do not modify it.
func (c *P2pConn) RemoteAddr() net.Addr {
	if !c.ok() {
		return nil
	}
	return &c.rAddr
}

// SetDeadline implements the Conn SetDeadline method.
func (c *P2pConn) SetDeadline(t time.Time) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := c.udpConn.SetDeadline(t); err != nil {
		return err
	}
	return nil
}

// SetReadDeadline implements the Conn SetReadDeadline method.
func (c *P2pConn) SetReadDeadline(t time.Time) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := c.udpConn.SetReadDeadline(t); err != nil {
		return err
	}
	return nil
}

// SetWriteDeadline implements the Conn SetWriteDeadline method.
func (c *P2pConn) SetWriteDeadline(t time.Time) error {
	if !c.ok() {
		return syscall.EINVAL
	}
	if err := c.udpConn.SetWriteDeadline(t); err != nil {
		return err
	}
	return nil
}
