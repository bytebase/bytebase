package vcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDoubleTimesAsteriskInTemplateValid(t *testing.T) {
	tests := []struct {
		template string
		err      bool
	}{
		{
			template: "**",
			err:      true,
		},
		{
			template: "bytebase/{{ENV_NAME}}/**",
			err:      true,
		},
		{
			template: "**/{{ENV_NAME}}/{{DB_NAME}}.sql",
			err:      true,
		},
		{
			template: "bytebase/**/{{ENV_NAME}}/**/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
			err:      false,
		},
		{
			template: "/**/{{ENV_NAME}}/**/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql",
			err:      false,
		},
		// Credit to Linear Issue BYT-1267
		{
			template: "/configure/configure/{{ENV_NAME}}/**/**/{{DESCRIPTION}}.sql",
			err:      false,
		},
	}
	for _, test := range tests {
		outputErr := isDoubleAsteriskInTemplateValid(test.template)
		if test.err {
			assert.Error(t, outputErr)
		} else {
			assert.NoError(t, outputErr)
		}
	}
}
