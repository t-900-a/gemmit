package feeds

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~adnano/go-gemini"
	"github.com/SlyMarbo/rss"
	"github.com/jackc/pgx/v4"
)

const (
	FEED_RSS = "rss"
	FEED_GEMINI = "gemini"
)

func fetchGemini(ctx context.Context, remoteURL *url.URL) (*rss.Feed, string, error) {
	client := &gemini.Client{}
	tctx, cancel := context.WithTimeout(ctx, 10 * time.Second)
	defer cancel()
	resp, err := client.Get(tctx, remoteURL.String())
	if err != nil {
		return nil, "", err
	}
	if resp.Status != gemini.StatusSuccess {
		return nil, "", fmt.Errorf("Unexpected Gemini response: %d %s",
			resp.Status, resp.Meta)
	}
	defer resp.Body.Close()

	mimetype, _, err := mime.ParseMediaType(resp.Meta)
	if err != nil {
		return nil, "", fmt.Errorf("Unintelligible content type: %s", resp.Meta)
	}

	reader := io.LimitReader(resp.Body, 1073741824) // 1 GiB

	switch mimetype {
	case "text/gemini":
		var feed rss.Feed
		feed.Link = remoteURL.String()
		text, err := gemini.ParseText(resp.Body)
		if err != nil {
			return nil, "", err
		}
		for _, line := range text {
			switch line := line.(type) {
			case gemini.LineHeading1:
				if feed.Title == "" {
					feed.Title = strings.TrimLeft(line.String(), "# ")
				}
			case gemini.LineLink:
				if line.Name == "" || len(line.Name) < 10 {
					continue
				}
				name := strings.TrimLeft(line.Name[:10], " ")
				date, err := time.Parse("2006-01-02", name)
				if err != nil {
					continue
				}
				link, err := url.Parse(line.URL)
				if err != nil {
					continue
				}
				link = remoteURL.ResolveReference(link)
				item := &rss.Item{}
				item.Title = strings.TrimLeft(line.Name[10:], ": ")
				item.Date = date
				item.Link = link.String()
				feed.Items = append(feed.Items, item)
			}
		}
		return &feed, FEED_GEMINI, nil
	case "text/xml",
		"application/rss+xml",
		"application/atom+xml",
		"application/xml":
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, "", err
		}
		feed, err := rss.Parse(data)
		return feed, FEED_RSS, err
	default:
		return nil, "", fmt.Errorf("Cannot interpret %s as a feed", resp.Meta)
	}
	panic("unreachable")
}
