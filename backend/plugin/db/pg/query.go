package pg

import (
	"context"
	"errors"
	"strings"

	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/jackc/pgx/v5/pgconn"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement.
// Calls are serialized, as the stream's one pooled connection was.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	explainFormat := strings.ToLower(util.OmniExplainFormat(
		queryContext.Option.GetExplainFormat(),
		"",
		v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON, v1pb.QueryOption_XML, v1pb.QueryOption_YAML,
	))
	results, err := d.QueryHeldSession(ctx, statement, util.OmniQueryOptions(queryContext, true, explainFormat), util.SessionQueryOptions{})
	if err != nil {
		return nil, err
	}

	converted := util.OmniResultsToQueryResultsWith(results, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		ClearExplainPlan:          queryContext.Explain,
		FillZeroAffectedRows:      true,
		FormatOutOfRangeTimestamp: pgdriver.FormatTimestamp,
	})
	for i, result := range results {
		if result.Err != nil {
			converted[i].DetailedError = postgresErrorDetail(result.Err)
		}
	}
	return converted, nil
}

// postgresErrorDetail is the server's error fields for a failed statement, for a
// client that shows them beside the message. The contract's result carries the
// error pgx returned, which is the server's *pgconn.PgError.
func postgresErrorDetail(err error) *v1pb.QueryResult_PostgresError_ {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	return &v1pb.QueryResult_PostgresError_{
		PostgresError: &v1pb.QueryResult_PostgresError{
			Severity:         pgErr.Severity,
			Code:             pgErr.Code,
			Message:          pgErr.Message,
			Detail:           pgErr.Detail,
			Hint:             pgErr.Hint,
			Position:         pgErr.Position,
			InternalPosition: pgErr.InternalPosition,
			InternalQuery:    pgErr.InternalQuery,
			Where:            pgErr.Where,
			SchemaName:       pgErr.SchemaName,
			TableName:        pgErr.TableName,
			ColumnName:       pgErr.ColumnName,
			DataTypeName:     pgErr.DataTypeName,
			ConstraintName:   pgErr.ConstraintName,
			File:             pgErr.File,
			Line:             pgErr.Line,
			Routine:          pgErr.Routine,
		},
	}
}
