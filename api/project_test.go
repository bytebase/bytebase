package api

import (
	"strings"
	"testing"

	"github.com/kr/pretty"
)

func TestGetTemplateTokens(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
	}{
		{
			"NoToken",
			"helloworld",
			nil,
		}, {
			"TwoTokens",
			"{{DB_NAME}}_{{LOCATION}}",
			[]string{"{{DB_NAME}}", "{{LOCATION}}"},
		}, {
			"ExtraPrefix",
			"hello_{{DB_NAME}}_{{LOCATION}}",
			[]string{"{{DB_NAME}}", "{{LOCATION}}"},
		},
	}

	for _, test := range tests {
		tokens := getTemplateTokens(test.template)
		diff := pretty.Diff(tokens, test.want)
		if len(diff) > 0 {
			t.Errorf("%q: getTemplateTokens(%q) got tokens %+v, want %+v, diff %+v.", test.name, test.template, tokens, test.want, diff)
		}
	}
}

func TestValidateRepositoryFilePathTemplate(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		tenantMode ProjectTenantMode
		errPart    string
	}{
		{
			"OK",
			"{{DB_NAME}}_{{TYPE}}_{{VERSION}}.sql",
			TenantModeDisabled,
			"",
		}, {
			"OK with optional tokens",
			"{{ENV_NAME}}/{{DB_NAME}}_{{TYPE}}_{{VERSION}}_{{DESCRIPTION}}.sql",
			TenantModeDisabled,
			"",
		}, {
			"Missing {{VERSION}}",
			"{{DB_NAME}}_{{TYPE}}.sql",
			TenantModeDisabled,
			"missing {{VERSION}}",
		}, {
			"UnknownToken",
			"{{DB_NAME}}_{{TYPE}}_{{VERSION}}_{{UNKNOWN}}.sql",
			TenantModeDisabled,
			"unknown token {{UNKNOWN}}",
		}, {
			"UnknownToken",
			"{{DB_NAME}}_{{TYPE}}_{{VERSION}}_{{UNKNOWN}}.sql",
			TenantModeDisabled,
			"unknown token {{UNKNOWN}}",
		}, {
			"Tenant mode {{ENV_NAME}}",
			"{{ENV_NAME}}/{{DB_NAME}}_{{TYPE}}.sql",
			TenantModeTenant,
			"not allowed in the template",
		},
	}

	for _, test := range tests {
		err := ValidateRepositoryFilePathTemplate(test.template, test.tenantMode)
		if err != nil {
			if test.errPart == "" {
				t.Errorf("%q: ValidateRepositoryFilePathTemplate(%q) got error %q, want OK.", test.name, test.template, err.Error())
			} else if !strings.Contains(err.Error(), test.errPart) {
				t.Errorf("%q: ValidateRepositoryFilePathTemplate(%q) got error %q, want errPart %q.", test.name, test.template, err.Error(), test.errPart)
			}
		} else {
			if test.errPart != "" {
				t.Errorf("%q: ValidateRepositoryFilePathTemplate(%q) got no error, want errPart %q.", test.name, test.template, test.errPart)
			}
		}
	}
}

func TestValidateRepositorySchemaPathTemplate(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		tenantMode ProjectTenantMode
		errPart    string
	}{
		{
			"OK",
			"{{DB_NAME}}_hello.sql",
			TenantModeDisabled,
			"",
		}, {
			"OK with optional tokens",
			"{{ENV_NAME}}/{{DB_NAME}}.sql",
			TenantModeDisabled,
			"",
		}, {
			"UnknownToken",
			"{{DB_NAME}}_{{TYPE}}.sql",
			TenantModeDisabled,
			"unknown token {{TYPE}}",
		}, {
			"Tenant mode {{ENV_NAME}}",
			"{{ENV_NAME}}/{{DB_NAME}}_{{TYPE}}.sql",
			TenantModeTenant,
			"not allowed in the template",
		},
	}

	for _, test := range tests {
		err := ValidateRepositorySchemaPathTemplate(test.template, test.tenantMode)
		if err != nil {
			if test.errPart == "" {
				t.Errorf("%q: ValidateRepositorySchemaPathTemplate(%q) got error %q, want OK.", test.name, test.template, err.Error())
			} else if !strings.Contains(err.Error(), test.errPart) {
				t.Errorf("%q: ValidateRepositorySchemaPathTemplate(%q) got error %q, want errPart %q.", test.name, test.template, err.Error(), test.errPart)
			}
		} else {
			if test.errPart != "" {
				t.Errorf("%q: ValidateRepositorySchemaPathTemplate(%q) got no error, want errPart %q.", test.name, test.template, test.errPart)
			}
		}
	}
}

func TestValidateProjectDBNameTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		errPart  string
	}{
		{
			"location",
			"{{DB_NAME}}_hello_{{LOCATION}}",
			"",
		}, {
			"tenant",
			"{{DB_NAME}}_{{TENANT}}.sql",
			"",
		}, {
			"InvalidToken",
			"{{DB_NAME}}_{{TYPE}}",
			"invalid token {{TYPE}}",
		}, {
			"DatabaseNameTokenNotExists",
			"{{TENANT}}",
			"must include token {{DB_NAME}}",
		},
	}

	for _, test := range tests {
		err := ValidateProjectDBNameTemplate(test.template)
		if err != nil {
			if !strings.Contains(err.Error(), test.errPart) {
				t.Errorf("%q: ValidateProjectDBNameTemplate(%q) got error %q, want errPart %q.", test.name, test.template, err.Error(), test.errPart)
			}
		} else {
			if test.errPart != "" {
				t.Errorf("%q: ValidateProjectDBNameTemplate(%q) got no error, want errPart %q.", test.name, test.template, test.errPart)
			}
		}
	}
}

func TestFormatTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		tokens   map[string]string
		want     string
		errPart  string
	}{
		{
			"valid",
			"{{DB_NAME}}_hello_{{LOCATION}}",
			map[string]string{
				"{{DB_NAME}}":  "db1",
				"{{LOCATION}}": "us-central1",
			},
			"db1_hello_us-central1",
			"",
		}, {
			"tokenNotFound",
			"{{DB_NAME}}_hello_{{LOCATION}}",
			map[string]string{
				"{{DB_NAME}}": "db1",
			},
			"",
			"not found",
		},
	}

	for _, test := range tests {
		got, err := FormatTemplate(test.template, test.tokens)
		if err != nil {
			if !strings.Contains(err.Error(), test.errPart) {
				t.Errorf("%q: FormatTemplate(%q, %+v) got error %q, want errPart %q.", test.name, test.template, test.tokens, err.Error(), test.errPart)
			}
		} else {
			if test.errPart != "" {
				t.Errorf("%q: FormatTemplate(%q, %+v) got no error, want errPart %q.", test.name, test.template, test.tokens, test.errPart)
			}
		}
		if got != test.want {
			t.Errorf("%q: FormatTemplate(%q, %+v) got %q, want %q.", test.name, test.template, test.tokens, got, test.want)
		}
	}
}

func TestGetBaseDatabaseName(t *testing.T) {
	tests := []struct {
		name         string
		databaseName string
		template     string
		labelsJSON   string
		want         string
		errPart      string
	}{
		{
			"no_template_success",
			"db1",
			"",
			"[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"tenant123\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
			"db1",
			"",
		},
		{
			"only_database_name_success",
			"db1",
			"{{DB_NAME}}",
			"[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"tenant123\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
			"db1",
			"",
		},
		{
			"only_database_name_no_label_success",
			"db1",
			"{{DB_NAME}}",
			"",
			"db1",
			"",
		},
		{
			"tenant_label_success",
			"db1_tenant123",
			"{{DB_NAME}}_{{TENANT}}",
			"[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"tenant123\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
			"db1",
			"",
		},
		{
			"tenant_location_label_success",
			"us-central1...db你好_tenant123",
			"{{LOCATION}}...{{DB_NAME}}_{{TENANT}}",
			"[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"tenant123\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
			"db你好",
			"",
		},
		{
			"tenant_label_fail",
			"db1_tenant123",
			"{{DB_NAME}}_{{LOCATION}}",
			"[{\"key\":\"bb.location\",\"value\":\"us-central1\"},{\"key\":\"bb.tenant\",\"value\":\"tenant123\"},{\"key\":\"bb.environment\",\"value\":\"Dev\"}]",
			"",
			"doesn't follow database name template",
		},
	}

	for _, test := range tests {
		got, err := GetBaseDatabaseName(test.databaseName, test.template, test.labelsJSON)
		if err != nil {
			if !strings.Contains(err.Error(), test.errPart) {
				t.Errorf("%q: GetBaseDatabaseName(%q, %q, %q) got error %q, want errPart %q.", test.name, test.databaseName, test.template, test.labelsJSON, err.Error(), test.errPart)
			}
		} else {
			if test.errPart != "" {
				t.Errorf("%q: GetBaseDatabaseName(%q, %q, %q) got no error, want errPart %q.", test.name, test.databaseName, test.template, test.labelsJSON, test.errPart)
			}
		}
		if got != test.want {
			t.Errorf("%q: GetBaseDatabaseName(%q, %q, %q) got %q, want %q.", test.name, test.databaseName, test.template, test.labelsJSON, got, test.want)
		}
	}
}
