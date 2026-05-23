package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func main() {
	secret := uint64(0xA5F3C9)
	realID := uint64(1141237606920028161)

	fmt.Println("secret: ", secret)

	publicID := EncodeID(realID, secret)
	fmt.Println("public:", publicID)

	backID, err := DecodeID(publicID, secret)
	if err != nil {
		panic(err)
	}
	fmt.Println("back:", backID)
}

func EncodeID(id uint64, secret uint64) string {
	randPart := uint64(rand.Uint32() & 0xFFFF) // 16 bits
	combined := (randPart << 48) | id          // assume id <= 48 bits

	obfuscated := combined ^ secret

	return base62Encode(obfuscated)
}

func DecodeID(encoded string, secret uint64) (uint64, error) {
	value, err := base62Decode(encoded)
	if err != nil {
		return 0, err
	}
	combined := value ^ secret
	id := combined & ((1 << 48) - 1)
	return id, nil
}

func base62Encode(num uint64) string {
	if num == 0 {
		return "0"
	}

	var result []byte
	for num > 0 {
		rem := num % 62
		result = append([]byte{base62Alphabet[rem]}, result...)
		num /= 62
	}
	return string(result)
}

func base62Decode(str string) (uint64, error) {
	var num uint64
	for i := 0; i < len(str); i++ {
		c := str[i]
		index := strings.IndexByte(base62Alphabet, c)
		if index < 0 {
			return 0, fmt.Errorf("invalid base62 character: %c", c)
		}
		num = num*62 + uint64(index)
	}
	return num, nil
}
