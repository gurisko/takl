package shared

import (
	"crypto/rand"
	"fmt"
	"time"
)

// DefaultIDGenerator generates issue IDs in the format "iss-abc123"
type DefaultIDGenerator struct{}

func (g DefaultIDGenerator) Generate() string {
	// Generate a simple ID like iss-abc123
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("iss-%06d", time.Now().UnixNano()%1000000)
	}

	return fmt.Sprintf("iss-%06x", bytes)
}
