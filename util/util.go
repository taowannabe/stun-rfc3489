package util

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"net"
	"unsafe"
)



func GetByteOrder() binary.ByteOrder {
	if isLittleEndian() {
		return binary.LittleEndian
	} else {
		return binary.BigEndian
	}
}
func isLittleEndian() bool  {
	u := uint16(0x0102)
	p := unsafe.Pointer(&u)
	bp := (*byte)(p)
	return *bp == 0x02
}
func hmacSha1 (message []byte,key []byte) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}
func Ip2l(ip net.IP) uint32 {
	l:= len(ip)
	if l < 4 {
		return 0
	}
	return binary.BigEndian.Uint32(ip[l-4:l])
}