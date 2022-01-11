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
		name     string
		template string
		errPart  string
	}{
		{
			"OK",
			"{{DB_NAME}}_{{TYPE}}_{{VERSION}}.sql",
			"",
		}, {
			"TwoTokens",
			"{{DB_NAME}}_{{TYPE}}.sql",
			"",
		}, {
			"UnknownToken",
			"{{DB_NAME}}_{{TYPE}}_{{VERSION}}_{{UNKNWON}}.sql",
			"unknown token {{UNKNWON}}",
		}, {
			"UnknownToken",
			"{{DB_NAME}}_{{TYPE}}_{{VERSION}}_{{UNKNWON}}.sql",
			"unknown token {{UNKNWON}}",
		},
	}

	for _, test := range tests {
		err := ValidateRepositoryFilePathTemplate(test.template)
		if err != nil {
			if !strings.Contains(err.Error(), test.errPart) {
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
		name     string
		template string
		errPart  string
	}{
		{
			"OK",
			"{{DB_NAME}}_hello.sql",
			"",
		}, {
			"TwoTokens",
			"{{DB_NAME}}_{{TYPE}}.sql",
			"invalid number of tokens",
		}, {
			"InvalidToken",
			"{{TYPE}}_hello.sql",
			"invalid token {{TYPE}}",
		},
	}

	for _, test := range tests {
		err := ValidateRepositorySchemaPathTemplate(test.template)
		if err != nil {
			if !strings.Contains(err.Error(), test.errPart) {
				t.Errorf("%q: ValidateRepositorySchemaPathTemplate(%q) got error %q, want errPart %q.", test.name, test.template, err.Error(), test.errPart)
			}
		} else {
			if test.errPart != "" {
				t.Errorf("%q: ValidateRepositorySchemaPathTemplate(%q) got no error, want errPart %q.", test.name, test.template, test.errPart)
			}
		}
	}
}
