package tests

import (
	"fmt"

	"github.com/google/uuid"
)

func generateRandomString(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:10])
}
