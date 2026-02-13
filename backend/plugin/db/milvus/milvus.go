package milvus

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	db.Register(storepb.Engine_MILVUS, newDriver)
}

type Driver struct {
	config     db.ConnectionConfig
	httpClient *http.Client
	baseURL    string
	token      string
}

func newDriver() db.Driver {
	return &Driver{}
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	if config.DataSource == nil {
		return nil, errors.New("data source is required")
	}
	scheme := "http"
	if config.DataSource.GetUseSsl() {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%s", scheme, config.DataSource.Host, config.DataSource.Port)
	token := ""
	if config.DataSource.Username != "" || config.Password != "" {
		token = fmt.Sprintf("%s:%s", config.DataSource.Username, config.Password)
	}
	return &Driver{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		token:      token,
	}, nil
}

func (*Driver) Close(context.Context) error {
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	_, err := d.callMilvus(ctx, "/v2/vectordb/collections/list", map[string]any{})
	if err != nil {
		return errors.Wrap(err, "failed to ping milvus")
	}
	return nil
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

func (*Driver) Execute(context.Context, string, db.ExecuteOptions) (int64, error) {
	return 0, errors.New("milvus driver is not implemented")
}

func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("milvus does not support EXPLAIN"))
	}
	_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_MILVUS, statement)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "milvus supports readonly query statements only"))
	}
	if !allQuery {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("milvus supports readonly query statements only"))
	}

	stmts, err := base.SplitMultiSQL(storepb.Engine_MILVUS, statement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split statements")
	}
	stmts = base.FilterEmptyStatements(stmts)
	if len(stmts) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		startTime := time.Now()
		queryResult, qErr := d.executeStatement(ctx, stmt.Text, queryContext.Limit)
		if qErr != nil {
			queryResult = &v1pb.QueryResult{Error: qErr.Error()}
		}
		queryResult.Latency = durationpb.New(time.Since(startTime))
		queryResult.Statement = stmt.Text
		queryResult.RowsCount = int64(len(queryResult.Rows))
		results = append(results, queryResult)
		if qErr != nil {
			break
		}
	}
	return results, nil
}

func (*Driver) SyncInstance(context.Context) (*db.InstanceMetadata, error) {
	return nil, errors.New("milvus driver is not implemented")
}

func (*Driver) SyncDBSchema(context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, errors.New("milvus driver is not implemented")
}

func (*Driver) Dump(context.Context, io.Writer, *storepb.DatabaseSchemaMetadata) error {
	return errors.New("milvus driver is not implemented")
}

var (
	showCollectionsRE = regexp.MustCompile(`(?i)^\s*show\s+collections\s*;?\s*$`)
	describeRE        = regexp.MustCompile(`(?i)^\s*desc(ribe)?\s+collection\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	selectRE          = regexp.MustCompile(`(?i)^\s*select\s+(.+?)\s+from\s+([A-Za-z0-9_]+)(?:\s+where\s+(.+?))?(?:\s+limit\s+([0-9]+))?\s*;?\s*$`)
)

func (d *Driver) executeStatement(ctx context.Context, stmt string, limit int) (*v1pb.QueryResult, error) {
	switch {
	case showCollectionsRE.MatchString(stmt):
		resp, err := d.callMilvus(ctx, "/v2/vectordb/collections/list", map[string]any{})
		if err != nil {
			return nil, err
		}
		return convertListCollectionsResult(resp), nil
	case describeRE.MatchString(stmt):
		matches := describeRE.FindStringSubmatch(stmt)
		resp, err := d.callMilvus(ctx, "/v2/vectordb/collections/describe", map[string]any{
			"collectionName": matches[2],
		})
		if err != nil {
			return nil, err
		}
		return convertDescribeResult(resp), nil
	case selectRE.MatchString(stmt):
		matches := selectRE.FindStringSubmatch(stmt)
		fields := parseSelectFields(matches[1])
		collection := matches[2]
		filter := strings.TrimSpace(matches[3])
		finalLimit := limit
		if matches[4] != "" {
			value, err := strconv.Atoi(matches[4])
			if err != nil {
				return nil, errors.Wrap(err, "invalid LIMIT value")
			}
			finalLimit = value
		}
		if finalLimit <= 0 {
			finalLimit = 100
		}
		payload := map[string]any{
			"collectionName": collection,
			"outputFields":   fields,
			"limit":          finalLimit,
		}
		if filter != "" {
			payload["filter"] = filter
		}
		resp, err := d.callMilvus(ctx, "/v2/vectordb/entities/query", payload)
		if err != nil {
			return nil, err
		}
		return convertQueryResult(resp), nil
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported milvus statement"))
	}
}

func (d *Driver) callMilvus(ctx context.Context, path string, payload map[string]any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode request")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	request.Header.Set("Content-Type", "application/json")
	if d.token != "" {
		request.Header.Set("Authorization", "Bearer "+d.token)
	}
	resp, err := d.httpClient.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call milvus")
	}
	defer resp.Body.Close()
	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}
	if resp.StatusCode >= 400 {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("milvus request failed with status %d", resp.StatusCode))
	}
	var parsed map[string]any
	if err := json.Unmarshal(rawResp, &parsed); err != nil {
		return nil, errors.Wrap(err, "failed to decode milvus response")
	}
	if code, ok := parsed["code"].(float64); ok && int(code) != 0 {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("milvus returned non-zero code %d", int(code)))
	}
	return parsed, nil
}

func parseSelectFields(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "*" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		field := strings.TrimSpace(part)
		if field != "" {
			fields = append(fields, field)
		}
	}
	if len(fields) == 0 {
		return []string{"*"}
	}
	return fields
}

func convertListCollectionsResult(resp map[string]any) *v1pb.QueryResult {
	result := &v1pb.QueryResult{
		ColumnNames:     []string{"collection"},
		ColumnTypeNames: []string{"TEXT"},
	}
	collections, _ := resp["data"].([]any)
	for _, collection := range collections {
		result.Rows = append(result.Rows, &v1pb.QueryRow{
			Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", collection)}},
			},
		})
	}
	return result
}

func convertDescribeResult(resp map[string]any) *v1pb.QueryResult {
	result := &v1pb.QueryResult{
		ColumnNames:     []string{"field", "value"},
		ColumnTypeNames: []string{"TEXT", "TEXT"},
	}
	obj, ok := resp["data"].(map[string]any)
	if !ok {
		return result
	}
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		value, _ := json.Marshal(obj[key])
		result.Rows = append(result.Rows, &v1pb.QueryRow{
			Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: key}},
				{Kind: &v1pb.RowValue_StringValue{StringValue: string(value)}},
			},
		})
	}
	return result
}

func convertQueryResult(resp map[string]any) *v1pb.QueryResult {
	result := &v1pb.QueryResult{}
	rows, ok := resp["data"].([]any)
	if !ok || len(rows) == 0 {
		return result
	}

	colSet := make(map[string]struct{})
	for _, row := range rows {
		if object, ok := row.(map[string]any); ok {
			for key := range object {
				colSet[key] = struct{}{}
			}
		}
	}
	columns := make([]string, 0, len(colSet))
	for key := range colSet {
		columns = append(columns, key)
	}
	slices.Sort(columns)
	result.ColumnNames = columns
	result.ColumnTypeNames = make([]string, len(columns))
	for i := range columns {
		result.ColumnTypeNames[i] = "TEXT"
	}

	for _, row := range rows {
		object, ok := row.(map[string]any)
		if !ok {
			continue
		}
		queryRow := &v1pb.QueryRow{}
		for _, column := range columns {
			queryRow.Values = append(queryRow.Values, toRowValue(object[column]))
		}
		result.Rows = append(result.Rows, queryRow)
	}
	return result
}

func toRowValue(v any) *v1pb.RowValue {
	switch value := v.(type) {
	case nil:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value}}
	case float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value}}
	case string:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value}}
	default:
		data, _ := json.Marshal(value)
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(data)}}
	}
}
