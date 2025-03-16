package main

import (
	"fmt"

	"github.com/venzy/gator/internal/config"
)

func main() {
	// Read config
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("Error reading config: %s\n", err)
		return
	}

	// Set new user name
	config.SetUser(cfg, "davidv")

	// Read config again
	cfg, err = config.Read()
	if err != nil {
		fmt.Printf("Error reading config: %s\n", err)
		return
	}

	// Print config to terminal
	fmt.Printf("Current config:\n%v\n", cfg)
}