package pg

import (
	"context"
	"database/sql"
	"errors"

	omndriver "github.com/bytebase/omni/driver"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement.
// The caller's *sql.Conn is unused: the contract's session is the driver's, not a
// *sql.DB's, and it is the one a statement's settings hold on. Calls are
// serialized, as the stream's one pooled connection was.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	d.sessionMu.Lock()
	defer d.sessionMu.Unlock()
	session, err := d.querySession(ctx)
	if err != nil {
		return nil, err
	}

	results, err := session.Query(ctx, statement, omndriver.QueryOptions{
		Schema:         queryContext.Schema,
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  explainFormatName(queryContext.Option.GetExplainFormat()),
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	})
	if err != nil {
		// The script could not be run at all; the session is not one to keep.
		_ = session.Close()
		d.session = nil
		return nil, err
	}
	if ctx.Err() != nil {
		_ = session.Close()
		d.session = nil
	}

	timestampsBeyondProtoRangeToText(results)
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		queryResult := util.OmniResultToQueryResult(result, queryContext.MaximumSQLResultSize)
		// The editor's result shape is the product's: a statement that reports no
		// count (ALTER TABLE, CREATE TABLE, SET) answers one "Affected Rows 0"
		// row, and the contract leaves the count nil for it.
		if len(result.Columns) == 0 && result.AffectedRows == nil && result.Err == nil {
			affected := util.BuildAffectedRowsResult(0, queryResult.Messages)
			queryResult.ColumnNames = affected.ColumnNames
			queryResult.ColumnTypeNames = affected.ColumnTypeNames
			queryResult.Rows = affected.Rows
			queryResult.RowsCount = 1
		}
		if result.Err != nil {
			queryResult.DetailedError = postgresErrorDetail(result.Err)
		}
		// An explain request asked for one plan, in the format it named; the
		// driver describes the statement it rewrote, which the caller already
		// knows, and the request's format replaces the caller's own EXPLAIN.
		if queryContext.Explain {
			queryResult.QueryPlan = nil
		}
		converted = append(converted, queryResult)
	}
	return converted, nil
}

// querySession returns the driver's session, opening it on first use. The caller
// holds sessionMu.
func (d *Driver) querySession(ctx context.Context) (omndriver.Session, error) {
	if d.session != nil {
		return d.session, nil
	}
	session, err := d.impl.Session(ctx)
	if err != nil {
		return nil, err
	}
	d.session = session
	return session, nil
}

// explainFormatName is the plan format the contract's QueryOptions names.
func explainFormatName(format v1pb.QueryOption_ExplainFormat) string {
	switch format {
	case v1pb.QueryOption_TEXT:
		return "text"
	case v1pb.QueryOption_JSON:
		return "json"
	case v1pb.QueryOption_XML:
		return "xml"
	case v1pb.QueryOption_YAML:
		return "yaml"
	default:
		return ""
	}
}

// timestampsBeyondProtoRangeToText rewrites a timestamp cell PostgreSQL allows
// and google.protobuf.Timestamp does not (it holds 0001-01-01 to 9999-12-31;
// PostgreSQL holds 4713 BC to 294276 AD) as the text PostgreSQL writes, which is
// what the conversion before the contract's answered. The protobuf value cannot
// carry it, and marshaling the response would fail.
func timestampsBeyondProtoRangeToText(results []*omndriver.Result) {
	for _, result := range results {
		for i, row := range result.Rows {
			for j, cell := range row {
				if cell.Kind != omndriver.Timestamp && cell.Kind != omndriver.TimestampTZ {
					continue
				}
				if err := timestamppb.New(cell.Time).CheckValid(); err == nil {
					continue
				}
				result.Rows[i][j] = omndriver.Value{
					Kind:   omndriver.String,
					String: pgdriver.FormatTimestamp(cell.Time, cell.Kind == omndriver.TimestampTZ),
				}
			}
		}
	}
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
