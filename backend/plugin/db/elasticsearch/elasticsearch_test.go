package elasticsearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	sigv4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	dbutil "github.com/bytebase/bytebase/backend/plugin/db/util"
)

type stubESDriver struct {
	session *stubESSession
	execFn  func(context.Context, string, omnidriver.ExecuteOptions) (int64, error)
}

func (*stubESDriver) Ping(context.Context) error { return nil }
func (*stubESDriver) Close() error               { return nil }
func (d *stubESDriver) Session(context.Context) (omnidriver.Session, error) {
	return d.session, nil
}
func (d *stubESDriver) Execute(ctx context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	if d.execFn != nil {
		return d.execFn(ctx, script, opts)
	}
	return 0, nil
}

type stubESSession struct {
	gotOpts omnidriver.QueryOptions
	results []*omnidriver.Result
	closed  bool
}

func (s *stubESSession) Query(_ context.Context, _ string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotOpts = opts
	return s.results, nil
}

func (s *stubESSession) Close() error {
	s.closed = true
	return nil
}

func TestOpenWithAWSAuth(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  db.ConnectionConfig
		wantErr string
	}{
		{
			name: "missing region",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:               "search-test.us-east-1.es.amazonaws.com",
					Port:               "443",
					AuthenticationType: storepb.DataSource_AWS_RDS_IAM,
				},
			},
			wantErr: "region is required for AWS IAM authentication",
		},
		{
			name: "basic auth",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:               "localhost",
					Port:               "9200",
					Username:           "elastic",
					AuthenticationType: storepb.DataSource_PASSWORD,
				},
				Password: "password123",
			},
		},
		{
			name: "a host that names https",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "https://localhost",
					Port:     "9200",
					Username: "elastic",
				},
				Password: "password123",
			},
		},
		{
			name: "a host that contradicts use_ssl",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "http://localhost",
					Port:     "9200",
					Username: "elastic",
					UseSsl:   true,
				},
				Password: "password123",
			},
			wantErr: "but TLS is required",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (&Driver{}).Open(context.Background(), storepb.Engine_ELASTICSEARCH, tc.config)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSecure(t *testing.T) {
	require.True(t, secure(&storepb.DataSource{UseSsl: true}))
	require.True(t, secure(&storepb.DataSource{Host: "https://search.example.com"}))
	require.True(t, secure(&storepb.DataSource{AuthenticationType: storepb.DataSource_AWS_RDS_IAM}))
	require.False(t, secure(&storepb.DataSource{Host: "search.example.com", Port: "9200"}))
}

func TestOpenWithAnSSHHost(t *testing.T) {
	_, err := (&Driver{}).Open(context.Background(), storepb.Engine_ELASTICSEARCH, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "127.0.0.1",
			Port:     "9200",
			Username: "elastic",
			SshHost:  "127.0.0.1",
			SshPort:  "1",
		},
		Password: "password123",
	})
	require.ErrorContains(t, err, "127.0.0.1:1")
}

func TestTLSSettings(t *testing.T) {
	t.Run("use_ssl with no material encrypts without verifying", func(t *testing.T) {
		tlsConfig, err := tlsSettings(&storepb.DataSource{UseSsl: true})
		require.NoError(t, err)
		require.True(t, tlsConfig.InsecureSkipVerify)
		require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	})

	t.Run("a CA certificate becomes the endpoint's roots", func(t *testing.T) {
		certificate := serverCertificatePEM(t)
		tlsConfig, err := tlsSettings(&storepb.DataSource{UseSsl: true, SslCa: certificate})
		require.NoError(t, err)
		require.NotNil(t, tlsConfig.RootCAs)
		require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	})

	t.Run("a CA that is not a certificate is an error", func(t *testing.T) {
		_, err := tlsSettings(&storepb.DataSource{UseSsl: true, SslCa: "not a certificate"})
		require.ErrorContains(t, err, "failed to add CA certificate")
	})
}

func serverCertificatePEM(t *testing.T) string {
	t.Helper()
	server := httptest.NewTLSServer(http.NotFoundHandler())
	defer server.Close()
	certificate := server.Certificate()
	require.NotNil(t, certificate)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw}))
}

func TestQueryConnAdapterSeam(t *testing.T) {
	t.Run("statement error fails the call", func(t *testing.T) {
		sess := &stubESSession{
			results: []*omnidriver.Result{
				{Statement: "GET /alpha/_doc/1\n"},
				{Statement: "GET /nope/_search\n", Err: errors.New("request failed with status code 500: index_not_found_exception")},
			},
		}
		driver := &Driver{OmniAdapter: dbutil.OmniAdapter{Driver: &stubESDriver{session: sess}}}

		results, err := driver.QueryConn(context.Background(), "GET /alpha/_doc/1\nGET /nope/_search", db.QueryContext{})
		require.Nil(t, results)
		require.True(t, sess.closed)
		require.ErrorContains(t, err, "index_not_found_exception")
	})

	t.Run("forwards explain and formats truncation message", func(t *testing.T) {
		const capBytes = int64(1)
		sess := &stubESSession{
			results: []*omnidriver.Result{{
				Statement: "POST /alpha/_search\n{\"profile\":true}\n",
				Columns:   []omnidriver.Column{{Name: "profile", TypeName: "ignored"}},
				Rows:      [][]omnidriver.Value{{{Kind: omnidriver.String, String: `{"profile":{}}`}}},
				Plan:      &omnidriver.Plan{Format: "JSON", Executed: true},
				Truncated: true,
			}},
		}
		driver := &Driver{OmniAdapter: dbutil.OmniAdapter{Driver: &stubESDriver{session: sess}}}

		results, err := driver.QueryConn(context.Background(), "POST /alpha/_search\n{}", db.QueryContext{
			Explain:              true,
			MaximumSQLResultSize: capBytes,
		})
		require.NoError(t, err)
		require.Equal(t, omnidriver.QueryOptions{Explain: true, MaxResultBytes: capBytes}, sess.gotOpts)
		require.Len(t, results, 1)
		require.Empty(t, results[0].ColumnTypeNames)
		require.Equal(t, dbutil.FormatMaximumSQLResultSizeMessage(capBytes), results[0].Error)
		require.Equal(t, &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_JSON, Executed: true}, results[0].GetQueryPlan())
	})
}

func TestExecuteReportsTheScriptSteps(t *testing.T) {
	const script = "POST /alpha/_bulk\n{\"index\":{}}\n{\"name\":\"row 1\"}"
	stub := &stubESDriver{
		execFn: func(_ context.Context, s string, opts omnidriver.ExecuteOptions) (int64, error) {
			opts.Log.CommandExecute(omnidriver.Command{Text: s, ByteStart: 0, ByteEnd: len(s)})
			opts.Log.CommandResponse(0, nil, nil)
			return 0, nil
		},
	}
	driver := &Driver{OmniAdapter: dbutil.OmniAdapter{Driver: stub}}

	var logs []*storepb.TaskRunLog
	affected, err := driver.Execute(context.Background(), script, db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), affected)
	require.Len(t, logs, 2)
	require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
	require.Equal(t, script, logs[0].GetCommandExecute().GetStatement())
	require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
	require.Empty(t, logs[1].GetCommandResponse().GetError())
}

// A managed OpenSearch domain's refusal names the role that could not be
// assumed, which is what a support case reads.
func TestPingKeepsTheAWSHints(t *testing.T) {
	for _, tc := range []struct {
		name   string
		driver *Driver
		inner  string
		want   string
	}{
		{
			name:   "a role that could not be assumed",
			driver: &Driver{aws: true, awsRoleARN: "arn:aws:iam::123456789012:role/search"},
			inner:  "ping failed: [403 Forbidden] {\"message\":\"no permissions\"}",
			want:   "authentication failed: unable to assume role arn:aws:iam::123456789012:role/search: ping failed: [403 Forbidden] {\"message\":\"no permissions\"}",
		},
		{
			name:   "no role configured",
			driver: &Driver{aws: true},
			inner:  "ping failed: [401 Unauthorized] {}",
			want:   "authentication failed (consider using cross-account role if accessing different AWS account): ping failed: [401 Unauthorized] {}",
		},
		{
			name:   "an endpoint that is not there",
			driver: &Driver{aws: true},
			inner:  "ping failed: [404 Not Found] {}",
			want:   "endpoint not found (check if path is correct): ping failed: [404 Not Found] {}",
		},
		{
			name:   "anything else",
			driver: &Driver{aws: true},
			inner:  "ping failed: [500 Internal Server Error] {}",
			want:   "failed to connect to OpenSearch: ping failed: [500 Internal Server Error] {}",
		},
		{
			name:   "not a managed domain",
			driver: &Driver{},
			inner:  "ping failed: [403 Forbidden] {}",
			want:   "ping failed: [403 Forbidden] {}",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.driver.connectionError(errors.New(tc.inner)).Error())
		})
	}
}

func TestToRowValue(t *testing.T) {
	require.Equal(t, &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "alpha"}},
		toRowValue(omnidriver.Value{Kind: omnidriver.String, String: "alpha"}))
	require.Equal(t, dbutil.NullRowValue, toRowValue(omnidriver.Value{Kind: omnidriver.Null}))
}

func TestToQueryPlan(t *testing.T) {
	convert := func(plan *omnidriver.Plan) *v1pb.QueryResult_QueryPlan {
		return dbutil.OmniResultToQueryResultWith(&omnidriver.Result{Plan: plan}, 0, dbutil.QueryResultOptions{
			TypedExplainPlan:   true,
			IncludePlanContent: true,
		}).QueryPlan
	}
	require.Nil(t, convert(nil))
	require.Equal(t, &v1pb.QueryResult_QueryPlan{Format: v1pb.QueryOption_JSON, Executed: true},
		convert(&omnidriver.Plan{Format: "JSON", Executed: true}))
	require.Equal(t, v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED,
		convert(&omnidriver.Plan{Format: "not a format"}).GetFormat())
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (c *trackingReadCloser) Close() error {
	c.closed = true
	return nil
}

func TestSignedHTTPRoundTrip(t *testing.T) {
	staticCreds := credentials.NewStaticCredentialsProvider("AKIAEXAMPLE", "secret-key", "session-token")

	t.Run("GET with nil body signs with empty SHA-256 and does not mutate caller request", func(t *testing.T) {
		var sent *http.Request
		transport := &signedHTTP{
			base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				sent = req
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
			}),
			signer:      sigv4.NewSigner(),
			credentials: staticCreds,
			region:      "us-west-2",
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://search.us-west-2.es.amazonaws.com/_cluster/health", nil)
		require.NoError(t, err)
		req.Header.Set("Accept", "application/json")

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotNil(t, sent)
		require.NotSame(t, req, sent)

		authHeader := sent.Header.Get("Authorization")
		require.True(t, strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 "), authHeader)
		require.Contains(t, authHeader, "Credential=AKIAEXAMPLE/")
		require.Contains(t, authHeader, "/us-west-2/es/aws4_request")
		require.Equal(t, emptyStringSHA256, sent.Header.Get("X-Amz-Content-Sha256"))
		require.Equal(t, "session-token", sent.Header.Get("X-Amz-Security-Token"))
		amzDate := sent.Header.Get("X-Amz-Date")
		require.NotEmpty(t, amzDate)
		_, err = time.Parse("20060102T150405Z", amzDate)
		require.NoError(t, err)

		// Caller's request headers must remain untouched.
		require.Empty(t, req.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("X-Amz-Date"))
		require.Empty(t, req.Header.Get("X-Amz-Content-Sha256"))
		require.Empty(t, req.Header.Get("X-Amz-Security-Token"))
		require.Equal(t, "application/json", req.Header.Get("Accept"))
	})

	t.Run("POST with JSON body preserves body and produces exact payload signature", func(t *testing.T) {
		const payload = `{"query":{"match_all":{}}}`
		sum := sha256.Sum256([]byte(payload))
		wantHash := hex.EncodeToString(sum[:])

		var sent *http.Request
		var sentBody []byte
		transport := &signedHTTP{
			base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				sent = req
				var readErr error
				sentBody, readErr = io.ReadAll(req.Body)
				require.NoError(t, readErr)
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
			}),
			signer:      sigv4.NewSigner(),
			credentials: staticCreds,
			region:      "eu-central-1",
		}

		body := &trackingReadCloser{Reader: strings.NewReader(payload)}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://search.eu-central-1.es.amazonaws.com/books/_search", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.True(t, body.closed)
		require.Equal(t, payload, string(sentBody))
		require.Equal(t, wantHash, sent.Header.Get("X-Amz-Content-Sha256"))
		require.NotEmpty(t, sent.Header.Get("X-Amz-Date"))
		require.Contains(t, sent.Header.Get("Authorization"), "/eu-central-1/es/aws4_request")

		// Re-sign an identical request at the same timestamp to prove the payload signature matches.
		signingTime, err := time.Parse("20060102T150405Z", sent.Header.Get("X-Amz-Date"))
		require.NoError(t, err)
		expectedReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://search.eu-central-1.es.amazonaws.com/books/_search", bytes.NewReader([]byte(payload)))
		require.NoError(t, err)
		expectedReq.Header.Set("Content-Type", "application/json")
		expectedReq.Header.Set("X-Amz-Content-Sha256", wantHash)
		creds, err := staticCreds.Retrieve(context.Background())
		require.NoError(t, err)
		require.NoError(t, sigv4.NewSigner().SignHTTP(context.Background(), creds, expectedReq, wantHash, "es", "eu-central-1", signingTime))
		require.Equal(t, expectedReq.Header.Get("Authorization"), sent.Header.Get("Authorization"))

		require.Empty(t, req.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("X-Amz-Date"))
	})

	t.Run("propagates credential retrieval failure and closes body", func(t *testing.T) {
		baseCalled := false
		transport := &signedHTTP{
			base: roundTripFunc(func(*http.Request) (*http.Response, error) {
				baseCalled = true
				return nil, nil
			}),
			signer: sigv4.NewSigner(),
			credentials: aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
				return aws.Credentials{}, errors.New("sts unavailable")
			}),
			region: "us-east-1",
		}

		body := &trackingReadCloser{Reader: strings.NewReader("{}")}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://search.us-east-1.es.amazonaws.com/_search", body)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		require.Nil(t, resp)
		require.EqualError(t, err, "sts unavailable")
		require.False(t, baseCalled)
		require.True(t, body.closed)
	})

	t.Run("rejects empty region", func(t *testing.T) {
		baseCalled := false
		transport := &signedHTTP{
			base: roundTripFunc(func(*http.Request) (*http.Response, error) {
				baseCalled = true
				return nil, nil
			}),
			signer:      sigv4.NewSigner(),
			credentials: staticCreds,
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://search.us-east-1.es.amazonaws.com/", nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		require.Nil(t, resp)
		require.ErrorContains(t, err, "region is required")
		require.False(t, baseCalled)
	})
}

func TestSignedTransport(t *testing.T) {
	t.Run("missing region returns error", func(t *testing.T) {
		ds := &storepb.DataSource{
			Host:               "search.us-east-1.es.amazonaws.com",
			AuthenticationType: storepb.DataSource_AWS_RDS_IAM,
		}
		_, err := signedTransport(context.Background(), db.ConnectionConfig{DataSource: ds}, ds, nil)
		require.ErrorContains(t, err, "region is required for AWS IAM authentication")
	})

	t.Run("static credentials produce signedHTTP transport", func(t *testing.T) {
		ds := &storepb.DataSource{
			Host:               "search.us-east-1.es.amazonaws.com",
			Region:             "us-east-1",
			AuthenticationType: storepb.DataSource_AWS_RDS_IAM,
			IamExtension: &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{
					AccessKeyId:     "AKIAEXAMPLE",
					SecretAccessKey: "secret-key",
				},
			},
		}
		rt, err := signedTransport(context.Background(), db.ConnectionConfig{DataSource: ds}, ds, &tls.Config{MinVersion: tls.VersionTLS12})
		require.NoError(t, err)
		signed, ok := rt.(*signedHTTP)
		require.True(t, ok)
		require.Equal(t, "us-east-1", signed.region)
		require.NotNil(t, signed.signer)
		_, isCache := signed.credentials.(*aws.CredentialsCache)
		require.True(t, isCache)
	})
}
