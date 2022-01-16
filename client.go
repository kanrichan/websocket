package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
)

func main() {
	conn, err := Dial("127.0.0.1:8000")
	if err != nil {
		panic(err)
	}
	conn.WriteMessage([]byte("xxx"))
	select {}
}

type Conn struct {
	conn net.Conn
}

func (conn Conn) WriteMessage(data []byte) {
	if len(data) >= 125 {
		return
	}
	// TODO
	// maskkey := []byte{byte(0xAB), byte(0xCD)}
	length := len(data)
	// payload := make([]byte, len(data))
	// for i := range data {
	// 	payload[i] = data[i] ^ maskkey[i%4]
	// }
	bw := bufio.NewWriter(conn.conn)
	bw.WriteByte(0x81)
	bw.WriteByte(byte(0x00) | byte(length))
	bw.Write(data)
	bw.Flush()
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
	return conn.conn.Close()
}

func gennonce() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}
