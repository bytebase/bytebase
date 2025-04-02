package main

import (
	"context"
	"fmt"
)

func run() error {
	ctx := context.Background()
	if _, err := login(ctx); err != nil {
		return err
	}
	return nil
}

func main() {
	platform := getJobPlatform()
	fmt.Printf("Hello, World - %s!\n", platform.String())
	if err := run(); err != nil {
		panic(err)
	}
}
