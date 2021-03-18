package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	feeds "github.com/t-900-a/gemmit/feeds"

	"github.com/jackc/pgx/v4/stdlib"
)

func main() {
	db, err := sql.Open("pgx", os.Args[1])
	if err != nil {
		panic(err)
	}

	lightwallet := os.Args[2] // https://api.mymonero.com:8443

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
		// update payment records for all authors
		rows, err := tx.Query(ctx, `
			SELECT id, view_key, address, registered, scan_height
			FROM accepted_payments 
			WHERE pay_type = $1;
		`, "application/monero-paymentrequest")
		if err != nil {
			panic(err)
		}
		var toRefresh []*struct {
			ID            int
			ViewKey       string
			Address       string
			Registered    bool
			ScannedHeight int
		}

		for rows.Next() {
			account := &struct {
				ID            int
				ViewKey       string
				Address       string
				Registered    bool
				ScannedHeight int
			}{}
			if err := rows.Scan(&account.ID, &account.ViewKey, &account.Address, &account.Registered, &account.ScannedHeight); err != nil {
				panic(err)
			}
			toRefresh = append(toRefresh, account)
		}

		for _, a := range toRefresh {
			if !a.Registered {
				// login for unregistered accounts
				method := "/login"
				strData := `{"address":"` + a.Address + `","view_key":"` + a.ViewKey + `","create_account":true,"generated_locally":false}`
				jsonData := []byte(strData)
				u, _ := url.ParseRequestURI(lightwallet)
				u.Path = method
				urlStr := u.String()
				client := &http.Client{}
				r, _ := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer(jsonData))
				r.Header.Add("Accept", "application/json")
				r.Header.Add("Accept-Encoding", "gzip, deflate, br")
				r.Header.Add("Content-Type", "application/json")
				r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))

				resp, err := client.Do(r)
				if err != nil {
					log.Println(err)
				}

				if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Println(err)
					}

					type Login struct {
						NewAddress  bool `json:"new_address"`
						StartHeight int  `json:"start_height"`
					}

					var l = new(Login)
					err = json.Unmarshal(body, &l)
					if err != nil {
						log.Println(err)
					}
					a.Registered = true
					tx.Exec(ctx, `
						UPDATE accepted_payments SET registered=$2 WHERE id = $1
						`, a.ID, a.Registered)
				} else {
					log.Println("Failed to register / login to ", lightwallet, " for Monero account ", a.Address)
				}
			}

			// get transactions for address
			method := "/get_address_txs"
			strData := `{"address":"` + a.Address + `","view_key":"` + a.ViewKey + `"}`
			jsonData := []byte(strData)
			u, _ := url.ParseRequestURI(lightwallet)
			u.Path = method
			urlStr := u.String()

			client := &http.Client{}
			r, _ := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer(jsonData))
			r.Header.Add("Accept", "application/json")
			r.Header.Add("Accept-Encoding", "gzip, deflate, br")
			r.Header.Add("Content-Type", "application/json")
			r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))

			resp, err := client.Do(r)
			if err != nil {
				log.Println(err)
			}

			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				body, err := ioutil.ReadAll(resp.Body)
				log.Println(string(body))
				if err != nil {
					log.Println(err)
				}

				type Transaction struct {
					Id            uint64    `json:"id"`
					Hash          byte      `json:"hash"`
					Timestamp     time.Time `json:"timestamp"`
					TotalReceived uint64    `json:"total_received"`
					TotalSent     uint64    `json:"total_sent"`
					UnlockTime    uint64    `json:"unlock_time"`
					Height        int       `json:"height"`
					//SpentOutputs  []struct{} `json:"spent_outputs"`
					//PaymentId     byte       `json:"payment_id"`
					Coinbase bool   `json:"coinbase"`
					Mempool  bool   `json:"mempool"`
					Mixin    uint32 `json:"mixin"`
				}
				type Txs struct {
					TotalReceived      uint64        `json:"total_received"`
					ScannedHeight      uint64        `json:"total_received"`
					ScannedBlockHeight uint64        `json:"scanned_block_height"`
					StartHeight        uint64        `json:"start_height"`
					BlockchainHeight   uint64        `json:"blockchain_height"`
					Transactions       []Transaction `json:"transaction"`
				}

				var ts = new(Txs)
				err = json.Unmarshal(body, &ts)
				fmt.Printf("%+v \n", ts)
				if err != nil {
					log.Println(err)
				}
				for _, t := range ts.Transactions {
					log.Println(t)
					if a.ScannedHeight < t.Height {
						// transaction hasn't been recorded
						if _, err := tx.Exec(ctx, `
							INSERT INTO payments (
								address, tx_id, tx_date, amount, accepted_payments_id
							) VALUES ($1, $2);
							`, a.Address, string(t.Hash), t.Timestamp, t.TotalReceived, a.ID); err != nil {
							return err
						}

						a.ScannedHeight = t.Height
					}
				}
			} else {
				log.Println("Failed to register / login to ", lightwallet, " for Monero account ", a.Address)
			}
			tx.Exec(ctx, `
				UPDATE accepted_payments SET scan_height=$2 WHERE id = $1
				`, a.ID, a.ScannedHeight)
			time.Sleep(1 * time.Second)
		}
		tx.Commit(ctx)
		return nil
	})
}
