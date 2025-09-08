package app

import "errors"

// Domain-specific errors that can be mapped to HTTP status codes by the transport layer
var (
	// Resource not found errors
	ErrProjectNotFound = errors.New("project not found")
	ErrIssueNotFound   = errors.New("issue not found")

	// Validation errors
	ErrInvalidInput    = errors.New("invalid input")
	ErrMissingRequired = errors.New("missing required field")

	// Concurrency control errors
	ErrVersionMismatch = errors.New("version mismatch")
	ErrStatusMismatch  = errors.New("status mismatch")

	// Business rule violations
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrCapacityExceeded  = errors.New("capacity exceeded")

	// Service unavailable errors
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrIndexUnavailable   = errors.New("index unavailable")
)
