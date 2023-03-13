package api

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
)

func TestGetTemplateTokens(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		tokens     []string
		delimiters []string
	}{
		{
			"NoToken",
			"helloworld",
			nil,
			nil,
		}, {
			"TwoTokens",
			"{{DB_NAME}}_{{LOCATION}}",
			[]string{"{{DB_NAME}}", "{{LOCATION}}"},
			[]string{"_"},
		}, {
			"ExtraPrefix",
			"hello_{{DB_NAME}}_{{LOCATION}}",
			[]string{"{{DB_NAME}}", "{{LOCATION}}"},
			[]string{"hello_", "_"},
		},
	}

	for _, test := range tests {
		tokens, delimiters := common.ParseTemplateTokens(test.template)
		require.Equal(t, test.tokens, tokens)
		require.Equal(t, test.delimiters, delimiters)
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
			"{{ENV_ID}}/{{DB_NAME}}_{{TYPE}}_{{VERSION}}_{{DESCRIPTION}}.sql",
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
		},
		{
			"Tenant mode {{ENV_ID}}",
			"{{ENV_ID}}/{{VERSION}}_{{TYPE}}.sql",
			TenantModeTenant,
			"in file path template",
		},
		{
			"Tenant mode {{DB_NAME}}",
			"{{DB_NAME}}_{{VERSION}}_{{TYPE}}.sql",
			TenantModeTenant,
			"in file path template",
		}, {
			"Tenant mode okay",
			"{{VERSION}}_{{TYPE}}.sql",
			TenantModeTenant,
			"",
		},
	}

	for _, test := range tests {
		// Fix the problem that closure in a for loop will always use the last element.
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := ValidateRepositoryFilePathTemplate(test.template, test.tenantMode)
			if test.errPart == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.errPart, test.name)
			}
		})
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
			"{{ENV_ID}}/{{DB_NAME}}.sql",
			TenantModeDisabled,
			"",
		}, {
			"UnknownToken",
			"{{DB_NAME}}_{{TYPE}}.sql",
			TenantModeDisabled,
			"unknown token {{TYPE}}",
		}, {
			"Tenant mode {{ENV_ID}}",
			"{{ENV_ID}}/LATEST.sql",
			TenantModeTenant,
			"in schema path template",
		}, {
			"Tenant mode {{DB_NAME}}",
			"{{DB_NAME}}_LATEST.sql",
			TenantModeTenant,
			"in schema path template",
		},
	}

	for _, test := range tests {
		err := ValidateRepositorySchemaPathTemplate(test.template, test.tenantMode)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, test.errPart, test.name)
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
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart)
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
		},
		{
			"tokenNotFound",
			"{{DB_NAME}}_hello_{{LOCATION}}",
			map[string]string{
				"{{DB_NAME}}": "db1",
			},
			"",
			"not found",
		},
		{
			"template with regex meta",
			"{{DB_NAME}}_hello_${{LOCATION}}",
			map[string]string{
				"{{DB_NAME}}":  "db1",
				"{{LOCATION}}": "us-central1",
			},
			"db1_hello_$us-central1",
			"",
		},
	}

	for _, test := range tests {
		got, err := FormatTemplate(test.template, test.tokens)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart)
		}
		require.Equal(t, got, test.want)
	}
}

func TestBuildTemplateExpr(t *testing.T) {
	tests := []struct {
		name     string
		template string
		tokens   map[string]string
		want     string
		errPart  string
	}{
		{
			"valid",
			"{{DB_NAME}}_hello-{{LOCATION}}",
			map[string]string{
				"{{DB_NAME}}":  "db1",
				"{{LOCATION}}": "us-central1",
			},
			"db1_hello-us-central1",
			"",
		},
		{
			"tokenNotFound",
			"{{DB_NAME}}_hello_{{LOCATION}}",
			map[string]string{
				"{{DB_NAME}}": "db1",
			},
			"",
			"not found",
		},
		{
			"template with regex meta",
			"{{DB_NAME}}_hello_${{LOCATION}}",
			map[string]string{
				"{{DB_NAME}}":  "db1",
				"{{LOCATION}}": "us-central1",
			},
			"db1_hello_\\$us-central1",
			"",
		},
	}

	for _, test := range tests {
		got, err := formatTemplateRegexp(test.template, test.tokens)
		if test.errPart == "" {
			require.NoError(t, err)
		} else {
			require.Contains(t, err.Error(), test.errPart)
		}
		require.Equal(t, got, test.want)
	}
}
