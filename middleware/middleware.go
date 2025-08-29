package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

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

type contextKey string

const (
	// uidKey    contextKey = "userID"
	ridKey    contextKey = "requestID"
	timeKey   contextKey = "requestTime"
	loggerKey contextKey = "requestLogger"
)

type eventKey string

const (
	completed eventKey = "http_request_completed"
	panic eventKey = "http_request_panic"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func (w *wrappedWriter) Write(b []byte) (int, error) {
        n, err := w.ResponseWriter.Write(b)
        w.size += n
        return n, err
    }

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ridKey, generateRequestID())
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// CORS handles CORS headers
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // or specific domain like "https://myapp.com"
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Continue to the next handler
		next.ServeHTTP(w, r)
	})
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Paths to skip logging
		skipPaths := map[string]bool{
			"/health": true,
		}

		// Skip logging for certain paths (health checks, metrics, etc.)
		if skipPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		rid, ok := r.Context().Value(ridKey).(string)
		if !ok {
			rid = "unknown"
			slog.Warn("missing_request_id",
				"path", r.URL.Path,
				"method", r.Method)
		}

		logger := slog.Default().With(
			"request_id", rid,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", getClientIP(r),
			// "user_agent", r.Header.Get("User-Agent"),
		)

		// Add logger and start time to context
		ctx := context.WithValue(r.Context(), loggerKey, logger)
		ctx = context.WithValue(ctx, timeKey, start)
		r = r.WithContext(ctx)

		// Panic recovery
		defer func() {
			if err := recover(); err != nil {
				logger.Error(string(panic),
					"error", err,
					"status", http.StatusInternalServerError,
					"duration_ms", time.Since(start).Milliseconds(),
				)
				http.Error(wrapped, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		logLevel := getLogLevelForStatus(wrapped.statusCode)

		logger.Log(r.Context(), logLevel, string(completed),
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
			"response_size", wrapped.size,
		)
	})
}

// Helper function to get client IP, handling proxies
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header (from load balancers/proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check for X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Helper to determine log level based on status code
func getLogLevelForStatus(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
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
