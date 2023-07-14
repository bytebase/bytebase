package ldap

import (
	"testing"

	"github.com/stretchr/testify/assert"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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
