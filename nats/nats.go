package main

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		panic(err)
	}
	messages := make(chan string)

	nc.QueueSubscribe("foo", "a", func(m *nats.Msg) {
		fmt.Printf("(1) Received a message: %s\n", string(m.Data))
	})
	nc.QueueSubscribe("foo", "a", func(m *nats.Msg) {
		fmt.Printf("(2) Received a message: %s\n", string(m.Data))
	})

	nc.Subscribe("bar", func(m *nats.Msg) {
		fmt.Println(string(m.Data))
		nc.Publish(m.Reply, []byte("Reply to you."))
	})

	nc.Publish("foo", []byte("I can help!"))
	nc.Publish("foo", []byte("I can help 2!"))
	nc.Publish("foo", []byte("I can help 3!"))

	ms, err := nc.Request("bar", []byte("Request with reply."), 10*time.Millisecond)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(ms.Data))

	msg := <-messages

	fmt.Println(msg)

	// Drain connection (Preferred for responders)
	// Close() not needed if this is called.
	nc.Drain()

	// Close connection
	nc.Close()

}
