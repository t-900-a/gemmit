package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/t-900-a/gemmit/feeds"

	"git.sr.ht/~adnano/go-gemini"
	"git.sr.ht/~adnano/go-gemini/certificate"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	hostname := os.Args[1]
	certpath := "/var/lib/gemini/certs"
	cs := os.Args[2]
	if len(os.Args) > 3 {
		certpath = os.Args[3]
	}
	db, err := sql.Open("pgx", cs)
	if err != nil {
		log.Fatalf("Failed to open a database connection: %v", err)
	}

	certificates := &certificate.Store{}
	if err := certificates.Load(certpath); err != nil {
		log.Fatal(err)
	}
	certificates.Register(hostname)

	mux := configureRoutes()

	server := &gemini.Server{
		Handler: feeds.DatabaseMiddleware(
			LoggingMiddleware(
				CertificateMiddleware(mux)),
			db),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   1 * time.Minute,
		GetCertificate: certificates.Get,
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	errch := make(chan error)
	go func() {
		ctx := context.Background()
		log.Println("Starting server")
		errch <- server.ListenAndServe(ctx)
	}()

	select {
	case err := <-errch:
		log.Fatal(err)
	case <-c:
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}
}
