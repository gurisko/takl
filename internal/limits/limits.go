package limits

// JSON size limits for API payloads and responses

const (
	// JSON is the standard size limit for API request/response payloads (1MB)
	JSON = 1 << 20

	// ErrorBody is the maximum size for error response bodies (1KB)
	// Used when parsing error messages from failed API calls
	ErrorBody = 1024
)
