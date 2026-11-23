//nolint:revive
package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestResolveTLSMaterialReadsPathFields(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(caPath, []byte("ca-pem"), 0o600))
	require.NoError(t, os.WriteFile(certPath, []byte("cert-pem"), 0o600))
	require.NoError(t, os.WriteFile(keyPath, []byte("key-pem"), 0o600))

	ds := &storepb.DataSource{
		UseSsl:      true,
		SslCa:       "stale-ca",
		SslCert:     "stale-cert",
		SslKey:      "stale-key",
		SslCaPath:   caPath,
		SslCertPath: certPath,
		SslKeyPath:  keyPath,
	}

	resolved, err := ResolveTLSMaterial(ds)
	require.NoError(t, err)
	require.Equal(t, "ca-pem", resolved.GetSslCa())
	require.Equal(t, "cert-pem", resolved.GetSslCert())
	require.Equal(t, "key-pem", resolved.GetSslKey())
	require.Equal(t, "stale-ca", ds.GetSslCa())
	require.Equal(t, caPath, ds.GetSslCaPath())
	require.Equal(t, certPath, ds.GetSslCertPath())
	require.Equal(t, keyPath, ds.GetSslKeyPath())
	require.Empty(t, resolved.GetSslCaPath())
	require.Empty(t, resolved.GetSslCertPath())
	require.Empty(t, resolved.GetSslKeyPath())
}

func TestResolveTLSMaterialPreservesInlineCrossSlotMaterial(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte("path-key"), 0o600))

	ds := &storepb.DataSource{
		UseSsl:     true,
		SslCa:      "inline-ca",
		SslCert:    "inline-cert",
		SslKeyPath: keyPath,
	}

	resolved, err := ResolveTLSMaterial(ds)
	require.NoError(t, err)
	require.NotSame(t, ds, resolved)
	require.Equal(t, "inline-ca", resolved.GetSslCa())
	require.Equal(t, "inline-cert", resolved.GetSslCert())
	require.Equal(t, "path-key", resolved.GetSslKey())
	require.Empty(t, resolved.GetSslKeyPath())
	require.Equal(t, keyPath, ds.GetSslKeyPath())
	require.Equal(t, "inline-ca", ds.GetSslCa())
	require.Equal(t, "inline-cert", ds.GetSslCert())
	require.Empty(t, ds.GetSslKey())
}

func TestResolveTLSMaterialRejectsRelativePath(t *testing.T) {
	_, err := ResolveTLSMaterial(&storepb.DataSource{
		UseSsl:    true,
		SslCaPath: "relative-ca.pem",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CA certificate path must be absolute")
	require.NotContains(t, err.Error(), "relative-ca.pem")
}

func TestResolveTLSMaterialRedactsReadErrors(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.pem")
	_, err := ResolveTLSMaterial(&storepb.DataSource{
		UseSsl:    true,
		SslCaPath: missingPath,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read CA certificate file")
	require.False(t, strings.Contains(err.Error(), missingPath), err.Error())
	require.NotContains(t, err.Error(), "no such file")
}

func TestResolveTLSMaterialRejectsNonRegularPath(t *testing.T) {
	dir := t.TempDir()

	_, err := ResolveTLSMaterial(&storepb.DataSource{
		UseSsl:    true,
		SslCaPath: dir,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CA certificate path must point to a regular file")
	require.NotContains(t, err.Error(), dir)
}

func TestResolveTLSMaterialRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(path, make([]byte, maxTLSMaterialFileSize+1), 0o600))

	_, err := ResolveTLSMaterial(&storepb.DataSource{
		UseSsl:    true,
		SslCaPath: path,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CA certificate file is too large")
	require.NotContains(t, err.Error(), path)
}

func TestGetTLSConfigRequiresResolvedTLSMaterial(t *testing.T) {
	_, err := GetTLSConfig(&storepb.DataSource{
		UseSsl:    true,
		SslCaPath: filepath.Join(t.TempDir(), "missing.pem"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "TLS material must be resolved before building TLS config")
	require.NotContains(t, err.Error(), "failed to read CA certificate file")
}

func TestGetTLSConfigIgnoresUnresolvedTLSMaterialWhenDisabled(t *testing.T) {
	tlsConfig, err := GetTLSConfig(&storepb.DataSource{
		UseSsl:    false,
		SslCaPath: filepath.Join(t.TempDir(), "missing.pem"),
	})
	require.NoError(t, err)
	require.Nil(t, tlsConfig)
}

func TestGetTLSConfigForHost(t *testing.T) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))

	signLeaf := func(serial int64, host string) []byte {
		leafKey, genErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, genErr)
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: host},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		if ip := net.ParseIP(host); ip != nil {
			tmpl.IPAddresses = []net.IP{ip}
		} else {
			tmpl.DNSNames = []string{host}
		}
		der, createErr := x509.CreateCertificate(rand.Reader, tmpl, caTemplate, &leafKey.PublicKey, caKey)
		require.NoError(t, createErr)
		return der
	}

	seedCert := signLeaf(2, "cluster.example.com")
	shardCert := signLeaf(3, "shard-01.cluster.example.com")
	ipv6Cert := signLeaf(4, "2001:db8::1")

	verifiedDS := &storepb.DataSource{
		Host:                 "cluster.example.com",
		UseSsl:               true,
		VerifyTlsCertificate: true,
		SslCa:                caPEM,
	}

	shardCfg, err := getTLSConfigForHost(verifiedDS, "shard-01.cluster.example.com")
	require.NoError(t, err)
	require.Equal(t, "shard-01.cluster.example.com", shardCfg.ServerName)
	require.NotNil(t, shardCfg.VerifyPeerCertificate)
	require.NoError(t, shardCfg.VerifyPeerCertificate([][]byte{shardCert}, nil))
	require.Error(t, shardCfg.VerifyPeerCertificate([][]byte{seedCert}, nil))

	fallbackCfg, err := getTLSConfigForHost(verifiedDS, "")
	require.NoError(t, err)
	require.Equal(t, "", fallbackCfg.ServerName)
	require.NoError(t, fallbackCfg.VerifyPeerCertificate([][]byte{seedCert}, nil))
	require.Error(t, fallbackCfg.VerifyPeerCertificate([][]byte{shardCert}, nil))

	ipv6Cfg, err := getTLSConfigForHost(verifiedDS, "[2001:db8::1]")
	require.NoError(t, err)
	require.Equal(t, "2001:db8::1", ipv6Cfg.ServerName)
	require.NoError(t, ipv6Cfg.VerifyPeerCertificate([][]byte{ipv6Cert}, nil))

	unverifiedDS := &storepb.DataSource{
		Host:                 "cluster.example.com",
		UseSsl:               true,
		VerifyTlsCertificate: false,
	}
	unverifiedCfg, err := getTLSConfigForHost(unverifiedDS, "shard-01.cluster.example.com")
	require.NoError(t, err)
	require.Equal(t, "shard-01.cluster.example.com", unverifiedCfg.ServerName)
	require.True(t, unverifiedCfg.InsecureSkipVerify)
	require.Nil(t, unverifiedCfg.VerifyPeerCertificate)
}
