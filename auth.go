package main

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"log"

	"github.com/t-900-a/gemmit/feeds"

	"git.sr.ht/~adnano/go-gemini"
)

var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

type UserContext struct {
	ID          int
	Certificate *x509.Certificate
	Hash        string
	NewUser     bool
}

func CertificateMiddleware(h gemini.Handler) gemini.Handler {
	return gemini.HandlerFunc(func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		certs := r.TLS().PeerCertificates
		if len(certs) == 0 {
			user := UserContext{
				Certificate: nil,
				Hash:        "empty20ceb2df1484eb936da53593733eb9aba167023657159390edff41caf07",
				NewUser:     false,
			}
			h.ServeGemini(context.WithValue(ctx, userCtxKey, &user), w, r)
			//w.WriteHeader(60, "A client certificate is required to use this service")
			return
		}
		if len(certs) != 1 {
			w.WriteHeader(62, "Expected a self-signed certificate")
			return
		}

		cert := certs[0]
		sum := sha256.Sum256(cert.Raw)
		hash := hex.EncodeToString(sum[:])

		user := UserContext{
			Certificate: cert,
			Hash:        hash,
			NewUser:     true,
		}

		if err := feeds.WithTx(ctx, nil, func(tx *sql.Tx) error {
			row := tx.QueryRowContext(ctx,
				`SELECT id FROM users WHERE certhash = $1`, hash)
			if err := row.Scan(&user.ID); err != sql.ErrNoRows {
				user.NewUser = false
				return err
			}

			row = tx.QueryRowContext(ctx, `
				INSERT INTO users (
					created,
					certhash
				) VALUES (
					NOW() at time zone 'utc',
					$1
				)
				ON CONFLICT ON CONSTRAINT users_certhash_key
				DO NOTHING
				RETURNING id;
			`, hash)
			if err := row.Scan(&user.ID); err != nil {
				return err
			}

			return nil
		}); err != nil {
			w.WriteHeader(40, "Internal server error")
			log.Println(err)
			return
		}

		h.ServeGemini(context.WithValue(ctx, userCtxKey, &user), w, r)
	})
}

func User(ctx context.Context) *UserContext {
	raw, ok := ctx.Value(userCtxKey).(*UserContext)
	if !ok {
		panic("Invalid authentication context")
	}
	return raw
}
