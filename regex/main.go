package main

import (
	"fmt"
	"regexp"
)

func main() {
	text := `Here are some links: https://example.com, http://test.org, and https://www.sample.net/page.`

	// Define the regex pattern for URLs
	urlPattern := `(https?://|www\.)[^\s]+`

	// Compile the regex
	re := regexp.MustCompile(urlPattern)

	// Find all matches
	matches := re.FindAllString(text, -1)

	// Remove trailing punctuation from matches
	for i, match := range matches {
		matches[i] = regexp.MustCompile(`[^\w/]+$`).ReplaceAllString(match, "")
	}

	// Print all matches
	for _, match := range matches {
		fmt.Println(match)
	}
}
