package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/venzy/gator/internal/database"
	"time"
)

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("agg requires one argument, the time between requests - a duration string like 1s, 1m , 1h5m3s etc")
	}

	timeBetweenReqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("Invalid time-between-requests argument (duration) '%s' - %s\n", cmd.args[0], err)
	}

	fmt.Printf("Collecting feeds every %s\n", timeBetweenReqs)

	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		scapeFeeds(s)
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("addfeed requires two arguments, the feed name and URL")
	}

	feedname := cmd.args[0]
	feedURL := cmd.args[1]

	now := time.Now()
	newFeed, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name:      feedname,
			Url:       feedURL,
			UserID:    user.ID,
		})
	if err != nil {
		return fmt.Errorf("Problem creating feed: %v", err)
	}

	// Auto-create new feed_follows entry
	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			UserID:    user.ID,
			FeedID:    newFeed.ID,
		})

	if err != nil {
		return fmt.Errorf("Problem creating new follow: %v", err)
	}

	fmt.Printf("New Feed: %v\n", newFeed)

	return nil
}

func handlerFeeds(s *state, _ command) error {
	feedList, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("Problem fetching all feeds: %v", err)
	}

	for _, feedData := range feedList {
		user, err := s.db.GetUserByID(context.Background(), feedData.UserID)
		if err != nil {
			return fmt.Errorf("Problem getting user name for userID %s associated with feed %s (%s)", feedData.UserID, feedData.Name, feedData.Url)
		}

		fmt.Printf("%s %s %s\n", feedData.Name, feedData.Url, user.Name)
	}

	return nil
}
