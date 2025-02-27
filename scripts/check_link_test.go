package tests

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/yterajima/go-sitemap"
)

var (
	// Extract all the links prefix with https://bytebase.com or https://www.bytebase.com in frontend code.
	linkMatcher = regexp.MustCompile(`(?m)https?:\/\/(www\.)?bytebase.com([-a-zA-Z0-9()@:%_\+.~#?&\/\/=]*)`)

	ignores = map[string]bool{
		"node_modules": true,
	}

	extensions = map[string]bool{
		".html": true,
		".js":   true,
		".json": true,
		".ts":   true,
		".vue":  true,
	}

	ignoreLinks = map[string]bool{
		"https://bytebase.com/changelog/bytebase-": true,
	}

	frontendDirectory = "../frontend"
)

func TestValidateLinks(t *testing.T) {
	a := require.New(t)
	links, err := extractLinkRecursive()
	a.NoError(err)

	sm, err := sitemap.Get("https://bytebase.com/sitemap.xml", nil)
	a.NoError(err)

	paths := map[string]struct{}{}
	for _, smu := range sm.URL {
		u, err := url.Parse(smu.Loc)
		a.NoError(err)
		paths[strings.TrimSuffix(u.Path, "/")] = struct{}{}
	}

	// Check all the links are reachable.
	for link := range links {
		u, err := url.Parse(link)
		a.NoError(err)
		if ignoreLinks[link] {
			continue
		}
		a.Contains(paths, strings.TrimSuffix(u.Path, "/"), link)
	}
}

func extractLinkRecursive() (map[string]bool, error) {
	// Initialize the result map.
	links := make(map[string]bool)

	// Define the function to be used with os.Walk.
	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if _, ok := ignores[info.Name()]; ok {
			return filepath.SkipDir
		}
		if info.IsDir() || (info.Mode()&os.ModeSymlink) == os.ModeSymlink {
			return nil
		}
		// Check if the file has a valid extension
		if _, ok := extensions[filepath.Ext(info.Name())]; !ok {
			return nil
		}

		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Find all matches using the regular expression
		matches := linkMatcher.FindAllString(string(content), -1)
		// Add matches to the result map
		for _, match := range matches {
			links[match] = true
		}

		return nil
	}

	// Start the recursive traversal using os.Walk
	if err := filepath.Walk(frontendDirectory, walkFn); err != nil {
		return nil, errors.Wrapf(err, "failed to walk directory: %s", frontendDirectory)
	}

	return links, nil
}
