package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"time"
)

type contextKey string

const (
	uidKey    contextKey = "userID"
	ridKey    contextKey = "requestID"
	timeKey   contextKey = "requestTime"
	loggerKey contextKey = "requestLogger"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

type Middleware func(http.Handler) http.Handler

func CreateStack(m ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(m) - 1; i >= 0; i-- {
			x := m[i]
			next = x(next)
		}

		return next
	}
}

func generateRequestID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		slog.Error("middleware_error", "error_message", err)
	}
	return hex.EncodeToString(bytes)
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ridKey, generateRequestID())
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		rid, ok := r.Context().Value(ridKey).(string)
		if !ok {
			slog.Error("middleware_error", "error_message", "missing requestID in context")
		}
		logger := slog.Default().With("method", r.Method).With("path", r.URL.Path).With("requestID", rid)
		ctx := context.WithValue(r.Context(), loggerKey, logger)
		ctx = context.WithValue(ctx, timeKey, start)
		r = r.WithContext(ctx)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		logger.Info("http_request", "status", wrapped.statusCode, "duration", duration.String())
	})
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"Status": "healthy"})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleHome)

	stack := CreateStack(RequestID, Logging)

	s := &http.Server{
		Addr:           ":8080",
		Handler:        stack(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
