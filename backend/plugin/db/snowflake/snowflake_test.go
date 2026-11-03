// Package snowflake is the plugin for Snowflake driver.
package snowflake

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/youmark/pkcs8"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestBuildSnowflakeDSN(t *testing.T) {
	testCases := []struct {
		input db.ConnectionConfig
		want  string
	}{
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1",
					Port:     "443",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
			},
			want: "bytebase:pwd@nb47110.ap-southeast-1.snowflakecomputing.com:443?ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1",
					Port:     "443",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "SAMPLE_DB",
				},
			},
			want: "bytebase:pwd@nb47110.ap-southeast-1.snowflakecomputing.com:443?database=%22SAMPLE_DB%22&ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1@10.0.0.1",
					Port:     "4182",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
			},
			want: "bytebase:pwd@10.0.0.1:443?account=nb47110&ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1@10.0.0.1",
					Port:     "4182",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "SAMPLE_DB",
				},
			},
			want: "bytebase:pwd@10.0.0.1:443?account=nb47110&database=%22SAMPLE_DB%22&ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
	}
	for _, testCase := range testCases {
		got, _ /* redacted */, err := buildSnowflakeDSN(testCase.input)
		require.NoError(t, err)
		require.Equal(t, testCase.want, got)
	}
}

func TestDecodeRSAPrivateKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	unencrypted, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	encrypted, err := pkcs8.MarshalPrivateKey(key, []byte("passphrase"), nil)
	require.NoError(t, err)
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	notRSA, err := x509.MarshalPKCS8PrivateKey(edKey)
	require.NoError(t, err)

	encodePEM := func(blockType string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}))
	}

	testCases := []struct {
		name       string
		key        string
		passphrase string
		wantErr    string
	}{
		{
			name: "unencrypted key",
			key:  encodePEM("PRIVATE KEY", unencrypted),
		},
		{
			name:       "unencrypted key with an unnecessary passphrase",
			key:        encodePEM("PRIVATE KEY", unencrypted),
			passphrase: "passphrase",
		},
		{
			name:       "encrypted key with its passphrase",
			key:        encodePEM("ENCRYPTED PRIVATE KEY", encrypted),
			passphrase: "passphrase",
		},
		{
			name:    "encrypted key without a passphrase",
			key:     encodePEM("ENCRYPTED PRIVATE KEY", encrypted),
			wantErr: "private key is encrypted, but no passphrase was provided",
		},
		{
			name:       "encrypted key with the wrong passphrase",
			key:        encodePEM("ENCRYPTED PRIVATE KEY", encrypted),
			passphrase: "wrong",
			wantErr:    "incorrect password",
		},
		{
			name:    "key that is not RSA",
			key:     encodePEM("PRIVATE KEY", notRSA),
			wantErr: "expected RSA private key",
		},
		{
			name:    "PKCS#1 block",
			key:     encodePEM("RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key)),
			wantErr: "unsupported pem block type: RSA PRIVATE KEY",
		},
		{
			name:    "not PEM at all",
			key:     "rsa_key.p8",
			wantErr: "failed to get private key PEM block",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := decodeRSAPrivateKey(testCase.key, testCase.passphrase)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.True(t, got.Equal(key))
		})
	}
}
