// Package mongodb is the plugin for MongoDB driver.
package mongodb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/bytebase/gomongo"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mongodbparser "github.com/bytebase/bytebase/backend/plugin/parser/mongodb"
)

var _ db.Driver = (*Driver)(nil)

func init() {
	db.Register(storepb.Engine_MONGODB, newDriver)
}

// Driver is the MongoDB driver.
type Driver struct {
	connCfg      db.ConnectionConfig
	client       *mongo.Client
	databaseName string
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a MongoDB driver.
func (d *Driver) Open(_ context.Context, _ storepb.Engine, connCfg db.ConnectionConfig) (db.Driver, error) {
	connectionURI := getBasicMongoDBConnectionURI(connCfg)
	opts := options.Client().ApplyURI(connectionURI)
	tlscfg, err := util.GetTLSConfig(connCfg.DataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get SSL config")
	}
	if tlscfg != nil {
		// Use the TLS config from util.GetTLSConfig which respects verify_tls_certificate setting
		opts.SetTLSConfig(tlscfg)
	}
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create MongoDB client")
	}
	d.client = client
	d.connCfg = connCfg
	d.databaseName = connCfg.ConnectionContext.DatabaseName
	return d, nil
}

// Close closes the MongoDB driver.
func (d *Driver) Close(ctx context.Context) error {
	if err := d.client.Disconnect(ctx); err != nil {
		return errors.Wrap(err, "failed to disconnect MongoDB")
	}
	return nil
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	if err := d.client.Ping(ctx, nil); err != nil {
		return errors.Wrap(err, "failed to ping MongoDB")
	}
	return nil
}

// GetDB gets the database.
func (*Driver) GetDB() *sql.DB {
	return nil
}

// Execute executes MongoDB statements one by one via gomongo.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	stmts, err := mongodbparser.SplitSQL(statement)
	if err != nil {
		return 0, errors.Wrap(err, "failed to split MongoDB statement")
	}

	stmts = base.FilterEmptyStatements(stmts)
	if len(stmts) == 0 {
		return 0, nil
	}

	gmClient := gomongo.NewClient(d.client)

	for _, stmt := range stmts {
		opts.LogCommandExecute(stmt.Range, stmt.Text)

		if _, err := gmClient.Execute(ctx, d.databaseName, stmt.Text); err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			return 0, err
		}
		opts.LogCommandResponse(0, nil, "")
	}

	return 0, nil
}

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}

// getBasicMongoDBConnectionURI returns the basic MongoDB connection URI, the following fields are excluded:
// - TLS related
// https://www.mongodb.com/docs/manual/reference/connection-string/
func getBasicMongoDBConnectionURI(connConfig db.ConnectionConfig) string {
	u := &url.URL{
		// In RFC, there can be no tailing slash('/') in the path if the path is empty and the query is not empty.
		// The Go driver throws "error parsing uri: must have a / before the query ?" without it.
		Scheme: "mongodb",
		Path:   "/",
	}
	if connConfig.DataSource.GetSrv() {
		u.Scheme = "mongodb+srv"
	}
	if connConfig.DataSource.Username != "" {
		u.User = url.UserPassword(connConfig.DataSource.Username, connConfig.Password)
	}
	u.Host = connConfig.DataSource.Host
	if connConfig.DataSource.Port != "" {
		u.Host = fmt.Sprintf("%s:%s", u.Host, connConfig.DataSource.Port)
	}
	for _, additionalAddress := range connConfig.DataSource.GetAdditionalAddresses() {
		address := additionalAddress.Host
		if additionalAddress.Port != "" {
			address = fmt.Sprintf("%s:%s", address, additionalAddress.Port)
		}
		u.Host = fmt.Sprintf("%s,%s", u.Host, address)
	}
	if connConfig.ConnectionContext.DatabaseName != "" {
		u.Path = connConfig.ConnectionContext.DatabaseName
	}
	authDatabase := "admin"
	if connConfig.DataSource.GetAuthenticationDatabase() != "" {
		authDatabase = connConfig.DataSource.GetAuthenticationDatabase()
	}

	values := u.Query()
	values.Add("authSource", authDatabase)
	if connConfig.DataSource.GetReplicaSet() != "" {
		values.Add("replicaSet", connConfig.DataSource.GetReplicaSet())
	}
	values.Add("appName", "bytebase")
	if connConfig.DataSource.GetDirectConnection() {
		values.Add("directConnection", "true")
	}

	for k, v := range connConfig.DataSource.GetExtraConnectionParameters() {
		if k == "" {
			continue
		}
		values.Add(k, v)
	}
	u.RawQuery = values.Encode()

	return u.String()
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, _ *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	if queryContext.Explain {
		return nil, errors.New("MongoDB does not support EXPLAIN")
	}

	statement = strings.Trim(statement, " \t\n\r\f;")
	startTime := time.Now()

	gmClient := gomongo.NewClient(d.client)
	var gmOpts []gomongo.ExecuteOption
	if queryContext.Limit > 0 {
		gmOpts = append(gmOpts, gomongo.WithMaxRows(int64(queryContext.Limit)))
	}
	result, err := gmClient.Execute(ctx, d.databaseName, statement, gmOpts...)
	if err != nil {
		return nil, err
	}
	return d.convertGomongoResult(result, statement, startTime), nil
}

func (*Driver) convertGomongoResult(res *gomongo.Result, statement string, startTime time.Time) []*v1pb.QueryResult {
	rows := []*v1pb.QueryRow{}
	for _, v := range res.Value {
		str, err := marshalValueToExtJSON(v)
		if err != nil {
			slog.Error("failed to marshal gomongo result value", log.BBError(err))
			str = fmt.Sprintf("%v", v)
		}
		rows = append(rows, &v1pb.QueryRow{
			Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: str}},
			},
		})
	}

	return []*v1pb.QueryResult{{
		ColumnNames:     []string{"result"},
		ColumnTypeNames: []string{"TEXT"},
		Rows:            rows,
		RowsCount:       int64(len(res.Value)),
		Latency:         durationpb.New(time.Since(startTime)),
		Statement:       statement,
	}}
}

// marshalValueToExtJSON marshals a value to Extended JSON (relaxed) format.
// The frontend expects each row to be a JSON object with key-value pairs,
// so primitive values are wrapped in an object with a "value" key.
func marshalValueToExtJSON(v any) (string, error) {
	var toMarshal any
	switch val := v.(type) {
	case string, int64, bool:
		// For primitive results, wrap as JSON object
		toMarshal = bson.M{"value": val}
	default:
		toMarshal = val
	}
	jsonBytes, err := bson.MarshalExtJSON(toMarshal, false, false)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
