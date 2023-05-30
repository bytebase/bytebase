package tests

import (
	"fmt"

	"github.com/google/uuid"
)

func generateRandomString(prefix string, length int) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:length])
}
