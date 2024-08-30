package directorysync

type AADUserEmail struct {
	Type    string `json:"type"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

type AADResourceMeta struct {
	ResourceType string `json:"resourceType"`
}

// AAD user schema
// Docs: https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#design-your-user-and-group-schema
type AADUser struct {
	ID         string   `json:"id"`
	Schemas    []string `json:"schemas"`
	ExternalID string   `json:"externalId"`
	// Map to userPrincipalName in AAD.
	UserName    string           `json:"userName"`
	Active      bool             `json:"active"`
	DisplayName string           `json:"displayName"`
	Emails      []*AADUserEmail  `json:"emails"`
	Meta        *AADResourceMeta `json:"meta"`
}

type AADGroup struct {
	ID          string           `json:"id"`
	Schemas     []string         `json:"schemas"`
	ExternalID  string           `json:"externalId"`
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
