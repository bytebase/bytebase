package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// Dump dumps database.
func (d *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	txn, err := d.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer txn.Rollback()

	query := fmt.Sprintf(`SELECT GET_DDL('DATABASE', '"%s"', true)`, d.databaseName)
	rows, err := txn.QueryContext(ctx, query)
	if err != nil {
		return util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databaseDDL string
	for rows.Next() {
		if err := rows.Scan(&databaseDDL); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Transform1: remove `create or replace database` statement.
	lines := strings.Split(databaseDDL, "\n")
	if len(lines) >= 2 {
		lines = lines[2:]
	}
	databaseDDL = strings.Join(lines, "\n")

	// Transform2: remove "create or replace schema PUBLIC;\n\n" because it's created by default.
	schemaStmt := fmt.Sprintf("create or replace schema %s.PUBLIC;", d.databaseName)
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt+"\n\n", "")
	// If this is the last statement.
	databaseDDL = strings.ReplaceAll(databaseDDL, schemaStmt, "")

	var transformedLines []string
	for _, line := range strings.Split(databaseDDL, "\n") {
		if strings.HasPrefix(strings.ToLower(line), "create ") {
			// Transform3: Remove "DEMO_DB." quantifier.
			line = strings.ReplaceAll(line, fmt.Sprintf(" %s.", d.databaseName), " ")

			// Transform4 (Important!): replace all `create or replace ` with `create ` to not break existing schema by any chance.
			line = strings.ReplaceAll(line, "create or replace ", "create ")
		}
		transformedLines = append(transformedLines, line)
	}
	databaseDDL = strings.Join(transformedLines, "\n")

	if _, err := io.WriteString(out, databaseDDL); err != nil {
		return err
	}

	return txn.Commit()
}
