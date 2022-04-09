package common

import (
	"math/rand"
	"path"
	"sort"
	"strings"
	"time"
)

// FindString returns the search index of sorted strings.
func FindString(strings []string, search string) int {
	sort.Strings(strings)
	i := sort.SearchStrings(strings, search)
	if i == len(strings) {
		return -1
	}
	return i
}

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandomString returns a random string with length n.
func RandomString(n int) string {
	var sb strings.Builder
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteRune(letters[rand.Intn(len(letters))])
	}
	return sb.String()
}

// HasPrefixes returns true if the string s has any of the given prefixes.
func HasPrefixes(src string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(src, prefix) {
			return true
		}
	}
	return false
}

// GetPostgresDataDir returns the postgres data directory of Bytebase.
func GetPostgresDataDir(dataDir string) string {
	return path.Join(dataDir, "pgdata")
}

// GetPostgresSocketDir returns the postgres socket directory of Bytebase.
func GetPostgresSocketDir() string {
	return "/tmp"
}

// DefaultMigrationVersion returns the default migration version string.
// Use the concatenation of current time in second to guarantee uniqueness in a monotonic increasing way.
// We cannot add task ID because tenant mode databases should use the same migration version string when applying a schema update.
func DefaultMigrationVersion() string {
	return time.Now().Format("20060102150405")
}
