package tests

import (
	"fmt"

	"github.com/google/uuid"
)

func generateRandomString(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:10])
}

// stringPtr returns a pointer to the string value.
func stringPtr(s string) *string {
	return &s
}
