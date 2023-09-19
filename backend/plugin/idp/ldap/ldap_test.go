package ldap

import (
	"crypto/tls"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/lor00x/goldap/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vjeantet/ldapserver"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Verbose() {
		ldapserver.Logger = ldapserver.DiscardingLogger
	}
	os.Exit(m.Run())
}

func TestNewIdentityProvider(t *testing.T) {
	tests := []struct {
		name        string
		config      IdentityProviderConfig
		containsErr string
	}{
		{
			name: "no security protocol",
			config: IdentityProviderConfig{
				Host:             "ldap.example.com",
				BindDN:           "uid=system,ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				BindPassword:     "pa$$word",
				BaseDN:           "ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				UserFilter:       "(&(objectClass=posixAccount)(uid=%s))",
				SecurityProtocol: "",
				FieldMapping: &storepb.FieldMapping{
					Identifier: "uid",
				},
			},
			containsErr: `the field "securityProtocol" must be either "starttls" or "ldaps"`,
		},
		{
			name: "no host",
			config: IdentityProviderConfig{
				Host:             "",
				BindDN:           "uid=system,ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				BindPassword:     "pa$$word",
				BaseDN:           "ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				UserFilter:       "(&(objectClass=posixAccount)(uid=%s))",
				SecurityProtocol: SecurityProtocolStartTLS,
				FieldMapping: &storepb.FieldMapping{
					Identifier: "uid",
				},
			},
			containsErr: `the field "host" is empty but required`,
		},
		{
			name: "no bindDn",
			config: IdentityProviderConfig{
				Host:             "ldap.example.com",
				BindDN:           "",
				BindPassword:     "pa$$word",
				BaseDN:           "ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				UserFilter:       "(&(objectClass=posixAccount)(uid=%s))",
				SecurityProtocol: SecurityProtocolStartTLS,
				FieldMapping: &storepb.FieldMapping{
					Identifier: "uid",
				},
			},
			containsErr: `the field "bindDn" is empty but required`,
		},
		{
			name: "no bindPassword",
			config: IdentityProviderConfig{
				Host:             "ldap.example.com",
				BindDN:           "uid=system,ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				BindPassword:     "",
				BaseDN:           "ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				UserFilter:       "(&(objectClass=posixAccount)(uid=%s))",
				SecurityProtocol: SecurityProtocolStartTLS,
				FieldMapping: &storepb.FieldMapping{
					Identifier: "uid",
				},
			},
			containsErr: `the field "bindPassword" is empty but required`,
		},
		{
			name: "no baseDn",
			config: IdentityProviderConfig{
				Host:             "ldap.example.com",
				BindDN:           "uid=system,ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				BindPassword:     "pa$$word",
				BaseDN:           "",
				UserFilter:       "(&(objectClass=posixAccount)(uid=%s))",
				SecurityProtocol: SecurityProtocolStartTLS,
				FieldMapping: &storepb.FieldMapping{
					Identifier: "uid",
				},
			},
			containsErr: `the field "baseDn" is empty but required`,
		},
		{
			name: "no userFilter",
			config: IdentityProviderConfig{
				Host:             "ldap.example.com",
				BindDN:           "uid=system,ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				BindPassword:     "pa$$word",
				BaseDN:           "ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				UserFilter:       "",
				SecurityProtocol: SecurityProtocolStartTLS,
				FieldMapping: &storepb.FieldMapping{
					Identifier: "uid",
				},
			},
			containsErr: `the field "userFilter" is empty but required`,
		},
		{
			name: "no fieldMapping.identifier",
			config: IdentityProviderConfig{
				Host:             "ldap.example.com",
				BindDN:           "uid=system,ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				BindPassword:     "pa$$word",
				BaseDN:           "ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
				UserFilter:       "(&(objectClass=posixAccount)(uid=%s))",
				SecurityProtocol: SecurityProtocolStartTLS,
				FieldMapping: &storepb.FieldMapping{
					DisplayName: "displayName",
				},
			},
			containsErr: `the field "fieldMapping.identifier" is empty but required`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewIdentityProvider(test.config)
			assert.ErrorContains(t, err, test.containsErr)
		})
	}
}

func newMockServer(t *testing.T, uid, displayName, mail string) (host string, port int) {
	// localhostCert is a PEM-encoded TLS cert with SAN IPs
	// "127.0.0.1" and "[::1]", expiring at Jan 29 16:00:00 2084 GMT.
	// generated from src/crypto/tls:
	// go run generate_cert.go  --rsa-bits 2048 --host 127.0.0.1,::1,example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
	var localhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIIDOTCCAiGgAwIBAgIQSRJrEpBGFc7tNb1fb5pKFzANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEA6Gba5tHV1dAKouAaXO3/ebDUU4rvwCUg/CNaJ2PT5xLD4N1Vcb8r
bFSW2HXKq+MPfVdwIKR/1DczEoAGf/JWQTW7EgzlXrCd3rlajEX2D73faWJekD0U
aUgz5vtrTXZ90BQL7WvRICd7FlEZ6FPOcPlumiyNmzUqtwGhO+9ad1W5BqJaRI6P
YfouNkwR6Na4TzSj5BrqUfP0FwDizKSJ0XXmh8g8G9mtwxOSN3Ru1QFc61Xyeluk
POGKBV/q6RBNklTNe0gI8usUMlYyoC7ytppNMW7X2vodAelSu25jgx2anj9fDVZu
h7AXF5+4nJS4AAt0n1lNY7nGSsdZas8PbQIDAQABo4GIMIGFMA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBStsdjh3/JCXXYlQryOrL4Sh7BW5TAuBgNVHREEJzAlggtleGFtcGxlLmNv
bYcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOCAQEAxWGI
5NhpF3nwwy/4yB4i/CwwSpLrWUa70NyhvprUBC50PxiXav1TeDzwzLx/o5HyNwsv
cxv3HdkLW59i/0SlJSrNnWdfZ19oTcS+6PtLoVyISgtyN6DpkKpdG1cOkW3Cy2P2
+tK/tKHRP1Y/Ra0RiDpOAmqn0gCOFGz8+lqDIor/T7MTpibL3IxqWfPrvfVRHL3B
grw/ZQTTIVjjh4JBSW3WyWgNo/ikC1lrVxzl4iPUGptxT36Cr7Zk2Bsg0XqwbOvK
5d+NTDREkSnUbie4GeutujmX3Dsx88UiV6UY/4lHJa6I5leHUNOHahRbpbWeOfs/
WkBKOclmOV2xlTVuPw==
-----END CERTIFICATE-----`)

	// localhostKey is the private key for localhostCert.
	var localhostKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDoZtrm0dXV0Aqi
4Bpc7f95sNRTiu/AJSD8I1onY9PnEsPg3VVxvytsVJbYdcqr4w99V3AgpH/UNzMS
gAZ/8lZBNbsSDOVesJ3euVqMRfYPvd9pYl6QPRRpSDPm+2tNdn3QFAvta9EgJ3sW
URnoU85w+W6aLI2bNSq3AaE771p3VbkGolpEjo9h+i42TBHo1rhPNKPkGupR8/QX
AOLMpInRdeaHyDwb2a3DE5I3dG7VAVzrVfJ6W6Q84YoFX+rpEE2SVM17SAjy6xQy
VjKgLvK2mk0xbtfa+h0B6VK7bmODHZqeP18NVm6HsBcXn7iclLgAC3SfWU1jucZK
x1lqzw9tAgMBAAECggEABWzxS1Y2wckblnXY57Z+sl6YdmLV+gxj2r8Qib7g4ZIk
lIlWR1OJNfw7kU4eryib4fc6nOh6O4AWZyYqAK6tqNQSS/eVG0LQTLTTEldHyVJL
dvBe+MsUQOj4nTndZW+QvFzbcm2D8lY5n2nBSxU5ypVoKZ1EqQzytFcLZpTN7d89
EPj0qDyrV4NZlWAwL1AygCwnlwhMQjXEalVF1ylXwU3QzyZ/6MgvF6d3SSUlh+sq
XefuyigXw484cQQgbzopv6niMOmGP3of+yV4JQqUSb3IDmmT68XjGd2Dkxl4iPki
6ZwXf3CCi+c+i/zVEcufgZ3SLf8D99kUGE7v7fZ6AQKBgQD1ZX3RAla9hIhxCf+O
3D+I1j2LMrdjAh0ZKKqwMR4JnHX3mjQI6LwqIctPWTU8wYFECSh9klEclSdCa64s
uI/GNpcqPXejd0cAAdqHEEeG5sHMDt0oFSurL4lyud0GtZvwlzLuwEweuDtvT9cJ
Wfvl86uyO36IW8JdvUprYDctrQKBgQDycZ697qutBieZlGkHpnYWUAeImVA878sJ
w44NuXHvMxBPz+lbJGAg8Cn8fcxNAPqHIraK+kx3po8cZGQywKHUWsxi23ozHoxo
+bGqeQb9U661TnfdDspIXia+xilZt3mm5BPzOUuRqlh4Y9SOBpSWRmEhyw76w4ZP
OPxjWYAgwQKBgA/FehSYxeJgRjSdo+MWnK66tjHgDJE8bYpUZsP0JC4R9DL5oiaA
brd2fI6Y+SbyeNBallObt8LSgzdtnEAbjIH8uDJqyOmknNePRvAvR6mP4xyuR+Bv
m+Lgp0DMWTw5J9CKpydZDItc49T/mJ5tPhdFVd+am0NAQnmr1MCZ6nHxAoGABS3Y
LkaC9FdFUUqSU8+Chkd/YbOkuyiENdkvl6t2e52jo5DVc1T7mLiIrRQi4SI8N9bN
/3oJWCT+uaSLX2ouCtNFunblzWHBrhxnZzTeqVq4SLc8aESAnbslKL4i8/+vYZlN
s8xtiNcSvL+lMsOBORSXzpj/4Ot8WwTkn1qyGgECgYBKNTypzAHeLE6yVadFp3nQ
Ckq9yzvP/ib05rvgbvrne00YeOxqJ9gtTrzgh7koqJyX1L4NwdkEza4ilDWpucn0
xiUZS4SoaJq6ZvcBYS62Yr1t8n09iG47YL8ibgtmH3L+svaotvpVxVK+d7BLevA/
ZboOWVe3icTy64BT3OQhmg==
-----END RSA PRIVATE KEY-----`)
	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	require.NoError(t, err)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "127.0.0.1",
	}

	server := ldapserver.NewServer()
	routes := ldapserver.NewRouteMux()
	routes.Bind(func(w ldapserver.ResponseWriter, m *ldapserver.Message) {
		w.Write(ldapserver.NewBindResponse(ldapserver.LDAPResultSuccess))
	})
	routes.Search(func(w ldapserver.ResponseWriter, m *ldapserver.Message) {
		e := ldapserver.NewSearchResultEntry(uid)
		e.AddAttribute("uid", message.AttributeValue(uid))
		e.AddAttribute("displayName", message.AttributeValue(displayName))
		e.AddAttribute("mail", message.AttributeValue(mail))
		w.Write(e)
		w.Write(ldapserver.NewSearchResultDoneResponse(ldapserver.LDAPResultSuccess))
	})
	server.Handle(routes)

	go func() {
		err := server.ListenAndServe(
			"127.0.0.1:10389",
			func(s *ldapserver.Server) {
				s.Listener = tls.NewListener(s.Listener, tlsConfig)
			},
		)
		require.NoError(t, err)
	}()
	t.Cleanup(func() { server.Stop() })

	// Give a second for the server to start
	time.Sleep(time.Second)
	return "127.0.0.1", 10389
}

func TestIdentityProvider(t *testing.T) {
	const (
		testUID         = "alice"
		testDisplayName = "Alice Smith"
		testMail        = "alice@example.com"
	)
	host, port := newMockServer(t, testUID, testDisplayName, testMail)
	ldap, err := NewIdentityProvider(
		IdentityProviderConfig{
			Host:             host,
			Port:             port,
			SkipTLSVerify:    true,
			BindDN:           "uid=system,ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
			BindPassword:     "pa$$word",
			BaseDN:           "ou=Users,o=6456a5e9c25dabb51ccad385,dc=example,dc=com",
			UserFilter:       "(&(objectClass=posixAccount)(uid=%s))",
			SecurityProtocol: SecurityProtocolLDAPS,
			FieldMapping: &storepb.FieldMapping{
				Identifier:  "uid",
				DisplayName: "displayName",
				Email:       "mail",
			},
		},
	)
	require.NoError(t, err)

	userInfo, err := ldap.Authenticate("alice", "pa$$word")
	require.NoError(t, err)

	wantUserInfo := &storepb.IdentityProviderUserInfo{
		Identifier:  testUID,
		DisplayName: testDisplayName,
		Email:       testMail,
	}
	assert.Equal(t, wantUserInfo, userInfo)
}
