package main

import (
	"fmt"
	"math/rand/v2"
)

type Lero struct {
	a string
	b string
}

func (l *Lero) String() string {
	return "toString"
}

func main() {
	fmt.Println(&Lero{
		a: "A",
		b: "B",
	})

	fmt.Println(1 + (rand.Int64() % ((5 - 1) + 1)))
}
