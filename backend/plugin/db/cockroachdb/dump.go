package cockroachdb

import (
	"context"
	"database/sql"
	"io"
	"strings"

	"github.com/cockroachdb/cockroach-go/v2/crdb"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, w io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	sb := &strings.Builder{}
	if err := crdb.ExecuteTx(ctx, driver.db, &sql.TxOptions{
		ReadOnly: true,
	}, func(tx *sql.Tx) error {
		createSchemas, err := dumpCreateSchemas(ctx, tx)
		if err != nil {
			return err
		}
		_, _ = sb.WriteString(createSchemas)

		createTables, err := dumpCreateTables(ctx, tx)
		if err != nil {
			return err
		}
		_, _ = sb.WriteString(createTables)

		return nil
	}); err != nil {
		return err
	}

	if _, err := w.Write([]byte(sb.String())); err != nil {
		return err
	}

	return nil
}

func dumpCreateTables(ctx context.Context, tx *sql.Tx) (string, error) {
	sb := &strings.Builder{}
	rows, err := tx.QueryContext(ctx, "SELECT create_statement FROM [SHOW CREATE ALL TABLES];")
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

	return sb.String(), nil
}

func dumpCreateSchemas(ctx context.Context, tx *sql.Tx) (string, error) {
	sb := &strings.Builder{}
	rows, err := tx.QueryContext(ctx, "SELECT create_statement FROM [SHOW CREATE ALL SCHEMAS];")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return "", err
		}
		// Skip public schema
		if strings.HasSuffix(payload, " public;") {
			continue
		}
		_, _ = sb.WriteString(payload)
		_, _ = sb.WriteString("\n\n")
	}

	if err := rows.Err(); err != nil {
		return "", err
	}

	return sb.String(), nil
}
