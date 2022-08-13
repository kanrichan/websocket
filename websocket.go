package main

import (
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

func genMaskKey() ([4]byte, error) {
	var p [4]byte
	_, err := io.ReadFull(rand.Reader, p[:])
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
	FIN        bool
	RSV        [3]bool
	OpCode     byte
	Length     int
	Mask       bool
	MaskingKey [4]byte
	Payload    []byte
	NextFrame  *Frame
}

func defaultFrame(opcode byte, payload []byte) (*Frame, error) {
	key, err := genMaskKey()
	if err == nil {
		return nil, err
	}
	return &Frame{
		FIN:        true,
		RSV:        [3]bool{false, false, false},
		OpCode:     opcode,
		Mask:       true,
		Length:     len(payload),
		MaskingKey: key,
		Payload:    payload,
	}, nil
}

func writeFrame(wr io.Writer, frame *Frame) error {
	var length = 2
	var i = 0
	switch {
	case frame.Length < 125:
		length += 0
	case frame.Length < 65536:
		length += 2
	default:
		length += 8
	}
	if frame.Mask {
		length += 4
	}
	if frame.OpCode == PingMessage || frame.OpCode == PongMessage {
		length += 1
	} else {
		length += frame.Length
	}
	var buf = make([]byte, length, length)
	// 0X80 -> FIN 1 RSV1 0 RSV2 0 RSV3 0
	frame.OpCode &= 0x0f
	if frame.FIN {
		buf[i] = 0x80 | frame.OpCode
	} else {
		buf[i] = frame.OpCode
	}
	if frame.OpCode == PingMessage || frame.OpCode == PongMessage {
		return nil
	}
	i++ // 1
	if frame.Mask {
		buf[i] = 0x80
	}
	switch {
	case frame.Length < 125:
		buf[i] |= byte(frame.Length)
		i++
	case frame.Length < 65536:
		buf[i] |= 0b01111110
		binary.BigEndian.PutUint16(buf[2:4], uint16(frame.Length))
		i += 3
	default:
		buf[i] |= 0b01111111
		binary.BigEndian.PutUint16(buf[2:10], uint16(frame.Length))
		i += 9
	}
	if frame.Mask {
		copy(buf[i:i+4], frame.MaskingKey[:])
		i += 4
	}
	for j := range frame.Payload {
		buf[i+j] = frame.Payload[j] ^ frame.MaskingKey[j%4]
	}
	n, err := wr.Write(buf)
	if n != length {
		return ErrNotSupportFinRsv
	}
	if !frame.FIN && frame.NextFrame != nil {
		err := writeFrame(wr, frame.NextFrame)
		if err != nil {
			return err
		}
	}
	return err
}

type Conn struct {
	net.Conn
}

func readFrame(rd io.Reader) (*Frame, error) {
	var frame = &Frame{}
	var b0 = make([]byte, 2)
	n, err := rd.Read(b0)
	if n != 2 {
		return frame, ErrShortBuffer
	}
	if err != nil {
		return frame, err
	}
	if b0[0]&0x80 == 0x80 {
		frame.FIN = true
	}
	if b0[0]&0x40 == 0x40 {
		frame.RSV[0] = true
	}
	if b0[0]&0x20 == 0x20 {
		frame.RSV[1] = true
	}
	if b0[0]&0x10 == 0x10 {
		frame.RSV[2] = true
	}
	frame.OpCode = b0[0] & 0x0f
	if b0[1]&0x80 == 0x80 {
		frame.Mask = true
	}
	if frame.OpCode == PingMessage || frame.OpCode == PongMessage {
		var t = make([]byte, 1)
		n, err := rd.Read(t)
		if n != 2 {
			return frame, ErrShortBuffer
		}
		if err != nil {
			return frame, nil
		}
		return frame, nil
	}
	switch {
	case b0[1]&0x7f <= 0b01111101:
		frame.Length = int(b0[1] & 0x7f)
	case b0[1]&0x7f == 0b01111110:
		var t = make([]byte, 2)
		n, err := rd.Read(t)
		if n != 2 {
			return frame, ErrShortBuffer
		}
		if err != nil {
			return frame, nil
		}
		frame.Length = int(binary.BigEndian.Uint16(t))
	case b0[1]&0x7f == 0b01111111:
		var t = make([]byte, 8)
		n, err := rd.Read(t)
		if n != 2 {
			return frame, ErrShortBuffer
		}
		if err != nil {
			return frame, err
		}
		frame.Length = int(binary.BigEndian.Uint64(t))
	}
	var length = frame.Length
	if frame.Mask {
		length += 4
	}
	var b1 = make([]byte, length)
	if frame.Mask {
		copy(frame.MaskingKey[:], b1[0:4])
		frame.Payload = b1[4:]
	} else {
		frame.Payload = b1
	}
	if !frame.FIN {
		frame.NextFrame, err = readFrame(rd)
		if err != nil {
			return frame, err
		}
	}
	return frame, nil
}
