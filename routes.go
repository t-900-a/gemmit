package main

import (
	"context"
	"database/sql"
	"errors"

	//"fmt"
	//"io"
	"log"
	//"mime"
	"net/url"
	//"os/exec"
	//"sort"
	//"strconv"
	//"strings"
	//"time"

	"github.com/t-900-a/gemmit/feeds"

	"git.sr.ht/~adnano/go-gemini"
	"github.com/jackc/pgx/v4/stdlib"
)

func configureRoutes() *gemini.ServeMux {
	mux := &gemini.ServeMux{}

	mux.HandleFunc("/", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		user := User(ctx)
		if user.NewUser {
			err := welcomePage.Execute(w, &WelcomePage{
				Cert: user.Certificate,
				Hash: user.Hash,
				Logo: gemmitLogo,
			})
			if err != nil {
				panic(err)
			}
			return
		}

		var top_feeds []*Feed
		if err := feeds.WithTx(ctx, &sql.TxOptions{
			Isolation: 0,
			ReadOnly:  true,
		}, func(tx *sql.Tx) error {
			rows, err := tx.QueryContext(ctx, `
				SELECT
					f.title, f.description, f.url, a.name, f.updated, votes.count, votes.amount
				FROM feeds f
				INNER JOIN authors a ON f.author_id = a.id
				INNER JOIN (SELECT ap.author_id, count(*) as count, sum(p.amount) as amount
				FROM payments p, accepted_payments ap
				WHERE p.accepted_payments_id = ap.id
				GROUP BY ap.author_id) as votes ON votes.author_id = f.author_id
				ORDER BY votes.count DESC
				LIMIT 10;
			`)
			if err != nil {
				return err
			}

			for rows.Next() {
				feed := &Feed{}
				if err := rows.Scan(&feed.Title, &feed.Description, &feed.URL,
					&feed.Author, &feed.Updated, &feed.VoteCnt, &feed.VoteAmt); err != nil {
					return err
				}
				top_feeds = append(top_feeds, feed)
			}

			return nil
		}); err != nil {
			log.Println(err)
			w.WriteHeader(40, "Internal server error")
			return
		}

		w.WriteHeader(20, "text/gemini")
		err := dashboardPage.Execute(w, &DashboardPage{
			Feeds: top_feeds,
			Logo:  gemmitLogo,
		})
		if err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/add", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		user := User(ctx)
		if r.URL.RawQuery == "" {
			w.WriteHeader(10, "Enter a feed URL")
			return
		}

		query, err := url.QueryUnescape(r.URL.RawQuery)
		if err != nil {
			w.WriteHeader(10, err.Error()+": Try again")
			return
		}
		feedURL, err := url.Parse(query)
		if err != nil {
			w.WriteHeader(10, err.Error()+": Try again")
			return
		}

		feed, kind, err := feeds.Fetch(ctx, feedURL)
		if err != nil {
			w.WriteHeader(10, err.Error()+": Try again")
			return
		}

		if feed.Author.Extension.Rel != "payment" {
			w.WriteHeader(10, "author payment data malformed")
			return
		}



		conn, err := feeds.ForContext(ctx)
		if err != nil {
			panic(err)
		}

		if err := conn.Raw(func(driverConn interface{}) error {
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
			rows, err := tx.Query(ctx, `
			SELECT id, url
			FROM feeds
			WHERE url = $1
		`, feedURL.String())
			if err != nil {
				panic(err)
			}
			var duplicates []*struct {
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
				log.Println(feed.URL)
				duplicates = append(duplicates, feed)
			}
			log.Println(len(duplicates))
			if len(duplicates) > 0 {
				return errors.New("Feed already exists")
			}

			// TODO Author validate strings
			if err := func() error {
				var id int
				row := tx.QueryRow(ctx, `
					INSERT INTO authors (
						name, created, updated, url, email
					) VALUES (
						$1,
						NOW() at time zone 'utc',
						NOW() at time zone 'utc',
						$2,
						$3
					)
					RETURNING id;
				`, feed.Author.Name, feed.Author.URI, feed.Author.Email)
				if err := row.Scan(&id); err != nil {
					return err
				}
				// TODO allow authors to specify multiple accepted payments
				// TODO monero view key
				// TODO payments validate strings
				row = tx.QueryRow(ctx, `
					INSERT INTO accepted_payments (
						author_id, pay_type, address, registered
					) VALUES (
						$1,
						$2,
						$3,
						$4
					)
					RETURNING id;
				`, id, feed.Author.Extension.Type, feed.Author.Extension.Href, false)
				if err := row.Scan(&id); err != nil {
					return err
				}

				row = tx.QueryRow(ctx, `
					INSERT INTO feeds (
						created, updated, author_id, kind, url,  title, description
					) VALUES (
						NOW() at time zone 'utc',
						NOW() at time zone 'utc',
						$1, $2, $3, $4, $5
					)
					ON CONFLICT ON CONSTRAINT feeds_url_key
					DO UPDATE SET
						(updated, title, description) =
						(EXCLUDED.updated, EXCLUDED.title, EXCLUDED.description)
					RETURNING id;
				`, id, kind, feedURL.String(), feed.Title, feed.Description)
				if err := row.Scan(&id); err != nil {
					return err
				}

				if _, err := tx.Exec(ctx, `
					INSERT INTO submissions (
						user_id, feed_id
					) VALUES ($1, $2)
					ON CONFLICT ON CONSTRAINT submissions_user_id_feed_id_key
					DO NOTHING;
				`, user.ID, id); err != nil {
					return err
				}

				return feeds.Index(ctx, tx, feed.Items, id)
			}(); err != nil {
				tx.Rollback(ctx)
				return err
			}

			tx.Commit(ctx)
			return nil
		}); err != nil {
			log.Println(err)
			if err.Error() == "Feed already exists" {
				w.WriteHeader(10, err.Error())
			} else {
				w.WriteHeader(40, "Internal server error")
			}
			return
		}

		w.WriteHeader(30, "/")
	})

	mux.HandleFunc("/about", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		w.WriteHeader(20, "text/gemini")
		err := aboutPage.Execute(w, &AboutPage{
			Logo: gemmitLogo,
		})
		if err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/earn", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		w.WriteHeader(20, "text/gemini")
		err := earnPage.Execute(w, &EarnPage{
			Logo: gemmitLogo,
		})
		if err != nil {
			panic(err)
		}
	})

	return mux
}
