package cassandra

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/cockroachdb/cockroachdb-parser/pkg/util/timeofday"
	"github.com/gocql/gocql"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/inf.v0"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func init() {
	db.Register(storepb.Engine_CASSANDRA, newDriver)
}

type Driver struct {
	config db.ConnectionConfig

	session *gocql.Session
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addrs := []string{
		formatAddress(config.DataSource.Host, config.DataSource.Port),
	}
	for _, addr := range config.DataSource.AdditionalAddresses {
		addrs = append(addrs, formatAddress(addr.Host, addr.Port))
	}
	cluster := gocql.NewCluster(addrs...)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: config.DataSource.Username,
		Password: config.Password,
	}
	cluster.Keyspace = config.ConnectionContext.DatabaseName

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session")
	}

	return &Driver{
		config:  config,
		session: session,
	}, nil
}

func (d *Driver) Close(context.Context) error {
	if d.session != nil {
		d.session.Close()
	}
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	var version string
	err := d.session.Query("SELECT release_version FROM system.local").WithContext(ctx).Scan(&version)
	if err != nil {
		return errors.Wrapf(err, "failed to ping")
	}
	return nil
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

func (*Driver) Execute(context.Context, string, db.ExecuteOptions) (int64, error) {
	return 0, status.Errorf(codes.Unimplemented, "Execute unimplemented")
}
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, rawStatement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmts, err := util.SanitizeSQL(rawStatement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split sql")
	}

	var results []*v1pb.QueryResult
	for _, stmt := range stmts {
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if _, _, err := base.ValidateSQLForEditor(storepb.Engine_CASSANDRA, stmt); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "support Cassandra SELECT statement only, err: %s", err.Error())
			}
			result := &v1pb.QueryResult{}
			pageSize := 0
			if queryContext.Limit > 0 {
				pageSize = queryContext.Limit
			}
			var pageState []byte
			for {
				nextPageState, err := func() ([]byte, error) {
					iter := d.session.Query(stmt).WithContext(ctx).PageSize(pageSize).PageState(pageState).Iter()
					defer iter.Close()
					nextPageState := iter.PageState()

					if len(result.ColumnNames) == 0 {
						for _, c := range iter.Columns() {
							result.ColumnNames = append(result.ColumnNames, c.Name)
							result.ColumnTypeNames = append(result.ColumnTypeNames, c.TypeInfo.Type().String())
						}
					}
					for {
						rowData, err := iter.RowData()
						if err != nil {
							return nil, errors.Wrap(err, "failed to fetch row data")
						}
						if !iter.Scan(rowData.Values...) {
							break
						}

						row := &v1pb.QueryRow{}
						for _, v := range rowData.Values {
							row.Values = append(row.Values, convertRowValue(v))
						}

						result.Rows = append(result.Rows, row)
						n := len(result.Rows)
						if (n&(n-1) == 0) && int64(proto.Size(result)) > queryContext.MaximumSQLResultSize {
							result.Error = common.FormatMaximumSQLResultSizeMessage(queryContext.MaximumSQLResultSize)
							break
						}

						if queryContext.Limit > 0 && queryContext.Limit == n {
							return nil, nil
						}
					}
					if err := iter.Close(); err != nil {
						return nil, errors.Wrapf(err, "iter close err")
					}

					return nextPageState, nil
				}()
				if err != nil {
					return nil, err
				}
				if len(nextPageState) == 0 {
					break
				}
				pageState = nextPageState
			}
			return result, nil
		}()
		stop := false
		if err != nil {
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}
		queryResult.Statement = stmt
		queryResult.Latency = durationpb.New(time.Since(startTime))
		queryResult.RowsCount = int64(len(queryResult.Rows))
		results = append(results, queryResult)
		if stop {
			break
		}
	}

	return results, nil
}

func convertRowValue(v any) *v1pb.RowValue {
	switch v := v.(type) {
	case *string:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: *v,
			},
		}
	case *int64:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_Int64Value{
				Int64Value: *v,
			},
		}
	case *time.Duration:
		if v == nil {
			return util.NullRowValue
		}
		// time of the day
		// e.g. '08:12:54'
		s, ns := v.Nanoseconds()/1_000_000_000, v.Nanoseconds()%1_000_000_000
		display := timeofday.FromTime(time.Unix(s, ns)).String()
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: display,
			},
		}
	case *time.Time:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_TimestampValue{
				TimestampValue: &v1pb.RowValue_Timestamp{
					GoogleTimestamp: timestamppb.New(*v),
					Accuracy:        3,
				},
			},
		}
	case *[]byte:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BytesValue{
				BytesValue: *v,
			},
		}
	case *bool:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BoolValue{
				BoolValue: *v,
			},
		}
	case *float32:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_FloatValue{
				FloatValue: *v,
			},
		}
	case *float64:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_DoubleValue{
				DoubleValue: *v,
			},
		}
	case *int:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_Int64Value{
				Int64Value: int64(*v),
			},
		}
	case *int16:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_Int32Value{
				Int32Value: int32(*v),
			},
		}
	case *int8:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_Int32Value{
				Int32Value: int32(*v),
			},
		}
	case *gocql.UUID:
		if v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: v.String(),
			},
		}
	case **inf.Dec:
		if v == nil || *v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: (*v).String(),
			},
		}
	case **big.Int:
		if v == nil || *v == nil {
			return util.NullRowValue
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: (*v).String(),
			},
		}
	case *gocql.Duration:
		if v == nil {
			return util.NullRowValue
		}
		display := time.Duration(v.Nanoseconds).String()
		if v.Days > 0 {
			display = fmt.Sprintf("%dd%s", v.Days, display)
		}
		if v.Months > 0 {
			display = fmt.Sprintf("%dmo%s", v.Months, display)
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: display,
			},
		}

	default:
		value, err := newValue(v)
		if err != nil {
			return &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{
					StringValue: fmt.Sprintf("failed to marshal value, err: %v", err),
				},
			}
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_ValueValue{
				ValueValue: value,
			},
		}
	}
}

func formatAddress(host, port string) string {
	if port == "" {
		return host
	}
	return host + ":" + port
}

func newValue(v any) (*structpb.Value, error) {
	switch v := v.(type) {
	case nil:
		return newNullValue(), nil
	case bool:
		return newBoolValue(v), nil
	case int:
		return newNumberValue(float64(v)), nil
	case int8:
		return newNumberValue(float64(v)), nil
	case int16:
		return newNumberValue(float64(v)), nil
	case int32:
		return newNumberValue(float64(v)), nil
	case int64:
		return newNumberValue(float64(v)), nil
	case uint:
		return newNumberValue(float64(v)), nil
	case uint8:
		return newNumberValue(float64(v)), nil
	case uint16:
		return newNumberValue(float64(v)), nil
	case uint32:
		return newNumberValue(float64(v)), nil
	case uint64:
		return newNumberValue(float64(v)), nil
	case float32:
		return newNumberValue(float64(v)), nil
	case float64:
		return newNumberValue(float64(v)), nil
	case json.Number:
		n, err := v.Float64()
		if err != nil {
			return nil, protoimpl.X.NewError("invalid number format %q, expected a float64: %v", v, err)
		}
		return newNumberValue(n), nil
	case string:
		if !utf8.ValidString(v) {
			return nil, protoimpl.X.NewError("invalid UTF-8 in string: %q", v)
		}
		return newStringValue(v), nil
	case []byte:
		s := base64.StdEncoding.EncodeToString(v)
		return newStringValue(s), nil
	case map[string]any:
		v2, err := newStruct(v)
		if err != nil {
			return nil, err
		}
		return newStructValue(v2), nil
	case []any:
		v2, err := newList(v)
		if err != nil {
			return nil, err
		}
		return newListValue(v2), nil
	default:
		if reflect.TypeOf(v).Kind() == reflect.Ptr {
			if reflect.ValueOf(v).IsNil() {
				return newNullValue(), nil
			}
			return newValue(reflect.ValueOf(v).Elem().Interface())
		}
		if reflect.TypeOf(v).Kind() == reflect.Slice || reflect.TypeOf(v).Kind() == reflect.Array {
			s := reflect.ValueOf(v)
			x := &structpb.ListValue{Values: make([]*structpb.Value, s.Len())}
			for i := 0; i < s.Len(); i++ {
				v2, err := newValue(s.Index(i).Interface())
				if err != nil {
					return nil, err
				}
				x.Values[i] = v2
			}
			return newListValue(x), nil
		}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			m := reflect.ValueOf(v)
			x := &structpb.Struct{Fields: make(map[string]*structpb.Value, m.Len())}
			for _, k := range m.MapKeys() {
				if !utf8.ValidString(k.String()) {
					return nil, protoimpl.X.NewError("invalid UTF-8 in string: %q", k.String())
				}
				v2, err := newValue(m.MapIndex(k).Interface())
				if err != nil {
					return nil, err
				}
				x.Fields[k.String()] = v2
			}
			return newStructValue(x), nil
		}
		return nil, protoimpl.X.NewError("invalid type: %T", v)
	}
}

func newList(v []any) (*structpb.ListValue, error) {
	x := &structpb.ListValue{Values: make([]*structpb.Value, len(v))}
	for i, v := range v {
		var err error
		x.Values[i], err = newValue(v)
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}

func newStruct(v map[string]any) (*structpb.Struct, error) {
	x := &structpb.Struct{Fields: make(map[string]*structpb.Value, len(v))}
	for k, v := range v {
		if !utf8.ValidString(k) {
			return nil, protoimpl.X.NewError("invalid UTF-8 in string: %q", k)
		}
		var err error
		x.Fields[k], err = newValue(v)
		if err != nil {
			return nil, err
		}
	}
	return x, nil
}

func newNullValue() *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
}

func newBoolValue(v bool) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: v}}
}

func newNumberValue(v float64) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: v}}
}

func newStringValue(v string) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: v}}
}

func newStructValue(v *structpb.Struct) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: v}}
}

func newListValue(v *structpb.ListValue) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: v}}
}
