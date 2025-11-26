package directorysync

type AADUserEmail struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type AADResourceMeta struct {
	ResourceType string `json:"resourceType"`
}

// AADUser represents the SCIM User schema for Azure AD (Entra ID) provisioning.
// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#design-your-user-and-group-schema
//
// SCIM ID mapping for users:
//   - id: Bytebase's user UID (numeric, returned as string). Used by Azure in subsequent API calls.
//   - externalId: Azure's objectId (optional). Azure sends this but we use userName (email) for user matching.
//   - userName: User's email address (maps to userPrincipalName in Azure). Primary identifier for user lookup.
//
// Unlike groups, users are matched by userName (email) rather than externalId because:
//   - Email is the natural unique identifier for users in Bytebase
//   - Azure's default attribute mapping uses userPrincipalName -> userName
//
// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#get-user-by-query
type AADUser struct {
	// id is Bytebase's user UID, used by Azure in subsequent requests (GET/PATCH/DELETE /Users/{id}).
	ID      string   `json:"id"`
	Schemas []string `json:"schemas"`
	// externalId is Azure's objectId. We don't use this for user matching; we use userName instead.
	ExternalID string `json:"externalId"`
	// userName maps to userPrincipalName in Azure. This is the primary identifier for user lookup.
	UserName    string           `json:"userName"`
	Active      bool             `json:"active"`
	DisplayName string           `json:"displayName"`
	Emails      []*AADUserEmail  `json:"emails"`
	Meta        *AADResourceMeta `json:"meta"`
}

// AADGroup represents the SCIM Group schema for Azure AD (Entra ID) provisioning.
// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups
//
// SCIM ID mapping:
//   - id: Bytebase's internal group identifier (returned to Azure, used in subsequent API calls)
//   - externalId: Azure's objectId for the group (sent by Azure to identify the group in their system)
//
// Docs: https://learn.microsoft.com/en-us/answers/questions/1394370/azure-ad-scim-provisioning-group-update-requests-w
// Docs: https://stackoverflow.com/questions/67198152/where-does-azuread-store-the-id-attribute-returned-by-a-scim-endpoint
type AADGroup struct {
	// id is returned by our SCIM server and used by Azure in subsequent requests (GET/PATCH/DELETE /Groups/{id}).
	ID      string   `json:"id"`
	Schemas []string `json:"schemas"`
	// externalId is Azure's objectId for the group. Azure sends this so we can correlate their group with ours.
	// By default, Azure maps objectId -> externalId in attribute mappings.
	ExternalID string `json:"externalId"`
	// email is a custom attribute mapped from Azure group's mail attribute.
	// Configure in Azure: Entra ID -> Enterprise apps -> Provisioning -> Attribute Mappings -> Groups
	// Add custom attribute "email" and map Azure "mail" -> SCIM "email".
	// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/customize-application-attributes
	Email       string           `json:"email"`
	DisplayName string           `json:"displayName"`
	Members     []string         `json:"members"`
	Meta        *AADResourceMeta `json:"meta"`
}

// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#get-user-by-query
type ListUsersResponse struct {
	Schemas      []string   `json:"schemas"`
	TotalResults int        `json:"totalResults"`
	Resources    []*AADUser `json:"Resources"`
}

type ListGroupsResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	Resources    []*AADGroup `json:"Resources"`
}

type PatchMember struct {
	Value string `json:"value"`
}

type PatchOperation struct {
	OP    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#update-user-multi-valued-properties
type PatchRequest struct {
	Schemas    []string          `json:"schemas"`
	Operations []*PatchOperation `json:"Operations"`
}
