package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// DefaultProjectID is the ID for the default project.
const DefaultProjectID = 1

// ProjectWorkflowType is the workflow type for projects.
type ProjectWorkflowType string

const (
	// UIWorkflow is the UI workflow.
	UIWorkflow ProjectWorkflowType = "UI"
	// VCSWorkflow is the VCS workflow.
	VCSWorkflow ProjectWorkflowType = "VCS"
)

func (e ProjectWorkflowType) String() string {
	switch e {
	case UIWorkflow:
		return "UI"
	case VCSWorkflow:
		return "VCS"
	}
	return ""
}

// ProjectVisibility is the visibility of a project.
type ProjectVisibility string

const (
	// Public is the project visibility for PUBLIC.
	Public ProjectVisibility = "PUBLIC"
	// Private is the project visibility for PRIVATE.
	Private ProjectVisibility = "PRIVATE"
)

func (e ProjectVisibility) String() string {
	switch e {
	case Public:
		return "PUBLIC"
	case Private:
		return "PRIVATE"
	}
	return ""
}

// ProjectTenantMode is the tenant mode setting for project.
type ProjectTenantMode string

const (
	// TenantModeDisabled is the DISABLED value for ProjectTenantMode.
	TenantModeDisabled ProjectTenantMode = "DISABLED"
	// TenantModeTenant is the TENANT value for ProjectTenantMode.
	TenantModeTenant ProjectTenantMode = "TENANT"
)

// Project is the API message for a project.
type Project struct {
	ID int `jsonapi:"primary,project"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectMemberList []*ProjectMember `jsonapi:"relation,projectMember"`

	// Domain specific fields
	Name         string              `jsonapi:"attr,name"`
	Key          string              `jsonapi:"attr,key"`
	WorkflowType ProjectWorkflowType `jsonapi:"attr,workflowType"`
	Visibility   ProjectVisibility   `jsonapi:"attr,visibility"`
	TenantMode   ProjectTenantMode   `jsonapi:"attr,tenantMode"`
	// DBNameTemplate is only used when a project is in tenant mode.
	// Empty value means {{DB_NAME}}.
	DBNameTemplate string `jsonapi:"attr,dbNameTemplate"`
}

// ProjectCreate is the API message for creating a project.
type ProjectCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name           string            `jsonapi:"attr,name"`
	Key            string            `jsonapi:"attr,key"`
	TenantMode     ProjectTenantMode `jsonapi:"attr,tenantMode"`
	DBNameTemplate string            `jsonapi:"attr,dbNameTemplate"`
}

// ProjectFind is the API message for finding projects.
type ProjectFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus

	// Domain specific fields
	// If present, will only find project containing PrincipalID as a member
	PrincipalID *int
}

func (find *ProjectFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// ProjectPatch is the API message for patching a project.
type ProjectPatch struct {
	ID int `jsonapi:"primary,projectPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name         *string              `jsonapi:"attr,name"`
	Key          *string              `jsonapi:"attr,key"`
	WorkflowType *ProjectWorkflowType `jsonapi:"attr,workflowType"`
}

var (
	// DBNameToken is the token for database name.
	DBNameToken = "{{DB_NAME}}"
	// EnvironemntToken is the token for environment.
	EnvironemntToken = "{{ENV_NAME}}"
	// LocationToken is the token for location.
	LocationToken = "{{LOCATION}}"
	// TenantToken is the token for tenant.
	TenantToken = "{{TENANT}}"

	// boolean indicates whether it's an required or optional token
	repositoryFilePathTemplateTokens = map[string]bool{
		"{{VERSION}}":     true,
		DBNameToken:       true,
		"{{TYPE}}":        true,
		EnvironemntToken:  false,
		"{{DESCRIPTION}}": false,
	}
	schemaPathTemplateTokens = map[string]bool{
		DBNameToken:      true,
		EnvironemntToken: false,
	}
	allowedProjectDBNameTemplateTokens = map[string]bool{
		DBNameToken:   true,
		LocationToken: true,
		TenantToken:   true,
	}
)

// ValidateRepositoryFilePathTemplate validates the repository file path template.
func ValidateRepositoryFilePathTemplate(filePathTemplate string) error {
	tokens := getTemplateTokens(filePathTemplate)
	tokenMap := make(map[string]bool)
	for _, token := range tokens {
		tokenMap[token] = true
	}

	for token, required := range repositoryFilePathTemplateTokens {
		if required {
			if _, ok := tokenMap[token]; !ok {
				return fmt.Errorf("missing %s in file path template", token)
			}
		}
	}
	for token := range tokenMap {
		if _, ok := repositoryFilePathTemplateTokens[token]; !ok {
			return fmt.Errorf("unknown token %s in file path template", token)
		}
	}
	return nil
}

// ValidateRepositorySchemaPathTemplate validates the repository schema path template.
func ValidateRepositorySchemaPathTemplate(schemaPathTemplate string) error {
	if schemaPathTemplate == "" {
		return nil
	}
	tokens := getTemplateTokens(schemaPathTemplate)
	tokenMap := make(map[string]bool)
	for _, token := range tokens {
		tokenMap[token] = true
	}

	for token, required := range schemaPathTemplateTokens {
		if required {
			if _, ok := tokenMap[token]; !ok {
				return fmt.Errorf("missing %s in schema path template", token)
			}
		}
	}
	for token := range tokenMap {
		if _, ok := schemaPathTemplateTokens[token]; !ok {
			return fmt.Errorf("unknown token %s in schema path template", token)
		}
	}
	return nil
}

// ValidateProjectDBNameTemplate validates the project database name template.
func ValidateProjectDBNameTemplate(template string) error {
	if template == "" {
		return nil
	}
	tokens := getTemplateTokens(template)
	// Must contain {{DB_NAME}}
	hasDBName := false
	for _, token := range tokens {
		if token == DBNameToken {
			hasDBName = true
		}
		if _, ok := allowedProjectDBNameTemplateTokens[token]; !ok {
			return fmt.Errorf("invalid token %v in database name template", token)
		}
	}
	if !hasDBName {
		return fmt.Errorf("project database name template must include token %v", DBNameToken)
	}
	return nil
}

// FormatTemplate formats the template.
func FormatTemplate(template string, tokens map[string]string) (string, error) {
	keys := getTemplateTokens(template)
	for _, key := range keys {
		if _, ok := tokens[key]; !ok {
			return "", fmt.Errorf("token %q not found", key)
		}
		template = strings.ReplaceAll(template, key, tokens[key])
	}
	return template, nil
}

func getTemplateTokens(template string) []string {
	r := regexp.MustCompile(`{{[^{}]+}}`)
	return r.FindAllString(template, -1)
}

// ProjectService is the service for projects.
type ProjectService interface {
	CreateProject(ctx context.Context, create *ProjectCreate) (*Project, error)
	FindProjectList(ctx context.Context, find *ProjectFind) ([]*Project, error)
	FindProject(ctx context.Context, find *ProjectFind) (*Project, error)
	PatchProject(ctx context.Context, patch *ProjectPatch) (*Project, error)
	// This is specifically used to update the ProjectWorkflowType when linking/unlinking the repository.
	PatchProjectTx(ctx context.Context, tx *sql.Tx, patch *ProjectPatch) (*Project, error)
}
