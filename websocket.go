package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

var (
	GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

var (
	ErrShortBuffer      = errors.New("short buffer")
	ErrNotSupportFinRsv = errors.New("not support fin or rsv")
)

var (
	TextMessage   byte = 0x01
	BinaryMessage byte = 0x02
	CloseMessage  byte = 0x08
	PingMessage   byte = 0x09
	PongMessage   byte = 0x0a
)

func genNonce() (string, error) {
	p := make([]byte, 16)
	n, err := rand.Reader.Read(p)
	if n != 16 {
		return "", ErrShortBuffer
	}
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}

func genMaskKey() ([]byte, error) {
	p := make([]byte, 4)
	_, err := io.ReadFull(rand.Reader, p)
	return p, err
}

func genNonceAccept(nonce string) (string, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(nonce)); err != nil {
		return "", err
	}
	if _, err := h.Write([]byte(GUID)); err != nil {
		return "", err
	}
	expected := make([]byte, 28)
	base64.StdEncoding.Encode(expected, h.Sum(nil))
	return string(expected), nil
}

func readByte(rd io.Reader) (byte, error) {
	var b = make([]byte, 1)
	n, err := rd.Read(b)
	if n != 1 {
		return 0, ErrShortBuffer
	}
	return b[0], err
}

func readBytes(rd io.Reader, p []byte) error {
	n, err := rd.Read(p)
	if n != len(p) {
		return ErrShortBuffer
	}
	return err
}

// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-------+-+-------------+-------------------------------+
// |F|R|R|R| opcode|M| Payload len |    Extended payload length    |
// |I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
// |N|V|V|V|       |S|             |   (if payload len==126/127)   |
// | |1|2|3|       |K|             |                               |
// +-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
// |     Extended payload length continued, if payload len == 127  |
// + - - - - - - - - - - - - - - - +-------------------------------+
// |                               |Masking-key, if MASK set to 1  |
// +-------------------------------+-------------------------------+
// | Masking-key (continued)       |          Payload Data         |
// +-------------------------------- - - - - - - - - - - - - - - - +
// :                     Payload Data continued ...                :
// + - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
// |                     Payload Data continued ...                |
// +---------------------------------------------------------------+

type Frame struct {
	FINRSV     byte
	OpCode     byte
	Mask       byte
	Length     int
	MaskingKey []byte
	Payload    []byte
}

func defaultFrame(opcode byte, payload []byte) *Frame {
	var mask byte
	key, err := genMaskKey()
	if err == nil {
		mask = 0x80
	}
	return &Frame{
		FINRSV:     0x80,
		OpCode:     opcode,
		Mask:       mask,
		Length:     len(payload),
		MaskingKey: key,
		Payload:    payload,
	}
}

func writeFrame(conn net.Conn, frame *Frame) error {
	bw := bufio.NewWriter(conn)
	// 0X80 -> FIN 1 RSV1 0 RSV2 0 RSV3 0
	if frame.FINRSV != 0x80 {
		frame.FINRSV = 0x80
	}
	if frame.Length != len(frame.Payload) {
		frame.Length = len(frame.Payload)
	}
	bw.WriteByte(frame.FINRSV | frame.OpCode)
	if frame.OpCode == PingMessage || frame.OpCode == PongMessage {
		bw.WriteByte(0x00) // MASK 0xxx xxxx LENGTH x000 0000
		return bw.Flush()
	}
	switch {
	case frame.Length < 125:
		bw.WriteByte(frame.Mask | byte(frame.Length))
	case frame.Length < 65536:
		bw.WriteByte(frame.Mask | 0b01111110)
		var extended = make([]byte, 2)
		binary.BigEndian.PutUint16(extended, uint16(frame.Length))
		bw.Write(extended)
	default:
		bw.WriteByte(frame.Mask | 0b01111111)
		var extended = make([]byte, 8)
		binary.BigEndian.PutUint64(extended, uint64(frame.Length))
		bw.Write(extended)
	}
	if frame.Mask != 0x80 {
		bw.Write(frame.Payload)
		return bw.Flush()
	}
	var temp = make([]byte, frame.Length)
	for i := range frame.Payload {
		temp[i] = frame.Payload[i] ^ frame.MaskingKey[i%4]
	}
	bw.Write(frame.MaskingKey)
	bw.Write(temp)
	return bw.Flush()
}

type Conn struct {
	net.Conn
}

func (conn *Conn) ReadByte() (byte, error) {
	var p = make([]byte, 1)
	n, err := conn.Read(p)
	if n != 1 {
		return 0, ErrShortBuffer
	}
	if err != nil {
		return 0, err
	}
	return p[0], nil
}

func (conn *Conn) WriteByte(p byte) error {
	n, err := conn.Write(p)
	if n != 1 {
		return ErrShortBuffer
	}
	if err != nil {
		return err
	}
	return b[0], nil
}

func readFrame(conn net.Conn) (*Frame, error) {
	var frame = &Frame{}
	b0, err := readByte(conn)
	frame.FINRSV = b0 & 0xf0
	frame.OpCode = b0 & 0x0f
	// FIN 1 RSV1 0 RSV2 0 RSV3 0
	if frame.FINRSV != 0x80 {
		return nil, ErrNotSupportFinRsv
	}

	b1, err := readByte(conn)
	if err != nil {
		return nil, err
	}
	var mask = b1 & 0x80
	var length int
	switch {
	case b1&0x7f <= 0b01111101:
		length = int(b1 & 0x7f)
	case b1&0x7f == 0b01111110:
		var t = make([]byte, 2)
		if err = readBytes(conn, t); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint16(t))
	case b1&0x7f == 0b01111111:
		var t = make([]byte, 8)
		if err = readBytes(conn, t); err != nil {
			return 0, nil, err
		}
		length = int(binary.BigEndian.Uint64(t))
	}
	var maskKey = make([]byte, 4)
	if mask == 0x80 {
		if err = readBytes(conn, maskKey); err != nil {
			return 0, nil, err
		}
	}
	var payload = make([]byte, length)
	if err = readBytes(conn, payload); err != nil {
		return 0, nil, err
	}
	if mask != 0x80 {
		return opcode, payload, nil
	}
	var content = make([]byte, length)
	for i := range payload {
		content[i] = payload[i] ^ maskKey[i%4]
	}
	return opcode, content, nil
}
