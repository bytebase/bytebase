package elasticsearch

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func elasticsearchDriverFunc(db.DriverConfig) db.Driver {
	return &Driver{}
}

func init() {
	db.Register(storepb.Engine_ELASTICSEARCH, elasticsearchDriverFunc)
}

var (
	_ db.Driver = (*Driver)(nil)
)

type Driver struct {
	client *elasticsearch.TypedClient
	config db.ConnectionConfig
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			// TODO(tommy): support multiple connections and HTTPS.
			fmt.Sprintf("http://%s:%s", config.Host, config.Port),
		},
		Username: config.Username,
		Password: config.Password,
	}

	client, err := elasticsearch.NewTypedClient(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create elasticsearch client")
	}

	// generate basic authentication string.
	encodedUsrAndPasswd := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", config.Username, config.Password)))
	config.AuthenticationPrivateKey = fmt.Sprintf("Basic %s", string(encodedUsrAndPasswd))

	return &Driver{
		client: client,
		config: config,
	}, nil
}

// ElasticSearch doesn't keep a live connection as it uses stateless HTTP.
func (*Driver) Close(_ context.Context) error {
	return nil
}

func (d *Driver) Ping(ctx context.Context) error {
	_, err := d.client.Ping().Do(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to ping db")
	}
	return nil
}

func (d *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	_, err := d.QueryConn(ctx, nil, statement, nil)
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (d *Driver) QueryConn(_ context.Context, _ *sql.Conn, statement string, _ *db.QueryContext) ([]*v1pb.QueryResult, error) {
	statements, err := SplitElasticsearchStatements(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, s := range statements {
		startTime := time.Now()
		// send HTTP request.
		req, err := http.NewRequest(s.method, fmt.Sprintf("http://%s:%s/%s",
			d.config.Host, d.config.Port, strings.TrimLeft(string(s.route), "/")), bytes.NewReader(s.queryString))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to init a HTTP request")
		}

		req.Header.Add("Authorization", d.config.AuthenticationPrivateKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to send HTTP request, status code message: %s", resp.Status)
		}
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read from HTTP response")
		}
		if err = resp.Body.Close(); err != nil {
			return nil, errors.Wrapf(err, "failed to close response body")
		}

		// structure results.
		var result v1pb.QueryResult
		var row v1pb.QueryRow

		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			pairs := map[string]any{}
			if err := json.Unmarshal(respBytes, &pairs); err != nil {
				return nil, errors.Wrapf(err, "failed to parse json body")
			}
			for key, val := range pairs {
				result.ColumnNames = append(result.ColumnNames, key)
				bytes, err := json.Marshal(val)
				if err != nil {
					return nil, err
				}
				row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(bytes)}})
			}
		} else if strings.Contains(contentType, "text/plain") {
			result.ColumnNames = append(result.ColumnNames, "plain/text")
			row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(respBytes)}})
		} else {
			return nil, errors.Errorf("Content-Type not supported: %s", contentType)
		}

		result.Rows = append(result.Rows, &row)
		result.Latency = durationpb.New(time.Since(startTime))
		result.Statement = fmt.Sprintf("%s %s\n%s", s.method, s.route, s.queryString)
		results = append(results, &result)
	}

	return results, nil
}

func (d *Driver) RunStatement(ctx context.Context, _ *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	results, err := d.QueryConn(ctx, nil, statement, nil)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (*Driver) GetDB() *sql.DB {
	return nil
}

func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_ELASTICSEARCH
}
