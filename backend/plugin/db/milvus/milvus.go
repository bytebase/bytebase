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

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, _ := d.fetchVersion(ctx)

	collections, err := d.listCollections(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sync milvus collections")
	}

	tables := make([]*storepb.TableMetadata, 0, len(collections))
	for _, collection := range collections {
		tables = append(tables, &storepb.TableMetadata{Name: collection})
	}

	return &db.InstanceMetadata{
		Version: version,
		Databases: []*storepb.DatabaseSchemaMetadata{
			{
				Name: d.databaseName(),
				Schemas: []*storepb.SchemaMetadata{
					{
						Tables: tables,
					},
				},
			},
		},
	}, nil
}

func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	collections, err := d.listCollections(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list milvus collections")
	}

	tables := make([]*storepb.TableMetadata, 0, len(collections))
	for _, collection := range collections {
		resp, err := d.callMilvus(ctx, "/v2/vectordb/collections/describe", map[string]any{
			"collectionName": collection,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to describe collection %q", collection)
		}
		table := &storepb.TableMetadata{
			Name:    collection,
			Columns: parseCollectionColumns(resp),
		}
		tables = append(tables, table)
	}

	return &storepb.DatabaseSchemaMetadata{
		Name: d.databaseName(),
		Schemas: []*storepb.SchemaMetadata{
			{
				Tables: tables,
			},
		},
	}, nil
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
	if code, ok := parseMilvusCode(parsed["code"]); ok && code != 0 {
		message := strings.TrimSpace(fmt.Sprintf("%v", parsed["message"]))
		if message == "" || message == "<nil>" {
			message = fmt.Sprintf("milvus returned non-zero code %d", code)
		}
		return nil, connect.NewError(connect.CodeInternal, errors.New(message))
	}
	return parsed, nil
}

func parseMilvusCode(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		n, err := strconv.Atoi(v.String())
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func (d *Driver) databaseName() string {
	if strings.TrimSpace(d.config.ConnectionContext.DatabaseName) != "" {
		return d.config.ConnectionContext.DatabaseName
	}
	return "default"
}

func (d *Driver) listCollections(ctx context.Context) ([]string, error) {
	resp, err := d.callMilvus(ctx, "/v2/vectordb/collections/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	return extractCollectionNames(resp), nil
}

func (d *Driver) fetchVersion(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, d.baseURL+"/version", nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create version request")
	}
	if d.token != "" {
		request.Header.Set("Authorization", "Bearer "+d.token)
	}
	resp, err := d.httpClient.Do(request)
	if err != nil {
		return "", errors.Wrap(err, "failed to call milvus version endpoint")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read version response")
	}
	if resp.StatusCode >= 400 {
		return "", errors.Errorf("milvus version request failed with status %d", resp.StatusCode)
	}
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return "", errors.New("empty version response")
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return bodyText, nil
	}
	version := extractVersion(payload)
	if version == "" {
		return "", errors.New("version field not found in response")
	}
	return version, nil
}

func extractVersion(v any) string {
	switch obj := v.(type) {
	case map[string]any:
		for _, key := range []string{"version", "buildVersion", "gitVersion"} {
			if raw, ok := obj[key]; ok {
				if version, ok := raw.(string); ok && strings.TrimSpace(version) != "" {
					return version
				}
			}
		}
		for _, key := range []string{"data", "result"} {
			if nested, ok := obj[key]; ok {
				if version := extractVersion(nested); version != "" {
					return version
				}
			}
		}
	case []any:
		for _, item := range obj {
			if version := extractVersion(item); version != "" {
				return version
			}
		}
	}
	return ""
}

func extractCollectionNames(resp map[string]any) []string {
	var collections []string
	appendNames := func(items []any) {
		for _, item := range items {
			name := strings.TrimSpace(fmt.Sprintf("%v", item))
			if name != "" && name != "<nil>" {
				collections = append(collections, name)
			}
		}
	}

	switch data := resp["data"].(type) {
	case []any:
		appendNames(data)
	case map[string]any:
		if values, ok := data["collections"].([]any); ok {
			appendNames(values)
		}
		if values, ok := data["collectionNames"].([]any); ok {
			appendNames(values)
		}
	}

	slices.Sort(collections)
	return slices.Compact(collections)
}

func parseCollectionColumns(resp map[string]any) []*storepb.ColumnMetadata {
	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil
	}
	fields, ok := data["fields"].([]any)
	if !ok {
		if schema, ok := data["schema"].(map[string]any); ok {
			fields, _ = schema["fields"].([]any)
		}
	}
	if len(fields) == 0 {
		return nil
	}

	columns := make([]*storepb.ColumnMetadata, 0, len(fields))
	for i, field := range fields {
		fieldMap, ok := field.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", fieldMap["name"]))
		if name == "" || name == "<nil>" {
			continue
		}
		columnType := strings.TrimSpace(fmt.Sprintf("%v", fieldMap["dataType"]))
		if columnType == "" || columnType == "<nil>" {
			columnType = strings.TrimSpace(fmt.Sprintf("%v", fieldMap["type"]))
		}
		if columnType == "" || columnType == "<nil>" {
			columnType = "UNKNOWN"
		}
		nullable := true
		switch v := fieldMap["nullable"].(type) {
		case bool:
			nullable = v
		}
		columns = append(columns, &storepb.ColumnMetadata{
			Name:     name,
			Position: int32(i + 1),
			Type:     columnType,
			Nullable: nullable,
		})
	}
	return columns
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
