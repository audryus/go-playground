package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
)

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	go func() {
		sig := <-c
		fmt.Printf("Got %s signal. Aborting...\n", sig)
		afterEffect()
		os.Exit(1)

	}()

	defer afterEffect()
	fmt.Println("teste")
	app := fiber.New()

	app.Hooks().OnShutdown(func() error {
		fmt.Print("Name: ")

		return nil
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))

}

func afterEffect() {
	fmt.Println("ending")
}
