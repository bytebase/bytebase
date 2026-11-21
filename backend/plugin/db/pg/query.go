package pg

import (
	"context"
	"errors"
	"strings"

	omndriver "github.com/bytebase/omni/driver"
	pgdriver "github.com/bytebase/omni/pg/driver"
	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// QueryConn runs the script on the driver's session, one result per statement.
// Calls are serialized, as the stream's one pooled connection was.
func (d *Driver) QueryConn(ctx context.Context, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
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
		ExplainFormat:  strings.ToLower(util.OmniExplainFormat(queryContext.Option.GetExplainFormat(), "", v1pb.QueryOption_TEXT, v1pb.QueryOption_JSON, v1pb.QueryOption_XML, v1pb.QueryOption_YAML)),
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
	converted := util.OmniResultsToQueryResults(results, queryContext.MaximumSQLResultSize, queryContext.Explain)
	for i, result := range results {
		// The editor's result shape is the product's: a statement that reports no
		// count (ALTER TABLE, CREATE TABLE, SET) answers one "Affected Rows 0"
		// row, and the contract leaves the count nil for it.
		util.FillZeroAffectedRows(result, converted[i])
		if result.Err != nil {
			converted[i].DetailedError = postgresErrorDetail(result.Err)
		}
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
