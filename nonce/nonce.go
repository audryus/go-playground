package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

func main() {
	fmt.Printf("Hex  %s\n", getHex())
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getHex() string {
	var b [4]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		fmt.Println("error")
	}
	var buf [8]byte
	hex.Encode(buf[:], b[:])
	return string(buf[:])
}
