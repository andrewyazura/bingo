package web

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type contextKey string

const TraceIDKey contextKey = "trace_id"

func generateTraceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type wrappedWriter struct {
	http.ResponseWriter
	status int
}

func (w *wrappedWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *wrappedWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *wrappedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := generateTraceID()

		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
		r = r.WithContext(ctx)

		logger := slog.With("trace_id", traceID, "method", r.Method, "path", r.URL.Path)

		ww := &wrappedWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		logger.Info("HTTP Request Completed",
			slog.Int("status", ww.status),
			slog.String("duration", duration.String()),
			slog.String("ip", r.RemoteAddr),
		)
	})
}
