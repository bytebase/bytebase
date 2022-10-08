package gitlab

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/plugin/vcs"
)

const (
	// CIFilePath is the local path for GitLab ci file.
	CIFilePath = ".gitlab-ci.yml"
	// SQLReviewCIFilePath is the local path for SQL review CI in GitLab repo.
	SQLReviewCIFilePath = ".gitlab/sql-review.yml"
)

// sqlReviewCI is the GitLab CI for SQL review in VCS workflow.
//
//go:embed sql-review.yml
var sqlReviewCI string

// SetupSQLReviewCI will setup the SQL review CI content with SQL review endpoint.
func SetupSQLReviewCI(endpoint string) string {
	return fmt.Sprintf(sqlReviewCI, endpoint, vcs.SQLReviewAPISecretName)
}

// SetupGitLabCI will update the GitLab CI content to add or update the SQL review CI.
func SetupGitLabCI(gitlabCI map[string]interface{}) (string, error) {
	if gitlabCI["sql-review"] == nil {
		// Add include for SQL review CI
		var includeList []interface{}
		// Docs for GitLab include syntax: https://docs.gitlab.com/ee/ci/yaml/includes.html
		switch include := gitlabCI["include"].(type) {
		case []interface{}:
			includeList = append(includeList, include...)
		case string, interface{}:
			includeList = append(includeList, include)
		}

		includeList = append(includeList, map[string]string{"local": SQLReviewCIFilePath})
		gitlabCI["include"] = includeList
	}

	newContent, err := yaml.Marshal(gitlabCI)
	if err != nil {
		return "", err
	}

	return string(newContent), nil
}
