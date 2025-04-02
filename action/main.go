package main

import (
	"fmt"
)

func main() {
	platform := getJobPlatform()
	fmt.Printf("Hello, World - %s!", platform.String())
}
