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
	"github.com/jackc/pgx/v4"
	"github.com/zaddok/rss"
)

const (
	FEED_RSS    = "rss"
	FEED_GEMINI = "gemini"
)

func fetchGemini(ctx context.Context, remoteURL *url.URL) (*rss.Feed, string, error) {
	client := &gemini.Client{}
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
				name := strings.TrimLeft(line.Name[:10], " -—")
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

func fetchHTTP(ctx context.Context, url *url.URL) (*rss.Feed, string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	req.Header.Add("User-Agent", "gemmit (https://github.com/t-900-a/gemmit)")
	if err != nil {
		return nil, "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("Unexpected HTTP response %s", resp.Status)
	}

	if resp.Header.Get("Content-Type") == "text/html" {
		// TODO
		return nil, "", fmt.Errorf("Extracting feed from HTML pages is unimplemented")
	}

	reader := io.LimitReader(resp.Body, 1073741824) // 1 GiB
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	feed, err := rss.Parse(data)
	return feed, FEED_RSS, err
}

func Fetch(ctx context.Context, url *url.URL) (*rss.Feed, string, error) {
	switch url.Scheme {
	case "gemini":
		return fetchGemini(ctx, url)
	case "https":
		return fetchHTTP(ctx, url)
	default:
		return nil, "", fmt.Errorf("Unsupported protocol '%s'", url.Scheme)
	}
}

func Index(ctx context.Context, tx pgx.Tx,
	items []*rss.Item, feedId int) error {
	_, err := tx.Exec(ctx,
		`CREATE TEMP TABLE IF NOT EXISTS articles_temp (
			title varchar,
			published timestamp,
			url varchar,
			feed_id INTEGER
		);`)
	if err != nil {
		return err
	}

	rows := make([][]interface{}, len(items))
	for i, item := range items {
		rows[i] = []interface{}{
			item.Title, item.Date, item.Link, feedId,
		}
	}

	_, err = tx.CopyFrom(ctx,
		pgx.Identifier{"articles_temp"},
		[]string{"title", "published", "url", "feed_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return err
	}

	result, err := tx.Exec(ctx, `
		INSERT INTO articles
		(title, published, url, feed_id)
		SELECT title, published, url, feed_id
		FROM articles_temp
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE feeds SET updated = NOW() at time zone 'utc' WHERE id = $1;
	`, feedId)
	if err != nil {
		return err
	}

	ra := result.RowsAffected()
	log.Printf("Imported %d items for feed %d", ra, feedId)
	return nil
}
