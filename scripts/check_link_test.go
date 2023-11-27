package tests

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestLinkHealth(t *testing.T) {
	// Extract all the links prefix with https://bytebase.com or https://www.bytebase.com in frontend code.
	regexp := regexp.MustCompile(`(?m)https?:\/\/(www\.)?bytebase.com([-a-zA-Z0-9()@:%_\+.~#?&\/\/=]*)`)
	// Our website may be re-deployed, so we need to retry a few times before we give up.
	defaultInterval := 1 * time.Minute
	defaultRetryTimes := 3

	directory := "../frontend"
	extensions := map[string]struct{}{
		".html": {},
		".js":   {},
		".vue":  {},
		".ts":   {},
		".json": {},
	}
	blackList := map[string]struct{}{
		"node_modules": {},
	}

	links, err := extractLinkRecursive(directory, blackList, extensions, regexp)
	require.NoError(t, err)

	// Check all the links are reachable.
	for link := range links {
		if err := checkLinkWithRetry(link, defaultRetryTimes, defaultInterval); err != nil {
			require.NoError(t, err)
		}
	}
}

func extractLinkRecursive(directory string, blackList map[string]struct{}, extensions map[string]struct{}, regexp *regexp.Regexp) (map[string]struct{}, error) {
	// Initialize the result map
	links := make(map[string]struct{})

	// Define the function to be used with os.Walk
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, ok := blackList[info.Name()]; ok {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			// Check if the file has a valid extension
			_, validExtension := extensions[filepath.Ext(info.Name())]
			if !validExtension {
				return nil
			}

			// Read the file content
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Find all matches using the regular expression
			matches := regexp.FindAllString(string(content), -1)

			// Add matches to the result map
			for _, match := range matches {
				links[match] = struct{}{}
			}
		}

		return nil
	}

	// Start the recursive traversal using os.Walk
	if err := filepath.Walk(directory, walkFn); err != nil {
		// Handle the error, e.g., log or return it
		return nil, errors.Wrapf(err, "failed to walk directory: %s", directory)
	}

	return links, nil
}

func checkLinkWithRetry(link string, retry int, interval time.Duration) error {
	for i := 0; i < retry; i++ {
		// Request the link and check the response status code is 200.
		res, err := http.Get(link)
		if err != nil {
			return errors.Wrapf(err, "failed to request link: %s", link)
		}
		if res.StatusCode != http.StatusOK {
			time.Sleep(interval)
			continue
		}
		return nil
	}

	return errors.Errorf("Link %s is not reachable after %d retries", link, retry)
}
