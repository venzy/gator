package main

import (
	"fmt"
	"log"
	"os"

	"github.com/venzy/gator/internal/config"
)

type state struct {
	cfg *config.Config
}

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

func main() {
	// Read config
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Error reading config: %s\n", err)
	}

	// Setup for CLI operation
	appState := state{&cfg}
	cliCommands := NewCommands()
	cliCommands.register("login", handlerLogin)

	// Get command line args
	if len(os.Args) < 2 {
		log.Fatalf("%s: CLI command required as first argument", os.Args[0])
	}

	cmd := command{os.Args[1], os.Args[2:]}

	// Run command
	if err := cliCommands.run(&appState, cmd); err != nil {
		log.Fatalf("ERROR: %s", err)
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

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("login requires a single argument, the username")
	}

	if err := s.cfg.SetUser(cmd.args[0]); err != nil {
		return err
	}

	fmt.Printf("Logged in as %s\n", cmd.args[0])

	return nil
}