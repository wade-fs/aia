package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	collectorHttp "github.com/wade/aia/internal/collector/http"
)

type RSSChannel struct {
	Title       string    `xml:"channel>title"`
	Description string    `xml:"channel>description"`
	Items       []RSSItem `xml:"channel>item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

type FeedFetcher struct {
	client *collectorHttp.Client
}

func NewFeedFetcher(client *collectorHttp.Client) *FeedFetcher {
	return &FeedFetcher{client: client}
}

// Fetch parses an RSS feed URL into structured items.
func (f *FeedFetcher) Fetch(ctx context.Context, feedURL string) ([]RSSItem, error) {
	bytes, statusCode, err := f.client.Get(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	if statusCode >= 400 {
		return nil, fmt.Errorf("feed server returned HTTP %d", statusCode)
	}

	var ch RSSChannel
	if err := xml.Unmarshal(bytes, &ch); err != nil {
		return nil, fmt.Errorf("failed to parse rss xml: %w", err)
	}

	return ch.Items, nil
}

// ParsePubDate attempts standard RFC1123, RFC822, and ISO formats.
func ParsePubDate(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, raw); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}
