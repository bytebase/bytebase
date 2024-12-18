package elasticsearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
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
	typedClient     *elasticsearch.Client
	basicAuthClient *BasicAuthClient
	config          db.ConnectionConfig
}

type BasicAuthClient struct {
	httpClient      *http.Client
	addrScheduler   *AddressScheduler
	basicAuthString string
}

func (client *BasicAuthClient) Do(method string, route []byte, queryString []byte) (*http.Response, error) {
	address := client.addrScheduler.GetNewAddress()
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", address, string(route)), bytes.NewReader(queryString))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to init a HTTP request")
	}

	req.Header.Add("Authorization", client.basicAuthString)
	req.Header.Set("Content-Type", "application/json")

	return client.httpClient.Do(req)
}

type AddressScheduler struct {
	addresses []string
	count     int
}

// Get a new address using round-robin. No failover mechanisms temporarily.
func (scheduler *AddressScheduler) GetNewAddress() string {
	address := scheduler.addresses[scheduler.count]
	scheduler.count = (scheduler.count + 1) % len(scheduler.addresses)
	return address
}

func (*Driver) Open(_ context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	addresse := fmt.Sprintf("%s:%s", config.Host, config.Port)
	u, err := url.Parse(addresse)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse addresse: %v", addresse)
	}
	if u.Scheme == "" {
		protocol := "http"
		if config.TLSConfig.UseSSL {
			protocol = "https"
		}
		addresse = fmt.Sprintf("%s://%s", protocol, addresse)
	}

	esConfig := elasticsearch.Config{
		Username:  config.Username,
		Password:  config.Password,
		Addresses: []string{addresse},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
		},
	}
	// default http client.
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
		},
	}

	if config.TLSConfig.SslCert != "" {
		certPool := x509.NewCertPool()
		if ok := certPool.AppendCertsFromPEM([]byte(config.TLSConfig.SslCert)); !ok {
			return nil, errors.New("cannot add CA cert to pool")
		}
		esConfig.CACert = []byte(config.TLSConfig.SslCert)
		esConfig.Transport = &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
		httpClient.Transport = esConfig.Transport
	}

	// typed elasticsearch client.
	typedClient, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create elasticsearch client")
	}

	// generate basic authentication string for http client.
	encodedUsrAndPasswd := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", config.Username, config.Password)))
	basicAuthString := fmt.Sprintf("Basic %s", string(encodedUsrAndPasswd))

	return &Driver{
		typedClient: typedClient,
		basicAuthClient: &BasicAuthClient{
			httpClient: httpClient,
			addrScheduler: &AddressScheduler{
				addresses: []string{addresse},
				count:     0,
			},
			basicAuthString: basicAuthString,
		},
		config: config,
	}, nil
}

// ElasticSearch doesn't keep a live connection as it uses stateless HTTP.
func (*Driver) Close(_ context.Context) error {
	return nil
}

func (d *Driver) Ping(_ context.Context) error {
	if _, err := d.typedClient.Ping(); err != nil {
		return errors.Wrapf(err, "failed to ping db")
	}
	return nil
}

func (d *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	_, err := d.QueryConn(ctx, nil, statement, db.QueryContext{})
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (d *Driver) QueryConn(_ context.Context, _ *sql.Conn, statement string, _ db.QueryContext) ([]*v1pb.QueryResult, error) {
	statements, err := splitElasticsearchStatements(statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, s := range statements {
		if err := func() error {
			startTime := time.Now()
			// send HTTP request.
			resp, err := d.basicAuthClient.Do(string(s.method), s.route, s.queryBody)
			if err != nil {
				return errors.Wrapf(err, "failed to send HTTP request")
			}
			defer resp.Body.Close()

			respBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			// structure results.
			var result v1pb.QueryResult
			var row v1pb.QueryRow

			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				pairs := map[string]any{}
				if err := json.Unmarshal(respBytes, &pairs); err != nil {
					return errors.Wrapf(err, "failed to parse json body")
				}
				for key, val := range pairs {
					result.ColumnNames = append(result.ColumnNames, key)
					bytes, err := json.Marshal(val)
					if err != nil {
						return err
					}
					row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(bytes)}})
				}
			} else if strings.Contains(contentType, "text/plain") {
				result.ColumnNames = append(result.ColumnNames, "text/plain")
				row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(respBytes)}})
			} else {
				return errors.Errorf("Content-Type not supported: %s", contentType)
			}
			result.Rows = append(result.Rows, &row)
			result.Latency = durationpb.New(time.Since(startTime))
			result.RowsCount = int64(len(result.Rows))
			result.Statement = fmt.Sprintf("%s %s\n%s", s.method, s.route, s.queryBody)
			// TODO(d): handle max size.
			results = append(results, &result)
			return nil
		}(); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (*Driver) GetDB() *sql.DB {
	return nil
}
