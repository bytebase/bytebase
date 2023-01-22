package api

// MemberStatus is the status of an member.
type MemberStatus string

const (
	// Unknown is the member status for UNKNOWN.
	Unknown MemberStatus = "UNKNOWN"
	// Invited is the member status for INVITED.
	Invited MemberStatus = "INVITED"
	// Active is the member status for ACTIVE.
	Active MemberStatus = "ACTIVE"
)

// Role is the type of a role.
type Role string

const (
	// Owner is the OWNER role.
	Owner Role = "OWNER"
	// DBA is the DBA role.
	DBA Role = "DBA"
	// Developer is the DEVELOPER role.
	Developer Role = "DEVELOPER"
	// UnknownRole is the unknown role.
	UnknownRole Role = "UNKNOWN"
)

// Member is the API message for a member.
type Member struct {
	ID int `jsonapi:"primary,member"`

	RowStatus RowStatus `jsonapi:"attr,rowStatus"`

	// Domain specific fields
	Status      MemberStatus `jsonapi:"attr,status"`
	Role        Role         `jsonapi:"attr,role"`
	PrincipalID int
	Principal   *Principal `jsonapi:"relation,principal"`
}

// MemberCreate is the API message for creating a member.
type MemberCreate struct {
	// Domain specific fields
	Role        Role `jsonapi:"attr,role"`
	PrincipalID int  `jsonapi:"attr,principalId"`
}

// MemberPatch is the API message for patching a member.
type MemberPatch struct {
	ID int

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Domain specific fields
	Role *string `jsonapi:"attr,role"`
}
