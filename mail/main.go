package main

import (
	"fmt"

	"github.com/resend/resend-go/v2"
)

func main() {
	apiKey := "YOUR_API_KEY"

	client := resend.NewClient(apiKey)

	params := &resend.SendEmailRequest{
		From:    "Name <noreply@example.com>",
		To:      []string{"to@example.com"},
		Html:    "<strong>hello world</strong>",
		Subject: "Hello from Golang",
	}

	_, err := client.Emails.Send(params)

	// Now send E-Mail
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

}
