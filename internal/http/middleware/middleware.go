// Package middleware provides HTTP middleware components for the TAKL HTTP server.
//
// This package implements Clean Architecture principles for HTTP middleware:
// - Centralized error handling with consistent response format
// - Request logging with unique request IDs
// - Common HTTP utilities and response writers
//
// Design Goals:
// - Reduce if/else error handling ladders in handlers
// - Provide consistent JSON error responses across all endpoints
// - Enable request tracing with unique identifiers
// - Centralize cross-cutting HTTP concerns
package middleware

import (
	"log/slog"
	"net/http"
)

// ChainMiddleware applies multiple middleware functions in order
func ChainMiddleware(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		// Apply middleware in reverse order so they execute in the specified order
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

// StandardMiddleware returns the standard middleware chain for TAKL HTTP handlers
func StandardMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return ChainMiddleware(
		RequestIDMiddleware(),     // Add request ID first
		LoggingMiddleware(logger), // Log with request ID context
		ErrorMiddleware(logger),   // Handle errors with logging
	)
}

// HTTPErrorHandler is an interface for handlers that can write structured errors
type HTTPErrorHandler interface {
	WriteError(err error)
}

// GetErrorHandler extracts the error handler from a response writer if available
func GetErrorHandler(w http.ResponseWriter) (HTTPErrorHandler, bool) {
	if eh, ok := w.(HTTPErrorHandler); ok {
		return eh, true
	}
	return nil, false
}
