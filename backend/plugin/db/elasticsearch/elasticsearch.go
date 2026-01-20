package elasticsearch

import (
	"bytes"
	"context"
	"crypto/tls"
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
	opensearch "github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/opensearch-project/opensearch-go/v4/signer/awsv2"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
)

func elasticsearchDriverFunc() db.Driver {
	return &Driver{}
}

func init() {
	db.Register(storepb.Engine_ELASTICSEARCH, elasticsearchDriverFunc)
}

var (
	_ db.Driver = (*Driver)(nil)
)

type Driver struct {
	// For Elasticsearch
	typedClient *elasticsearch.Client
	// For OpenSearch
	opensearchClient *opensearch.Client
	opensearchAPI    *opensearchapi.Client
	// Common basic auth client for REST operations
	basicAuthClient *BasicAuthClient
	config          db.ConnectionConfig
	isOpenSearch    bool
}

type BasicAuthClient struct {
	httpClient      *http.Client
	addrScheduler   *AddressScheduler
	basicAuthString string
}

func (client *BasicAuthClient) Do(method string, route []byte, queryString []byte) (*http.Response, error) {
	address := client.addrScheduler.GetNewAddress()

	// Parse base URL and join with route path
	baseURL, err := url.Parse(address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse base URL")
	}

	// Split route into path and query parameters
	// The route may contain query parameters (e.g., "/_mapping?pretty")
	// We need to separate them to avoid URL-encoding the '?' character
	routeStr := string(route)
	pathPart, queryPart, hasQuery := strings.Cut(routeStr, "?")

	// Join only the path part (without query parameters)
	// This ensures JoinPath only processes the actual path
	fullURL := baseURL.JoinPath(pathPart)

	// Add query parameters if present in the route
	// Assign directly to RawQuery to preserve the query string as-is
	if hasQuery {
		fullURL.RawQuery = queryPart
	}

	req, err := http.NewRequest(method, fullURL.String(), bytes.NewReader(queryString))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to init a HTTP request")
	}

	// Only add basic auth header if it exists (not for AWS IAM auth)
	if client.basicAuthString != "" {
		req.Header.Add("Authorization", client.basicAuthString)
	}
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

func (*Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	address := fmt.Sprintf("%s:%s", config.DataSource.Host, config.DataSource.Port)

	// Check if address already has a protocol
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		protocol := "http"
		if config.DataSource.GetUseSsl() {
			protocol = "https"
		}
		// AWS OpenSearch always requires HTTPS (port 443)
		// Even if UseSsl is false, AWS managed services enforce HTTPS
		if config.DataSource.GetAuthenticationType() == storepb.DataSource_AWS_RDS_IAM {
			protocol = "https"
		}
		address = fmt.Sprintf("%s://%s", protocol, address)
	}

	if _, err := url.Parse(address); err != nil {
		return nil, errors.Wrapf(err, "failed to parse address: %v", address)
	}

	if config.DataSource.GetAuthenticationType() == storepb.DataSource_AWS_RDS_IAM {
		return openWithOpenSearchClient(ctx, config, address)
	}

	return openWithBasicAuth(ctx, config, address)
}

func openWithBasicAuth(_ context.Context, config db.ConnectionConfig, address string) (db.Driver, error) {
	// Get TLS config that respects verify_tls_certificate setting
	tlsConfig, err := util.GetTLSConfig(config.DataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get TLS config")
	}

	// Default to insecure if no SSL is configured
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
	} else {
		// Ensure minimum TLS version
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	esConfig := elasticsearch.Config{
		Username:  config.DataSource.Username,
		Password:  config.Password,
		Addresses: []string{address},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 5 * time.Second,
			TLSClientConfig:       tlsConfig,
		},
	}
	// default http client.
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// If CA cert is provided, update the config
	// Note: util.GetTLSConfig already handles CA cert configuration
	if config.DataSource.GetSslCert() != "" {
		esConfig.CACert = []byte(config.DataSource.GetSslCert())
	}

	// typed elasticsearch client.
	typedClient, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create elasticsearch client")
	}

	// generate basic authentication string for http client.
	encodedUsrAndPasswd := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", config.DataSource.Username, config.Password)))
	basicAuthString := fmt.Sprintf("Basic %s", string(encodedUsrAndPasswd))

	return &Driver{
		typedClient: typedClient,
		basicAuthClient: &BasicAuthClient{
			httpClient: httpClient,
			addrScheduler: &AddressScheduler{
				addresses: []string{address},
				count:     0,
			},
			basicAuthString: basicAuthString,
		},
		config: config,
	}, nil
}

func openWithOpenSearchClient(ctx context.Context, config db.ConnectionConfig, address string) (db.Driver, error) {
	// Keep debugInfo for error messages
	debugInfo := fmt.Sprintf("OpenSearch config - address: %s, authType: %v", address, config.DataSource.GetAuthenticationType())

	// Get TLS config that respects verify_tls_certificate setting
	tlsConfig, err := util.GetTLSConfig(config.DataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get TLS config")
	}

	// Default to insecure if no SSL is configured
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		}
	} else {
		// Ensure minimum TLS version
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	baseTransport := &http.Transport{
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: 5 * time.Second, // Increase timeout for AWS
		TLSClientConfig:       tlsConfig,
	}

	osConfig := opensearch.Config{
		Addresses: []string{address},
		Transport: baseTransport,
	}

	// Configure authentication
	switch config.DataSource.GetAuthenticationType() {
	case storepb.DataSource_AWS_RDS_IAM:
		// Validate AWS-specific requirements
		if config.DataSource.GetRegion() == "" {
			return nil, errors.New("region is required for AWS IAM authentication")
		}

		// Load AWS configuration (uses specific credentials from UI if provided, otherwise default chain)
		awsCfg, err := util.GetAWSConnectionConfig(ctx, config)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load AWS config")
		}

		// Handle cross-account role assumption if configured
		if err := util.AssumeRoleIfNeeded(ctx, &awsCfg, config.ConnectionContext, config.DataSource.GetAwsCredential()); err != nil {
			return nil, err
		}

		// Verify credentials are available
		_, err = awsCfg.Credentials.Retrieve(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrieve AWS credentials")
		}

		// Create AWS signer
		signer, err := awsv2.NewSigner(awsCfg)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create AWS signer")
		}

		osConfig.Signer = signer

	default:
		// Basic auth
		osConfig.Username = config.DataSource.Username
		osConfig.Password = config.Password
	}

	// Note: util.GetTLSConfig already handles CA cert configuration,
	// so we don't need to manually handle it here anymore

	// Create OpenSearch client
	osClient, err := opensearch.NewClient(osConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create OpenSearch client; %s", debugInfo)
	}

	// Create OpenSearch API client
	apiClient, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: osConfig,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create OpenSearch API client; %s", debugInfo)
	}

	// For consistency, still create a basic auth client for REST operations
	httpClient := &http.Client{
		Transport: osConfig.Transport,
	}

	basicAuthString := ""
	// For AWS IAM, we don't use basic auth - the AWS signer handles authentication
	if config.DataSource.GetAuthenticationType() != storepb.DataSource_AWS_RDS_IAM &&
		config.DataSource.Username != "" && config.Password != "" {
		encodedUsrAndPasswd := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", config.DataSource.Username, config.Password)))
		basicAuthString = fmt.Sprintf("Basic %s", string(encodedUsrAndPasswd))
	}

	return &Driver{
		opensearchClient: osClient,
		opensearchAPI:    apiClient,
		basicAuthClient: &BasicAuthClient{
			httpClient: httpClient,
			addrScheduler: &AddressScheduler{
				addresses: []string{address},
				count:     0,
			},
			basicAuthString: basicAuthString,
		},
		config:       config,
		isOpenSearch: true,
	}, nil
}

// ElasticSearch doesn't keep a live connection as it uses stateless HTTP.
func (*Driver) Close(_ context.Context) error {
	return nil
}

func (d *Driver) Ping(_ context.Context) error {
	if d.isOpenSearch && d.opensearchAPI != nil {
		ctx := context.Background()
		info, err := d.opensearchAPI.Info(ctx, &opensearchapi.InfoReq{})
		if err != nil {
			// Check if it's an authentication or connection issue
			errStr := err.Error()
			if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
				// Check if role assumption was attempted
				if d.config.DataSource.GetAwsCredential() != nil &&
					d.config.DataSource.GetAwsCredential().RoleArn != "" {
					return errors.Errorf("authentication failed: unable to assume role %s: %v",
						d.config.DataSource.GetAwsCredential().RoleArn, err)
				}
				return errors.Errorf("authentication failed (consider using cross-account role if accessing different AWS account): %v", err)
			}
			if strings.Contains(errStr, "404") {
				return errors.Errorf("endpoint not found (check if path is correct): %v", err)
			}
			// For any other error, return the full error
			return errors.Wrapf(err, "failed to connect to OpenSearch")
		}
		if info == nil || info.Version.Number == "" {
			return errors.New("invalid response from server")
		}
		return nil
	}

	// Use Elasticsearch client
	if d.typedClient != nil {
		res, err := d.typedClient.Ping()
		if err != nil {
			return errors.Wrapf(err, "failed to ping db")
		}
		defer res.Body.Close()
		if res.IsError() {
			return errors.Errorf("ping failed: %s", res.String())
		}
		return nil
	}

	// Fallback to basic auth client
	if d.basicAuthClient != nil {
		resp, err := d.basicAuthClient.Do("GET", []byte("/"), nil)
		if err != nil {
			return errors.Wrapf(err, "failed to ping db")
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return errors.Errorf("ping failed with status %d: %s", resp.StatusCode, body)
		}
		return nil
	}

	return errors.New("no client available for ping")
}

func (d *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	_, err := d.QueryConn(ctx, nil, statement, db.QueryContext{})
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (d *Driver) QueryConn(_ context.Context, _ *sql.Conn, statement string, _ db.QueryContext) ([]*v1pb.QueryResult, error) {
	parseResult, err := parser.ParseElasticsearchREST(statement)
	if err != nil {
		return nil, err
	}
	if len(parseResult.Errors) > 0 {
		return nil, parseResult.Errors[0]
	}
	if parseResult.Requests == nil {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, request := range parseResult.Requests {
		if err := func() error {
			startTime := time.Now()
			// send HTTP request.
			var data []byte
			for _, item := range request.Data {
				data = append(data, []byte(item)...)
				data = append(data, '\n')
			}
			var resp *http.Response

			// For AWS IAM auth, we need to use the OpenSearch client which has the signer
			if d.isOpenSearch && d.opensearchClient != nil && d.config.DataSource.GetAuthenticationType() == storepb.DataSource_AWS_RDS_IAM {
				// Create request for OpenSearch client
				// Ensure the path starts with "/" for proper HTTP request
				requestPath := strings.TrimSpace(request.URL)
				if requestPath != "" && !strings.HasPrefix(requestPath, "/") {
					requestPath = "/" + requestPath
				}

				req, err := http.NewRequest(request.Method, requestPath, bytes.NewReader(data))
				if err != nil {
					return errors.Wrapf(err, "failed to create HTTP request")
				}
				req.Header.Set("Content-Type", "application/json")

				// Use OpenSearch client's Perform method which handles AWS signing
				resp, err = d.opensearchClient.Perform(req)
				if err != nil {
					return errors.Wrapf(err, "failed to send HTTP request via OpenSearch client")
				}
			} else {
				// For basic auth, use the basic auth client as before
				resp, err = d.basicAuthClient.Do(request.Method, []byte(request.URL), data)
				if err != nil {
					return errors.Wrapf(err, "failed to send HTTP request")
				}
			}
			defer resp.Body.Close()

			respBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return errors.Wrap(err, "failed to read response body")
			}

			// Check HTTP status code for errors (4xx, 5xx)
			if resp.StatusCode >= 400 {
				return errors.Errorf("request failed with status code %d: %s", resp.StatusCode, string(respBytes))
			}

			// structure results.
			var result v1pb.QueryResult
			var row v1pb.QueryRow

			contentType := resp.Header.Get("Content-Type")

			// Debug HTML responses
			if strings.Contains(contentType, "text/html") && resp.StatusCode >= 400 {
				return errors.Errorf("received HTML error response (status %d): %s", resp.StatusCode, string(respBytes))
			}

			switch {
			case strings.Contains(contentType, "application/json"):
				// First unmarshal into any to determine the JSON type
				var data any
				if err := json.Unmarshal(respBytes, &data); err != nil {
					return errors.Wrapf(err, "failed to parse json body")
				}

				// Handle based on the actual type
				switch v := data.(type) {
				case map[string]any:
					// Object response: create one column per key
					for key, val := range v {
						result.ColumnNames = append(result.ColumnNames, key)
						bytes, err := json.Marshal(val)
						if err != nil {
							return err
						}
						row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(bytes)}})
					}
				case []any:
					// Array response: create a single column with array data
					result.ColumnNames = append(result.ColumnNames, "result")
					bytes, err := json.Marshal(v)
					if err != nil {
						return err
					}
					row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(bytes)}})
				default:
					// Primitive response (string, number, boolean, null): single column
					result.ColumnNames = append(result.ColumnNames, "result")
					bytes, err := json.Marshal(v)
					if err != nil {
						return err
					}
					row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(bytes)}})
				}
			case strings.Contains(contentType, "text/plain"):
				result.ColumnNames = append(result.ColumnNames, "text/plain")
				row.Values = append(row.Values, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: string(respBytes)}})
			default:
				return errors.Errorf("Content-Type not supported: %s", contentType)
			}
			result.Rows = append(result.Rows, &row)
			result.Latency = durationpb.New(time.Since(startTime))
			result.RowsCount = int64(len(result.Rows))
			result.Statement = fmt.Sprintf("%s %s\n%s", request.Method, request.URL, string(data))
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
