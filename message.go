package stun

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
)

var bin = binary.BigEndian
const (
	transactionIDSize = 16                                                      // transactionId size of bytes
	messageTypeSize   = 2                                                       // message type size of bytes
	messageLengthSize = 2                                                       // message length size of bytes
	messageHeaderSize = messageTypeSize + messageLengthSize + transactionIDSize // message header size of bytes
	attrTypeSize      = 2                                                       // attr type size of bytes
	attrLengthSize    = 2                                                       // attr length size of bytes
	integritySize     = 20 + attrTypeSize + attrLengthSize                      // message_integrity attr size of bytes
)

type message struct {
	messageType   MessageType
	length        uint16 // len(Raw) not including header
	transactionID [transactionIDSize]byte
	attributes    []Attribute
}

func (m *message) TransactionId() [transactionIDSize]byte {
	return m.transactionID
}

func (m *message) Length() uint16 {
	return m.length
}

func (m *message) MessageType() MessageType {
	return m.messageType
}

func (m *message) GetAttribute(attrType AttrType) interface{} {
	for _, attribute := range m.attributes {
		if attribute.attrType != attrType {
			continue
		}
		switch attrType {
		case AttrMappedAddress,
			AttrResponseAddress,
			AttrSourceAddress,
			AttrChangedAddress:
			return bytes2Address(attribute.value)
		case AttrChangeRequest:
			f := attribute.value[1]
			return [2]bool{f ^ 0x04 == 0x04,f ^ 0x02 == 0x02}
		case AttrUsername:
		case AttrPassword:
		case AttrMessageIntegrity:
		case AttrErrorCode:
		case AttrUnknownAttributes:
		case AttrReflectedFrom:
		}
	}
	return nil
}

func IsMessage(bytes []byte) bool {
	if _, err := detectMessageType(bytes); err != nil {
		return false
	}
	return true
}
func detectMessageType(bytes []byte) (MessageType, error) {
	if len(bytes) < messageTypeSize {
		return 0, errors.New("no message header")
	}
	messageType := MessageType(bin.Uint16(bytes[:messageTypeSize]))
	if messageType != BindReq &&
		messageType != BindResp &&
		messageType != BindErrorResp &&
		messageType != ShareSecretReq &&
		messageType != ShareSecretResp &&
		messageType != ShareSecretErrorResp {
		return 0, errors.New("invalid message header")
	}
	return messageType, nil
}
func ToMessage(bytes []byte) (OutMessage, error) {
	messageType, err := detectMessageType(bytes)
	if err != nil {
		return nil, err
	}
	m := message{}
	m.messageType = messageType

	p := messageTypeSize
	m.length = bin.Uint16(bytes[p : p+messageLengthSize])
	p += messageLengthSize

	copy(m.transactionID[:], bytes[p:p+transactionIDSize])
	p += transactionIDSize

	attributes := make([]Attribute, 0, 8)
	for len(bytes) > p+attrTypeSize {
		attrType := AttrType(bin.Uint16(bytes[p : p+attrTypeSize]))
		p += attrTypeSize

		if attrType > AttrReflectedFrom {
			break
		}
		attrLength := bin.Uint16(bytes[p : p+attrLengthSize])
		p += attrLengthSize

		attrValue := make([]byte, attrLength)
		copy(attrValue, bytes[p:p+int(attrLength)])
		p += int(attrLength)

		attributes = append(attributes, Attribute{attrType: attrType, length: attrLength, value: attrValue})
	}
	m.attributes = attributes
	return &m, nil
}
func (m *message) ToRaw() []byte {
	m.sumLength()
	raw := make([]byte, 4, messageHeaderSize+m.length)
	bin.PutUint16(raw, uint16(m.messageType))
	bin.PutUint16(raw[2:], m.length)
	raw = append(raw, m.transactionID[:]...)
	for _, a := range m.attributes {
		attrBytes, err := a.toRaw()
		if err != nil {
			fmt.Printf("%e", err)
			continue
		}
		raw = append(raw, attrBytes...)
	}
	return raw
}
func (m *message) AddIntegrityAttrAnd2Raw() []byte {
	raw := m.ToRaw()
	// todo AddIntegrity
	return raw
}
func (m *message) sumLength() {
	m.length = 0
	for _, a := range m.attributes {
		m.length += attrTypeSize + attrLengthSize + a.length
	}
	//m.length = m.length + integritySize
}
func (m *message) ToString() string {
	t := fmt.Sprintf("%x%x",bin.Uint64(m.transactionID[:8]),bin.Uint64(m.transactionID[8:]))
	str := fmt.Sprintf("message: {messageType:%s, length:%d, transactionId: %s, attributes: [",
		MessageTypeName(m.messageType),m.length,t)
	for _, a := range m.attributes {
		str += fmt.Sprintf(AttrTypeName(a.attrType) + ": %v, ",m.GetAttribute(a.attrType))
	}
	str += "]}"
	return str
}

const (
	BindReq              MessageType = 0x0001 //捆绑请求
	BindResp             MessageType = 0x0101 //捆绑响应
	BindErrorResp        MessageType = 0x0111 //捆绑错误响应
	ShareSecretReq       MessageType = 0x0002 //共享私密请求
	ShareSecretResp      MessageType = 0x0102 //共享私密响应
	ShareSecretErrorResp MessageType = 0x0112 //共享私密错误响应
)
func MessageTypeName(messageType MessageType) string {
	switch messageType {
		case BindReq: return "BindReq"
		case BindResp: return "BindResp"
		case BindErrorResp: return "BindErrorResp"
		case ShareSecretReq: return "ShareSecretReq"
		case ShareSecretResp: return "ShareSecretResp"
		case ShareSecretErrorResp: return "ShareSecretErrorResp"
	}
	return ""
}

type MessageType uint16

func NewBindRequest(transactionID []byte, responseAddress string, changeIp, changePort bool) (InMessage, error) {
	var traId [transactionIDSize]byte
	if transactionID == nil || len(transactionID) != transactionIDSize {
		traId = NewTransactionID()
	} else {
		copy(traId[:], transactionID)
	}
	attributes := make([]Attribute, 0, 2)
	if changeIp {
		addressAttr, err := newAttrResponseAddress(responseAddress)
		if err == nil {
			attributes = append(attributes, addressAttr)
		}
	}
	if changeIp || changePort {
		changeReqAttr, err := newAttrChangeRequest(changeIp, changePort)
		if err != nil {
			fmt.Errorf("%e", err)
		} else {
			attributes = append(attributes, changeReqAttr)
		}
	}
	message := message{BindReq, uint16(0), traId, attributes}
	message.sumLength()
	return &message, nil
}
func NewBindResponse(transactionID []byte, mappedAddress, sourceAddress, changedAddress string) (InMessage, error) {
	var traId [transactionIDSize]byte
	if transactionID == nil || len(transactionID) != transactionIDSize {
		traId = NewTransactionID()
	} else {
		copy(traId[:], transactionID)
	}
	attributes := make([]Attribute, 0, 4)

	mappedAddressAttr, err := newAttrMappedAddress(mappedAddress)
	if err != nil {
		fmt.Errorf("%e", err)
		return nil, err
	} else {
		attributes = append(attributes, mappedAddressAttr)
	}

	sourceAddressAttr, err := newAttrSourceAddress(sourceAddress)
	if err != nil {
		fmt.Errorf("%e", err)
		return nil, err
	} else {
		attributes = append(attributes, sourceAddressAttr)
	}

	changedAddressAttr, err := newAttrChangedAddress(changedAddress)
	if err != nil {
		fmt.Errorf("%e", err)
		return nil, err
	} else {
		attributes = append(attributes, changedAddressAttr)
	}

	message := message{BindResp, uint16(0), traId, attributes}
	message.sumLength()
	return &message, nil
}
func NewBindErrorResponse() (InMessage, error) {
	message := message{}
	return &message, nil
}
func NewShareSecretRequest() (InMessage, error) {
	message := message{}
	return &message, nil
}
func NewShareSecretResponse() (InMessage, error) {
	message := message{}
	return &message, nil
}
func NewShareSecretErrorResponse() (InMessage, error) {
	message := message{}
	return &message, nil
}

// NewTransactionID returns new random transaction ID using math/rand
// as source.

func NewTransactionID() (b [transactionIDSize]byte) {
	rand.Read(b[:])
	return b
}

type (
	InMessage interface {
		ToRaw() []byte
		AddIntegrityAttrAnd2Raw() []byte
		ToString() string
	}
	OutMessage interface {
		TransactionId() [transactionIDSize]byte
		Length() uint16
		MessageType() MessageType
		GetAttribute(attrType AttrType) interface{}
		ToString() string
	}
)
