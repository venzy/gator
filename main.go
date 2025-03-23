package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/venzy/gator/internal/config"
	"github.com/venzy/gator/internal/database"

	_ "github.com/lib/pq"
)

type state struct {
	db *database.Queries
	cfg *config.Config
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
	cliCommands.register("agg", handlerAgg)
	cliCommands.register("addfeed", handlerAddFeed)
	cliCommands.register("feeds", handlerFeeds)
	cliCommands.register("follow", handlerFollow)
	cliCommands.register("following", handlerFollowing)

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