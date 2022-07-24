package main

import (
	"crypto/tls"
	"testing"
)

func TestClient(t *testing.T) {
	ws, err := NewClient("ws://127.0.0.1:8080/ws")
	if err != nil {
		t.Fatal(err)
	}
	ws.Config = &tls.Config{InsecureSkipVerify: true}
	err = ws.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	t.Log(ws.Response)
	ws.WriteFrame(TextMessage, []byte("hello world!"))
	opcode, content, err := ws.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(opcode, string(content))
}
