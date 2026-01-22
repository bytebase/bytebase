package ghost

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ghostDirectiveRegex matches: -- gh-ost = {"key":"value",...} or -- gh-ost = {} /*comment*/
// Captures the JSON object, allows optional trailing /* comment */
var ghostDirectiveRegex = regexp.MustCompile(`(?im)^\s*--\s*gh-ost\s*=\s*(\{[^}]*\})\s*(?:/\*.*\*/)?\s*$`)

// ParseGhostDirective extracts ghost configuration from sheet content.
// Returns nil if no ghost directive is found.
func ParseGhostDirective(content string) (map[string]string, error) {
	match := ghostDirectiveRegex.FindStringSubmatch(content)
	if len(match) < 2 {
		return nil, nil
	}

	var flags map[string]string
	if err := json.Unmarshal([]byte(match[1]), &flags); err != nil {
		return nil, err
	}

	return flags, nil
}

// IsGhostEnabled checks if ghost is enabled by checking for directive presence.
func IsGhostEnabled(content string) bool {
	return ghostDirectiveRegex.MatchString(content)
}

// RemoveGhostDirective removes the ghost directive from sheet content.
func RemoveGhostDirective(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	for _, line := range lines {
		if !ghostDirectiveRegex.MatchString(line) {
			result = append(result, line)
		}
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}
