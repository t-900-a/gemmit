package main

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"
	"time"

	feeds "git.sr.ht/~sircmpwn/gemreader/feeds"

	"github.com/jackc/pgx/v4/stdlib"
)

func main() {
	db, err := sql.Open("pgx", os.Args[1])
	if err != nil {
		panic(err)
	}

	ctx := feeds.DBContext(context.TODO(), db)
	conn, err := db.Conn(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	conn.Raw(func(driverConn interface{}) error {
		conn := driverConn.(*stdlib.Conn).Conn()
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}

		defer func() {
			if err := recover(); err != nil {
				tx.Rollback(ctx)
				panic(err)
			}
		}()

		since := time.Now().UTC().Add(-4 * time.Hour)
		rows, err := tx.Query(ctx, `
			SELECT id, url
			FROM feeds
			WHERE updated < $1
		`, since)
		if err != nil {
			panic(err)
		}
		var toUpdate []*struct {
			ID  int
			URL string
		}
		for rows.Next() {
			feed := &struct {
				ID  int
				URL string
			}{}
			if err := rows.Scan(&feed.ID, &feed.URL); err != nil {
				panic(err)
			}
			toUpdate = append(toUpdate, feed)
		}

		for _, f := range toUpdate {
			log.Printf("Fetching %s", f.URL)
			u, _ := url.Parse(f.URL)
			feed, _, err := feeds.Fetch(ctx, u)
			if err != nil {
				log.Println("Error: %v", err)
				continue
			}
			err = feeds.Index(ctx, tx, feed.Items, f.ID)
			if err != nil {
				log.Println("Error: %v", err)
				continue
			}
		}

		tx.Exec(ctx, `
			UPDATE feeds SET updated = NOW() at time zone 'utc' WHERE updated < $1
		`, since)
		tx.Commit(ctx)

		return nil
	})
}
