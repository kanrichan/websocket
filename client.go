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
	_, err := Dial("127.0.0.1:8000")
	if err != nil {
		panic(err)
	}
}

type Conn struct {
	conn net.Conn
	recv chan []byte
	send chan []byte
}

func (conn *Conn) WriteMessage(data []byte) error {
	conn.send <- data
	return nil
}

func (conn *Conn) ReadMessage() ([]byte, error) {
	res, ok := <-conn.recv
	if !ok {
		return nil, io.EOF
	}
	return res, nil
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
	br := bufio.NewReader(conn)
	x, _, err := br.ReadLine()
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
