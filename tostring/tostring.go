package main

import "fmt"

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
}
