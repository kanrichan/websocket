package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	urlpkg "net/url"
	"strings"
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

func main() {
	ws, err := NewClient("ws://127.0.0.1:8000/ws/")
	if err != nil {
		panic(err)
	}
	err = ws.Connect()
	if err != nil {
		panic(err)
	}
	defer ws.Close()
	fmt.Println(ws.Response)
	ws.WriteFrame(TextMessage, []byte("hello world!"))
}

type Conn struct {
	Conn net.Conn
}

type Client struct {
	URL    *url.URL
	Header http.Header

	Response *http.Response

	Conn net.Conn
}

func NewClient(url string) (*Client, error) {
	u, err := urlpkg.Parse(url)
	if err != nil {
		return nil, err
	}
	nonce, err := genNonce()
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	header.Set("Connection", "Upgrade")
	header.Set("Upgrade", "websocket")
	header.Set("Origin", "http://"+u.Host)
	header.Set("Host", u.Host)
	header.Set("Sec-WebSocket-Version", "13")
	header.Set("Sec-WebSocket-Key", nonce)
	return &Client{
		URL:    u,
		Header: header,
	}, nil
}

func (cli *Client) Connect() error {
	var err error
	switch cli.URL.Scheme {
	case "ws":
		cli.Conn, err = net.Dial("tcp", cli.URL.Host)
	case "wss":
		cli.Conn, err = tls.Dial("tcp", cli.URL.Host, nil)
	}
	if err != nil {
		return err
	}
	err = func() error {
		var err error
		bw := bufio.NewWriter(cli.Conn)
		bw.WriteString("GET " + cli.URL.Path + " HTTP/1.1\r\n")
		cli.Header.Write(bw)

		bw.WriteString("\r\n")
		err = bw.Flush()
		if err != nil {
			return err
		}
		br := bufio.NewReader(cli.Conn)
		cli.Response, err = http.ReadResponse(br, &http.Request{Method: "GET"})
		if err != nil {
			return err
		}
		if cli.Response.StatusCode != 101 {
			return errors.New("bad status")
		}
		if strings.ToLower(cli.Response.Header.Get("Connection")) != "upgrade" ||
			strings.ToLower(cli.Response.Header.Get("Upgrade")) != "websocket" {
			return errors.New("bad upgrade")
		}
		nonceAccept, err := genNonceAccept(cli.Header.Get("Sec-WebSocket-Key"))
		if err != nil {
			return err
		}
		if cli.Response.Header.Get("Sec-Websocket-Accept") != string(nonceAccept) {
			return errors.New("mismatch challenge/response")
		}
		return nil
	}()
	if err != nil {
		cli.Conn.Close()
		return err
	}
	return nil
}

func (cli *Client) WriteFrame(opcode byte, data []byte) error {
	bw := bufio.NewWriter(cli.Conn)
	bw.WriteByte(0x80 | opcode) // 0X80 -> FIN 1 RSV1 0 RSV2 0 RSV3 0
	if opcode == PingMessage || opcode == PongMessage {
		bw.WriteByte(0x00) // MASK 0xxx xxxx LENGTH x000 0000
		return bw.Flush()
	}
	var mask byte = 0x80 // MASK 1xxx xxxx
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

func (cli *Client) Close() error {
	cli.WriteFrame(CloseMessage, []byte{0x03, 0xe8})
	return cli.Conn.Close()
}

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
