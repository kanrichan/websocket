package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
)

var (
	TextMessage   byte = 0x01
	BinaryMessage byte = 0x02
	CloseMessage  byte = 0x08
	PingMessage   byte = 0x09
	PongMessage   byte = 0x0a
)

func main() {
	conn, err := Dial("127.0.0.1:8000")
	if err != nil {
		panic(err)
	}
	conn.WriteFrame(TextMessage, []byte(strings.Repeat("01231456789", 1)))
	conn.Close()
}

type Conn struct {
	Conn net.Conn
}

func (conn Conn) WriteFrame(opcode byte, data []byte) error {
	bw := bufio.NewWriter(conn.Conn)
	bw.WriteByte(0x80 | opcode) // 0X80 -> FIN 1 RSV1 0 RSV2 0 RSV3 0
	if opcode == PingMessage {
		bw.WriteByte(0x00)
		return bw.Flush()
	}
	var mask byte = 0x80
	var length = len(data)
	switch {
	case length < 125:
		bw.WriteByte(mask | byte(length))
	case length < 65536:
		bw.WriteByte(mask | 0b01111110)
		var extended = make([]byte, 2)
		binary.BigEndian.PutUint16(extended, uint16(length))
		bw.Write(extended)
	default:
		bw.WriteByte(mask | 0b01111111)
		var extended = make([]byte, 8)
		binary.BigEndian.PutUint64(extended, uint64(length))
		bw.Write(extended)
	}
	// TODO
	var maskkey, err = genMaskKey()
	if err != nil {
		return err
	}
	bw.Write(maskkey)
	payload := make([]byte, len(data))
	for i := range data {
		payload[i] = data[i] ^ maskkey[i%4]
	}
	bw.Write(payload)
	return bw.Flush()
}

func Dial(address string) (Conn, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return Conn{}, err
	}
	nonce, err := gennonce()
	bw := bufio.NewWriter(conn)
	bw.WriteString("GET / HTTP/1.1\r\n")
	bw.WriteString("Host: 127.0.0.1:8000\r\n")
	bw.WriteString("Connection: Upgrade\r\n")
	bw.WriteString("Upgrade: websocket\r\n")
	bw.WriteString("Origin: http://127.0.0.1:8000\r\n")
	bw.WriteString("Sec-WebSocket-Version: 13\r\n")
	bw.WriteString("Sec-WebSocket-Key: " + nonce + "\r\n")
	bw.WriteString("\r\n")
	err = bw.Flush()
	if err != nil {
		return Conn{}, err
	}
	var x = make([]byte, 256)
	conn.Read(x)
	if err != nil {
		return Conn{}, err
	}
	fmt.Println(string(x))
	return Conn{conn}, nil
}

func (conn *Conn) Close() error {
	conn.WriteFrame(CloseMessage, []byte{0x03, 0xe8})
	return conn.Conn.Close()
}

func gennonce() (string, error) {
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
