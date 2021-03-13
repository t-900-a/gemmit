package main

import (
	"context"
	"log"

	"git.sr.ht/~adnano/go-gemini"
)

func LoggingMiddleware(h gemini.Handler) gemini.Handler {
	return gemini.HandlerFunc(func(ctx context.Context, w gemini.ResponseWriter, r *gemini.Request) {
		lw := &logResponseWriter{rw: w}
		h.ServeGemini(ctx, lw, r)
		host := r.TLS().ServerName
		log.Printf("gemini: %s %q %d %d", host, r.URL, lw.Status, lw.Wrote)
	})
}

type logResponseWriter struct {
	Status      gemini.Status
	Wrote       int
	rw          gemini.ResponseWriter
	mediatype   string
	wroteHeader bool
}

func (w *logResponseWriter) SetMediaType(mediatype string) {
	w.mediatype = mediatype
}

func (w *logResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(gemini.StatusSuccess, w.mediatype)
	}
	n, err := w.rw.Write(b)
	w.Wrote += n
	return n, err
}

func (w *logResponseWriter) WriteHeader(status gemini.Status, meta string) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.Status = status
	w.Wrote += len(meta) + 5
	w.rw.WriteHeader(status, meta)
}

func (w *logResponseWriter) Flush() error {
	return nil
}
