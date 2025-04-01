// pkg/uuid/uuid.go
package uuid

import (
	"github.com/gofrs/uuid"
)

// Generate creates a UUID v7 - time-ordered UUID
func Generate() string {

	// Create a UUID v7 (time-ordered UUID)
	id, err := uuid.NewV7()
	if err != nil {
		// Fall back to UUID v4 if v7 fails for any reason
		id, _ = uuid.NewV4()
	}

	return id.String()
}
