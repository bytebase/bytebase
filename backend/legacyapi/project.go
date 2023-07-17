package api

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

const (
	// DefaultProjectUID is the UID for the default project.
	DefaultProjectUID = 1
	// DefaultProjectID is the resource ID for the default project.
	DefaultProjectID = "default"

	// Below are defined in LATEST_DATA.sql.

	// DefaultTestEnvironmentID is the initial resource ID for the test environment.
	// This can be mutated by the user. But for now this is only used by onboarding flow to create
	// a test instance after first signup, so it's safe to refer it.
	DefaultTestEnvironmentID = "test"
	// DefaultTestEnvironmentUID is the initial resource UID for the test environment.
	DefaultTestEnvironmentUID = 101

	// DefaultProdEnvironmentID is the initial resource ID for the prod environment.
	// This can be mutated by the user. But for now this is only used by onboarding flow to create
	// a prod instance after first signup, so it's safe to refer it.
	DefaultProdEnvironmentID = "prod"
	// DefaultProdEnvironmentUID is the initial resource UID for the prod environment.
	DefaultProdEnvironmentUID = 102
)

// ProjectWorkflowType is the workflow type for projects.
type ProjectWorkflowType string

const (
	// UIWorkflow is the UI workflow.
	UIWorkflow ProjectWorkflowType = "UI"
	// VCSWorkflow is the VCS workflow.
	VCSWorkflow ProjectWorkflowType = "VCS"
)

// ProjectVisibility is the visibility of a project.
type ProjectVisibility string

const (
	// Public is the project visibility for PUBLIC.
	Public ProjectVisibility = "PUBLIC"
	// Private is the project visibility for PRIVATE.
	Private ProjectVisibility = "PRIVATE"
)

// ProjectTenantMode is the tenant mode setting for project.
type ProjectTenantMode string

const (
	// TenantModeDisabled is the DISABLED value for ProjectTenantMode.
	TenantModeDisabled ProjectTenantMode = "DISABLED"
	// TenantModeTenant is the TENANT value for ProjectTenantMode.
	TenantModeTenant ProjectTenantMode = "TENANT"
)

// ProjectSchemaChangeType is the schema change type for projects.
type ProjectSchemaChangeType string

const (
	// ProjectSchemaChangeTypeDDL is the Data Definition Language (DDL) schema
	// migration.
	ProjectSchemaChangeTypeDDL ProjectSchemaChangeType = "DDL"
	// ProjectSchemaChangeTypeSDL is the Schema Definition Language (SDL) schema
	// migration.
	ProjectSchemaChangeTypeSDL ProjectSchemaChangeType = "SDL"
)

// Project is the API message for a project.
type Project struct {
	ID         int       `jsonapi:"primary,project"`
	ResourceID string    `jsonapi:"attr,resourceId"`
	RowStatus  RowStatus `jsonapi:"attr,rowStatus"`

	// Domain specific fields
	Name         string              `jsonapi:"attr,name"`
	Key          string              `jsonapi:"attr,key"`
	WorkflowType ProjectWorkflowType `jsonapi:"attr,workflowType"`
	Visibility   ProjectVisibility   `jsonapi:"attr,visibility"`
	TenantMode   ProjectTenantMode   `jsonapi:"attr,tenantMode"`
	// DBNameTemplate is only used when a project is in tenant mode and the name of tenant databases follows a format.
	// {{DB_NAME}} is used for each tenant belonging to an individual database instance and all tenant databases have the same database name.
	// The template can include label keys such as {{DB_NAME}}_{{TENANT}}. It allows all tenant databases to belong to one or a few database instances.
	// All database with the same {{DB_NAME}} (base database name) belong to one group.
	//
	// Empty value means all tenant databases in the project belonging to the same group.
	DBNameTemplate string `jsonapi:"attr,dbNameTemplate"`
	// SchemaChangeType is the type of the schema migration script.
	SchemaChangeType ProjectSchemaChangeType `jsonapi:"attr,schemaChangeType"`
}

var (
	// DBNameToken is the token for database name.
	DBNameToken = "{{DB_NAME}}"
	// EnvironmentToken is the token for environment.
	EnvironmentToken = "{{ENV_ID}}"
	// LocationToken is the token for location.
	LocationToken = "{{LOCATION}}"
	// TenantToken is the token for tenant.
	TenantToken = "{{TENANT}}"

	// boolean indicates whether it's a required or optional token.
	repositoryFilePathTemplateTokens = map[string]bool{
		"{{VERSION}}":     true,
		DBNameToken:       true,
		"{{TYPE}}":        true,
		EnvironmentToken:  false,
		"{{DESCRIPTION}}": false,
	}
	tenantRepositoryFilePathTemplateTokens = map[string]bool{
		"{{VERSION}}":     true,
		"{{TYPE}}":        true,
		"{{DESCRIPTION}}": false,
	}
	schemaPathTemplateTokens = map[string]bool{
		DBNameToken:      true,
		EnvironmentToken: false,
	}
	tenantSchemaPathTemplateTokens     = map[string]bool{}
	allowedProjectDBNameTemplateTokens = map[string]bool{
		DBNameToken:   true,
		LocationToken: true,
		TenantToken:   true,
	}
)

// ValidateRepositoryFilePathTemplate validates the repository file path template.
func ValidateRepositoryFilePathTemplate(filePathTemplate string, tenantMode ProjectTenantMode) error {
	tokens, _ := common.ParseTemplateTokens(filePathTemplate)
	tokenMap := make(map[string]bool)
	for _, token := range tokens {
		tokenMap[token] = true
	}

	filePathTemplateTokens := repositoryFilePathTemplateTokens
	if tenantMode == TenantModeTenant {
		filePathTemplateTokens = tenantRepositoryFilePathTemplateTokens
	}
	for token, required := range filePathTemplateTokens {
		// Skip checking tokens that are not required
		if !required {
			continue
		}

		if _, ok := tokenMap[token]; !ok {
			return errors.Errorf("missing %s in file path template", token)
		}
	}
	for token := range tokenMap {
		if _, ok := filePathTemplateTokens[token]; !ok {
			return errors.Errorf("unknown token %s in file path template", token)
		}
	}
	return nil
}

// ValidateRepositorySchemaPathTemplate validates the repository schema path template.
func ValidateRepositorySchemaPathTemplate(schemaPathTemplate string, tenantMode ProjectTenantMode) error {
	if schemaPathTemplate == "" {
		return nil
	}
	tokens, _ := common.ParseTemplateTokens(schemaPathTemplate)
	tokenMap := make(map[string]bool)
	for _, token := range tokens {
		tokenMap[token] = true
	}

	allowedTokens := schemaPathTemplateTokens
	if tenantMode == TenantModeTenant {
		allowedTokens = tenantSchemaPathTemplateTokens
	}

	for token, required := range allowedTokens {
		if required {
			if _, ok := tokenMap[token]; !ok {
				return errors.Errorf("missing %s in schema path template", token)
			}
		}
	}

	for token := range tokenMap {
		if _, ok := allowedTokens[token]; !ok {
			return errors.Errorf("unknown token %s in schema path template", token)
		}
	}
	return nil
}

// ValidateProjectDBNameTemplate validates the project database name template.
func ValidateProjectDBNameTemplate(template string) error {
	if template == "" {
		return nil
	}
	tokens, _ := common.ParseTemplateTokens(template)
	// Must contain {{DB_NAME}}
	hasDBName := false
	for _, token := range tokens {
		if token == DBNameToken {
			hasDBName = true
		}
		if _, ok := allowedProjectDBNameTemplateTokens[token]; !ok {
			return errors.Errorf("invalid token %v in database name template", token)
		}
	}
	if !hasDBName {
		return errors.Errorf("project database name template must include token %v", DBNameToken)
	}
	return nil
}

// FormatTemplate formats the template by using the tokens as a replacement mapping.
// Note that the returned (modified) template should not be used as a regexp.
func FormatTemplate(template string, tokens map[string]string) (string, error) {
	keys, _ := common.ParseTemplateTokens(template)
	for _, key := range keys {
		if _, ok := tokens[key]; !ok {
			return "", errors.Errorf("token %q not found", key)
		}
		template = strings.ReplaceAll(template, key, tokens[key])
	}
	return template, nil
}

// Similar to FormatTemplate, except that it will also escape special regexp characters in the delimiters
// of the template string, which will produce the correct regexp string.
func formatTemplateRegexp(template string, tokens map[string]string) (string, error) {
	keys, delimiters := common.ParseTemplateTokens(template)
	for _, key := range keys {
		if _, ok := tokens[key]; !ok {
			return "", errors.Errorf("token %q not found", key)
		}
		template = strings.ReplaceAll(template, key, tokens[key])
	}
	for _, delimiter := range delimiters {
		quoted := regexp.QuoteMeta(delimiter)
		if quoted != delimiter {
			// The delimiter is a special regexp character, we should escape it.
			template = strings.Replace(template, delimiter, quoted, 1)
		}
	}
	return template, nil
}
