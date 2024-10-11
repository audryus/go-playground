package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/sqids/sqids-go"
)

func main() {

	hex := getHex()
	fmt.Printf("Hex  %s\n", hex)
	//var hexes []string
	//hexes = append(hexes, hex)

	/* for i := 0; i < 1000; i++ {
		h := getHex()
		if contains(hexes, h) {
			panic("Duplicate hex: " + h)
		}
		hexes = append(hexes, h)
	} */

	s, _ := sqids.New()
	id, _ := s.Encode([]uint64{1, 2, 3}) // "86Rf07"
	//numbers := s.Decode(id)              // [1, 2, 3]

	fmt.Println(id)

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
	fmt.Println(string(b[:]))
	var buf [8]byte
	hex.Encode(buf[:], b[:])
	return string(buf[:])
}
