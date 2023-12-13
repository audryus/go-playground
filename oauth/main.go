package main

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

var (
	authConfig       *oauth2.Config
	oauthStateString = "pseudo-random"
)

func main() {
	fmt.Printf("asd")

	authConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/callback",
		ClientID:     "8niRRugvWdL4yqa8vR0irC2FkfDjJHQZTdbXyvEu",
		ClientSecret: "acZBPsF8qMdAFu3CTKAMrUChI2A8xj8s0XbyFUSh8PgqThCgDFv1cpojpNEksKP3g2LHYsb5JyBEqsmtxK3q5F0igQM8JLetsog2p7Lfpftus4k2IjUY2HtznO1HEuzT",
		Endpoint: oauth2.Endpoint{
			AuthURL:   "http://localhost:9000/application/o/authorize/",
			TokenURL:  "http://localhost:9000/application/o/token/",
			AuthStyle: 0,
		},
	}

	app := fiber.New()

	app.Get("/login", func(c *fiber.Ctx) error {
		url := authConfig.AuthCodeURL(oauthStateString)
		return c.Redirect(url)
	})

	app.Get("/callback", func(c *fiber.Ctx) error {
		state := c.Query("state")
		code := c.Query("code")
		cont, err := getUserInfo(state, code)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.SendString(string(cont))
	})

	app.Listen(":8080")

}

func getUserInfo(state string, code string) ([]byte, error) {
	if state != oauthStateString {
		return nil, fmt.Errorf("invalid oauth state")
	}
	token, err := authConfig.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	fmt.Println("token", token)
	response, err := http.Get("http://localhost:9000/application/o/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}
	return contents, nil
}
