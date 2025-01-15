// Package cosmosdb is the plugin for CosmosDB driver.
package cosmosdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"sort"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_COSMOSDB, newDriver)
}

// Driver is the CosmosDB driver.
type Driver struct {
	client       *azcosmos.Client
	connCfg      db.ConnectionConfig
	databaseName string
}

func newDriver(_ db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a CosmosDB driver.
func (driver *Driver) Open(_ context.Context, _ storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	endpoint := connCfg.Host
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to found default Azure credential")
	}
	client, err := azcosmos.NewClient(endpoint, credential, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create CosmosDB client")
	}
	driver.client = client
	driver.databaseName = connCfg.Database
	driver.connCfg = connCfg
	return driver, nil
}

// Close closes the CosmosDB driver.
func (*Driver) Close(_ context.Context) error {
	return nil
}

// Ping pings the database.
func (driver *Driver) Ping(_ context.Context) error {
	queryPager := driver.client.NewQueryDatabasesPager("select 1", nil)
	for queryPager.More() {
		_, err := queryPager.NextPage(context.Background())
		if err != nil {
			// TODO(zp): Deserialize the error into azcore.ResponseError
			return errors.Wrapf(err, "failed to ping CosmosDB")
		}
	}
	return nil
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

func (*Driver) Execute(_ context.Context, _ string, _ db.ExecuteOptions) (int64, error) {
	return 0, status.Errorf(codes.Unimplemented, "method Execute unimplemented")
}

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Container == "" {
		return nil, status.Errorf(codes.InvalidArgument, "container argument is required for CosmosDB")
	}

	startTime := time.Now()
	container, err := driver.client.NewContainer(driver.databaseName, queryContext.Container)
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrapf(err, "failed to create container").Error())
	}

	var queryOption *azcosmos.QueryOptions
	// TODO(zp): Rewrite limit while parser supported.
	if queryContext.Limit > 0 && queryContext.Limit < 1000 {
		// Set the page size to limit to avoid large page size.
		queryOption = &azcosmos.QueryOptions{
			PageSizeHint: int32(queryContext.Limit),
		}
	}

	pager := container.NewCrossPartitionQueryItemsPager(statement, queryOption)
	var items [][]byte
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, errors.Wrapf(err, "failed to read more items").Error())
		}
		var stop bool
		for _, item := range response.Items {
			items = append(items, item)
			if queryContext.Limit > 0 && len(items) >= queryContext.Limit {
				stop = true
				break
			}
		}
		if stop {
			break
		}
	}

	var unmarshalledItems []map[string]any
	var illegal bool
	for _, item := range items {
		var m map[string]any
		if err := json.Unmarshal(item, &m); err != nil {
			slog.Warn("failed to unmarshal JSON", slog.String("item", string(item)), log.BBError(err))
			illegal = true
			break
		}
		unmarshalledItems = append(unmarshalledItems, m)
	}

	if illegal {
		var rows []*v1pb.QueryRow
		for _, item := range items {
			rows = append(rows, &v1pb.QueryRow{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_StringValue{StringValue: string(item)}},
				},
			})
		}
		return []*v1pb.QueryResult{
			{
				Rows: rows,
			},
		}, nil
	}

	columns, columnTypeMap, columnIndexMap := getColumns(unmarshalledItems)
	result := &v1pb.QueryResult{
		ColumnNames: columns,
	}
	for _, column := range columns {
		result.ColumnTypeNames = append(result.ColumnTypeNames, columnTypeMap[column])
	}

	for _, m := range unmarshalledItems {
		values := make([]*v1pb.RowValue, len(columns))
		for k, v := range m {
			switch v := v.(type) {
			case string:
				values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: v}}
			case float64:
				// Decide the target type for float64
				if v == float64(int32(v)) {
					values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(v)}}
				} else if v == float64(int64(v)) {
					values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: int64(v)}}
				} else if v >= 0 && v == float64(uint32(v)) {
					values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(v)}}
				} else if v >= 0 && v == float64(uint64(v)) {
					values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: uint64(v)}}
				} else {
					// Default to DoubleValue if it's not an integer type
					values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: v}}
				}
			case bool:
				values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: v}}
			case map[string]any:
				// Handle nested objects if necessary
				// Convert to JSON string representation for example
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					slog.Warn("failed to marshal JSON", slog.Any("object", v), log.BBError(err))
					illegal = true
					break
				}
				values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(jsonBytes)}}
			case []any:
				// Handle arrays if necessary
				// Convert to JSON string representation for example
				jsonBytes, err := json.Marshal(v)
				if err != nil {
					slog.Warn("failed to marshal JSON", slog.Any("array", v), log.BBError(err))
					illegal = true
					break
				}
				values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(jsonBytes)}}
			case nil:
				values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{}}
			default:
				// Handle unknown types
				values[columnIndexMap[k]] = &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "unknow"}}
			}
			for i := 0; i < len(values); i++ {
				if values[i] == nil {
					values[i] = &v1pb.RowValue{Kind: &v1pb.RowValue_NullValue{}}
				}
			}
		}
		if illegal {
			break
		}
		result.Rows = append(result.Rows, &v1pb.QueryRow{
			Values: values,
		})
	}
	result.Latency = durationpb.New(time.Since(startTime))

	if illegal {
		var rows []*v1pb.QueryRow
		for _, item := range items {
			rows = append(rows, &v1pb.QueryRow{
				Values: []*v1pb.RowValue{
					{Kind: &v1pb.RowValue_StringValue{StringValue: string(item)}},
				},
			})
		}
		return []*v1pb.QueryResult{
			{
				Rows: rows,
			},
		}, nil
	}

	return []*v1pb.QueryResult{result}, nil
}

func getColumns(unmarshalledItems []map[string]any) (columnNames []string, columnTypes map[string]string, columnIndexMap map[string]int) {
	columnNamesSet := make(map[string]bool)
	columnTypes = make(map[string]string)

	for _, m := range unmarshalledItems {
		for k, v := range m {
			if _, ok := columnNamesSet[k]; ok {
				continue
			}
			columnNamesSet[k] = true
			columnTypes[k] = getType(v)
		}
	}
	columnNames, columnIndexMap = getOrderedColumns(columnNamesSet)
	return columnNames, columnTypes, columnIndexMap
}

func getOrderedColumns(columnSet map[string]bool) ([]string, map[string]int) {
	var columns []string
	for k := range columnSet {
		columns = append(columns, k)
	}
	// Put built-in columns at the end.
	builtInColumns := map[string]bool{
		"_rid":         true,
		"_self":        true,
		"_etag":        true,
		"_attachments": true,
		"_ts":          true,
	}
	// TODO(zp): Put id and parititon key columns at the front.
	sort.SliceStable(columns, func(i, j int) bool {
		// "id" should come first
		if columns[i] == "id" {
			return true
		}
		if columns[j] == "id" {
			return false
		}

		// Built-in columns should come last
		_, isIBuiltIn := builtInColumns[columns[i]]
		_, isJBuiltIn := builtInColumns[columns[j]]

		if isIBuiltIn && !isJBuiltIn {
			return false
		}
		if !isIBuiltIn && isJBuiltIn {
			return true
		}

		// Otherwise, sort lexicographically
		return columns[i] < columns[j]
	})
	columnIndexMap := make(map[string]int)

	for i, column := range columns {
		columnIndexMap[column] = i
	}
	return columns, columnIndexMap
}

func getType(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number" // JSON numbers are unmarshalled as float64
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}
