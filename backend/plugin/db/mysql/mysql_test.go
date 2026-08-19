package mysql

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestValidateMySQLExtraConnectionParameters(t *testing.T) {
	tests := []struct {
		name      string
		params    map[string]string
		wantError bool
	}{
		{
			name:      "empty parameters",
			params:    map[string]string{},
			wantError: false,
		},
		{
			name: "safe parameters",
			params: map[string]string{
				"timeout":      "10s",
				"readTimeout":  "30s",
				"writeTimeout": "30s",
			},
			wantError: false,
		},
		{
			name: "dangerous parameter - allowAllFiles",
			params: map[string]string{
				"allowAllFiles": "true",
			},
			wantError: true,
		},
		{
			name: "dangerous parameter - allowAllFiles lowercase",
			params: map[string]string{
				"allowallfiles": "true",
			},
			wantError: true,
		},
		{
			name: "dangerous parameter - allowAllFiles with mixed case",
			params: map[string]string{
				"AllowAllFiles": "true",
			},
			wantError: true,
		},
		{
			name: "mixed safe and dangerous parameters",
			params: map[string]string{
				"timeout":       "10s",
				"allowAllFiles": "true",
			},
			wantError: true,
		},
		{
			name: "parameter with whitespace",
			params: map[string]string{
				"  allowAllFiles  ": "true",
			},
			wantError: true,
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		t.Run(tc.name, func(_ *testing.T) {
			err := validateMySQLExtraConnectionParameters(tc.params)
			if tc.wantError {
				a.Error(err, "expected error for test case: %s", tc.name)
			} else {
				a.NoError(err, "expected no error for test case: %s", tc.name)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version  string
		want     string
		wantRest string
	}{
		{
			version:  "8.0.27",
			want:     "8.0.27",
			wantRest: "",
		},
		{
			version:  "5.7.22-log",
			want:     "5.7.22",
			wantRest: "-log",
		},
		{
			version:  "5.6.29_ddm_3.0.1.7",
			want:     "5.6.29",
			wantRest: "_ddm_3.0.1.7",
		},
		{
			version:  "10.4.7-MariaDB",
			want:     "10.4.7",
			wantRest: "-MariaDB",
		},
	}

	a := require.New(t)
	for _, tc := range tests {
		version, rest, err := parseVersion(tc.version)
		a.NoError(err)
		a.Equal(tc.want, version)
		a.Equal(tc.wantRest, rest)
	}
}

func TestBuildExecuteCommandsNormalizesDelimiter(t *testing.T) {
	statement := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\nDELIMITER ;\n"

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.NotContains(t, commands[0].Text, "DELIMITER")
	require.Contains(t, commands[0].Text, "CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND")
}

func TestBuildExecuteCommandsDoesNotNormalizeDelimiterForTooManyCommands(t *testing.T) {
	var statement strings.Builder
	statement.WriteString("DELIMITER //\n")
	statement.WriteString("CREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\n")
	statement.WriteString("DELIMITER ;\n")
	statement.WriteString("/*!50003 SET @OLD_SQL_MODE=@@SQL_MODE */;\n")
	for i := 0; i < common.MaximumCommands; i++ {
		statement.WriteString("SELECT 1;\n")
	}

	commands, err := buildExecuteCommands(statement.String())
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, statement.String(), commands[0].Text)
}

func TestBuildExecuteCommandsDoesNotNormalizeDelimiterForLargeSheet(t *testing.T) {
	statement := "DELIMITER //\nCREATE PROCEDURE p()\nBEGIN\n  SELECT 1;\nEND//\nDELIMITER ;\n" +
		strings.Repeat(" ", common.MaxSheetCheckSize)

	commands, err := buildExecuteCommands(statement)
	require.NoError(t, err)
	require.Len(t, commands, 1)
	require.Equal(t, statement, commands[0].Text)
}

// selfSignedCAPEM builds a throwaway CA so the tests can assert that a supplied
// ssl_ca is parsed and that a served bundle is cached, without shipping a
// certificate that eventually expires.
func selfSignedCAPEM(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "bytebase-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// serveRDSCertBundle points rdsCertBundleURL at a local server and returns the
// request counter, so a test can assert that no download happened at all.
func serveRDSCertBundle(t *testing.T, handler http.HandlerFunc) *atomic.Int32 {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	previousURL := rdsCertBundleURL
	rdsCertBundleURL = server.URL
	rdsCertPool.Store(nil)
	t.Cleanup(func() {
		rdsCertBundleURL = previousURL
		rdsCertPool.Store(nil)
	})
	return &requests
}

func TestRDSTLSConfigSkipsDownloadWhenVerificationDisabled(t *testing.T) {
	ca := selfSignedCAPEM(t)
	requests := serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(ca))
	})

	cfg, err := rdsTLSConfig(context.Background(), &storepb.DataSource{
		Host:                 "db.example.com",
		VerifyTlsCertificate: false,
	})
	require.NoError(t, err)
	require.True(t, cfg.InsecureSkipVerify)
	require.Nil(t, cfg.RootCAs)
	require.Nil(t, cfg.VerifyPeerCertificate)
	require.Equal(t, int32(0), requests.Load())
}

func TestRDSTLSConfigSkipsDownloadWhenSslCaConfigured(t *testing.T) {
	ca := selfSignedCAPEM(t)
	requests := serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(ca))
	})

	cfg, err := rdsTLSConfig(context.Background(), &storepb.DataSource{
		Host:                 "db.example.com",
		VerifyTlsCertificate: true,
		SslCa:                ca,
	})
	require.NoError(t, err)
	require.NotNil(t, cfg.RootCAs)
	require.NotNil(t, cfg.VerifyPeerCertificate)
	require.Equal(t, int32(0), requests.Load())
}

func TestRDSTLSConfigDownloadsBundleOnlyOnce(t *testing.T) {
	ca := selfSignedCAPEM(t)
	requests := serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(ca))
	})

	dataSource := &storepb.DataSource{Host: "db.example.com", VerifyTlsCertificate: true}
	for range 5 {
		cfg, err := rdsTLSConfig(context.Background(), dataSource)
		require.NoError(t, err)
		require.NotNil(t, cfg.RootCAs)
	}
	require.Equal(t, int32(1), requests.Load())
}

func TestRDSCertPoolDoesNotCacheFailures(t *testing.T) {
	ca := selfSignedCAPEM(t)
	var fail atomic.Bool
	fail.Store(true)
	requests := serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(ca))
	})

	_, err := cachedRDSCertPool(context.Background())
	require.Error(t, err)

	fail.Store(false)
	pool, err := cachedRDSCertPool(context.Background())
	require.NoError(t, err)
	require.NotNil(t, pool)
	require.Equal(t, int32(2), requests.Load())
}

func TestRDSCertPoolReportsHTTPStatus(t *testing.T) {
	serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<Error>AccessDenied</Error>"))
	})

	_, err := getRDSCertPool(context.Background())
	require.ErrorContains(t, err, "403")
}

func TestRDSRootCAsRejectsUnparseableSslCa(t *testing.T) {
	_, err := rdsRootCAs(context.Background(), &storepb.DataSource{SslCa: "not a certificate"})
	require.ErrorContains(t, err, "ssl_ca")
}

func TestRDSCertPoolRejectsOversizedResponse(t *testing.T) {
	serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), int(maxRDSCertBundleSize)+1024))
	})
	_, err := getRDSCertPool(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds")
}

func TestCreateCertificateVerifierValidatesChainAndHostname(t *testing.T) {
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "bytebase-test-root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	require.NoError(t, err)
	rootCert, err := x509.ParseCertificate(rootDER)
	require.NoError(t, err)
	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCert)

	foreignKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	foreignTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "bytebase-test-foreign-root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	foreignDER, err := x509.CreateCertificate(rand.Reader, foreignTemplate, foreignTemplate, &foreignKey.PublicKey, foreignKey)
	require.NoError(t, err)
	foreignCert, err := x509.ParseCertificate(foreignDER)
	require.NoError(t, err)

	// leafFor returns a DER-encoded leaf signed by parent, valid for dnsName.
	leafFor := func(parent *x509.Certificate, parentKey *ecdsa.PrivateKey, dnsName string) []byte {
		leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		leaf := &x509.Certificate{
			SerialNumber: big.NewInt(100),
			Subject:      pkix.Name{CommonName: dnsName},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			DNSNames:     []string{dnsName},
		}
		der, err := x509.CreateCertificate(rand.Reader, leaf, parent, &leafKey.PublicKey, parentKey)
		require.NoError(t, err)
		return der
	}

	const host = "db.example.com"
	verifier := util.CreateCertificateVerifier(rootPool, host)

	require.NoError(t, verifier([][]byte{leafFor(rootCert, rootKey, host)}, nil))
	require.Error(t, verifier([][]byte{leafFor(rootCert, rootKey, "other.example.com")}, nil))
	require.Error(t, verifier([][]byte{leafFor(foreignCert, foreignKey, host)}, nil))
}

func TestRDSCertPoolCollapsesConcurrentDownloads(t *testing.T) {
	const callers = 32
	ca := selfSignedCAPEM(t)
	// release blocks the first response until every caller is inside the fetch,
	// so a per-caller download cannot pass by finishing before the others start.
	release := make(chan struct{})
	requests := serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		<-release
		_, _ = w.Write([]byte(ca))
	})

	dataSource := &storepb.DataSource{Host: "db.example.com", VerifyTlsCertificate: true}
	var ready, done sync.WaitGroup
	ready.Add(callers)
	done.Add(callers)
	pools := make([]*x509.CertPool, callers)
	errs := make([]error, callers)
	for i := range callers {
		go func() {
			defer done.Done()
			ready.Done()
			cfg, err := rdsTLSConfig(context.Background(), dataSource)
			errs[i] = err
			if err == nil {
				pools[i] = cfg.RootCAs
			}
		}()
	}
	ready.Wait()
	close(release)
	done.Wait()

	for i := range callers {
		require.NoError(t, errs[i])
		require.NotNil(t, pools[i])
		require.Same(t, pools[0], pools[i])
	}
	require.Equal(t, int32(1), requests.Load())
}

func TestRDSCertPoolWaiterObservesItsOwnCancellation(t *testing.T) {
	ca := selfSignedCAPEM(t)
	release := make(chan struct{})
	inFlight := make(chan struct{}, 1)
	var leader sync.WaitGroup
	// Registered before serveRDSCertBundle, so it runs last: the leader must finish
	// before the helper resets rdsCertPool, or it re-stores this test's pool
	// afterwards and a later cache test never contacts its own server.
	t.Cleanup(leader.Wait)
	serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		inFlight <- struct{}{}
		<-release
		_, _ = w.Write([]byte(ca))
	})
	// Registered after serveRDSCertBundle so this runs before its server.Close,
	// which would otherwise wait forever on the blocked handler.
	t.Cleanup(func() { close(release) })

	dataSource := &storepb.DataSource{Host: "db.example.com", VerifyTlsCertificate: true}
	// Leader occupies the flight and stays blocked in the handler.
	leader.Add(1)
	go func() {
		defer leader.Done()
		_, _ = cachedRDSCertPool(context.Background())
	}()
	<-inFlight

	// The waiter joins the in-flight download, then gives up on its own context.
	ctx, cancel := context.WithCancel(context.Background())
	waiterDone := make(chan error, 1)
	leader.Add(1)
	go func() {
		defer leader.Done()
		_, err := rdsTLSConfig(ctx, dataSource)
		waiterDone <- err
	}()
	cancel()

	select {
	case err := <-waiterDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		require.Fail(t, "canceled waiter blocked on the shared download")
	}
}

func TestRDSCertPoolLeaderCancellationDoesNotFailWaiter(t *testing.T) {
	ca := selfSignedCAPEM(t)
	release := make(chan struct{})
	inFlight := make(chan struct{}, 1)
	var leader sync.WaitGroup
	// See the note in TestRDSCertPoolWaiterObservesItsOwnCancellation: this runs
	// last, so no background store lands after the cache reset.
	t.Cleanup(leader.Wait)
	serveRDSCertBundle(t, func(w http.ResponseWriter, _ *http.Request) {
		inFlight <- struct{}{}
		<-release
		_, _ = w.Write([]byte(ca))
	})

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	leader.Add(1)
	go func() {
		defer leader.Done()
		_, _ = cachedRDSCertPool(leaderCtx)
	}()
	<-inFlight

	// A second caller with a live context is waiting on the same flight.
	waiterDone := make(chan error, 1)
	leader.Add(1)
	go func() {
		defer leader.Done()
		_, err := cachedRDSCertPool(context.Background())
		waiterDone <- err
	}()

	// The leader walks away; the shared fetch must still complete for the waiter.
	cancelLeader()
	close(release)

	select {
	case err := <-waiterDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		require.Fail(t, "waiter never completed after leader cancellation")
	}
}
