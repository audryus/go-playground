package main

import "fmt"

type MyErr struct {
	cod string
}

var ErrNotFound MyErr = MyErr{
	cod: "not found",
}

// Error returns the error message of the MyErr type.
//
// No parameters.
// Returns a string.
func (e MyErr) Error() string {
	return e.cod
}

func main() {
	e := lero()

	fmt.Printf("Erro: %+v\n", e)
}

func lero() error {
	return ErrNotFound
}
