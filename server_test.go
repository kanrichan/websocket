package main

import "testing"

func TestServer(t *testing.T) {
	server, err := NewServer("127.0.0.1:8000")
	if err != nil {
		t.Fatal(err)
	}
	err = server.Listen()
	if err != nil {
		t.Fatal(err)
	}
}
