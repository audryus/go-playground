package main

import (
	"fmt"
	"math/big"
)

const (
	ABANDON int64 = iota
	ABILITY
	ABLE
	ABOUT
	ABOVE
	ABSENT
	ABSORB
	ABSTRACT
	ABSURD
	ZERO
	ZONE
	ZOO
)

func main() {
	a := big.NewInt(ZOO)
	b := big.NewInt(ZONE)

	// 1. Precisamos saber quantos bits tem o 'b' para fazer o shift no 'a'
	bitLen := b.BitLen()

	// 2. Criar cópia de 'a' e fazer shift para a esquerda (<<)
	resultado := new(big.Int).Lsh(a, uint(bitLen))

	// 3. Fazer OR bit a bit (ou soma, se não houver sobreposição) para juntar
	resultado.Or(resultado, b)
	teste1 := new(big.Int).And(resultado, b)
	if teste1.Cmp(b) == 0 {
		fmt.Println("[X] Zone está presente")
	} else {
		fmt.Println("[ ] Zone NÃO está presente")
	}
	fmt.Printf("A: %b\n", a)
	fmt.Printf("B: %b\n", b)
	fmt.Printf("Concatenado (Bin): %s\n", resultado.Text(2))
	fmt.Printf("Concatenado (Dec): %s\n", resultado.String())
}
