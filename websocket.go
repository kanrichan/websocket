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
	TextMessage   byte = 0x01
	BinaryMessage byte = 0x02
	CloseMessage  byte = 0x08
	PingMessage   byte = 0x09
	PongMessage   byte = 0x0a
)

func genNonce() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
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
func WriteFrame(conn net.Conn, opcode byte, content []byte) error {
	bw := bufio.NewWriter(conn)
	bw.WriteByte(0x80 | opcode) // 0X80 -> FIN 1 RSV1 0 RSV2 0 RSV3 0
	if opcode == PingMessage || opcode == PongMessage {
		bw.WriteByte(0x00) // MASK 0xxx xxxx LENGTH x000 0000
		return bw.Flush()
	}
	var mask byte = 0x00 // MASK 0xxx xxxx
	var maskKey, err = genMaskKey()
	var payload = make([]byte, len(content))
	if err == nil {
		mask = 0x80 // MASK 1xxx xxxx
		for i := range content {
			payload[i] = content[i] ^ maskKey[i%4]
		}
	}
	switch {
	case len(content) < 125:
		bw.WriteByte(mask | byte(len(content)))
	case len(content) < 65536:
		bw.WriteByte(mask | 0b01111110)
		var extended = make([]byte, 2)
		binary.BigEndian.PutUint16(extended, uint16(len(content)))
		bw.Write(extended)
	default:
		bw.WriteByte(mask | 0b01111111)
		var extended = make([]byte, 8)
		binary.BigEndian.PutUint64(extended, uint64(len(content)))
		bw.Write(extended)
	}
	if mask == 0x00 {
		bw.Write(content)
	} else {
		bw.Write(maskKey)
		bw.Write(payload)
	}
	return bw.Flush()
}

func readByte(rd io.Reader) (byte, error) {
	var b = make([]byte, 1)
	n, err := rd.Read(b)
	if n != 1 {
		return 0, errors.New("???")
	}
	return b[0], err
}

func readBytes(rd io.Reader, p []byte) error {
	n, err := rd.Read(p)
	if n != len(p) {
		return errors.New("???")
	}
	return err
}

func ReadFrame(conn net.Conn) (byte, []byte, error) {
	b0, err := readByte(conn)
	// FIN 1 RSV1 0 RSV2 0 RSV3 0
	if b0&0xf0 != 0x80 {
		return 0, nil, errors.New("?????")
	}
	var opcode = b0 & 0x0f
	b1, err := readByte(conn)
	if err != nil {
		return 0, nil, err
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
