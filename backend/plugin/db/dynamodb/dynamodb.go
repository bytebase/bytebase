// Package dynamodb is the plugin for DynamoDB driver.
package dynamodb

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_DYNAMODB, newDriver)
}

// Driver is the BigQuery driver.
type Driver struct {
	config    db.ConnectionConfig
	connCtx   db.ConnectionContext
	client    *dynamodb.Client
	awsConfig aws.Config
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a BigQuery driver. It must connect to a specific database.
// If database isn't provided, part of the driver cannot function.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, conf db.ConnectionConfig) (db.Driver, error) {
	d.config = conf
	d.connCtx = conf.ConnectionContext

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load AWS config")
	}
	d.awsConfig = cfg
	client := dynamodb.NewFromConfig(cfg)
	d.client = client
	return d, nil
}

// Close closes the driver.
func (*Driver) Close(_ context.Context) error {
	return nil
}

// Ping pings the instance.
func (d *Driver) Ping(ctx context.Context) error {
	// DynamoDB does not support ping method, we list tables instead. To avoid network overhead,
	// we set the limit to 1.
	var limit int32 = 1
	_, err := d.client.ListTables(ctx, &dynamodb.ListTablesInput{
		Limit: &limit,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to list dynamodb tables")
	}
	return nil
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes a SQL statement.
// Execute is usually used for DDL/DML statements without read-operation. The dynamodb driver supports three types of execute:
//
// BatchExecuteStatement: This operation allows you to perform batch reads or writes on data stored in DynamoDB, using PartiQL. Each read statement in a BatchExecuteStatement must specify an equality condition on all key attributes. This enforces that each SELECT statement in a batch returns at most a single item.
//
// ExecuteStatement: This operation allows you to perform reads and singleton writes on data stored in DynamoDB, using PartiQL.
// For PartiQL reads ( SELECT statement), if the total number of processed items exceeds the maximum dataset size limit of 1 MB, the read stops and results are returned to the user as a LastEvaluatedKey value to continue the read in a subsequent operation. If the filter criteria in WHERE clause does not match any data, the read will return an empty result set.
// A single SELECT statement response can return up to the maximum number of items (if using the Limit parameter) or a maximum of 1 MB of data (and then apply any filtering to the results using WHERE clause).
//
// ExecuteTransaction: This operation allows you to perform transactional reads or writes on data stored in DynamoDB, using PartiQL.
// The entire transaction must consist of either read statements or write statements, you cannot mix both in one transaction. The EXISTS function is an exception and can be used to check the condition of specific attributes of the item in a similar manner to ConditionCheck in the TransactWriteItems API.
//
// NOTE: Each api contains some constraints which do not be described in api docs. For example, in ExecuteTransaction, cannot include multiple operations on one item.
// So we use a simple solution here, use parser to split the statement and execute them one by one, unfortunately, we lose the transaction feature.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	statements, err := base.SplitMultiSQL(storepb.Engine_DYNAMODB, statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split multi statement")
	}
	nonEmptyStatements := base.FilterEmptyStatements(statements)

	for _, statement := range nonEmptyStatements {
		opts.LogCommandExecute(statement.Range, statement.Text)
		_, err := d.client.ExecuteTransaction(ctx, &dynamodb.ExecuteTransactionInput{
			TransactStatements: []types.ParameterizedStatement{
				{
					Statement: &statement.Text,
				},
			},
		})
		if err != nil {
			opts.LogCommandResponse(0, []int64{0}, err.Error())
			return 0, err
		}
		opts.LogCommandResponse(0, []int64{0}, "")
	}

	return 0, nil
}

// QueryConn queries a SQL statement in a given connection.
// The result.Rows.Values can be nil in DynamoDB, which means the column is not set in the row.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return nil, errors.New("DynamoDB does not support EXPLAIN")
	}

	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_DYNAMODB, statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split multi statement")
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		startTime := time.Now()
		result, err := d.querySinglePartiQL(ctx, singleSQL.Text, queryContext)
		stop := false
		if err != nil {
			result = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}
		result.Latency = durationpb.New(time.Since(startTime))
		result.Statement = statement
		result.RowsCount = int64(len(result.Rows))
		results = append(results, result)
		if stop {
			break
		}
	}
	return results, nil
}

type dynamodbQueryResultMeta struct {
	columnType string
	value      *v1pb.RowValue
}

func (d *Driver) querySinglePartiQL(ctx context.Context, statement string, queryContext db.QueryContext) (*v1pb.QueryResult, error) {
	result := &v1pb.QueryResult{}
	input := &dynamodb.ExecuteStatementInput{
		Statement: &statement,
	}
	if queryContext.Limit > 0 {
		limit := int32(queryContext.Limit)
		input.Limit = &limit
	}

	var nextToken *string
	rowMap := make(map[string][]*v1pb.RowValue)
	// TODO(zp): Our proto is not designed for NoSQL, whose data is not fixed. So we only use the last row to determine the column type.
	columnTypeMap := make(map[string]string)
	totalRowCount := 0
	for {
		input.NextToken = nextToken
		output, err := d.client.ExecuteStatement(ctx, input)
		if err != nil {
			return nil, err
		}
		for _, item := range output.Items {
			totalRowCount++
			meta := convertAttributeValueMapToRow(item)
			allKeySet := make(map[string]bool)
			for key := range rowMap {
				allKeySet[key] = true
			}
			curKeySet := make(map[string]*dynamodbQueryResultMeta, len(meta))
			for key, value := range meta {
				curKeySet[key] = value
				allKeySet[key] = true
			}
			for key := range allKeySet {
				_, inRowMap := rowMap[key]
				_, inCurKeySet := curKeySet[key]
				// 1. The key appears in the rowMap, and appears in the current row, we append the value to the row.
				if inRowMap && inCurKeySet {
					rowMap[key] = append(rowMap[key], curKeySet[key].value)
					columnTypeMap[key] = curKeySet[key].columnType
				}
				// 2.If the key appears in the row map, but does not appear in the current row, we append nil to the row.
				if inRowMap && !inCurKeySet {
					rowMap[key] = append(rowMap[key], nil)
				}
				// 3. If the key appears in the current row, but does not appear in the row map, it means that the current row has a new column, we should
				// backfill the previous rows with nil.
				if !inRowMap && inCurKeySet {
					for i := 0; i < totalRowCount-1; i++ {
						rowMap[key] = append(rowMap[key], nil)
					}
					rowMap[key] = append(rowMap[key], curKeySet[key].value)
					columnTypeMap[key] = curKeySet[key].columnType
				}
			}
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	sortedColumnNames := make([]string, 0, len(rowMap))
	for key := range rowMap {
		sortedColumnNames = append(sortedColumnNames, key)
	}
	slices.Sort(sortedColumnNames)
	columnTypes := make([]string, 0, len(sortedColumnNames))
	for _, key := range sortedColumnNames {
		columnTypes = append(columnTypes, columnTypeMap[key])
	}
	// Flatten the row map to rows.
	if len(rowMap) > 0 {
		for i := 0; i < totalRowCount; i++ {
			row := &v1pb.QueryRow{}
			for _, columnName := range sortedColumnNames {
				row.Values = append(row.Values, rowMap[columnName][i])
			}
			result.Rows = append(result.Rows, row)
		}
	}
	result.ColumnTypeNames = columnTypes
	result.ColumnNames = sortedColumnNames
	return result, nil
}

func convertAttributeValueMapToRow(items map[string]types.AttributeValue) map[string]*dynamodbQueryResultMeta {
	result := make(map[string]*dynamodbQueryResultMeta, len(items))
	var columnType string
	for item, attributeValue := range items {
		switch attributeValue.(type) {
		case *types.AttributeValueMemberB:
			columnType = "Binary"
		case *types.AttributeValueMemberBOOL:
			columnType = "Boolean"
		case *types.AttributeValueMemberBS:
			columnType = "BinarySet"
		case *types.AttributeValueMemberL:
			columnType = "List"
		case *types.AttributeValueMemberM:
			columnType = "Map"
		case *types.AttributeValueMemberN:
			columnType = "Number"
		case *types.AttributeValueMemberNS:
			columnType = "NumberSet"
		case *types.AttributeValueMemberNULL:
			columnType = "Null"
		case *types.AttributeValueMemberS:
			columnType = "String"
		case *types.AttributeValueMemberSS:
			columnType = "StringSet"
		default:
			columnType = "Unknown"
		}
		value, err := convertAttributeValueToRowValue(attributeValue)
		var r *v1pb.RowValue
		if err != nil {
			r = &v1pb.RowValue{
				Kind: &v1pb.RowValue_StringValue{StringValue: err.Error()},
			}
		} else {
			r = value
		}
		result[item] = &dynamodbQueryResultMeta{
			columnType: columnType,
			value:      r,
		}
	}
	return result
}

func convertAttributeValueToRowValue(attributeValue types.AttributeValue) (*v1pb.RowValue, error) {
	switch attributeValue := attributeValue.(type) {
	case *types.AttributeValueMemberB:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{StringValue: string(attributeValue.Value)},
		}, nil
	case *types.AttributeValueMemberBOOL:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_BoolValue{BoolValue: attributeValue.Value},
		}, nil
	case *types.AttributeValueMemberBS:
		a := convertAttributeValueToGoPrimitives(attributeValue)
		b, err := json.Marshal(a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal attribute value")
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(b),
			},
		}, nil
	case *types.AttributeValueMemberL:
		a := convertAttributeValueToGoPrimitives(attributeValue)
		b, err := json.Marshal(a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal attribute value")
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(b),
			},
		}, nil
	case *types.AttributeValueMemberM:
		a := convertAttributeValueToGoPrimitives(attributeValue)
		b, err := json.Marshal(a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal attribute value")
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(b),
			},
		}, nil
	case *types.AttributeValueMemberN:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: attributeValue.Value,
			},
		}, nil
	case *types.AttributeValueMemberNS:
		a := convertAttributeValueToGoPrimitives(attributeValue)
		b, err := json.Marshal(a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal attribute value")
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(b),
			},
		}, nil
	case *types.AttributeValueMemberNULL:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_NullValue{
				NullValue: structpb.NullValue_NULL_VALUE,
			},
		}, nil
	case *types.AttributeValueMemberS:
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: attributeValue.Value,
			},
		}, nil
	case *types.AttributeValueMemberSS:
		a := convertAttributeValueToGoPrimitives(attributeValue)
		b, err := json.Marshal(a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal attribute value")
		}
		return &v1pb.RowValue{
			Kind: &v1pb.RowValue_StringValue{
				StringValue: string(b),
			},
		}, nil
	}
	return nil, errors.Errorf("unsupported attribute value type: %T", attributeValue)
}

func convertAttributeValueToGoPrimitives(av types.AttributeValue) any {
	switch av := av.(type) {
	case *types.AttributeValueMemberB:
		return string(av.Value)
	case *types.AttributeValueMemberBOOL:
		return av.Value
	case *types.AttributeValueMemberBS:
		ss := make([]string, 0, len(av.Value))
		for _, b := range av.Value {
			ss = append(ss, string(b))
		}
		return ss
	case *types.AttributeValueMemberL:
		ss := make([]any, 0, len(av.Value))
		for _, v := range av.Value {
			ss = append(ss, convertAttributeValueToGoPrimitives(v))
		}
		return ss
	case *types.AttributeValueMemberM:
		m := make(map[string]any, len(av.Value))
		for k, v := range av.Value {
			m[k] = convertAttributeValueToGoPrimitives(v)
		}
		return m
	case *types.AttributeValueMemberN:
		return av.Value
	case *types.AttributeValueMemberNS:
		return av.Value
	case *types.AttributeValueMemberNULL:
		return nil
	case *types.AttributeValueMemberS:
		return av.Value
	case *types.AttributeValueMemberSS:
		return av.Value
	}
	return nil
}
