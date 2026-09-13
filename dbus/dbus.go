package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"
)

func main() {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// -------------------------------------------------
	// 1. Listen for ActionInvoked signals
	// -------------------------------------------------
	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/freedesktop/Notifications"),
		dbus.WithMatchInterface("org.freedesktop.Notifications"),
		dbus.WithMatchMember("ActionInvoked"),
	); err != nil {
		log.Fatal(err)
	}

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)

	go func() {
		for sig := range c {
			if sig.Name != "org.freedesktop.Notifications.ActionInvoked" {
				continue
			}

			id := sig.Body[0].(uint32)
			action := sig.Body[1].(string)

			fmt.Printf("Notification %d → action clicked: %q\n", id, action)

			switch action {
			case "accept":
				fmt.Println("→ User accepted!")
				// do your accept logic here
			case "decline":
				fmt.Println("→ User declined!")
				// do your decline logic here
			case "later":
				fmt.Println("→ Remind later")
			default:
				fmt.Println("→ Unknown action")
			}
		}
	}()

	// -------------------------------------------------
	// 2. Send the notification with multiple buttons
	// -------------------------------------------------
	obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")

	// actions = ["id1", "Label1", "id2", "Label2", ...]
	actions := []string{
		"accept", "Accept",
		"decline", "Decline",
		"later", "Remind later",
	}

	call := obj.Call("org.freedesktop.Notifications.Notify", 0,
		"MyGoApp",                             // app_name
		uint32(0),                             // replaces_id
		"dialog-question",                     // icon
		"Meeting invitation",                  // summary
		"John invited you to a call at 15:00", // body
		actions,                               // ← the buttons
		map[string]dbus.Variant{ // hints
			"urgency": dbus.MakeVariant(byte(1)),
		},
		int32(0), // 0 = never expire (so user can click the buttons)
	)

	if call.Err != nil {
		log.Fatal(call.Err)
	}

	var nid uint32
	if err := call.Store(&nid); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Notification sent, ID =", nid)
	fmt.Println("Waiting for button clicks… (Ctrl+C to quit)")

	// Keep the program alive so we can receive the signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
