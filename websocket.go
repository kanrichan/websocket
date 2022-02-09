package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"io"
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
