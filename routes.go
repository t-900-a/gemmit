package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"mime"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~adnano/go-gemini"
	"github.com/jackc/pgx/v4/stdlib"
)

func configureRoutes() *gemini.ServeMux {
	mux := &gemini.ServeMux{}

	mux.HandleFunc("/about", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		w.WriteHeader(20, "text/gemini")
		err := aboutPage.Execute(w, nil)
		if err != nil {
			panic(err)
		}
	})

	mux.HandleFunc("/earn", func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		w.WriteHeader(20, "text/gemini")
		err := aboutPage.Execute(w, nil)
		if err != nil {
			panic(err)
		}
	})

	return mux
}
