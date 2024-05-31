// Package dynamodb is the plugin for DynamoDB driver.
package dynamodb

import (
	"context"
	"database/sql"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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

func newDriver(_ db.DriverConfig) db.Driver {
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

// GetType returns the database type.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_DYNAMODB
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
	statements, err := base.SplitMultiSQL(d.GetType(), statement)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to split multi statement")
	}
	nonEmptyStatements, idxMap := base.FilterEmptySQLWithIndexes(statements)
	commandsTotal := len(nonEmptyStatements)

	for currentIndex, statement := range nonEmptyStatements {
		if opts.UpdateExecutionStatus != nil {
			opts.UpdateExecutionStatus(&v1pb.TaskRun_ExecutionDetail{
				CommandsTotal:     int32(commandsTotal),
				CommandsCompleted: int32(idxMap[currentIndex]),
				CommandStartPosition: &v1pb.TaskRun_ExecutionDetail_Position{
					Line:   int32(statement.FirstStatementLine),
					Column: int32(statement.FirstStatementColumn),
				},
				CommandEndPosition: &v1pb.TaskRun_ExecutionDetail_Position{
					Line:   int32(statement.LastLine),
					Column: int32(statement.LastColumn),
				},
			})
		}
		opts.LogCommandExecute([]int32{int32(idxMap[currentIndex])})
		_, err := d.client.ExecuteTransaction(ctx, &dynamodb.ExecuteTransactionInput{
			TransactStatements: []types.ParameterizedStatement{
				{
					Statement: &statement.Text,
				},
			},
		})
		if err != nil {
			return 0, &db.ErrorWithPosition{
				Err: errors.Wrapf(err, "failed to execute statement: %s", statement.Text),
				Start: &storepb.TaskRunResult_Position{
					Line:   int32(statement.FirstStatementLine),
					Column: int32(statement.FirstStatementColumn),
				},
				End: &storepb.TaskRunResult_Position{
					Line:   int32(statement.LastLine),
					Column: int32(statement.LastColumn),
				},
			}
		}
		opts.LogCommandResponse([]int32{int32(currentIndex)}, 0, []int32{}, "")
	}

	return 0, nil
}

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	panic("implement me")
}

// RunStatement executes a SQL statement.
func (*Driver) RunStatement(_ context.Context, _ *sql.Conn, _ string) ([]*v1pb.QueryResult, error) {
	panic("implement me")
}
