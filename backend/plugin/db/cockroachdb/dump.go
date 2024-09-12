package cockroachdb

import (
	"context"
	"io"
	"strings"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, w io.Writer) (string, error) {
	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	sb := &strings.Builder{}
	rows, err := conn.QueryContext(ctx, "SELECT create_statement FROM [SHOW CREATE ALL TABLES];")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return "", err
		}
		_, _ = sb.WriteString(payload)
		_, _ = sb.WriteString("\n\n")
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	if _, err := w.Write([]byte(sb.String())); err != nil {
		return "", err
	}

	return sb.String(), nil
}
