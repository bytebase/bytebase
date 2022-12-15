package v1

import (
	"fmt"
	"strings"
)

// PGRoleAttributeStr is the attribute string for role.
type PGRoleAttributeStr string

const (
	// SUPERUSER is the role attribute for rolsuper.
	SUPERUSER PGRoleAttributeStr = "SUPERUSER"
	// LOGIN is the role attribute for rolcanlogin.
	LOGIN PGRoleAttributeStr = "LOGIN"
	// NOINHERIT is the role attribute for rolinherit.
	// INHERIT is the default value for rolinherit, so we need to use the NOINHERIT.
	NOINHERIT PGRoleAttributeStr = "NOINHERIT"
	// CREATEDB is the role attribute for rolcreatedb.
	CREATEDB PGRoleAttributeStr = "CREATEDB"
	// CREATEROLE is the role attribute for rolcreaterole.
	CREATEROLE PGRoleAttributeStr = "CREATEROLE"
	// REPLICATION is the role attribute for rolreplication.
	REPLICATION PGRoleAttributeStr = "REPLICATION"
	// BYPASSRLS is the role attribute for rolbypassrls.
	BYPASSRLS PGRoleAttributeStr = "BYPASSRLS"
)

// ToString returns the string value for role attribute.
func (a PGRoleAttributeStr) ToString() string {
	return string(a)
}

// PGRoleAttribute is the attribute for role.
type PGRoleAttribute struct {
	SuperUser   bool `json:"superUser"`
	NoInherit   bool `json:"noInherit"`
	CreateRole  bool `json:"createRole"`
	CreateDB    bool `json:"createDB"`
	CanLogin    bool `json:"canLogin"`
	Replication bool `json:"replication"`
	ByPassRLS   bool `json:"byPassRLS"`
}

// PGRole is the API message for role.
type PGRole struct {
	Name            string           `json:"name"`
	InstanceID      int              `json:"instanceId"`
	ConnectionLimit int              `json:"connectionLimit"`
	ValidUntil      *string          `json:"validUntil"`
	Attribute       *PGRoleAttribute `json:"attribute"`
}

// PGRoleUpsert is the API message for upserting a new role.
type PGRoleUpsert struct {
	Name            string           `json:"name"`
	Password        *string          `json:"password"`
	ConnectionLimit *int             `json:"connectionLimit"`
	ValidUntil      *string          `json:"validUntil"`
	Attribute       *PGRoleAttribute `json:"attribute"`
}

// ToAttributeStatement returns the attribute statemnt to create or update the role.
func (r *PGRoleUpsert) ToAttributeStatement() string {
	attributeList := []string{}

	if r.Attribute != nil {
		if r.Attribute.SuperUser {
			attributeList = append(attributeList, SUPERUSER.ToString())
		}
		if r.Attribute.NoInherit {
			attributeList = append(attributeList, NOINHERIT.ToString())
		}
		if r.Attribute.CanLogin {
			attributeList = append(attributeList, LOGIN.ToString())
		}
		if r.Attribute.CreateRole {
			attributeList = append(attributeList, CREATEROLE.ToString())
		}
		if r.Attribute.CreateDB {
			attributeList = append(attributeList, CREATEDB.ToString())
		}
		if r.Attribute.Replication {
			attributeList = append(attributeList, REPLICATION.ToString())
		}
		if r.Attribute.ByPassRLS {
			attributeList = append(attributeList, BYPASSRLS.ToString())
		}
	}

	if v := r.Password; v != nil {
		attributeList = append(attributeList, fmt.Sprintf("ENCRYPTED PASSWORD '%s'", *v))
	}
	if v := r.ValidUntil; v != nil {
		attributeList = append(attributeList, fmt.Sprintf("VALID UNTIL '%s'", *v))
	}
	if v := r.ConnectionLimit; v != nil {
		attributeList = append(attributeList, fmt.Sprintf("CONNECTION LIMIT %d", *v))
	}

	attribute := ""
	if len(attributeList) > 0 {
		attribute = fmt.Sprintf("WITH %s", strings.Join(attributeList, " "))
	}

	return attribute
}
