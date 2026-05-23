package app

import (
	"context"
	"fmt"
	"time"
)

func Activity(ctx context.Context, data string) (string, error) {
	fmt.Printf("Activity: Hello %s ! \n\n", data)
	time.Sleep(10 * time.Second)

	return fmt.Sprintf("Activity: Hello %s ! \n\n", data), nil
}
