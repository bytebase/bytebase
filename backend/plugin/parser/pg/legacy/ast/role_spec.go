package ast

// RoleSpecType is the type of a role specification.
type RoleSpecType int

const (
	// RoleSpecTypeNone is the default value for RoleSpecType.
	RoleSpecTypeNone RoleSpecType = iota
	// RoleSpecTypeUser is a user role.
	RoleSpecTypeUser
	// RoleSpecTypeCurrentRole is CURRENT_ROLE.
	RoleSpecTypeCurrentRole
	// RoleSpecTypeCurrentUser is CURRENT_USER.
	RoleSpecTypeCurrentUser
	// RoleSpecTypeSessionUser is SESSION_USER.
	RoleSpecTypeSessionUser
)

// RoleSpec is the struct for role specification.
type RoleSpec struct {
	Type RoleSpecType
	// Value only used when Tp is RoleSpecTypeUser.
	Value string
}
