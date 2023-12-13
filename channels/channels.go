package main

import (
	"fmt"
	"time"
)

type Future struct {
}

func lero(c chan string) {
	time.Sleep(10 * time.Second)
	c <- "channel"
}

func lero2(c chan string) {
	x := <-c
	fmt.Println(x)
}

func main() {
	c := make(chan string)
	go lero(c)
	go lero2(c)
	fmt.Println("done ...")
}
