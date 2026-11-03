//nolint:revive
package util

import (
	"strings"

	"github.com/pkg/errors"
)

// pgValueEscaper escapes the two characters that end or escape a quoted value
// in a libpq keyword/value connection string.
var pgValueEscaper = strings.NewReplacer(`\`, `\\`, `'`, `\'`)

// QuotePGValue returns value quoted for a libpq keyword/value connection
// string. Unquoted, whitespace ends a value and pgx reads the rest as more
// keywords, the last of each winning: a port of "5432 host=other.example.com"
// connects to other.example.com. Surrounding whitespace is trimmed, as the
// unquoted form ignored it.
func QuotePGValue(value string) string {
	return "'" + pgValueEscaper.Replace(strings.TrimSpace(value)) + "'"
}

// PGHostPort returns the host and port keywords of a libpq keyword/value
// connection string, each value quoted. A blank host is refused: pgx gives an
// empty host its default, a Unix socket directory on this server.
func PGHostPort(host, port string) (string, error) {
	if strings.TrimSpace(host) == "" {
		return "", errors.New("host must be set")
	}
	return "host=" + QuotePGValue(host) + " port=" + QuotePGValue(port), nil
}
