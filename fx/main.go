package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// CreateServer creates a new GoFiber server instance
func CreateServer() (*fiber.App, error) {
	fiberConfig := fiber.Config{
		ReadTimeout:  time.Second * 20,
		WriteTimeout: time.Second * 20,
	}
	app := fiber.New(fiberConfig)
	// do setup and other stuff
	return app, nil
}

// Register registers a handler with the GoFiber server
func Register(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("teste")
	})
}
func main() {
	app := fx.New(
		fx.Provide(
			CreateServer,
		),
		fx.Invoke(Register),
		fx.Invoke(func(lifecycle fx.Lifecycle, app *fiber.App) {
			lifecycle.Append(fx.Hook{
				OnStart: func(context.Context) error {
					go app.Listen(":8080")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					fmt.Println("shuting down ...")
					return app.Shutdown()
				},
			})
		}),
	)

	app.Run()

	fmt.Println("teste")
}
