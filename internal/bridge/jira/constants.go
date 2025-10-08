package jira

const (
	// MaxJSONPayloadSize is the maximum size for incoming JSON payloads (1MB)
	MaxJSONPayloadSize = 1 << 20

	// MaxSearchResponseSize is the maximum size for Jira search API responses (10MB)
	MaxSearchResponseSize = 10 << 20

	// MaxErrorBodySize is the maximum bytes to read from error response bodies
	MaxErrorBodySize = 1024

	// SearchPageSize is the number of issues to fetch per API request
	SearchPageSize = 100

	// MaxSearchResults is the maximum total number of issues to fetch
	MaxSearchResults = 1000
)
