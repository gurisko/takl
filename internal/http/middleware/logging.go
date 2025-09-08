package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// Context key for request ID
type contextKey string

const RequestIDKey contextKey = "request_id"

// Global request counter for generating request IDs
var requestCounter atomic.Uint64

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate unique request ID
			requestID := fmt.Sprintf("req-%d", requestCounter.Add(1))

			// Add request ID to context
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			r = r.WithContext(ctx)

			// Add request ID to response headers
			w.Header().Set("X-Request-ID", requestID)

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetRequestID extracts the request ID from the request context
func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(RequestIDKey).(string); ok {
		return id
	}
	return "unknown"
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := GetRequestID(r)

			// Log incoming request
			logger.Info("HTTP request started",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)

			// Wrap response writer to capture status code and response size
			lrw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default status
			}

			// Process request
			next.ServeHTTP(lrw, r)

			// Log response
			duration := time.Since(start)
			logger.Info("HTTP request completed",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"status_code", lrw.statusCode,
				"response_size", lrw.responseSize,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture response metrics
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int
}

// WriteHeader captures the status code
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (lrw *loggingResponseWriter) Write(data []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(data)
	lrw.responseSize += n
	return n, err
}

// Unwrap returns the underlying ResponseWriter (for middleware compatibility)
func (lrw *loggingResponseWriter) Unwrap() http.ResponseWriter {
	return lrw.ResponseWriter
}
