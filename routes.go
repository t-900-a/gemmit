package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/t-900-a/gemmit/feeds"

	"git.sr.ht/~adnano/go-gemini"
	"github.com/jackc/pgx/v4/stdlib"
)

func configureRoutes() *gemini.ServeMux {
	mux := &gemini.ServeMux{}

	mux.HandleFunc("/", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		var top_feeds []*Feed
		if err := feeds.WithTx(ctx, &sql.TxOptions{
			Isolation: 0,
			ReadOnly:  true,
		}, func(tx *sql.Tx) error {
			rows, err := tx.QueryContext(ctx, `
				SELECT
					f.title, f.description, f.url, a.name, f.updated, COALESCE(votes.count, 0), COALESCE(votes.amount, 0)
				FROM feeds f
				INNER JOIN authors a ON f.author_id = a.id
				LEFT JOIN (SELECT ap.author_id, count(*) as count, sum(p.amount) as amount
				FROM payments p, accepted_payments ap
				WHERE p.accepted_payments_id = ap.id
				GROUP BY ap.author_id) as votes ON votes.author_id = f.author_id
				WHERE approved = true
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
			Feeds:   top_feeds,
			Logo:    gemmitLogo,
			Newline: "\n",
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

		if len(feed.Author.Extensions) > 0 {
			for _, ext := range feed.Author.Extensions {
				if ext.Rel != "payment" {
					w.WriteHeader(10, "Author's payment data within feed is malformed")
					return
				}
			}
		} else {
			w.WriteHeader(10, "No accepted Payments found within feed")
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
			if len(duplicates) > 0 {
				return errors.New("Feed already exists")
			}

			// TODO Author validate strings
			if err := func() error {
				var id int
				// feed must include an author, insert author first
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
				// TODO at add bitcoin input address validation
				// TODO viewkey input validation
				// go doesn't allow arrays of arbitrary length
				// arbitrarily capping the max accepted payments to 15
				accepted_payments := make([]*AcceptedPayment, 0, 15)
				re := regexp.MustCompile(`application\/.+-paymentrequest`)
				// outer loop finds paymentrequests within the author extensions
				// if the payment request is not a monero one, then add to accepted_payments array as is
				// inner loop is to find monero view key within extensions
				// both view key and address are added together to accepted_payments
				for _, outer_ext := range feed.Author.Extensions {
					if re.MatchString(outer_ext.Type) {
						split_index := strings.Index(outer_ext.Href, ":")
						address := ""
						if split_index > -1 {
							address = outer_ext.Href[split_index+1:]
						} else {
							return errors.New("Author's payment data within feed is malformed")
						}
						if outer_ext.Type == "application/monero-paymentrequest" {
							re = regexp.MustCompile("4[a-zA-Z\\d]{94}")
							if re.MatchString(address) {
								for _, inner_ext := range feed.Author.Extensions {
									if inner_ext.Type == "application/monero-viewkey" {
										accepted_payments = append(accepted_payments, &AcceptedPayment{
											PayType:    outer_ext.Type,
											ViewKey:    inner_ext.Href,
											Address:    address,
											Registered: false,
										})
									}
								}
							} else {
								return errors.New("Author's payment data within feed is malformed")
							}
						} else {
							accepted_payments = append(accepted_payments, &AcceptedPayment{
								PayType:    outer_ext.Type,
								ViewKey:    "",
								Address:    outer_ext.Href,
								Registered: false,
							})
						}
					}
				}
				if len(accepted_payments) < 1 {
					return errors.New("Failed to process Author's accepted payments")
				}
				// add the accepted payments to db table after we found them within the loop
				for _, pymnt := range accepted_payments {
					row = tx.QueryRow(ctx, `
					INSERT INTO accepted_payments (
						author_id, pay_type, view_key, address, registered, scan_height
					) VALUES (
						$1,
						$2,
						$3,
						$4,
						$5,
						$6
					)
					RETURNING id;
				`, id, pymnt.PayType, pymnt.ViewKey, pymnt.Address, pymnt.Registered, 0)
					if err := row.Scan(&id); err != nil {
						return err
					}
				}

				row = tx.QueryRow(ctx, `
					INSERT INTO feeds (
						created, updated, author_id, kind, url,  title, description, approved, feed_url
					) VALUES (
						NOW() at time zone 'utc',
						NOW() at time zone 'utc',
						$1, $2, $3, $4, $5, $6, $7
					)
					ON CONFLICT ON CONSTRAINT feeds_url_key
					DO UPDATE SET
						(updated, title, description) =
						(EXCLUDED.updated, EXCLUDED.title, EXCLUDED.description)
					RETURNING id;
				`, id, kind, feed.Link, feed.Title, feed.Description, true, feedURL.String())
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

	mux.HandleFunc("/vote", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		w.WriteHeader(20, "text/gemini")
		err := votePage.Execute(w, &EarnPage{
			Logo: gemmitLogo,
		})
		if err != nil {
			panic(err)
		}
	})

	return mux
}
