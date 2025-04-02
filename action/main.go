package main

import (
	"fmt"
	"os"
)

func run() error {
	baseURL := os.Getenv("BYTEBASE_URL")
	email := os.Getenv("BYTEBASE_SERVICE_ACCOUNT")
	password := os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET")
	_, err := NewClient(baseURL, email, password)
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
