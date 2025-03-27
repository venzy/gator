package main

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/venzy/gator/internal/database"
)

func handlerBrowse(s *state, cmd command, user database.User) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf("browse requires at most one argument, the max number of posts to see")
	}
	var limit int32
	if len(cmd.args) == 1 {
		parsedLimit, err := strconv.ParseInt(cmd.args[0], 0, 32)
		if err != nil {
			return fmt.Errorf("Problem parsing max posts argument '%s': %v", cmd.args[0], err)
		}
		if parsedLimit < 1 || parsedLimit > math.MaxInt32 {
			return fmt.Errorf("Out of range max posts argument '%s': must be from 1 to %v", cmd.args[0], math.MaxInt32)
		}
		limit = int32(parsedLimit)
	} else {
		// Default
		limit = 2
	}
	
	// Get posts
	posts, err := s.db.GetPostsForUser(
		context.Background(),
		database.GetPostsForUserParams{
			UserID: user.ID,
			Limit: limit,
		})
	if err != nil {
		return fmt.Errorf("Problem fetching posts for user '%s': %v", s.cfg.CurrentUserName, err)
	}

	for _, post := range posts {
		fmt.Printf("%s | %s | %s\n", post.PublishedAt.Local().Format("2006-01-02 15:04:05 MST"), post.FeedName, post.Title)
	}

	return nil
}