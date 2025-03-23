package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/venzy/gator/internal/database"
	"time"
)

func handlerFollow(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("follow requires one argument, the feed URL")
	}

	feedURL := cmd.args[0]

	// Get logged in user ID
	user, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return fmt.Errorf("Current user '%s' not in database!", s.cfg.CurrentUserName)
	}

	// Get feed ID from URL
	feed, err := s.db.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("Feed URL '%s' not in database!", feedURL)
	}

	now := time.Now()
	newFeedFollow, err := s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			UserID:    user.ID,
			FeedID:    feed.ID,
		})

	if err != nil {
		return fmt.Errorf("Problem creating feed_follows record for user '%s' and feed URL '%s'", user.Name, feed.Url)
	}

	fmt.Printf("User '%s' now following feed '%s'\n", newFeedFollow.UserName, newFeedFollow.FeedName)

	return nil
}

func handlerFollowing(s *state, _ command) error {
	// Get user ID
	user, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return fmt.Errorf("User '%s' not in database!", s.cfg.CurrentUserName)
	}

	// Get follows
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("Problem fetching follows for user '%s': %v", s.cfg.CurrentUserName, err)
	}

	for _, follow := range follows {
		fmt.Printf("%s\n", follow.FeedName)
	}

	return nil
}
