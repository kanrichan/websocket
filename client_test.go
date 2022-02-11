package main

import (
	"crypto/tls"
	"fmt"
	"testing"
)

func TestClient(t *testing.T) {
	ws, err := NewClient("wss://127.0.0.1:8000/ws")
	if err != nil {
		t.Fatal(err)
	}
	ws.Config = &tls.Config{InsecureSkipVerify: true}
	err = ws.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	fmt.Println(ws.Response)
	ws.WriteFrame(TextMessage, []byte("hello world!"))
}
