//nolint:revive
package util

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jackc/pgx/v5/stdlib"
)

// PgColumnTypeModifiers describes statement on conn without executing it and returns the
// PostgreSQL type modifier (atttypmod) of each result column, in column order. For the
// temporal types it is the declared fractional-seconds precision (0-6), or -1 when the
// precision is unspecified; callers map -1 to PostgreSQL's default of 6.
//
// It supplies the fractional-seconds precision of timestamp/timestamptz query results, which
// database/sql's ColumnType cannot report: the precision rides on the wire in the field
// description, and the text value drops trailing zeros, so neither the value nor the catalog
// (a result column need not map to one) recovers it. Reading the field description directly
// is why Bytebase needs no patched pgx driver to surface it.
//
// Call it on the same conn and after any SET that affects the statement, immediately before
// running that statement, so describe resolves names under the same search_path as execution.
// It returns nil, so the caller falls back to the default precision, when the connection is
// inside a transaction or the statement cannot be described; either way the statement itself
// still runs and reports its own errors.
func PgColumnTypeModifiers(ctx context.Context, conn *sql.Conn, statement string) []int32 {
	var modifiers []int32
	// The error is handled inside the callback so it never propagates out of Raw and marks
	// the pooled connection suspect; a describe failure only costs the precision refinement.
	_ = conn.Raw(func(driverConn any) error {
		pgxConn, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return nil
		}
		pgConn := pgxConn.Conn().PgConn()
		// Inside a transaction a describe that fails (say, a missing table) aborts it, and the
		// caller's own execution would then report "25P02 current transaction is aborted"
		// instead of the real error. Skip describe there; 'I' is the idle status, and outside a
		// transaction a failed describe leaves no trace, so the caller still reports the real
		// error.
		if pgConn.TxStatus() != 'I' {
			return nil
		}
		// An empty statement name describes the anonymous prepared statement, creating no
		// server-side prepared statement.
		sd, err := pgConn.Prepare(ctx, "", statement, nil)
		if err != nil {
			slog.Debug("failed to describe statement for column type modifiers", slog.String("error", err.Error()))
			return nil
		}
		modifiers = make([]int32, len(sd.Fields))
		for i, f := range sd.Fields {
			modifiers[i] = f.TypeModifier
		}
		return nil
	})
	return modifiers
}
