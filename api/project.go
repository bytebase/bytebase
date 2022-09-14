package api

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common"
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
	DBNameTemplate string              `jsonapi:"attr,dbNameTemplate"`
	RoleProvider   ProjectRoleProvider `jsonapi:"attr,roleProvider"`
	// SchemaChangeType is the type of the schema migration script.
	SchemaChangeType ProjectSchemaChangeType `jsonapi:"attr,schemaChangeType"`
}

// ProjectCreate is the API message for creating a project.
type ProjectCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name             string                  `jsonapi:"attr,name"`
	Key              string                  `jsonapi:"attr,key"`
	TenantMode       ProjectTenantMode       `jsonapi:"attr,tenantMode"`
	DBNameTemplate   string                  `jsonapi:"attr,dbNameTemplate"`
	RoleProvider     ProjectRoleProvider     `jsonapi:"attr,roleProvider"`
	SchemaChangeType ProjectSchemaChangeType `jsonapi:"attr,schemaChangeType"`
}

// ProjectFind is the API message for finding projects.
type ProjectFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus

	// Domain specific fields
	// If present, will only find project containing PrincipalID as an active member
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
	Name             *string `jsonapi:"attr,name"`
	WorkflowType     *string `jsonapi:"attr,workflowType"` // NOTE: We can't use *ProjectWorkflowType because "google/jsonapi" doesn't support.
	RoleProvider     *string `jsonapi:"attr,roleProvider"`
	SchemaChangeType *string `jsonapi:"attr,schemaChangeType"` // NOTE: We can't use *ProjectSchemaChangeType because "google/jsonapi" doesn't support.
}

var (
	// DBNameToken is the token for database name.
	DBNameToken = "{{DB_NAME}}"
	// EnvironmentToken is the token for environment.
	EnvironmentToken = "{{ENV_NAME}}"
	// LocationToken is the token for location.
	LocationToken = "{{LOCATION}}"
	// TenantToken is the token for tenant.
	TenantToken = "{{TENANT}}"

	// boolean indicates whether it's an required or optional token.
	repositoryFilePathTemplateTokens = map[string]bool{
		"{{VERSION}}":     true,
		DBNameToken:       true,
		"{{TYPE}}":        true,
		EnvironmentToken:  false,
		"{{DESCRIPTION}}": false,
	}
	schemaPathTemplateTokens = map[string]bool{
		DBNameToken:      true,
		EnvironmentToken: false,
	}
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
	if tenantMode == TenantModeTenant {
		if _, ok := tokenMap[EnvironmentToken]; ok {
			return &common.Error{Code: common.Invalid, Err: errors.Errorf("%q is not allowed in the template for projects in tenant mode", EnvironmentToken)}
		}
	}

	for token, required := range repositoryFilePathTemplateTokens {
		if required {
			if _, ok := tokenMap[token]; !ok {
				return errors.Errorf("missing %s in file path template", token)
			}
		}
	}
	for token := range tokenMap {
		if _, ok := repositoryFilePathTemplateTokens[token]; !ok {
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
	if tenantMode == TenantModeTenant {
		if _, ok := tokenMap[EnvironmentToken]; ok {
			return &common.Error{Code: common.Invalid, Err: errors.Errorf("%q is not allowed in the template for projects in tenant mode", EnvironmentToken)}
		}
	}

	for token, required := range schemaPathTemplateTokens {
		if required {
			if _, ok := tokenMap[token]; !ok {
				return errors.Errorf("missing %s in schema path template", token)
			}
		}
	}
	for token := range tokenMap {
		if _, ok := schemaPathTemplateTokens[token]; !ok {
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

// GetBaseDatabaseName will return the base database name given the database name, dbNameTemplate, labelsJSON.
func GetBaseDatabaseName(databaseName, dbNameTemplate, labelsJSON string) (string, error) {
	if dbNameTemplate == "" {
		return databaseName, nil
	}
	var labels []*DatabaseLabel
	if labelsJSON != "" {
		if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
			return "", err
		}
	}
	labelMap := map[string]string{}
	for _, label := range labels {
		switch label.Key {
		case LocationLabelKey:
			labelMap[LocationToken] = label.Value
		case TenantLabelKey:
			labelMap[TenantToken] = label.Value
		}
	}
	labelMap["{{DB_NAME}}"] = "(?P<NAME>.+)"

	expr, err := formatTemplateRegexp(dbNameTemplate, labelMap)
	if err != nil {
		return "", errors.Wrapf(err, "FormatTemplate(%q, %+v) failed", dbNameTemplate, labelMap)
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return "", errors.Wrapf(err, "regexp %q compiled failure", expr)
	}
	names := re.FindStringSubmatch(databaseName)
	if len(names) != 2 || names[1] == "" {
		return "", errors.Errorf("database name %q doesn't follow database name template %q", databaseName, dbNameTemplate)
	}
	return names[1], nil
}
