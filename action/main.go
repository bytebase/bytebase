package main

import (
	"fmt"
)

func run() error {
	_, err := NewClient("https://demo.bytebase.com", "ci@service.bytebase.com", "bbs_iqysPHMqhNpG4rQ5SFEJ")
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
