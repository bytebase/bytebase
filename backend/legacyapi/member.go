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
	// Exporter is the EXPORTER role.
	Exporter Role = "EXPORTER"
	// Querier is the QUERIER role.
	Querier Role = "QUERIER"
	// Releaser is the RELEASER role.
	Releaser Role = "RELEASER"
	// UnknownRole is the unknown role.
	UnknownRole Role = "UNKNOWN"
)

func (r Role) String() string {
	return string(r)
}
