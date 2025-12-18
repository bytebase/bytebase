package directorysync

type SCIMUserEmail struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type SCIMResourceMeta struct {
	ResourceType string `json:"resourceType"`
}

// SCIMUser represents the SCIM 2.0 User schema for identity provider provisioning.
// Supports both Azure Entra ID and Okta SCIM clients.
// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#design-your-user-and-group-schema
// Docs: https://developer.okta.com/docs/reference/scim/scim-20/
//
// SCIM ID mapping for users:
//   - id: Bytebase's user UID (numeric, returned as string). Used by IdP in subsequent API calls.
//   - externalId: IdP's user identifier (optional). We use userName (email) for user matching instead.
//   - userName: User's email address. Primary identifier for user lookup.
//
// Unlike groups, users are matched by userName (email) rather than externalId because:
//   - Email is the natural unique identifier for users in Bytebase
//   - Both Azure and Okta default attribute mappings use email -> userName
type SCIMUser struct {
	// id is Bytebase's user UID, used by IdP in subsequent requests (GET/PATCH/DELETE /Users/{id}).
	ID      string   `json:"id"`
	Schemas []string `json:"schemas"`
	// externalId is the IdP's user identifier. We don't use this for user matching; we use userName instead.
	ExternalID string `json:"externalId"`
	// userName is the user's email address. This is the primary identifier for user lookup.
	UserName    string            `json:"userName"`
	Active      bool              `json:"active"`
	DisplayName string            `json:"displayName"`
	Emails      []*SCIMUserEmail  `json:"emails"`
	Meta        *SCIMResourceMeta `json:"meta"`
}

// SCIMGroup represents the SCIM 2.0 Group schema for identity provider provisioning.
// Supports both Azure Entra ID and Okta SCIM clients.
// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups
// Docs: https://developer.okta.com/docs/reference/scim/scim-20/
//
// SCIM ID mapping:
//   - id: Bytebase's internal group identifier (returned to IdP, used in subsequent API calls)
//   - externalId: IdP's group identifier (sent by IdP to identify the group in their system)
type SCIMGroup struct {
	// id is returned by our SCIM server and used by IdP in subsequent requests (GET/PATCH/DELETE /Groups/{id}).
	ID      string   `json:"id"`
	Schemas []string `json:"schemas"`
	// externalId is the IdP's group identifier. The IdP sends this so we can correlate their group with ours.
	ExternalID string `json:"externalId"`
	// email is a custom attribute mapped from the IdP's group mail attribute.
	// Configure in IdP: Add custom attribute "email" and map group mail -> SCIM "email".
	// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/customize-application-attributes
	Email       string            `json:"email"`
	DisplayName string            `json:"displayName"`
	Members     []*SCIMMember     `json:"members"`
	Meta        *SCIMResourceMeta `json:"meta"`
}

// ListUsersResponse is the SCIM 2.0 list response for users.
// Docs: https://datatracker.ietf.org/doc/html/rfc7644#section-3.4.2
type ListUsersResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	Resources    []*SCIMUser `json:"Resources"`
}

// ListGroupsResponse is the SCIM 2.0 list response for groups.
type ListGroupsResponse struct {
	Schemas      []string     `json:"schemas"`
	TotalResults int          `json:"totalResults"`
	Resources    []*SCIMGroup `json:"Resources"`
}

// SCIMMember represents a member in SCIM group operations.
// Used in both POST /Groups (with members) and PATCH /Groups (add/remove members).
// Okta sends: {"value": "101", "display": "user@example.com"}
// Azure sends: {"value": "101"}
type SCIMMember struct {
	Value   string `json:"value"`             // Bytebase user UID
	Display string `json:"display,omitempty"` // User's email/display name (optional)
}

// PatchOperation represents a single operation in a SCIM PATCH request.
// Docs: https://datatracker.ietf.org/doc/html/rfc7644#section-3.5.2
type PatchOperation struct {
	OP    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

// PatchRequest represents a SCIM 2.0 PATCH request.
// Docs: https://datatracker.ietf.org/doc/html/rfc7644#section-3.5.2
type PatchRequest struct {
	Schemas    []string          `json:"schemas"`
	Operations []*PatchOperation `json:"Operations"`
}

// ServiceProviderConfig represents the SCIM 2.0 ServiceProviderConfig response.
// This endpoint allows SCIM clients to discover the capabilities of the SCIM server.
// Docs: https://datatracker.ietf.org/doc/html/rfc7644#section-4
type ServiceProviderConfig struct {
	Schemas               []string                       `json:"schemas"`
	DocumentationURI      string                         `json:"documentationUri,omitempty"`
	Patch                 ServiceProviderConfigPatch     `json:"patch"`
	Bulk                  ServiceProviderConfigBulk      `json:"bulk"`
	Filter                ServiceProviderConfigFilter    `json:"filter"`
	ChangePassword        ServiceProviderConfigSupported `json:"changePassword"`
	Sort                  ServiceProviderConfigSupported `json:"sort"`
	Etag                  ServiceProviderConfigSupported `json:"etag"`
	AuthenticationSchemes []AuthenticationScheme         `json:"authenticationSchemes"`
}

// ServiceProviderConfigPatch indicates PATCH operation support.
type ServiceProviderConfigPatch struct {
	Supported bool `json:"supported"`
}

// ServiceProviderConfigBulk indicates bulk operation support.
type ServiceProviderConfigBulk struct {
	Supported      bool `json:"supported"`
	MaxOperations  int  `json:"maxOperations"`
	MaxPayloadSize int  `json:"maxPayloadSize"`
}

// ServiceProviderConfigFilter indicates filter support.
type ServiceProviderConfigFilter struct {
	Supported  bool `json:"supported"`
	MaxResults int  `json:"maxResults"`
}

// ServiceProviderConfigSupported indicates feature support.
type ServiceProviderConfigSupported struct {
	Supported bool `json:"supported"`
}

// AuthenticationScheme describes an authentication method supported by the SCIM server.
type AuthenticationScheme struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
