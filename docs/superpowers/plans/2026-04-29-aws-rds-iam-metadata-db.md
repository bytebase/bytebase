# AWS RDS IAM Metadata DB Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add native AWS RDS IAM authentication for Bytebase's metadata PostgreSQL database while preserving existing `PG_URL` behavior.

**Architecture:** Keep the feature inside `backend/store`, where metadata DB connections are created today. Parse and strip reserved `bytebase_aws_*` query parameters before pgx sees the URL, then use pgx stdlib `OptionBeforeConnect` to generate an AWS RDS auth token for each new physical connection. Helm renders the same URL shape for Kubernetes deployments, and docs explain operator prerequisites.

**Tech Stack:** Go, pgx v5 stdlib, AWS SDK for Go v2 `config` and `feature/rds/auth`, Helm templates, Markdown docs.

---

## File Structure

- Create `backend/store/metadata_db_auth.go`: private parser, auth config, token provider, and `BeforeConnect` hook for metadata DB IAM auth.
- Create `backend/store/metadata_db_auth_test.go`: unit tests for URL parsing, stripping, validation, and hook behavior with fake token providers.
- Modify `backend/store/db_connection.go`: call the parser before `pgx.ParseConfig` and pass `stdlib.OptionBeforeConnect` when IAM auth is enabled.
- Modify `helm-charts/bytebase/values.yaml`: add `bytebase.option.externalPg.awsRdsIam.enabled` and `region` defaults.
- Modify `helm-charts/bytebase/templates/statefulset.yaml`: render passwordless IAM `PG_URL` when `awsRdsIam.enabled` is true.
- Modify `helm-charts/bytebase/README.md`: document the new chart values.
- Modify `docs/operations/high-availability.md`: add a short note that all HA replicas using RDS IAM must share the same IAM-enabled metadata `PG_URL` shape and AWS principal setup.

## Task 1: Add Metadata DB IAM URL Parser

**Files:**
- Create: `backend/store/metadata_db_auth.go`
- Create: `backend/store/metadata_db_auth_test.go`

- [ ] **Step 1: Write parser tests**

Create `backend/store/metadata_db_auth_test.go` with:

```go
package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMetadataDBAuthConfigDisabled(t *testing.T) {
	cleanURL, authConfig, err := parseMetadataDBAuthConfig("postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full")
	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.Equal(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full", cleanURL)
}

func TestParseMetadataDBAuthConfigEnabled(t *testing.T) {
	cleanURL, authConfig, err := parseMetadataDBAuthConfig("postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")
	require.NoError(t, err)
	require.Equal(t, "postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full", cleanURL)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
}

func TestParseMetadataDBAuthConfigStripsDisabledBytebaseAWSParams(t *testing.T) {
	cleanURL, authConfig, err := parseMetadataDBAuthConfig("postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=false&bytebase_aws_region=us-east-1&sslmode=verify-full")
	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.Equal(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full", cleanURL)
}

func TestParseMetadataDBAuthConfigRequiresFields(t *testing.T) {
	tests := []struct {
		name    string
		pgURL   string
		wantErr string
	}{
		{
			name:    "region",
			pgURL:   "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true",
			wantErr: "bytebase_aws_region is required",
		},
		{
			name:    "user",
			pgURL:   "postgres://example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
			wantErr: "database user is required",
		},
		{
			name:    "host",
			pgURL:   "postgres://bb@:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
			wantErr: "database host is required",
		},
		{
			name:    "port",
			pgURL:   "postgres://bb@example.us-east-1.rds.amazonaws.com/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
			wantErr: "database port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseMetadataDBAuthConfig(tt.pgURL)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestParseMetadataDBAuthConfigRejectsIAMForNonURI(t *testing.T) {
	_, _, err := parseMetadataDBAuthConfig("host=example.us-east-1.rds.amazonaws.com port=5432 user=bb bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1")
	require.ErrorContains(t, err, "metadata database AWS RDS IAM auth requires a postgres:// or postgresql:// URL")
}

func TestParseMetadataDBAuthConfigRequiresVerifyFullSSLMode(t *testing.T) {
	tests := []struct {
		name  string
		pgURL string
	}{
		{
			name:  "missing",
			pgURL: "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
		},
		{
			name:  "disable",
			pgURL: "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=disable&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
		},
		{
			name:  "require",
			pgURL: "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=require&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseMetadataDBAuthConfig(tt.pgURL)
			require.ErrorContains(t, err, "sslmode=verify-full is required")
		})
	}
}
```

- [ ] **Step 2: Run parser tests to verify they fail**

Run:

```bash
go test -v -count=1 ./backend/store -run '^TestParseMetadataDBAuthConfig'
```

Expected: FAIL with `undefined: parseMetadataDBAuthConfig`.

- [ ] **Step 3: Implement the parser**

Create `backend/store/metadata_db_auth.go` with:

```go
package store

import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

const (
	metadataDBAWSRDSIAMParam = "bytebase_aws_rds_iam"
	metadataDBAWSRegionParam = "bytebase_aws_region"
)

type metadataDBAuthConfig struct {
	enabled  bool
	region   string
	endpoint string
	user     string
}

type metadataDBTokenProvider interface {
	BuildAuthToken(ctx context.Context, endpoint, region, user string) (string, error)
}

func parseMetadataDBAuthConfig(pgURL string) (string, *metadataDBAuthConfig, error) {
	if !strings.HasPrefix(pgURL, "postgres://") && !strings.HasPrefix(pgURL, "postgresql://") {
		if strings.Contains(pgURL, metadataDBAWSRDSIAMParam) || strings.Contains(pgURL, metadataDBAWSRegionParam) {
			return "", nil, errors.New("metadata database AWS RDS IAM auth requires a postgres:// or postgresql:// URL")
		}
		return pgURL, nil, nil
	}

	u, err := url.Parse(pgURL)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to parse database URL")
	}

	q := u.Query()
	iamEnabled := q.Get(metadataDBAWSRDSIAMParam) == "true"
	region := q.Get(metadataDBAWSRegionParam)
	q.Del(metadataDBAWSRDSIAMParam)
	q.Del(metadataDBAWSRegionParam)
	u.RawQuery = q.Encode()
	cleanURL := u.String()

	if !iamEnabled {
		return cleanURL, nil, nil
	}

	if region == "" {
		return "", nil, errors.Errorf("%s is required when metadata database AWS RDS IAM auth is enabled", metadataDBAWSRegionParam)
	}

	if q.Get("sslmode") != "verify-full" {
		return "", nil, errors.New("sslmode=verify-full is required when metadata database AWS RDS IAM auth is enabled")
	}

	user := u.User.Username()
	if user == "" {
		return "", nil, errors.New("database user is required when metadata database AWS RDS IAM auth is enabled")
	}

	host := u.Hostname()
	if host == "" {
		return "", nil, errors.New("database host is required when metadata database AWS RDS IAM auth is enabled")
	}

	port := u.Port()
	if port == "" {
		return "", nil, errors.New("database port is required when metadata database AWS RDS IAM auth is enabled")
	}

	return cleanURL, &metadataDBAuthConfig{
		enabled:  true,
		region:   region,
		endpoint: net.JoinHostPort(host, port),
		user:     user,
	}, nil
}

func newMetadataDBBeforeConnect(authConfig *metadataDBAuthConfig, tokenProvider metadataDBTokenProvider) func(context.Context, *pgx.ConnConfig) error {
	return func(ctx context.Context, connConfig *pgx.ConnConfig) error {
		token, err := tokenProvider.BuildAuthToken(ctx, authConfig.endpoint, authConfig.region, authConfig.user)
		if err != nil {
			return errors.Wrapf(err, "failed to build metadata database AWS RDS IAM auth token for endpoint %q, region %q, user %q", authConfig.endpoint, authConfig.region, authConfig.user)
		}
		connConfig.Password = token
		return nil
	}
}
```

- [ ] **Step 4: Run parser tests to verify they pass**

Run:

```bash
go test -v -count=1 ./backend/store -run '^TestParseMetadataDBAuthConfig'
```

Expected: PASS.

- [ ] **Step 5: Commit parser**

Run:

```bash
gofmt -w backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git add backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git commit -m "feat(store): parse metadata DB AWS IAM auth config"
```

## Task 2: Add Token Provider and BeforeConnect Tests

**Files:**
- Modify: `backend/store/metadata_db_auth.go`
- Modify: `backend/store/metadata_db_auth_test.go`

- [ ] **Step 1: Add hook tests**

Update the import block in `backend/store/metadata_db_auth_test.go` to:

```go
import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)
```

Then append this test code:

```go
type fakeMetadataDBTokenProvider struct {
	token string
	err   error
	calls []fakeMetadataDBTokenProviderCall
}

type fakeMetadataDBTokenProviderCall struct {
	endpoint string
	region   string
	user     string
}

func (p *fakeMetadataDBTokenProvider) BuildAuthToken(_ context.Context, endpoint, region, user string) (string, error) {
	p.calls = append(p.calls, fakeMetadataDBTokenProviderCall{
		endpoint: endpoint,
		region:   region,
		user:     user,
	})
	return p.token, p.err
}

func TestNewMetadataDBBeforeConnectSetsPassword(t *testing.T) {
	provider := &fakeMetadataDBTokenProvider{token: "generated-token"}
	hook := newMetadataDBBeforeConnect(&metadataDBAuthConfig{
		enabled:  true,
		region:   "us-east-1",
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		user:     "bb_meta",
	}, provider)

	connConfig := &pgx.ConnConfig{}
	require.NoError(t, hook(context.Background(), connConfig))
	require.Equal(t, "generated-token", connConfig.Password)
	require.Equal(t, []fakeMetadataDBTokenProviderCall{{
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		region:   "us-east-1",
		user:     "bb_meta",
	}}, provider.calls)
}

func TestNewMetadataDBBeforeConnectReturnsTokenError(t *testing.T) {
	provider := &fakeMetadataDBTokenProvider{err: errors.New("credential chain failed")}
	hook := newMetadataDBBeforeConnect(&metadataDBAuthConfig{
		enabled:  true,
		region:   "us-east-1",
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		user:     "bb_meta",
	}, provider)

	connConfig := &pgx.ConnConfig{}
	err := hook(context.Background(), connConfig)
	require.ErrorContains(t, err, "failed to build metadata database AWS RDS IAM auth token")
	require.ErrorContains(t, err, "example.us-east-1.rds.amazonaws.com:5432")
	require.ErrorContains(t, err, "us-east-1")
	require.ErrorContains(t, err, "bb_meta")
	require.Empty(t, connConfig.Password)
}
```

- [ ] **Step 2: Run hook tests**

Run:

```bash
go test -v -count=1 ./backend/store -run '^TestNewMetadataDBBeforeConnect'
```

Expected: PASS. These tests use the hook already added in Task 1.

- [ ] **Step 3: Add AWS production provider test seam**

Modify the imports in `backend/store/metadata_db_auth.go` to include AWS SDK packages:

```go
import (
	"context"
	"net"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)
```

Append the production provider to `backend/store/metadata_db_auth.go`. AWS config is loaded once while the metadata DB pool is being created; each `BeforeConnect` call reuses the cached credential provider and only builds a fresh RDS auth token:

```go
type awsMetadataDBTokenProvider struct {
	credentials aws.CredentialsProvider
}

func newAWSMetadataDBTokenProvider(ctx context.Context, region string) (*awsMetadataDBTokenProvider, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS config")
	}

	return &awsMetadataDBTokenProvider{
		credentials: cfg.Credentials,
	}, nil
}

func (p *awsMetadataDBTokenProvider) BuildAuthToken(ctx context.Context, endpoint, region, user string) (string, error) {
	token, err := auth.BuildAuthToken(ctx, endpoint, region, user, p.credentials)
	if err != nil {
		return "", errors.Wrap(err, "failed to create authentication token")
	}
	return token, nil
}
```

- [ ] **Step 4: Run focused store tests**

Run:

```bash
go test -v -count=1 ./backend/store -run '^(TestParseMetadataDBAuthConfig|TestNewMetadataDBBeforeConnect)'
```

Expected: PASS.

- [ ] **Step 5: Commit hook and provider**

Run:

```bash
gofmt -w backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git add backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git commit -m "feat(store): add metadata DB AWS IAM token hook"
```

## Task 3: Wire IAM Auth into Metadata DB Connection Creation

**Files:**
- Modify: `backend/store/db_connection.go`
- Modify: `backend/store/metadata_db_auth.go`

- [ ] **Step 1: Add connection option helper test**

Append this test to `backend/store/metadata_db_auth_test.go`:

```go
func TestMetadataDBOpenOptions(t *testing.T) {
	require.Empty(t, metadataDBOpenOptions(nil, &fakeMetadataDBTokenProvider{}))

	authConfig := &metadataDBAuthConfig{
		enabled:  true,
		region:   "us-east-1",
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		user:     "bb_meta",
	}
	require.Len(t, metadataDBOpenOptions(authConfig, &fakeMetadataDBTokenProvider{}), 1)
}
```

- [ ] **Step 2: Run helper test to verify it fails**

Run:

```bash
go test -v -count=1 ./backend/store -run '^TestMetadataDBOpenOptions$'
```

Expected: FAIL with `undefined: metadataDBOpenOptions`.

- [ ] **Step 3: Add the connection option helper**

Modify `backend/store/metadata_db_auth.go` imports to include pgx stdlib:

```go
	"github.com/jackc/pgx/v5/stdlib"
```

Append this function to `backend/store/metadata_db_auth.go`:

```go
func metadataDBOpenOptions(authConfig *metadataDBAuthConfig, tokenProvider metadataDBTokenProvider) []stdlib.OptionOpenDB {
	if authConfig == nil || !authConfig.enabled {
		return nil
	}
	return []stdlib.OptionOpenDB{
		stdlib.OptionBeforeConnect(newMetadataDBBeforeConnect(authConfig, tokenProvider)),
	}
}
```

- [ ] **Step 4: Wire parser and open options into `createConnectionWithTracer`**

Modify `createConnectionWithTracer` in `backend/store/db_connection.go` from:

```go
func createConnectionWithTracer(ctx context.Context, pgURL string) (*sql.DB, error) {
	pgxConfig, err := pgx.ParseConfig(pgURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse database URL")
	}

	pgxConfig.Tracer = &metadataDBTracer{}
	db := stdlib.OpenDB(*pgxConfig)
```

to:

```go
func createConnectionWithTracer(ctx context.Context, pgURL string) (*sql.DB, error) {
	cleanURL, authConfig, err := parseMetadataDBAuthConfig(pgURL)
	if err != nil {
		return nil, err
	}

	pgxConfig, err := pgx.ParseConfig(cleanURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse database URL")
	}

	pgxConfig.Tracer = &metadataDBTracer{}
	var tokenProvider metadataDBTokenProvider
	if authConfig != nil && authConfig.enabled {
		tokenProvider, err = newAWSMetadataDBTokenProvider(ctx, authConfig.region)
		if err != nil {
			return nil, err
		}
	}
	db := stdlib.OpenDB(*pgxConfig, metadataDBOpenOptions(authConfig, tokenProvider)...)
```

- [ ] **Step 5: Run focused store tests**

Run:

```bash
go test -v -count=1 ./backend/store -run '^(TestParseMetadataDBAuthConfig|TestNewMetadataDBBeforeConnect|TestMetadataDBOpenOptions)'
```

Expected: PASS.

- [ ] **Step 6: Commit connection wiring**

Run:

```bash
gofmt -w backend/store/db_connection.go backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git add backend/store/db_connection.go backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git commit -m "feat(store): enable AWS IAM auth for metadata DB connections"
```

## Task 4: Add Helm Values and Rendering

**Files:**
- Modify: `helm-charts/bytebase/values.yaml`
- Modify: `helm-charts/bytebase/templates/statefulset.yaml`
- Modify: `helm-charts/bytebase/README.md`

- [ ] **Step 1: Add default Helm values**

Modify `helm-charts/bytebase/values.yaml` under `bytebase.option.externalPg`:

```yaml
      awsRdsIam:
        enabled: false
        region: ""
```

- [ ] **Step 2: Add template variables**

Modify the variable block near the top of `helm-charts/bytebase/templates/statefulset.yaml` by adding:

```gotemplate
{{- $externalPgAWSRDSIAMEnabled := .Values.bytebase.option.externalPg.awsRdsIam.enabled -}}
{{- $externalPgAWSRDSIAMRegion := .Values.bytebase.option.externalPg.awsRdsIam.region -}}
```

- [ ] **Step 3: Render passwordless IAM `PG_URL` for direct externalPg fields**

Modify the constructed `PG_URL` branch in `helm-charts/bytebase/templates/statefulset.yaml` so the `{{- else }}` branch after `{{- else if $externalPgURL }}` starts with an IAM branch:

```gotemplate
          {{- if $externalPgAWSRDSIAMEnabled }}
          - name: PG_URL
            value: postgres://{{ $externalPgUsername }}@{{ $externalPgHost }}:{{ $externalPgPort }}/{{ $externalPgDatabase }}?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region={{ required "bytebase.option.externalPg.awsRdsIam.region is required when bytebase.option.externalPg.awsRdsIam.enabled is true" $externalPgAWSRDSIAMRegion }}
          {{- else if and $externalPgExistingPgPasswordSecret (not $externalPgEscapePgPassword) }}
```

Keep the existing password-secret and password branches under the new `else if`.

- [ ] **Step 4: Disable escaped-password init when IAM auth is enabled**

Change both `if $externalPgEscapePgPassword` template guards in `helm-charts/bytebase/templates/statefulset.yaml` to:

```gotemplate
{{- if and $externalPgEscapePgPassword (not $externalPgAWSRDSIAMEnabled) }}
```

This affects the init container block and the shell `export PG_URL=...` block. IAM auth does not use a password, so it must not run the password-escaping path.

- [ ] **Step 5: Document Helm values**

Add these rows to the parameters table in `helm-charts/bytebase/README.md` after `bytebase.option.externalPg.escapePassword`:

```markdown
|    `bytebase.option.externalPg.awsRdsIam.enabled`    | Enables AWS RDS IAM authentication for the Bytebase metadata PostgreSQL database when the chart constructs `PG_URL` from externalPg fields, including `sslmode=verify-full`.                                      | false |
|     `bytebase.option.externalPg.awsRdsIam.region`    | AWS region used to sign RDS IAM authentication tokens for the Bytebase metadata PostgreSQL database. Required when `awsRdsIam.enabled` is true.                                  | ""    |
```

- [ ] **Step 6: Render chart with IAM auth enabled**

Run:

```bash
helm template bytebase-release helm-charts/bytebase \
  --set bytebase.option.externalPg.pgHost=example.us-east-1.rds.amazonaws.com \
  --set bytebase.option.externalPg.pgPort=5432 \
  --set bytebase.option.externalPg.pgUsername=bb_meta \
  --set bytebase.option.externalPg.pgPassword=unused \
  --set bytebase.option.externalPg.pgDatabase=bytebase \
  --set bytebase.option.externalPg.awsRdsIam.enabled=true \
  --set bytebase.option.externalPg.awsRdsIam.region=us-east-1 \
  | rg 'PG_URL|bytebase_aws_rds_iam|PG_PASSWORD|init-container'
```

Expected output includes:

```text
- name: PG_URL
value: postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1
```

Expected output does not include `PG_PASSWORD` or `init-container`.

- [ ] **Step 7: Render chart with IAM auth enabled and missing region**

Run:

```bash
helm template bytebase-release helm-charts/bytebase \
  --set bytebase.option.externalPg.pgHost=example.us-east-1.rds.amazonaws.com \
  --set bytebase.option.externalPg.pgPort=5432 \
  --set bytebase.option.externalPg.pgUsername=bb_meta \
  --set bytebase.option.externalPg.pgDatabase=bytebase \
  --set bytebase.option.externalPg.awsRdsIam.enabled=true
```

Expected: FAIL with `bytebase.option.externalPg.awsRdsIam.region is required when bytebase.option.externalPg.awsRdsIam.enabled is true`.

- [ ] **Step 8: Render chart with IAM auth disabled**

Run:

```bash
helm template bytebase-release helm-charts/bytebase \
  --set bytebase.option.externalPg.pgHost=example.us-east-1.rds.amazonaws.com \
  --set bytebase.option.externalPg.pgPort=5432 \
  --set bytebase.option.externalPg.pgUsername=bb_meta \
  --set bytebase.option.externalPg.pgPassword=secret \
  --set bytebase.option.externalPg.pgDatabase=bytebase \
  | rg 'PG_URL|bytebase_aws_rds_iam'
```

Expected output includes:

```text
value: postgres://bb_meta:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase
```

Expected output does not include `bytebase_aws_rds_iam`.

- [ ] **Step 9: Commit Helm changes**

Run:

```bash
git add helm-charts/bytebase/values.yaml helm-charts/bytebase/templates/statefulset.yaml helm-charts/bytebase/README.md
git commit -m "feat(helm): render metadata DB AWS IAM PG_URL"
```

## Task 5: Add Operator Documentation

**Files:**
- Modify: `docs/operations/high-availability.md`

- [ ] **Step 1: Add metadata DB IAM note**

Add this section near the external PostgreSQL requirement in `docs/operations/high-availability.md`:

````markdown
### AWS RDS IAM authentication for metadata PostgreSQL

When the shared metadata PostgreSQL database is AWS RDS or Aurora PostgreSQL, every Bytebase replica can use the same IAM-enabled `PG_URL` shape:

```text
postgres://bb_meta@mydb.abc.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1
```

The PostgreSQL user must be granted `rds_iam`, and the AWS principal used by each Bytebase process must have `rds-db:connect` for that database user. Use `sslmode=verify-full` with a trust store that validates the RDS certificate. Do not put AWS access keys in `PG_URL`; use the standard AWS SDK credential chain through the runtime environment, such as EKS IRSA, ECS task roles, EC2 instance profiles, environment variables, or shared AWS config.
````

- [ ] **Step 2: Run docs diff review**

Run:

```bash
git diff -- docs/operations/high-availability.md
```

Expected: The diff contains only the RDS IAM metadata database note.

- [ ] **Step 3: Commit docs**

Run:

```bash
git add docs/operations/high-availability.md
git commit -m "docs: describe metadata DB AWS IAM setup"
```

## Task 6: Final Verification

**Files:**
- Verify all changed files from Tasks 1 through 5.

- [ ] **Step 1: Run focused Go tests**

Run:

```bash
go test -v -count=1 ./backend/store -run '^(TestParseMetadataDBAuthConfig|TestNewMetadataDBBeforeConnect|TestMetadataDBOpenOptions)'
```

Expected: PASS.

- [ ] **Step 2: Run Go formatting**

Run:

```bash
gofmt -w backend/store/db_connection.go backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git diff --check
```

Expected: `git diff --check` exits 0.

- [ ] **Step 3: Run backend lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: PASS. If it reports issues in files changed by this plan, fix them and rerun the command until it exits 0.

- [ ] **Step 4: Run backend build**

Run:

```bash
go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go
```

Expected: PASS and produces `./bytebase-build/bytebase`.

- [ ] **Step 5: Render Helm IAM configuration**

Run:

```bash
helm template bytebase-release helm-charts/bytebase \
  --set bytebase.option.externalPg.pgHost=example.us-east-1.rds.amazonaws.com \
  --set bytebase.option.externalPg.pgPort=5432 \
  --set bytebase.option.externalPg.pgUsername=bb_meta \
  --set bytebase.option.externalPg.pgDatabase=bytebase \
  --set bytebase.option.externalPg.awsRdsIam.enabled=true \
  --set bytebase.option.externalPg.awsRdsIam.region=us-east-1 \
  >/tmp/bytebase-iam-rendered.yaml

rg 'postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase\\?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1' /tmp/bytebase-iam-rendered.yaml
```

Expected: `rg` finds exactly the rendered IAM `PG_URL`.

- [ ] **Step 6: Render Helm default password configuration**

Run:

```bash
helm template bytebase-release helm-charts/bytebase \
  --set bytebase.option.externalPg.pgHost=example.us-east-1.rds.amazonaws.com \
  --set bytebase.option.externalPg.pgPort=5432 \
  --set bytebase.option.externalPg.pgUsername=bb_meta \
  --set bytebase.option.externalPg.pgPassword=secret \
  --set bytebase.option.externalPg.pgDatabase=bytebase \
  >/tmp/bytebase-password-rendered.yaml

rg 'postgres://bb_meta:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase' /tmp/bytebase-password-rendered.yaml
```

Expected: `rg` finds the existing password-style `PG_URL`.

- [ ] **Step 7: Review accumulated diff**

Run:

```bash
git diff origin/main...HEAD -- backend/store helm-charts/bytebase docs/operations/high-availability.md
```

Expected: The diff is limited to metadata DB IAM auth, Helm rendering, and operator docs.

- [ ] **Step 8: Final commit if verification required fixes**

If Steps 1 through 7 required fixes after the Task 5 commit, run:

```bash
git add backend/store helm-charts/bytebase docs/operations/high-availability.md
git commit -m "fix: complete metadata DB AWS IAM verification fixes"
```

Expected: a commit is created only when verification changed files.
