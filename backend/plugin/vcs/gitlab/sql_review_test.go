package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const mockGitLabCIContentYAMLStr = `
image: busybox:latest

include:
  - local: .gitlab/example.yml

before_script:
  - echo "Before script section"

after_script:
  - echo "After script section"
`

func Test_SetupGitLabCI(t *testing.T) {
	content := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(mockGitLabCIContentYAMLStr), &content)
	require.NoError(t, err)

	newContent, err := SetupGitLabCI(content)
	require.NoError(t, err)

	err = yaml.Unmarshal([]byte(newContent), &content)
	require.NoError(t, err)

	assert.NotNil(t, content["image"])
	assert.NotNil(t, content["include"])
	assert.NotNil(t, content["before_script"])
	assert.NotNil(t, content["after_script"])

	include, ok := content["include"].([]interface{})
	assert.Equal(t, ok, true)

	sqlReviewCI, ok := findSQLReviewCI(include)
	assert.Equal(t, ok, true)
	assert.NotNil(t, sqlReviewCI)
}
