package registry

import "github.com/google/uuid"

// GenerateProjectID generates a new unique project ID using UUID v4
func GenerateProjectID() string {
	return uuid.New().String()
}
