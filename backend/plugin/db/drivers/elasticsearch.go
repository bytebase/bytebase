package drivers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sigv4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	odriver "github.com/bytebase/omni/driver"
	esdriver "github.com/bytebase/omni/elasticsearch/driver"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// elasticsearchSpec opens without reaching the endpoint, though an SSH data
// source dials its jump host and an AWS IAM login resolves its credentials.
var elasticsearchSpec = spec{
	open:    elasticsearchOpen,
	adapter: util.AdapterOptions{WithInstanceMetadata: true},
	query:   elasticsearchQuery,
	ping: func(ctx context.Context, d *Driver) error {
		if err := d.OmniAdapter.Ping(ctx); err != nil {
			return openSearchConnectionError(d.config.DataSource, err)
		}
		return nil
	},
}

func elasticsearchOpen(ctx context.Context, d *Driver, config db.ConnectionConfig) (odriver.Driver, error) {
	dataSource := config.DataSource
	cfg := odriver.Config{
		Host: elasticsearchHost(dataSource),
		Port: dataSource.GetPort(),
	}
	var tlsConfig *tls.Config
	if elasticsearchSecure(dataSource) {
		var err error
		tlsConfig, err = elasticsearchTLSConfig(dataSource)
		if err != nil {
			return nil, err
		}
		cfg.TLS = func(string) (*tls.Config, error) { return tlsConfig, nil }
	}
	if !isOpenSearchIAM(dataSource) {
		cfg.User = dataSource.GetUsername()
		cfg.Password = config.Password
		if err := util.SetupSSHDial(dataSource, &cfg, &d.OmniAdapter, true); err != nil {
			return nil, err
		}
		return esdriver.Open(ctx, cfg, esdriver.Options{})
	}
	// A managed domain signs each request through a transport of the caller's,
	// which then owns its own connections, so this login path cannot also take
	// the tunnel.
	transport, err := signedTransport(ctx, config, tlsConfig)
	if err != nil {
		return nil, err
	}
	return esdriver.Open(ctx, cfg, esdriver.Options{OpenSearch: true, Transport: transport})
}

// isOpenSearchIAM reports whether the endpoint is a managed OpenSearch domain
// reached with SigV4.
func isOpenSearchIAM(dataSource *storepb.DataSource) bool {
	return dataSource.GetAuthenticationType() == storepb.DataSource_AWS_RDS_IAM
}

// elasticsearchHost is the host with its port glued on when the host carries
// its own scheme: the contract then ignores Config.Port.
func elasticsearchHost(dataSource *storepb.DataSource) string {
	host := dataSource.GetHost()
	if port := dataSource.GetPort(); port != "" && strings.Contains(host, "://") {
		return host + ":" + port
	}
	return host
}

// elasticsearchSecure reports whether the endpoint is reached over https:
// use_ssl asks for it, a managed OpenSearch domain requires it, and so does a
// host that names https in its own scheme.
func elasticsearchSecure(dataSource *storepb.DataSource) bool {
	return dataSource.GetUseSsl() ||
		isOpenSearchIAM(dataSource) ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(dataSource.GetHost())), "https://")
}

// elasticsearchTLSConfig returns the TLS settings the endpoint is verified
// with. A data source with no TLS settings encrypts without verifying, and a CA
// certificate joins the endpoint's roots whether or not verification is on.
func elasticsearchTLSConfig(dataSource *storepb.DataSource) (*tls.Config, error) {
	// The certificate is checked against the host's name, so a host that
	// carries its own scheme and port is checked by the name inside it.
	if host := dataSource.GetHost(); strings.Contains(host, "://") {
		u, err := url.Parse(strings.TrimSpace(host))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse address %q", host)
		}
		dataSource = proto.CloneOf(dataSource)
		dataSource.Host = u.Hostname()
	}
	tlsConfig, err := util.GetTLSConfig(dataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get TLS config")
	}
	if tlsConfig == nil {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}
	tlsConfig.MinVersion = tls.VersionTLS12
	if ca := dataSource.GetSslCa(); ca != "" {
		if tlsConfig.RootCAs == nil {
			tlsConfig.RootCAs = x509.NewCertPool()
		}
		if ok := tlsConfig.RootCAs.AppendCertsFromPEM([]byte(ca)); !ok {
			return nil, errors.New("failed to add CA certificate from PEM-encoded certificate")
		}
	}
	return tlsConfig, nil
}

const (
	openSearchService = "es"
	emptyStringSHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// signedTransport resolves the AWS credentials and signs every request the
// driver sends with them.
func signedTransport(ctx context.Context, config db.ConnectionConfig, tlsConfig *tls.Config) (http.RoundTripper, error) {
	if config.DataSource.GetRegion() == "" {
		return nil, errors.New("region is required for AWS IAM authentication")
	}
	awsCfg, err := util.LoadAWSConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	if _, ok := awsCfg.Credentials.(*aws.CredentialsCache); !ok {
		awsCfg.Credentials = aws.NewCredentialsCache(awsCfg.Credentials)
	}
	if _, err := awsCfg.Credentials.Retrieve(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve AWS credentials")
	}
	return &signedHTTP{
		base: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 5 * time.Second,
			TLSClientConfig:       tlsConfig,
		},
		signer:      sigv4.NewSigner(),
		credentials: awsCfg.Credentials,
		region:      awsCfg.Region,
	}, nil
}

// signedHTTP signs each request with AWS SigV4 before the transport under it
// sends it, which a managed OpenSearch domain requires.
type signedHTTP struct {
	base        http.RoundTripper
	signer      *sigv4.Signer
	credentials aws.CredentialsProvider
	region      string
}

func (t *signedHTTP) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if t.region == "" {
		closeRequestBody(req.Body)
		return nil, errors.New("region is required for AWS IAM authentication")
	}
	if t.credentials == nil {
		closeRequestBody(req.Body)
		return nil, errors.New("AWS credentials provider is not configured")
	}
	creds, err := t.credentials.Retrieve(ctx)
	if err != nil {
		closeRequestBody(req.Body)
		return nil, err
	}
	cloned := req.Clone(ctx)
	payloadHash, err := hashAndRestoreBody(req, cloned)
	if err != nil {
		return nil, err
	}
	cloned.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := t.signer.SignHTTP(ctx, creds, cloned, payloadHash, openSearchService, t.region, time.Now().UTC()); err != nil {
		return nil, err
	}
	return t.base.RoundTrip(cloned)
}

func hashAndRestoreBody(orig, cloned *http.Request) (string, error) {
	if orig.Body == nil || orig.Body == http.NoBody {
		return emptyStringSHA256, nil
	}
	bodyBytes, readErr := io.ReadAll(orig.Body)
	closeErr := orig.Body.Close()
	if readErr != nil {
		return "", readErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	sum := sha256.Sum256(bodyBytes)
	cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	cloned.ContentLength = int64(len(bodyBytes))
	return hex.EncodeToString(sum[:]), nil
}

func closeRequestBody(body io.ReadCloser) {
	if body != nil {
		_ = body.Close()
	}
}

// openSearchConnectionError names what a managed OpenSearch domain's refusal
// means, for a support case to read. The status is in the driver's error text.
func openSearchConnectionError(dataSource *storepb.DataSource, err error) error {
	if !isOpenSearchIAM(dataSource) {
		return err
	}
	text := err.Error()
	switch {
	case strings.Contains(text, "401") || strings.Contains(text, "403"):
		if roleARN := dataSource.GetAwsCredential().GetRoleArn(); roleARN != "" {
			return errors.Errorf("authentication failed: unable to assume role %s: %v", roleARN, err)
		}
		return errors.Errorf("authentication failed (consider using cross-account role if accessing different AWS account): %v", err)
	case strings.Contains(text, "404"):
		return errors.Errorf("endpoint not found (check if path is correct): %v", err)
	default:
		return errors.Wrapf(err, "failed to connect to OpenSearch")
	}
}

// elasticsearchQuery runs each request of a read-only script. A request that
// fails fails the call, so the results before it are not answered either.
func elasticsearchQuery(ctx context.Context, d *Driver, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	stmtResults, err := d.QueryEphemeralSession(ctx, statement, odriver.QueryOptions{
		Explain:        queryContext.Explain,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
	})
	if err != nil {
		return nil, err
	}
	var results []*v1pb.QueryResult
	for _, stmtResult := range stmtResults {
		if stmtResult.Err != nil {
			return nil, stmtResult.Err
		}
		results = append(results, util.OmniResultToQueryResultWith(stmtResult, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
			SkipColumnTypes:    true,
			SkipMessages:       true,
			TypedExplainPlan:   true,
			IncludePlanContent: true,
			TruncatedWins:      true,
			// Every answer is text: a JSON value re-encoded, or the body as the
			// endpoint wrote it.
			ConvertValue: func(value odriver.Value) *v1pb.RowValue {
				switch value.Kind {
				case odriver.Null:
					return util.NullRowValue
				case odriver.String:
					return stringValue(value.String)
				default:
					return printedValue(value)
				}
			},
		}))
	}
	return results, nil
}
