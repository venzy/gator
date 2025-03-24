package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/venzy/gator/internal/database"
	"time"
)

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("login requires a single argument, the username")
	}

	username := cmd.args[0]
	_, err := s.db.GetUserByName(context.Background(), username)
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
	user, err := s.db.GetUserByName(context.Background(), username)
	if err == nil && user.Name == username {
		return fmt.Errorf("User '%s' already exists!", username)
	}

	now := time.Now()
	user, err = s.db.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name:      username,
		})
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

// 'middleware' as boot.dev likes to call it
func withLoggedInUser(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("User '%s' not in database!", s.cfg.CurrentUserName)
		}

		return handler(s, cmd, user)
	}
}