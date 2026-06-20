package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	enet "github.com/codecat/go-enet"
)

const (
	port = 7777
)

func main() {
	enet.Initialize()
	defer enet.Deinitialize()

	done := make(chan struct{})

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runServer(done)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runClient(done)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	<-sig

	fmt.Println("\nshutting down...")

	close(done)

	wg.Wait()
}

func runServer(done <-chan struct{}) {
	host, err := enet.NewHost(
		enet.NewListenAddress(port),
		32,
		1,
		0,
		0,
	)
	if err != nil {
		panic(err)
	}
	defer host.Destroy()

	fmt.Println("[server] listening on:", port)

	for {
		select {
		case <-done:
			fmt.Println("[server] we're done")
			return
		default:
		}

		ev := host.Service(10)

		switch ev.GetType() {
		case enet.EventConnect:
			fmt.Println("[server] peer connected")

		case enet.EventDisconnect:
			fmt.Println("[server] peer disconnected")

		case enet.EventReceive:
			packet := ev.GetPacket()

			msg := string(packet.GetData())

			fmt.Printf("[server] received: %q\n", msg)

			err := ev.GetPeer().SendString(
				"ACK: "+msg,
				0,
				enet.PacketFlagReliable,
			)

			if err != nil {
				fmt.Println(err)
			}

			packet.Destroy()
		}
	}
}

func runClient(done <-chan struct{}) {
	host, err := enet.NewHost(nil, 1, 1, 0, 0)
	if err != nil {
		panic(err)
	}
	defer host.Destroy()

	peer, err := host.Connect(
		enet.NewAddress("127.0.0.1", port),
		1,
		0,
	)
	if err != nil {
		panic(err)
	}

	connected := make(chan struct{})

	var eventWG sync.WaitGroup

	eventWG.Add(1)

	go func() {
		defer eventWG.Done()

		for {
			select {
			case <-done:
				fmt.Println("[client] event loop stopped")
				return
			default:
			}

			ev := host.Service(10)

			switch ev.GetType() {
			case enet.EventConnect:
				fmt.Println("[client] connected")

				select {
				case <-connected:
				default:
					close(connected)
				}

			case enet.EventDisconnect:
				fmt.Println("[client] disconnected")

			case enet.EventReceive:
				packet := ev.GetPacket()

				fmt.Printf(
					"[client] server says: %s\n",
					string(packet.GetData()),
				)

				packet.Destroy()
			}
		}
	}()

	<-connected

	fmt.Println("[client] type something:")

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		select {
		case <-done:
			eventWG.Wait()
			return
		default:
		}

		text := strings.TrimSpace(scanner.Text())

		if text == "" {
			continue
		}

		err := peer.SendString(
			text,
			0,
			enet.PacketFlagReliable,
		)

		if err != nil {
			fmt.Println(err)
		}
	}

	eventWG.Wait()
}
