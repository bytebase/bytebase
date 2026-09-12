package tests

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func generateRandomString(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:10])
}

// shortInstanceID is an eight-character instance ID, for the tests that keep
// names short. Instance IDs are unique per workspace, which the tests share.
func shortInstanceID() string {
	return "in" + strings.ReplaceAll(uuid.New().String(), "-", "")[:6]
}

// uniqueDB makes a database name unique to one test, which the shared target
// Postgres needs: two tests asking for "history_db" would be one database.
func uniqueDB(name string) string {
	return fmt.Sprintf("%s_%s", name, strings.ReplaceAll(uuid.New().String(), "-", "")[:8])
}
