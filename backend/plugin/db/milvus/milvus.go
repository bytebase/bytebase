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

func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	stmts, err := base.SplitMultiSQL(storepb.Engine_MILVUS, statement)
	if err != nil {
		return 0, errors.Wrap(err, "failed to split statements")
	}
	stmts = base.FilterEmptyStatements(stmts)
	if len(stmts) == 0 {
		return 0, nil
	}

	const (
		defaultMaxRetries = 2
		baseBackoff       = 100 * time.Millisecond
	)

	maxRetries := opts.MaximumRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

	for _, stmt := range stmts {
		endpoint, payload, err := parseExecuteOperation(stmt.Text)
		if err != nil {
			return 0, err
		}

		attempts := maxRetries + 1
		for attempt := 0; attempt < attempts; attempt++ {
			_, callErr := d.callMilvus(ctx, endpoint, payload)
			if callErr == nil {
				break
			}
			if attempt == attempts-1 || !isRetryableMilvusError(callErr) {
				return 0, errors.Wrapf(callErr, "failed to execute milvus operation %q", strings.TrimSpace(stmt.Text))
			}
			backoff := baseBackoff * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return 0, errors.Wrap(ctx.Err(), "milvus operation retry canceled")
			case <-time.After(backoff):
			}
		}
	}

	return 0, nil
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

		partitions, err := d.listCollectionPartitions(ctx, collection)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list partitions for collection %q", collection)
		}
		aliases, err := d.listCollectionAliases(ctx, collection)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list aliases for collection %q", collection)
		}
		loadState, err := d.getCollectionLoadState(ctx, collection)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get load state for collection %q", collection)
		}
		table := &storepb.TableMetadata{
			Name:          collection,
			Columns:       parseCollectionColumns(resp),
			Indexes:       parseCollectionIndexes(resp),
			Partitions:    toTablePartitions(partitions),
			CreateOptions: buildCollectionCreateOptions(resp, aliases, loadState),
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

func (*Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *storepb.DatabaseSchemaMetadata) error {
	if dbMetadata == nil {
		return errors.New("database metadata is required")
	}

	var buf strings.Builder
	buf.WriteString("-- Milvus schema dump generated by Bytebase\n")
	if dbName := strings.TrimSpace(dbMetadata.GetName()); dbName != "" {
		buf.WriteString(fmt.Sprintf("-- Database: %s\n\n", dbName))
	}

	schema := &storepb.SchemaMetadata{}
	if len(dbMetadata.Schemas) > 0 && dbMetadata.Schemas[0] != nil {
		schema = dbMetadata.Schemas[0]
	}
	tables := slices.Clone(schema.GetTables())
	slices.SortFunc(tables, func(a, b *storepb.TableMetadata) int {
		return strings.Compare(a.GetName(), b.GetName())
	})

	for i, table := range tables {
		if err := ctx.Err(); err != nil {
			return err
		}
		if table == nil || strings.TrimSpace(table.GetName()) == "" {
			continue
		}

		createPayload := buildDumpCreateCollectionPayload(table)
		if createPayload == "" {
			buf.WriteString(fmt.Sprintf("create collection %s;\n", table.GetName()))
		} else {
			buf.WriteString(fmt.Sprintf("create collection %s with %s;\n", table.GetName(), createPayload))
		}

		indexes := slices.Clone(table.GetIndexes())
		slices.SortFunc(indexes, func(a, b *storepb.IndexMetadata) int {
			if cmp := strings.Compare(a.GetName(), b.GetName()); cmp != 0 {
				return cmp
			}
			return strings.Compare(strings.Join(a.GetExpressions(), ","), strings.Join(b.GetExpressions(), ","))
		})
		for _, index := range indexes {
			if index == nil {
				continue
			}
			fieldName := "vector"
			if len(index.Expressions) > 0 && strings.TrimSpace(index.Expressions[0]) != "" {
				fieldName = strings.TrimSpace(index.Expressions[0])
			}
			payload := indexMetadataToDumpPayload(index)
			if payload == "" {
				buf.WriteString(fmt.Sprintf("create index on %s field %s;\n", table.GetName(), fieldName))
				continue
			}
			buf.WriteString(fmt.Sprintf("create index on %s field %s with %s;\n", table.GetName(), fieldName, payload))
		}

		partitions := slices.Clone(table.GetPartitions())
		slices.SortFunc(partitions, func(a, b *storepb.TablePartitionMetadata) int {
			return strings.Compare(a.GetName(), b.GetName())
		})
		for _, partition := range partitions {
			if partition == nil || strings.TrimSpace(partition.GetName()) == "" {
				continue
			}
			buf.WriteString(fmt.Sprintf("create partition %s in %s;\n", partition.GetName(), table.GetName()))
		}

		for _, alias := range extractAliasesFromCreateOptions(table.GetCreateOptions()) {
			buf.WriteString(fmt.Sprintf("create alias %s for %s;\n", alias, table.GetName()))
		}
		if strings.EqualFold(extractLoadStateFromCreateOptions(table.GetCreateOptions()), "loaded") {
			buf.WriteString(fmt.Sprintf("load collection %s;\n", table.GetName()))
		}

		if i < len(tables)-1 {
			buf.WriteString("\n")
		}
	}

	if _, err := io.WriteString(out, buf.String()); err != nil {
		return errors.Wrap(err, "failed to write dump content")
	}
	return nil
}

var (
	showCollectionsRE = regexp.MustCompile(`(?i)^\s*show\s+collections\s*;?\s*$`)
	describeRE        = regexp.MustCompile(`(?i)^\s*desc(ribe)?\s+collection\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	selectRE          = regexp.MustCompile(`(?i)^\s*select\s+(.+?)\s+from\s+([A-Za-z0-9_]+)(?:\s+where\s+(.+?))?(?:\s+limit\s+([0-9]+))?\s*;?\s*$`)
	searchRE          = regexp.MustCompile(`(?is)^\s*search\s+([A-Za-z0-9_]+)\s+with\s+(\{.*\})\s*;?\s*$`)
	hybridSearchRE    = regexp.MustCompile(`(?is)^\s*hybrid\s+search\s+([A-Za-z0-9_]+)\s+with\s+(\{.*\})\s*;?\s*$`)

	createCollectionRE = regexp.MustCompile(`(?is)^\s*create\s+collection\s+([A-Za-z0-9_]+)(?:\s+with\s+(\{.*\}))?\s*;?\s*$`)
	dropCollectionRE   = regexp.MustCompile(`(?i)^\s*drop\s+collection\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	alterCollectionRE  = regexp.MustCompile(`(?is)^\s*alter\s+collection\s+([A-Za-z0-9_]+)\s+with\s+(\{.*\})\s*;?\s*$`)
	loadCollectionRE   = regexp.MustCompile(`(?i)^\s*load\s+collection\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	releaseCollectRE   = regexp.MustCompile(`(?i)^\s*release\s+collection\s+([A-Za-z0-9_]+)\s*;?\s*$`)

	createPartRE  = regexp.MustCompile(`(?i)^\s*create\s+partition\s+([A-Za-z0-9_]+)\s+in\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	dropPartRE    = regexp.MustCompile(`(?i)^\s*drop\s+partition\s+([A-Za-z0-9_]+)\s+in\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	loadPartRE    = regexp.MustCompile(`(?i)^\s*load\s+partition\s+([A-Za-z0-9_]+)\s+in\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	releasePartRE = regexp.MustCompile(`(?i)^\s*release\s+partition\s+([A-Za-z0-9_]+)\s+in\s+([A-Za-z0-9_]+)\s*;?\s*$`)

	createIndexRE = regexp.MustCompile(`(?is)^\s*create\s+index\s+on\s+([A-Za-z0-9_]+)\s+field\s+([A-Za-z0-9_]+)(?:\s+with\s+(\{.*\}))?\s*;?\s*$`)
	dropIndexRE   = regexp.MustCompile(`(?i)^\s*drop\s+index\s+on\s+([A-Za-z0-9_]+)(?:\s+name\s+([A-Za-z0-9_]+))?\s*;?\s*$`)
	alterIndexRE  = regexp.MustCompile(`(?is)^\s*alter\s+index\s+on\s+([A-Za-z0-9_]+)\s+name\s+([A-Za-z0-9_]+)\s+with\s+(\{.*\})\s*;?\s*$`)

	createAliasRE = regexp.MustCompile(`(?i)^\s*create\s+alias\s+([A-Za-z0-9_]+)\s+for\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	dropAliasRE   = regexp.MustCompile(`(?i)^\s*drop\s+alias\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	alterAliasRE  = regexp.MustCompile(`(?i)^\s*alter\s+alias\s+([A-Za-z0-9_]+)\s+to\s+([A-Za-z0-9_]+)\s*;?\s*$`)

	createUserRE = regexp.MustCompile(`(?is)^\s*create\s+user\s+([A-Za-z0-9_]+)\s+with\s+password\s+'([^']+)'\s*;?\s*$`)
	dropUserRE   = regexp.MustCompile(`(?i)^\s*drop\s+user\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	createRoleRE = regexp.MustCompile(`(?i)^\s*create\s+role\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	dropRoleRE   = regexp.MustCompile(`(?i)^\s*drop\s+role\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	grantRoleRE  = regexp.MustCompile(`(?i)^\s*grant\s+role\s+([A-Za-z0-9_]+)\s+to\s+user\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	revokeRoleRE = regexp.MustCompile(`(?i)^\s*revoke\s+role\s+([A-Za-z0-9_]+)\s+from\s+user\s+([A-Za-z0-9_]+)\s*;?\s*$`)
	insertDataRE = regexp.MustCompile(`(?is)^\s*insert\s+into\s+([A-Za-z0-9_]+)\s+values\s+(\[.*\]|\{.*\})\s*;?\s*$`)
	upsertDataRE = regexp.MustCompile(`(?is)^\s*upsert\s+into\s+([A-Za-z0-9_]+)\s+values\s+(\[.*\]|\{.*\})\s*;?\s*$`)
	deleteDataRE = regexp.MustCompile(`(?is)^\s*delete\s+from\s+([A-Za-z0-9_]+)\s+where\s+(.+?)\s*;?\s*$`)
)

func parseExecuteOperation(stmt string) (string, map[string]any, error) {
	if matches := createCollectionRE.FindStringSubmatch(stmt); len(matches) > 0 {
		payload := map[string]any{"collectionName": matches[1]}
		extra, err := parseJSONObject(matches[2])
		if err != nil {
			return "", nil, err
		}
		mergePayload(payload, extra)
		return "/v2/vectordb/collections/create", payload, nil
	}
	if matches := dropCollectionRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/collections/drop", map[string]any{"collectionName": matches[1]}, nil
	}
	if matches := alterCollectionRE.FindStringSubmatch(stmt); len(matches) > 0 {
		extra, err := parseJSONObject(matches[2])
		if err != nil {
			return "", nil, err
		}
		payload := map[string]any{"collectionName": matches[1]}
		mergePayload(payload, extra)
		return "/v2/vectordb/collections/alter", payload, nil
	}
	if matches := loadCollectionRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/collections/load", map[string]any{"collectionName": matches[1]}, nil
	}
	if matches := releaseCollectRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/collections/release", map[string]any{"collectionName": matches[1]}, nil
	}
	if matches := createPartRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/partitions/create", map[string]any{
			"collectionName": matches[2],
			"partitionName":  matches[1],
		}, nil
	}
	if matches := dropPartRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/partitions/drop", map[string]any{
			"collectionName": matches[2],
			"partitionName":  matches[1],
		}, nil
	}
	if matches := loadPartRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/partitions/load", map[string]any{
			"collectionName": matches[2],
			"partitionNames": []string{matches[1]},
		}, nil
	}
	if matches := releasePartRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/partitions/release", map[string]any{
			"collectionName": matches[2],
			"partitionNames": []string{matches[1]},
		}, nil
	}
	if matches := createIndexRE.FindStringSubmatch(stmt); len(matches) > 0 {
		extra, err := parseJSONObject(matches[3])
		if err != nil {
			return "", nil, err
		}
		indexParams, err := normalizeCreateIndexParams(matches[2], extra)
		if err != nil {
			return "", nil, err
		}
		payload := map[string]any{
			"collectionName": matches[1],
			"indexParams":    indexParams,
		}
		return "/v2/vectordb/indexes/create", payload, nil
	}
	if matches := dropIndexRE.FindStringSubmatch(stmt); len(matches) > 0 {
		payload := map[string]any{"collectionName": matches[1]}
		if matches[2] != "" {
			payload["indexName"] = matches[2]
		}
		return "/v2/vectordb/indexes/drop", payload, nil
	}
	if matches := alterIndexRE.FindStringSubmatch(stmt); len(matches) > 0 {
		payload := map[string]any{"collectionName": matches[1]}
		extra, err := parseJSONObject(matches[3])
		if err != nil {
			return "", nil, err
		}
		payload["indexName"] = matches[2]
		if indexParams, ok := extra["indexParams"]; ok {
			payload["indexParams"] = indexParams
			delete(extra, "indexParams")
		}
		if params, ok := extra["params"]; ok {
			payload["params"] = params
			delete(extra, "params")
		}
		if len(extra) > 0 {
			payload["params"] = extra
		}
		return "/v2/vectordb/indexes/alter", payload, nil
	}
	if matches := createAliasRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/aliases/create", map[string]any{
			"aliasName":      matches[1],
			"collectionName": matches[2],
		}, nil
	}
	if matches := dropAliasRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/aliases/drop", map[string]any{"aliasName": matches[1]}, nil
	}
	if matches := alterAliasRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/aliases/alter", map[string]any{
			"aliasName":      matches[1],
			"collectionName": matches[2],
		}, nil
	}
	if matches := createUserRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/users/create", map[string]any{
			"userName": matches[1],
			"password": matches[2],
		}, nil
	}
	if matches := dropUserRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/users/drop", map[string]any{"userName": matches[1]}, nil
	}
	if matches := createRoleRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/roles/create", map[string]any{"roleName": matches[1]}, nil
	}
	if matches := dropRoleRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/roles/drop", map[string]any{"roleName": matches[1]}, nil
	}
	if matches := grantRoleRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/users/grant_role", map[string]any{
			"roleName": matches[1],
			"userName": matches[2],
		}, nil
	}
	if matches := revokeRoleRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/users/revoke_role", map[string]any{
			"roleName": matches[1],
			"userName": matches[2],
		}, nil
	}
	if matches := insertDataRE.FindStringSubmatch(stmt); len(matches) > 0 {
		data, err := parseJSONArrayOrObject(matches[2])
		if err != nil {
			return "", nil, err
		}
		return "/v2/vectordb/entities/insert", map[string]any{
			"collectionName": matches[1],
			"data":           data,
		}, nil
	}
	if matches := upsertDataRE.FindStringSubmatch(stmt); len(matches) > 0 {
		data, err := parseJSONArrayOrObject(matches[2])
		if err != nil {
			return "", nil, err
		}
		return "/v2/vectordb/entities/upsert", map[string]any{
			"collectionName": matches[1],
			"data":           data,
		}, nil
	}
	if matches := deleteDataRE.FindStringSubmatch(stmt); len(matches) > 0 {
		return "/v2/vectordb/entities/delete", map[string]any{
			"collectionName": matches[1],
			"filter":         strings.TrimSpace(matches[2]),
		}, nil
	}

	return "", nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported milvus execute statement"))
}

func parseJSONObject(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid JSON payload"))
	}
	return payload, nil
}

func parseJSONArrayOrObject(raw string) ([]map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("data payload is required"))
	}

	if strings.HasPrefix(raw, "{") {
		var one map[string]any
		if err := json.Unmarshal([]byte(raw), &one); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid JSON payload"))
		}
		return []map[string]any{one}, nil
	}

	var many []map[string]any
	if err := json.Unmarshal([]byte(raw), &many); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid JSON payload"))
	}
	return many, nil
}

func normalizeCreateIndexParams(fieldName string, extra map[string]any) ([]map[string]any, error) {
	if extra == nil {
		return []map[string]any{{"fieldName": fieldName}}, nil
	}

	if raw, ok := extra["indexParams"]; ok {
		switch v := raw.(type) {
		case []any:
			params := make([]map[string]any, 0, len(v))
			for _, item := range v {
				obj, ok := item.(map[string]any)
				if !ok {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("indexParams entries must be JSON objects"))
				}
				fieldNameValue := strings.TrimSpace(fmt.Sprintf("%v", obj["fieldName"]))
				if fieldNameValue == "" || fieldNameValue == "<nil>" {
					obj["fieldName"] = fieldName
				}
				params = append(params, obj)
			}
			return params, nil
		case map[string]any:
			fieldNameValue := strings.TrimSpace(fmt.Sprintf("%v", v["fieldName"]))
			if fieldNameValue == "" || fieldNameValue == "<nil>" {
				v["fieldName"] = fieldName
			}
			return []map[string]any{v}, nil
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("indexParams must be an object or array of objects"))
		}
	}

	param := map[string]any{"fieldName": fieldName}
	for k, v := range extra {
		param[k] = v
	}
	return []map[string]any{param}, nil
}

func mergePayload(basePayload map[string]any, extra map[string]any) {
	for k, v := range extra {
		basePayload[k] = v
	}
}

func isRetryableMilvusError(err error) bool {
	code := connect.CodeOf(err)
	return code == connect.CodeUnavailable || code == connect.CodeDeadlineExceeded || code == connect.CodeResourceExhausted
}

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
	case searchRE.MatchString(stmt):
		matches := searchRE.FindStringSubmatch(stmt)
		payload, err := parseSearchPayload(matches[1], matches[2], limit)
		if err != nil {
			return nil, err
		}
		resp, err := d.callMilvus(ctx, "/v2/vectordb/entities/search", payload)
		if err != nil {
			return nil, err
		}
		return convertSearchResult(resp), nil
	case hybridSearchRE.MatchString(stmt):
		matches := hybridSearchRE.FindStringSubmatch(stmt)
		payload, err := parseHybridSearchPayload(matches[1], matches[2], limit)
		if err != nil {
			return nil, err
		}
		resp, err := d.callMilvus(ctx, "/v2/vectordb/entities/hybrid_search", payload)
		if err != nil {
			return nil, err
		}
		return convertSearchResult(resp), nil
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported milvus statement"))
	}
}

func parseSearchPayload(collectionName, rawJSON string, limit int) (map[string]any, error) {
	payload, err := parseJSONObject(rawJSON)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]any)
	}
	payload["collectionName"] = collectionName
	applyQueryLimit(payload, limit)
	return payload, nil
}

func parseHybridSearchPayload(collectionName, rawJSON string, limit int) (map[string]any, error) {
	payload, err := parseJSONObject(rawJSON)
	if err != nil {
		return nil, err
	}
	if payload == nil {
		payload = make(map[string]any)
	}
	payload["collectionName"] = collectionName
	applyQueryLimit(payload, limit)
	return payload, nil
}

func applyQueryLimit(payload map[string]any, limit int) {
	if _, ok := payload["limit"]; ok {
		return
	}
	if limit > 0 {
		payload["limit"] = limit
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
		return nil, connect.NewError(connect.CodeUnavailable, errors.Wrapf(err, "failed to call milvus endpoint %q", path))
	}
	defer resp.Body.Close()
	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response")
	}
	if resp.StatusCode >= 400 {
		return nil, connect.NewError(mapHTTPStatusToConnectCode(resp.StatusCode), errors.Errorf("milvus request to %q failed with status %d", path, resp.StatusCode))
	}
	var parsed map[string]any
	if err := json.Unmarshal(rawResp, &parsed); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to decode milvus response"))
	}
	if code, ok := parseMilvusCode(parsed["code"]); ok && code != 0 {
		message := strings.TrimSpace(fmt.Sprintf("%v", parsed["message"]))
		if message == "" || message == "<nil>" {
			message = fmt.Sprintf("milvus returned non-zero code %d", code)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("milvus endpoint %q returned code %d: %s", path, code, message))
	}
	return parsed, nil
}

func (d *Driver) callMilvusOptional(ctx context.Context, path string, payload map[string]any) (map[string]any, bool, error) {
	resp, err := d.callMilvus(ctx, path, payload)
	if err == nil {
		return resp, true, nil
	}
	switch connect.CodeOf(err) {
	case connect.CodeNotFound, connect.CodeUnimplemented:
		return nil, false, nil
	default:
		return nil, false, err
	}
}

func mapHTTPStatusToConnectCode(status int) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeAlreadyExists
	case http.StatusTooManyRequests:
		return connect.CodeResourceExhausted
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return connect.CodeDeadlineExceeded
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	default:
		if status >= 500 {
			return connect.CodeInternal
		}
		return connect.CodeUnknown
	}
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

func parseCollectionIndexes(resp map[string]any) []*storepb.IndexMetadata {
	data, ok := resp["data"].(map[string]any)
	if !ok {
		return nil
	}
	candidates := []any{}
	for _, key := range []string{"indexes", "indexDescriptions", "indexParams"} {
		if values, ok := data[key].([]any); ok {
			candidates = append(candidates, values...)
		}
	}
	if len(candidates) == 0 {
		return nil
	}

	indexes := make([]*storepb.IndexMetadata, 0, len(candidates))
	for _, item := range candidates {
		indexMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := firstNonEmpty(indexMap, "indexName", "name")
		fieldName := firstNonEmpty(indexMap, "fieldName", "field")
		indexType := firstNonEmpty(indexMap, "indexType", "type")

		definitionBytes, _ := json.Marshal(indexMap)
		index := &storepb.IndexMetadata{
			Name:       name,
			Type:       indexType,
			Definition: string(definitionBytes),
		}
		if fieldName != "" {
			index.Expressions = []string{fieldName}
		}
		indexes = append(indexes, index)
	}

	slices.SortFunc(indexes, func(a, b *storepb.IndexMetadata) int {
		if cmp := strings.Compare(a.GetName(), b.GetName()); cmp != 0 {
			return cmp
		}
		return strings.Compare(strings.Join(a.GetExpressions(), ","), strings.Join(b.GetExpressions(), ","))
	})
	return indexes
}

func toTablePartitions(names []string) []*storepb.TablePartitionMetadata {
	if len(names) == 0 {
		return nil
	}
	partitions := make([]*storepb.TablePartitionMetadata, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) == "" {
			continue
		}
		partitions = append(partitions, &storepb.TablePartitionMetadata{Name: name})
	}
	slices.SortFunc(partitions, func(a, b *storepb.TablePartitionMetadata) int {
		return strings.Compare(a.GetName(), b.GetName())
	})
	return partitions
}

func (d *Driver) listCollectionPartitions(ctx context.Context, collectionName string) ([]string, error) {
	payload := map[string]any{"collectionName": collectionName}
	for _, path := range []string{"/v2/vectordb/partitions/list", "/v2/vectordb/partitions/show"} {
		resp, ok, err := d.callMilvusOptional(ctx, path, payload)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		return extractPartitionNames(resp), nil
	}
	return nil, nil
}

func extractPartitionNames(resp map[string]any) []string {
	names := make([]string, 0)
	appendName := func(value any) {
		name := strings.TrimSpace(fmt.Sprintf("%v", value))
		if name != "" && name != "<nil>" {
			names = append(names, name)
		}
	}
	appendFromArray := func(items []any) {
		for _, item := range items {
			if entry, ok := item.(map[string]any); ok {
				appendName(firstNonEmpty(entry, "partitionName", "name"))
				continue
			}
			appendName(item)
		}
	}

	switch data := resp["data"].(type) {
	case []any:
		appendFromArray(data)
	case map[string]any:
		for _, key := range []string{"partitionNames", "partitions"} {
			if items, ok := data[key].([]any); ok {
				appendFromArray(items)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func (d *Driver) listCollectionAliases(ctx context.Context, collectionName string) ([]string, error) {
	for _, payload := range []map[string]any{{"collectionName": collectionName}, {}} {
		resp, ok, err := d.callMilvusOptional(ctx, "/v2/vectordb/aliases/list", payload)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		return extractAliases(resp, collectionName), nil
	}
	return nil, nil
}

func extractAliases(resp map[string]any, collectionName string) []string {
	aliases := make([]string, 0)
	appendAlias := func(name string) {
		name = strings.TrimSpace(name)
		if name != "" {
			aliases = append(aliases, name)
		}
	}
	appendFromArray := func(items []any) {
		for _, item := range items {
			switch value := item.(type) {
			case string:
				appendAlias(value)
			case map[string]any:
				boundCollection := firstNonEmpty(value, "collectionName", "collection")
				if boundCollection != "" && boundCollection != collectionName {
					continue
				}
				appendAlias(firstNonEmpty(value, "aliasName", "name", "alias"))
			}
		}
	}

	switch data := resp["data"].(type) {
	case []any:
		appendFromArray(data)
	case map[string]any:
		for _, key := range []string{"aliases", "aliasNames"} {
			if values, ok := data[key].([]any); ok {
				appendFromArray(values)
			}
		}
	}

	slices.Sort(aliases)
	return slices.Compact(aliases)
}

func (d *Driver) getCollectionLoadState(ctx context.Context, collectionName string) (string, error) {
	resp, ok, err := d.callMilvusOptional(ctx, "/v2/vectordb/collections/get_load_state", map[string]any{
		"collectionName": collectionName,
	})
	if err != nil || !ok {
		return "", err
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		return "", nil
	}
	state := firstNonEmpty(data, "state", "loadState", "status")
	return strings.TrimSpace(state), nil
}

func buildCollectionCreateOptions(resp map[string]any, aliases []string, loadState string) string {
	data, ok := resp["data"].(map[string]any)
	if !ok {
		return ""
	}
	options := make(map[string]any)
	for _, key := range []string{"description", "consistencyLevel", "enableDynamicField", "numShards", "shardsNum", "properties", "schema", "fields"} {
		if value, ok := data[key]; ok && value != nil {
			options[key] = value
		}
	}
	if len(aliases) > 0 {
		options["aliases"] = aliases
	}
	if loadState != "" {
		options["loadState"] = loadState
	}
	if len(options) == 0 {
		return ""
	}
	content, err := json.Marshal(options)
	if err != nil {
		return ""
	}
	return string(content)
}

func firstNonEmpty(data map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprintf("%v", data[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func buildDumpCreateCollectionPayload(table *storepb.TableMetadata) string {
	options := parseCreateOptions(table.GetCreateOptions())
	delete(options, "aliases")
	delete(options, "loadState")
	if _, ok := options["fields"]; !ok && len(table.GetColumns()) > 0 {
		fields := make([]map[string]any, 0, len(table.GetColumns()))
		for _, column := range table.GetColumns() {
			if column == nil || strings.TrimSpace(column.GetName()) == "" {
				continue
			}
			fields = append(fields, map[string]any{
				"name":     column.GetName(),
				"dataType": column.GetType(),
				"nullable": column.GetNullable(),
			})
		}
		if len(fields) > 0 {
			options["fields"] = fields
		}
	}
	if len(options) == 0 {
		return ""
	}
	content, err := json.Marshal(options)
	if err != nil {
		return ""
	}
	return string(content)
}

func indexMetadataToDumpPayload(index *storepb.IndexMetadata) string {
	payload := map[string]any{}
	if index.GetName() != "" {
		payload["indexName"] = index.GetName()
	}
	if index.GetType() != "" {
		payload["indexType"] = index.GetType()
	}
	if index.GetDefinition() != "" {
		var definition map[string]any
		if err := json.Unmarshal([]byte(index.GetDefinition()), &definition); err == nil {
			if params, ok := definition["params"]; ok {
				payload["params"] = params
			}
			if metricType, ok := definition["metricType"]; ok {
				payload["metricType"] = metricType
			}
		}
	}
	if len(payload) == 0 {
		return ""
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(content)
}

func parseCreateOptions(raw string) map[string]any {
	options := map[string]any{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return options
	}
	_ = json.Unmarshal([]byte(raw), &options)
	return options
}

func extractAliasesFromCreateOptions(raw string) []string {
	options := parseCreateOptions(raw)
	values, ok := options["aliases"].([]any)
	if !ok {
		return nil
	}
	aliases := make([]string, 0, len(values))
	for _, value := range values {
		name := strings.TrimSpace(fmt.Sprintf("%v", value))
		if name != "" && name != "<nil>" {
			aliases = append(aliases, name)
		}
	}
	slices.Sort(aliases)
	return slices.Compact(aliases)
}

func extractLoadStateFromCreateOptions(raw string) string {
	options := parseCreateOptions(raw)
	return strings.TrimSpace(fmt.Sprintf("%v", options["loadState"]))
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

func convertSearchResult(resp map[string]any) *v1pb.QueryResult {
	rows, ok := resp["data"].([]any)
	if !ok || len(rows) == 0 {
		return &v1pb.QueryResult{}
	}

	normalized := make([]any, 0, len(rows))
	for _, row := range rows {
		object, ok := row.(map[string]any)
		if !ok {
			continue
		}
		item := make(map[string]any, len(object)+1)
		for k, v := range object {
			item[k] = v
		}
		if _, hasDistance := item["distance"]; !hasDistance {
			if score, hasScore := item["score"]; hasScore {
				item["distance"] = score
			}
		}
		if _, hasScore := item["score"]; !hasScore {
			if distance, hasDistance := item["distance"]; hasDistance {
				item["score"] = distance
			}
		}
		normalized = append(normalized, item)
	}

	return convertQueryResult(map[string]any{"data": normalized})
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
