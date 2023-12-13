package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	total := 0.0
	dodged := 0.0

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		d10_1 := r.Intn(9)
		d10_2 := r.Intn(90)
		total++
		d10 := d10_1 + d10_2
		if d10 == 0 || d10 <= 90 {
			dodged++
		}
	}

	fmt.Printf("%f / %f = %f", dodged, total, ((dodged / total) * 100))
}
