package main

import (
	"fmt"
	"time"
)

type Future struct {
}

func lero(c chan string) {
	//time.Sleep(10 * time.Second)
	c <- "channel"
}

func lero2(c chan string) {
	x := <-c
	fmt.Println("lero2", x)
}

func lero3(c chan string) {
	x := <-c
	fmt.Println("lero3", x)
}

func main() {
	c := make(chan string)
	go lero2(c)
	go lero3(c)
	go lero(c)
	fmt.Println("done ...")
	time.Sleep(1 * time.Second)
}
