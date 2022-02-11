package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type Server struct {
	Address  string
	Listener net.Listener
}

func NewServer(address string) (*Server, error) {
	return &Server{Address: address}, nil
}

func (srv *Server) Listen() error {
	var err error
	srv.Listener, err = net.Listen("tcp", srv.Address)
	if err != nil {
		return err
	}
	for {
		conn, err := srv.Listener.Accept()
		if err != nil {
			return err
		}
		br := bufio.NewReader(conn)
		req, err := http.ReadRequest(br)
		if err != nil {
			return err
		}
		fmt.Println(req.URL.Path)
		bw := bufio.NewWriter(conn)
		if req.Method != "GET" {
			bw.WriteString("HTTP/1.1 405 Method Not Allowed\r\n")
			bw.WriteString("\r\n")
			bw.Flush()
			return errors.New("bad method")
		}
		if strings.ToLower(req.Header.Get("Upgrade")) != "websocket" ||
			strings.ToLower(req.Header.Get("Connection")) != "upgrade" {
			bw.WriteString("HTTP/1.1 400 Bad Request\r\n")
			bw.WriteString("\r\n")
			bw.Flush()
			return errors.New("missing or bad upgrade")
		}
		if req.Header.Get("Sec-Websocket-Key") == "" {
			bw.WriteString("HTTP/1.1 400 Bad Request\r\n")
			bw.WriteString("\r\n")
			bw.Flush()
			return errors.New("mismatch challenge/response")
		}
		if req.Header.Get("Sec-WebSocket-Version") != "13" {
			bw.WriteString("HTTP/1.1 400 Bad Request\r\n")
			bw.WriteString("Sec-WebSocket-Version: 13\r\n")
			bw.WriteString("\r\n")
			bw.Flush()
			return errors.New("missing or bad WebSocket Version")
		}
		accept, err := genNonceAccept(req.Header.Get("Sec-Websocket-Key"))
		if err != nil {
			return err
		}
		bw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
		bw.WriteString("Upgrade: websocket\r\n")
		bw.WriteString("Connection: Upgrade\r\n")
		bw.WriteString("Sec-WebSocket-Accept: " + accept + "\r\n")
		bw.Flush()
		conn.Close()
	}
}
