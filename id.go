package main

import (
	"crypto/rand"
	"encoding/base32"
)

func NewID() string {
	var buffer [16]byte
	_, err := rand.Read(buffer[:])
	if err != nil {
		panic(err)
	}
	return base32.StdEncoding.EncodeToString(buffer[:])[:8]
}
