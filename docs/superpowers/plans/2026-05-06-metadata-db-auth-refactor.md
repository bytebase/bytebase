# Metadata DB Auth Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move metadata database auth setup into one provider-neutral `backend/store/dbauth` package while preserving existing AWS RDS IAM behavior.

**Architecture:** `backend/store/db_connection.go` remains responsible for reading `PG_URL`, parsing pgx config, tracing, pinging, and pool sizing. The new `backend/store/dbauth` package owns metadata database auth runtime params, provider dispatch, AWS validation, and pgx open options. Managed-instance auth under `backend/plugin/db` is not changed.

**Tech Stack:** Go, pgx v5, pgx stdlib `OptionOpenDB`, AWS SDK v2 RDS IAM auth, testify, gofmt, golangci-lint.

---

## File Structure

- Create `backend/store/dbauth/auth.go`: public metadata DB auth entrypoints and provider runtime-param detection.
- Create `backend/store/dbauth/aws.go`: AWS RDS IAM metadata DB auth implementation moved from `backend/store/metadata_db_auth.go`.
- Create `backend/store/dbauth/aws_test.go`: tests moved from `backend/store/metadata_db_auth_test.go`, adjusted for package `dbauth`.
- Modify `backend/store/db_connection.go`: import `dbauth`, delegate auth configuration, and delegate provider-specific keyword-value runtime param detection.
- Modify `backend/store/db_connection_test.go`: keep file-path detection coverage after provider param names move out of `store`.
- Delete `backend/store/metadata_db_auth.go`.
- Delete `backend/store/metadata_db_auth_test.go`.

## Task 1: Create `dbauth` Package API With Failing Tests

**Files:**
- Create: `backend/store/dbauth/auth.go`
- Create: `backend/store/dbauth/aws_test.go`

- [ ] **Step 1: Create a minimal package shell**

Create `backend/store/dbauth/auth.go`:

```go
package dbauth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Configure applies metadata database authentication settings to pgxConfig.
func Configure(_ context.Context, _ *pgx.ConnConfig) ([]stdlib.OptionOpenDB, error) {
	return nil, nil
}

// IsKeywordValueRuntimeParam reports whether key is a Bytebase metadata DB auth runtime parameter.
func IsKeywordValueRuntimeParam(_ string) bool {
	return false
}
```

- [ ] **Step 2: Add failing AWS auth tests**

Create `backend/store/dbauth/aws_test.go`:

```go
package dbauth

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type fakeTokenProvider struct {
	token string
	err   error
	calls []fakeTokenProviderCall
}

type fakeTokenProviderCall struct {
	endpoint string
	region   string
	user     string
}

func (p *fakeTokenProvider) BuildAuthToken(_ context.Context, endpoint, region, user string) (string, error) {
	p.calls = append(p.calls, fakeTokenProviderCall{
		endpoint: endpoint,
		region:   region,
		user:     user,
	})
	return p.token, p.err
}

func TestAWSConfigFromPGXConfigDisabled(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
}

func TestAWSConfigFromPGXConfigEnabledURL(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb_meta@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRegionParam)
}

func TestAWSConfigFromPGXConfigEnabledKeywordValue(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "host=example.us-east-1.rds.amazonaws.com port=5432 user=bb_meta dbname=bytebase sslmode=verify-full bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.NotNil(t, authConfig)
	require.True(t, authConfig.enabled)
	require.Equal(t, "us-east-1", authConfig.region)
	require.Equal(t, "example.us-east-1.rds.amazonaws.com:5432", authConfig.endpoint)
	require.Equal(t, "bb_meta", authConfig.user)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRegionParam)
}

func TestAWSConfigFromPGXConfigStripsDisabledParams(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb:secret@example.us-east-1.rds.amazonaws.com:5432/bytebase?bytebase_aws_rds_iam=false&bytebase_aws_region=us-east-1&sslmode=verify-full")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRDSIAMParam)
	require.NotContains(t, pgxConfig.RuntimeParams, awsRegionParam)
}

func TestAWSConfigFromPGXConfigRequiresFields(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*pgx.ConnConfig)
		wantErr   string
	}{
		{
			name:      "region",
			configure: func(config *pgx.ConnConfig) { delete(config.RuntimeParams, awsRegionParam) },
			wantErr:   "bytebase_aws_region is required",
		},
		{
			name:      "user",
			configure: func(config *pgx.ConnConfig) { config.User = "" },
			wantErr:   "database user is required",
		},
		{
			name:      "host",
			configure: func(config *pgx.ConnConfig) { config.Host = "" },
			wantErr:   "database host is required",
		},
		{
			name:      "port",
			configure: func(config *pgx.ConnConfig) { config.Port = 0 },
			wantErr:   "database port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")
			tt.configure(pgxConfig)

			_, err := awsConfigFromPGXConfig(pgxConfig)

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestAWSConfigFromPGXConfigNonURIAllowsParamNamesInValues(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "host=example.us-east-1.rds.amazonaws.com port=5432 user=bb password=bytebase_aws_region application_name=bytebase_aws_rds_iam")

	authConfig, err := awsConfigFromPGXConfig(pgxConfig)

	require.NoError(t, err)
	require.Nil(t, authConfig)
}

func TestAWSConfigFromPGXConfigRequiresVerifiedTLS(t *testing.T) {
	tests := []struct {
		name  string
		pgURL string
	}{
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
			pgxConfig := mustParsePGXConfig(t, tt.pgURL)

			_, err := awsConfigFromPGXConfig(pgxConfig)

			require.ErrorContains(t, err, "verified TLS is required")
		})
	}
}

func TestAWSConfigFromPGXConfigRejectsFallbacks(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=prefer&bytebase_aws_rds_iam=true&bytebase_aws_region=us-east-1")

	_, err := awsConfigFromPGXConfig(pgxConfig)

	require.ErrorContains(t, err, "fallback hosts or TLS fallback")
}

func TestNewAWSBeforeConnectSetsPassword(t *testing.T) {
	tokenProvider := &fakeTokenProvider{token: "generated-token"}
	hook := newAWSBeforeConnect(&awsConfig{
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		region:   "us-east-1",
		user:     "bb_meta",
	}, tokenProvider)
	connConfig := &pgx.ConnConfig{}

	err := hook(context.Background(), connConfig)

	require.NoError(t, err)
	require.Equal(t, "generated-token", connConfig.Password)
	require.Equal(t, []fakeTokenProviderCall{
		{
			endpoint: "example.us-east-1.rds.amazonaws.com:5432",
			region:   "us-east-1",
			user:     "bb_meta",
		},
	}, tokenProvider.calls)
}

func TestNewAWSBeforeConnectReturnsTokenError(t *testing.T) {
	tokenProvider := &fakeTokenProvider{err: errors.New("credential chain failed")}
	hook := newAWSBeforeConnect(&awsConfig{
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		region:   "us-east-1",
		user:     "bb_meta",
	}, tokenProvider)
	connConfig := &pgx.ConnConfig{}

	err := hook(context.Background(), connConfig)

	require.ErrorContains(t, err, "failed to build metadata database AWS RDS IAM auth token")
	require.ErrorContains(t, err, "example.us-east-1.rds.amazonaws.com:5432")
	require.ErrorContains(t, err, "us-east-1")
	require.ErrorContains(t, err, "bb_meta")
	require.Empty(t, connConfig.Password)
}

func TestAWSOpenOptions(t *testing.T) {
	require.Empty(t, awsOpenOptions(nil, &fakeTokenProvider{}))

	authConfig := &awsConfig{
		enabled:  true,
		region:   "us-east-1",
		endpoint: "example.us-east-1.rds.amazonaws.com:5432",
		user:     "bb_meta",
	}

	require.Len(t, awsOpenOptions(authConfig, &fakeTokenProvider{}), 1)
}

func TestConfigure(t *testing.T) {
	pgxConfig := mustParsePGXConfig(t, "postgres://bb@example.us-east-1.rds.amazonaws.com:5432/bytebase?sslmode=verify-full")

	options, err := Configure(context.Background(), pgxConfig)

	require.NoError(t, err)
	require.Empty(t, options)
}

func TestIsKeywordValueRuntimeParam(t *testing.T) {
	require.True(t, IsKeywordValueRuntimeParam("bytebase_aws_rds_iam"))
	require.True(t, IsKeywordValueRuntimeParam("bytebase_aws_region"))
	require.False(t, IsKeywordValueRuntimeParam("host"))
}

func mustParsePGXConfig(t *testing.T, pgURL string) *pgx.ConnConfig {
	t.Helper()

	pgxConfig, err := pgx.ParseConfig(pgURL)
	require.NoError(t, err)
	return pgxConfig
}
```

- [ ] **Step 3: Run test to verify failure**

Run:

```bash
go test -v -count=1 ./backend/store/dbauth
```

Expected: FAIL with undefined identifiers including `awsConfigFromPGXConfig`, `awsRDSIAMParam`, `awsRegionParam`, `newAWSBeforeConnect`, and `awsOpenOptions`.

## Task 2: Move AWS Implementation Into `dbauth`

**Files:**
- Create: `backend/store/dbauth/aws.go`
- Modify: `backend/store/dbauth/auth.go`

- [ ] **Step 1: Implement provider-neutral dispatch**

Replace `backend/store/dbauth/auth.go` with:

```go
package dbauth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Configure applies metadata database authentication settings to pgxConfig.
func Configure(ctx context.Context, pgxConfig *pgx.ConnConfig) ([]stdlib.OptionOpenDB, error) {
	authConfig, err := awsConfigFromPGXConfig(pgxConfig)
	if err != nil {
		return nil, err
	}
	if authConfig == nil || !authConfig.enabled {
		return nil, nil
	}

	tokenProvider, err := newAWSMetadataDBTokenProvider(ctx, authConfig.region)
	if err != nil {
		return nil, err
	}
	return awsOpenOptions(authConfig, tokenProvider), nil
}

// IsKeywordValueRuntimeParam reports whether key is a Bytebase metadata DB auth runtime parameter.
func IsKeywordValueRuntimeParam(key string) bool {
	switch key {
	case awsRDSIAMParam, awsRegionParam:
		return true
	default:
		return false
	}
}
```

- [ ] **Step 2: Implement AWS provider**

Create `backend/store/dbauth/aws.go`:

```go
package dbauth

import (
	"context"
	"net"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
)

const (
	awsRDSIAMParam = "bytebase_aws_rds_iam"
	awsRegionParam = "bytebase_aws_region"
)

type awsConfig struct {
	enabled  bool
	region   string
	endpoint string
	user     string
}

type awsTokenProvider interface {
	BuildAuthToken(ctx context.Context, endpoint, region, user string) (string, error)
}

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

func awsConfigFromPGXConfig(pgxConfig *pgx.ConnConfig) (*awsConfig, error) {
	iamEnabled := pgxConfig.RuntimeParams[awsRDSIAMParam] == "true"
	region := pgxConfig.RuntimeParams[awsRegionParam]
	delete(pgxConfig.RuntimeParams, awsRDSIAMParam)
	delete(pgxConfig.RuntimeParams, awsRegionParam)

	if !iamEnabled {
		return nil, nil
	}

	if region == "" {
		return nil, errors.Errorf("%s is required when metadata database AWS RDS IAM auth is enabled", awsRegionParam)
	}

	if pgxConfig.User == "" {
		return nil, errors.New("database user is required when metadata database AWS RDS IAM auth is enabled")
	}

	if pgxConfig.Host == "" {
		return nil, errors.New("database host is required when metadata database AWS RDS IAM auth is enabled")
	}

	if pgxConfig.Port == 0 {
		return nil, errors.New("database port is required when metadata database AWS RDS IAM auth is enabled")
	}

	if len(pgxConfig.Fallbacks) > 0 {
		return nil, errors.New("metadata database AWS RDS IAM auth does not support fallback hosts or TLS fallback")
	}

	if pgxConfig.TLSConfig == nil || pgxConfig.TLSConfig.InsecureSkipVerify || pgxConfig.TLSConfig.ServerName == "" {
		return nil, errors.New("verified TLS is required when metadata database AWS RDS IAM auth is enabled")
	}

	return &awsConfig{
		enabled:  true,
		region:   region,
		endpoint: net.JoinHostPort(pgxConfig.Host, strconv.FormatUint(uint64(pgxConfig.Port), 10)),
		user:     pgxConfig.User,
	}, nil
}

func newAWSBeforeConnect(authConfig *awsConfig, tokenProvider awsTokenProvider) func(context.Context, *pgx.ConnConfig) error {
	return func(ctx context.Context, connConfig *pgx.ConnConfig) error {
		token, err := tokenProvider.BuildAuthToken(ctx, authConfig.endpoint, authConfig.region, authConfig.user)
		if err != nil {
			return errors.Wrapf(err, "failed to build metadata database AWS RDS IAM auth token for endpoint %q, region %q, user %q", authConfig.endpoint, authConfig.region, authConfig.user)
		}
		connConfig.Password = token
		return nil
	}
}

func awsOpenOptions(authConfig *awsConfig, tokenProvider awsTokenProvider) []stdlib.OptionOpenDB {
	if authConfig == nil || !authConfig.enabled {
		return nil
	}
	return []stdlib.OptionOpenDB{stdlib.OptionBeforeConnect(newAWSBeforeConnect(authConfig, tokenProvider))}
}
```

- [ ] **Step 3: Run dbauth tests**

Run:

```bash
go test -v -count=1 ./backend/store/dbauth
```

Expected: PASS.

- [ ] **Step 4: Commit**

Run:

```bash
git add backend/store/dbauth/auth.go backend/store/dbauth/aws.go backend/store/dbauth/aws_test.go
git commit -m "refactor(store): add metadata DB auth package"
```

## Task 3: Wire `db_connection.go` To `dbauth` And Remove Old Store Auth

**Files:**
- Modify: `backend/store/db_connection.go`
- Modify: `backend/store/db_connection_test.go`
- Delete: `backend/store/metadata_db_auth.go`
- Delete: `backend/store/metadata_db_auth_test.go`

- [ ] **Step 1: Update imports and keyword-value detection**

In `backend/store/db_connection.go`, add:

```go
	"github.com/bytebase/bytebase/backend/store/dbauth"
```

Update the `isKeywordValuePGURL` switch so provider-specific keys are delegated:

```go
			case "host",
				"hostaddr",
				"port",
				"dbname",
				"database",
				"user",
				"password",
				"passfile",
				"connect_timeout",
				"sslmode",
				"sslrootcert",
				"sslcert",
				"sslkey",
				"service",
				"servicefile",
				"target_session_attrs",
				"application_name":
				return true
			default:
				if dbauth.IsKeywordValueRuntimeParam(field[:eqIdx]) {
					return true
				}
			}
```

- [ ] **Step 2: Delegate metadata auth setup**

Replace the auth setup block in `createConnectionWithTracer`:

```go
	authConfig, err := metadataDBAuthConfigFromPGXConfig(pgxConfig)
	if err != nil {
		return nil, err
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

with:

```go
	openOptions, err := dbauth.Configure(ctx, pgxConfig)
	if err != nil {
		return nil, err
	}

	pgxConfig.Tracer = &metadataDBTracer{}
	db := stdlib.OpenDB(*pgxConfig, openOptions...)
```

- [ ] **Step 3: Keep file-path detection test coverage**

In `backend/store/db_connection_test.go`, keep the existing `"keyword value IAM DSN"` case exactly:

```go
{
	name: "keyword value IAM DSN",
	in:   "host=example.com port=5432 user=bb dbname=bytebase bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1 sslmode=verify-full",
	want: false,
},
```

This verifies `db_connection.go` still recognizes auth runtime params through `dbauth.IsKeywordValueRuntimeParam`.

- [ ] **Step 4: Delete old store-level auth files**

Use `apply_patch`:

```diff
*** Begin Patch
*** Delete File: backend/store/metadata_db_auth.go
*** Delete File: backend/store/metadata_db_auth_test.go
*** End Patch
```

- [ ] **Step 5: Format modified Go files**

Run:

```bash
gofmt -w backend/store/db_connection.go backend/store/db_connection_test.go backend/store/dbauth/*.go
```

- [ ] **Step 6: Run focused tests**

Run:

```bash
go test -v -count=1 ./backend/store ./backend/store/dbauth
```

Expected: PASS.

- [ ] **Step 7: Commit**

Run:

```bash
git add backend/store/db_connection.go backend/store/db_connection_test.go backend/store/dbauth backend/store/metadata_db_auth.go backend/store/metadata_db_auth_test.go
git commit -m "refactor(store): move metadata DB auth out of store package"
```

## Task 4: Final Validation

**Files:**
- Verify only.

- [ ] **Step 1: Run Go lint**

Run:

```bash
golangci-lint run --allow-parallel-runners
```

Expected: PASS. If it reports issues, fix the reported issues and run the same command again until it passes.

- [ ] **Step 2: Run focused tests again**

Run:

```bash
go test -v -count=1 ./backend/store ./backend/store/dbauth
```

Expected: PASS.

- [ ] **Step 3: Inspect final diff**

Run:

```bash
git diff --stat origin/main...HEAD
git diff --name-status origin/main...HEAD
```

Expected: the diff includes the spec and plan docs plus the metadata DB auth package move. It should not include managed-instance auth files under `backend/plugin/db`.

- [ ] **Step 4: Commit validation fixes if any**

If lint or tests required code changes, run:

```bash
git add backend/store backend/store/dbauth
git commit -m "fix(store): address metadata DB auth refactor validation"
```

If no validation fixes were needed, do not create an empty commit.
