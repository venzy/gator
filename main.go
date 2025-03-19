package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/venzy/gator/internal/config"
	"github.com/venzy/gator/internal/database"

	_ "github.com/lib/pq"
)

type state struct {
	db *database.Queries
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
	appState := state{db: nil, cfg: &cfg}
	cliCommands := NewCommands()
	cliCommands.register("login", handlerLogin)
	cliCommands.register("register", handlerRegister)
	cliCommands.register("reset", handlerReset)
	cliCommands.register("users", handlerUsers)

	// Get command line args
	if len(os.Args) < 2 {
		log.Fatalf("%s: CLI command required as first argument", os.Args[0])
	}

	cmd := command{os.Args[1], os.Args[2:]}

	// Open DB connection
	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		log.Fatalf("Cannot connect to database: %s", err)
	}

	dbQueries := database.New(db)
	appState.db = dbQueries

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

	username := cmd.args[0]
	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return fmt.Errorf("Could not login user '%s'", username)
	}

	if err := s.cfg.SetUser(username); err != nil {
		return err
	}

	fmt.Printf("Logged in as %s\n", username)

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("register requires a single argument, the username")
	}

	username := cmd.args[0]
	user, err := s.db.GetUser(context.Background(), username)
	if err == nil && user.Name == username {
		return fmt.Errorf("User '%s' already exists!", username)
	}

	now := time.Now()
	user, err = s.db.CreateUser(context.Background(), database.CreateUserParams{ID: uuid.New(), CreatedAt: now, UpdatedAt: now, Name: username})
	if err != nil {
		return err
	}

	err = s.cfg.SetUser(user.Name)
	if err != nil {
		return err
	}

	fmt.Printf("Registered new user: %v\n", user)

	return nil
}

func handlerReset(s *state, _ command) error {
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return err
	}
	fmt.Println("Successfully reset users table")
	return nil
}

func handlerUsers(s *state, _ command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		fmt.Printf("* %s", user.Name)
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf(" (current)")
		}
		fmt.Printf("\n")
	}

	return nil
}