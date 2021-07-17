package stun

import (
	"errors"
	"net"
)

type Attribute struct {
	attrType AttrType
	length   uint16
	value    []byte
}

func (a *Attribute) toRaw() ([]byte, error) {
	raw := make([]byte, 4, a.length+4)
	bin.PutUint16(raw, uint16(a.attrType))
	bin.PutUint16(raw[2:], a.length)
	raw = append(raw, a.value...)
	return raw, nil
}

type AttrType uint16

const (
	AttrMappedAddress     AttrType = 0x0001 //
	AttrResponseAddress   AttrType = 0x0002 //
	AttrChangeRequest     AttrType = 0x0003 //
	AttrSourceAddress     AttrType = 0x0004 //
	AttrChangedAddress    AttrType = 0x0005 //
	AttrUsername          AttrType = 0x0006 //
	AttrPassword          AttrType = 0x0007 //
	AttrMessageIntegrity  AttrType = 0x0008 //
	AttrErrorCode         AttrType = 0x0009 //
	AttrUnknownAttributes AttrType = 0x000a //
	AttrReflectedFrom     AttrType = 0x000b //
)

func AttrTypeName(attrType AttrType) string {
	switch attrType {
	case AttrMappedAddress:
		return "AttrMappedAddress"
	case AttrResponseAddress:
		return "AttrResponseAddress"
	case AttrChangeRequest:
		return "AttrChangeRequest"
	case AttrSourceAddress:
		return "AttrSourceAddress"
	case AttrChangedAddress:
		return "AttrChangedAddress"
	case AttrUsername:
		return "AttrUsername"
	case AttrPassword:
		return "AttrPassword"
	case AttrMessageIntegrity:
		return "AttrMessageIntegrity"
	case AttrErrorCode:
		return "AttrErrorCode"
	case AttrUnknownAttributes:
		return "AttrUnknownAttributes"
	case AttrReflectedFrom:
		return "AttrReflectedFrom"
	}
	return ""
}

func address2bytes(address string) ([]byte, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil || addr.IP == nil {
		return nil, errors.New("invalid address")
	}
	addressBytes := make([]byte, 8)
	bin.PutUint16(addressBytes, uint16(0x0001))
	port := uint16(addr.Port)
	bin.PutUint16(addressBytes[2:], port)
	copy(addressBytes[4:8], addr.IP[len(addr.IP)-4:len(addr.IP)])
	return addressBytes, nil
}

func bytes2Address(bytes []byte) string {
	if len(bytes) < 8 {
		return ""
	}
	addr := net.UDPAddr{}
	addr.IP, addr.Port = bytes[4:8], int(bin.Uint16(bytes[2:4]))
	return addr.String()
}

func newAttrMappedAddress(mappedAddress string) (Attribute, error) {
	addrBytes, err := address2bytes(mappedAddress)
	if err != nil {
		return Attribute{}, err
	}
	return Attribute{AttrMappedAddress, uint16(8), addrBytes}, nil
}
func newAttrResponseAddress(respAddress string) (Attribute, error) {
	addrBytes, err := address2bytes(respAddress)
	if err != nil {
		return Attribute{}, err
	}
	return Attribute{AttrResponseAddress, uint16(8), addrBytes}, nil
}
func newAttrChangeRequest(changeIp bool, changePort bool) (Attribute, error) {
	value := uint8(0x00)
	if changeIp {
		value = value | uint8(0x04)
	}
	if changePort {
		value = value | uint8(0x02)
	}
	bytes := [4]byte{0, 0, 0, value}
	return Attribute{AttrChangeRequest, uint16(4), bytes[:]}, nil
}
func newAttrSourceAddress(sourceAddress string) (Attribute, error) {
	addrBytes, err := address2bytes(sourceAddress)
	if err != nil {
		return Attribute{}, err
	}
	return Attribute{AttrSourceAddress, uint16(8), addrBytes}, nil
}
func newAttrChangedAddress(changedAddress string) (Attribute, error) {
	addrBytes, err := address2bytes(changedAddress)
	if err != nil {
		return Attribute{}, err
	}
	return Attribute{AttrChangedAddress, uint16(8), addrBytes}, nil
}
func newAttrUsername() (Attribute, error)          { return Attribute{}, nil }
func newAttrPassword() (Attribute, error)          { return Attribute{}, nil }
func newAttrMessageIntegrity() (Attribute, error)  { return Attribute{}, nil }
func newAttrErrorCode() (Attribute, error)         { return Attribute{}, nil }
func newAttrUnknownAttributes() (Attribute, error) { return Attribute{}, nil }
func newAttrReflectedFrom() (Attribute, error)     { return Attribute{}, nil }
