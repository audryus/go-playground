package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v2"
)

type Res struct {
	Version int
	Header  string
}

var results []Res
var wg sync.WaitGroup

func main() {
	go server()
	client()
	wg.Wait()

}

func client() {
	wg.Add(3)
	client := &http.Client{}

	makeReq := func(url string) {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("X-Custom-Header", url)
		client.Do(req)
	}

	makeReq("http://127.0.0.1:3000?version=11")
	makeReq("http://127.0.0.1:3000?version=2")
	makeReq("http://127.0.0.1:3000?version=33")
}

func server() {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		fmt.Printf("Received Header X-Custom-Header: %s\n", c.Get("X-Custom-Header"))
		var version int
		if c.Query("version") != "" {
			version = c.QueryInt("version")
		}
		results = append(results, Res{
			Version: version,
			Header:  c.Get("X-Custom-Header"),
		})
		fmt.Printf("Results: %+v\n", results)
		wg.Done()
		return c.SendStatus(fiber.StatusAccepted)
	})
	app.Listen(":3000")
}
