package stun

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"unsafe"
)

var bin = getByteOrder()

func getByteOrder() binary.ByteOrder {
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