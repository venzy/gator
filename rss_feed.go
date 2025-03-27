package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/venzy/gator/internal/database"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
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

	var rssFeed RSSFeed
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

func parseDateTime(input string) (time.Time, error) {
    // List of possible formats
    formats := []string{
        "Mon, 02 Jan 2006 15:04:05 -0700",
    }

    var parsedDate time.Time
    var err error

    // Attempt to parse the date using each format
    for _, format := range formats {
        parsedDate, err = time.Parse(format, input)
        if err == nil {
            return parsedDate, nil
        }
    }

    // If none of the formats work, return an error
    return time.Time{}, fmt.Errorf("could not parse date '%s': %v", input, err)
}

func scapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	// Not sure why we mark it fetched before fetch success
	err = s.db.MarkFeedFetched(
		context.Background(),
		database.MarkFeedFetchedParams{
			ID: feed.ID,
			LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
		})
	
	rssFeed, err := fetchFeed(context.Background(), feed.Url)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, item := range rssFeed.Channel.Item {
		pubTime, err := parseDateTime(item.PubDate)
		if err != nil {
			fmt.Printf("Problem parsing publication date '%v' for item '%s', assuming 'now': %v\n", item.PubDate, item.Title, err)
			pubTime = now
		}
		_, err = s.db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID : uuid.New(),
				CreatedAt: now,
				UpdatedAt: now,
				Title: item.Title,
				Url: item.Link,
				Description: sql.NullString{String: item.Description, Valid: true},
				PublishedAt: pubTime,
				FeedID: feed.ID,
			})
		
		if err != nil && !strings.HasSuffix(err.Error(), "\"posts_url_key\"") {
			fmt.Printf("Problem adding post '%s': %v\n", item.Title, err)
		}
	}

	return nil
}