package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/venzy/gator/internal/config"
	"github.com/venzy/gator/internal/database"
	"github.com/venzy/gator/internal/feed"

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
			ID: uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name: username,
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

func handlerAgg(s *state, _ command) error {
	url := "https://www.wagslane.dev/index.xml"
	feed, err := fetchFeed(context.Background(), url)
	if err != nil {
		return fmt.Errorf("Problem fetching feed: %s", err)
	}
	fmt.Printf("%v", feed)
	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*feed.RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %s", err)
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making request: %s", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %s", err)
	}

	var rssFeed feed.RSSFeed
	err = xml.Unmarshal(body, &rssFeed)
	if err != nil {
		return nil, fmt.Errorf("Error parsing XML: %s", err)
	}

	// Unescape various strings
	rssFeed.Channel.Title = html.UnescapeString(rssFeed.Channel.Title)
	rssFeed.Channel.Description = html.UnescapeString(rssFeed.Channel.Description)
	for idx, item := range rssFeed.Channel.Item {
		rssFeed.Channel.Item[idx].Title = html.UnescapeString(item.Title)
		rssFeed.Channel.Item[idx].Description = html.UnescapeString(item.Description)
	}

	return &rssFeed, nil
}

func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("addfeed requires two arguments, the feed name and URL")
	}

	feedname := cmd.args[0]
	feedURL := cmd.args[1]

	user, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return fmt.Errorf("Current user '%s' not in database!", s.cfg.CurrentUserName)
	}

	now := time.Now()
	newFeed, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID: uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name: feedname,
			Url: feedURL,
			UserID: user.ID,
		})
	if err != nil {
		return fmt.Errorf("Problem creating feed: %v", err)
	}

	// Auto-create new feed_follows entry
	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID: uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			UserID: user.ID,
			FeedID: newFeed.ID,
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
			ID: uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			UserID: user.ID,
			FeedID: feed.ID,
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