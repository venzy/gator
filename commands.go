package main

import (
	"fmt"
	"log"
)

type command struct {
	name string
	args []string
}

type commands struct {
	registry map[string]func(*state, command) error
}

func NewCommands() *commands {
	return &commands{
		registry: make(map[string]func(*state, command) error),
	}
}

func (c *commands) register(name string, f func(*state, command) error) {
	if _, exists := c.registry[name]; exists {
		log.Fatalf("Attempt to double-register command '%s'", name)
	}

	c.registry[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	if cmdFunc, ok := c.registry[cmd.name]; ok {
		return cmdFunc(s, cmd)
	}

	return fmt.Errorf("Command does not exist: '%s'", cmd.name)
}
