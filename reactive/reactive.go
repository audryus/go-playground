package main

import (
	"fmt"
	"time"
)

type Reactive struct {
	f fn
}

type Result struct {
	success bool
	err     error
	result  any
}

func lero(f fn, c complete) {
	result, err := f()
	if err != nil {
		go c(Result{
			success: false,
			err:     err,
		})
	}

	go c(Result{
		success: true,
		result:  result,
	})
}

func (r Reactive) onComplete(complete complete) {
	go lero(r.f, complete)

}

type fn func() (any, error)

type complete func(Result)

func doSomething(s string) fn {
	return func() (any, error) {
		time.Sleep(5 * time.Second)
		return s, nil
	}
}
func readSomething(r Result) {
	if r.success {
		fmt.Printf("Sucesso %+v\n", r.result)
	} else {
		fmt.Println("Erro")
	}
}
func new(f fn) *Reactive {
	return &Reactive{
		f: f,
	}
}

func main() {
	fmt.Println("Before complete")
	new(doSomething("teste")).onComplete(readSomething)
	fmt.Println("After complete")
}
