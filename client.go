package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	urlpkg "net/url"
	"strings"
)

type Client struct {
	URL *url.URL

	Header http.Header
	Dialer *net.Dialer
	Config *tls.Config

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
	if cli.Dialer == nil {
		cli.Dialer = &net.Dialer{}
	}
	switch cli.URL.Scheme {
	case "ws":
		cli.Conn, err = cli.Dialer.Dial("tcp", cli.URL.Host)
	case "wss":
		cli.Conn, err = tls.DialWithDialer(cli.Dialer, "tcp", cli.URL.Host, cli.Config)
	}
	if err != nil {
		return err
	}

	bw := bufio.NewWriter(cli.Conn)
	bw.WriteString("GET " + cli.URL.Path + " HTTP/1.1\r\n")
	cli.Header.Write(bw)

	bw.WriteString("\r\n")
	err = bw.Flush()
	if err != nil {
		cli.Close()
		return err
	}
	br := bufio.NewReader(cli.Conn)
	cli.Response, err = http.ReadResponse(br, &http.Request{Method: "GET"})
	if err != nil {
		cli.Close()
		return err
	}
	if cli.Response.StatusCode != 101 {
		cli.Close()
		return errors.New("bad status")
	}
	if strings.ToLower(cli.Response.Header.Get("Connection")) != "upgrade" ||
		strings.ToLower(cli.Response.Header.Get("Upgrade")) != "websocket" {
		cli.Close()
		return errors.New("bad upgrade")
	}
	nonceAccept, err := genNonceAccept(cli.Header.Get("Sec-WebSocket-Key"))
	if err != nil {
		cli.Close()
		return err
	}
	if cli.Response.Header.Get("Sec-Websocket-Accept") != string(nonceAccept) {
		cli.Close()
		return errors.New("mismatch challenge/response")
	}
	return nil
}

func (cli *Client) WriteFrame(opcode byte, content []byte) error {
	return writeFrame(cli.Conn, opcode, content)
}

func (cli *Client) ReadFrame() (byte, []byte, error) {
	return readFrame(cli.Conn)
}

func (cli *Client) Close() error {
	if cli.Conn == nil {
		return nil
	}
	cli.WriteFrame(CloseMessage, []byte{0x03, 0xe8})
	return cli.Conn.Close()
}
