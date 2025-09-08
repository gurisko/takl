package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// Standard HTTP errors that map to specific status codes
var (
	ErrNotFound      = errors.New("not_found")
	ErrConflict      = errors.New("conflict")
	ErrInvalid       = errors.New("invalid")
	ErrUnprocessable = errors.New("unprocessable")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrBadRequest    = errors.New("bad_request")
	ErrInternal      = errors.New("internal_error")
)

// HTTPError represents a structured HTTP error response
type HTTPError struct {
	Code    int               `json:"-"`                 // HTTP status code
	Key     string            `json:"error"`             // Machine-readable error key
	Msg     string            `json:"message"`           // Human-readable message
	Details map[string]string `json:"details,omitempty"` // Additional context
}

// Error implements the error interface
func (e HTTPError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("%s: %s", e.Key, e.Msg)
	}
	return e.Key
}

// WriteErr writes a structured error response to the HTTP writer
func WriteErr(w http.ResponseWriter, he HTTPError) {
	// Set content type
	w.Header().Set("Content-Type", "application/json")

	// Set status code
	if he.Code == 0 {
		he.Code = http.StatusInternalServerError
	}
	w.WriteHeader(he.Code)

	// Encode and write JSON response
	if err := json.NewEncoder(w).Encode(he); err != nil {
		// Fallback if JSON encoding fails
		http.Error(w, `{"error":"internal_error","message":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

// MapAppError maps common application errors to HTTP errors
func MapAppError(err error) HTTPError {
	if err == nil {
		return HTTPError{}
	}

	errMsg := err.Error()

	// Map by error message patterns
	switch {
	// 404 Not Found
	case strings.Contains(errMsg, "not found"):
		return HTTPError{
			Code: http.StatusNotFound,
			Key:  "not_found",
			Msg:  "Resource not found",
		}
	// 409 Conflict
	case strings.Contains(errMsg, "version mismatch"):
		return HTTPError{
			Code: http.StatusConflict,
			Key:  "conflict",
			Msg:  "Version mismatch - resource was modified concurrently",
		}
	case strings.Contains(errMsg, "status mismatch"):
		return HTTPError{
			Code: http.StatusConflict,
			Key:  "conflict",
			Msg:  "Status mismatch - resource is in unexpected state",
		}
	// 400 Bad Request
	case strings.Contains(errMsg, "invalid"):
		return HTTPError{
			Code: http.StatusBadRequest,
			Key:  "invalid",
			Msg:  "Invalid request data",
		}
	case strings.Contains(errMsg, "missing required"):
		return HTTPError{
			Code: http.StatusBadRequest,
			Key:  "invalid",
			Msg:  "Missing required field",
		}
	case strings.Contains(errMsg, "method not allowed"):
		return HTTPError{
			Code: http.StatusMethodNotAllowed,
			Key:  "method_not_allowed",
			Msg:  "HTTP method not allowed for this endpoint",
		}
	// 422 Unprocessable Entity
	case strings.Contains(errMsg, "invalid transition"):
		return HTTPError{
			Code: http.StatusUnprocessableEntity,
			Key:  "unprocessable",
			Msg:  "Invalid state transition",
		}
	case strings.Contains(errMsg, "capacity exceeded"):
		return HTTPError{
			Code: http.StatusUnprocessableEntity,
			Key:  "unprocessable",
			Msg:  "Operation would exceed capacity limits",
		}
	// 503 Service Unavailable
	case strings.Contains(errMsg, "service unavailable") || strings.Contains(errMsg, "not initialized"):
		return HTTPError{
			Code: http.StatusServiceUnavailable,
			Key:  "service_unavailable",
			Msg:  "Service temporarily unavailable",
		}
	// 500 Internal Server Error (default)
	default:
		return HTTPError{
			Code: http.StatusInternalServerError,
			Key:  "internal_error",
			Msg:  "Internal server error",
		}
	}
}

// ErrorMiddleware provides centralized error handling for handlers
func ErrorMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a custom response writer to capture errors
			rw := &errorResponseWriter{
				ResponseWriter: w,
				logger:         logger,
				requestID:      GetRequestID(r),
			}

			next.ServeHTTP(rw, r)
		})
	}
}

// errorResponseWriter wraps http.ResponseWriter to provide error handling
type errorResponseWriter struct {
	http.ResponseWriter
	logger    *slog.Logger
	requestID string
	written   bool
}

// WriteError is a helper method for handlers to write errors
func (rw *errorResponseWriter) WriteError(err error) {
	if rw.written {
		return // Don't write twice
	}

	httpErr := MapAppError(err)

	// Log the error with context
	if rw.logger != nil {
		rw.logger.Error("HTTP error",
			"request_id", rw.requestID,
			"error", err.Error(),
			"status_code", httpErr.Code,
			"error_key", httpErr.Key,
		)
	}

	WriteErr(rw.ResponseWriter, httpErr)
	rw.written = true
}

// WriteHeader tracks if response has been written
func (rw *errorResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write tracks if response has been written
func (rw *errorResponseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(data)
}
