package main

import (
	"fmt"
	"os"
)

func run() error {
	_, err := NewClient(os.Getenv("BYTEBASE_URL"), os.Getenv("BYTEBASE_SERVICE_ACCOUNT"), os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET"))
	if err != nil {
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
