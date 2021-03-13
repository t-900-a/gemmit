package feeds

import (
	"context"
	"database/sql"

	"git.sr.ht/~adnano/go-gemini"
)

var dbCtxKey = &contextKey{"database"}

type contextKey struct {
	name string
}

func DatabaseMiddleware(h gemini.Handler, db *sql.DB) gemini.Handler {
	return gemini.HandlerFunc(func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		h.ServeGemini(DBContext(ctx, db), w, r)
	})
}

func DBContext(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, dbCtxKey, db)
}

func ForContext(ctx context.Context) (*sql.Conn, error) {
	raw, ok := ctx.Value(dbCtxKey).(*sql.DB)
	if !ok {
		panic("Invalid database context")
	}
	return raw.Conn(ctx)
}

func DBForContext(ctx context.Context) *sql.DB {
	raw, ok := ctx.Value(dbCtxKey).(*sql.DB)
	if !ok {
		panic("Invalid database context")
	}
	return raw
}

func WithTx(ctx context.Context, opts *sql.TxOptions, fn func(tx *sql.Tx) error) error {
	conn, err := ForContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
		tx.Commit()
	}()
	err = fn(tx)
	if err != nil {
		tx.Rollback()
	}
	return err
}
