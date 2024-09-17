package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/django/v3"
)

type Ticket struct {
	Name    string
	Seating string
	Price   string
}

func getSeating(idx int) string {
	if idx%3 == 0 {
		return "Seated"
	}
	return "Stading"
}
func getPrice(idx int) string {
	return fmt.Sprintf("%.2f", (float32(idx+1) * 527.57))
}

func lero() []Ticket {
	var tickers []Ticket

	for i := 0; i < 5; i++ {
		tickers = append(tickers, Ticket{
			Name:    fmt.Sprintf("My Name %d", i+1),
			Seating: getSeating(i),
			Price:   getPrice(i),
		})
	}

	return tickers
}

func main() {
	engine := django.NewFileSystem(http.Dir("./views"), ".html")
	app := fiber.New(fiber.Config{Views: engine})

	app.Static("/css", "./css")
	app.Static("/fonts", "./fonts")
	app.Static("/images", "./images")
	app.Static("/js", "./js")

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Tickets": lero(),
		})
	})
	app.Get("/tick", func(c *fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{
			Name:  "test",
			Value: "SomeThing",
		})
		/* return c.JSON(fiber.Map{
			"tickets": lero(),
		}) */

		return c.Render("tick/index", fiber.Map{
			"tickets": lero(),
		})
	})

	log.Fatal(app.Listen("127.0.0.1:3000"))
}
