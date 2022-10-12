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
	// sqlReviewCIFileRelativePathInGitLabCI is the keyword for relative path for SQL review CI file in .gitlab-ci.yml.
	sqlReviewCIFileRelativePathKeywordInGitLabCI = "local"
	// gitlabCIIncludeKeyword is the keyword for "include" in .gitlab-ci.yml
	// Docs for GitLab include syntax: https://docs.gitlab.com/ee/ci/yaml/includes.html
	gitlabCIIncludeKeyword = "include"
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
	// Add include for SQL review CI
	var includeList []interface{}
	// Docs for GitLab include syntax: https://docs.gitlab.com/ee/ci/yaml/includes.html
	switch include := gitlabCI[gitlabCIIncludeKeyword].(type) {
	case []interface{}:
		includeList = append(includeList, include...)
	case string, interface{}:
		includeList = append(includeList, include)
	}

	if _, ok := findSQLReviewCI(includeList); !ok {
		includeList = append(includeList, map[string]string{sqlReviewCIFileRelativePathKeywordInGitLabCI: SQLReviewCIFilePath})
	}

	gitlabCI[gitlabCIIncludeKeyword] = includeList
	newContent, err := yaml.Marshal(gitlabCI)
	if err != nil {
		return "", err
	}

	return string(newContent), nil
}

func findSQLReviewCI(include []interface{}) (map[string]interface{}, bool) {
	for _, data := range include {
		if val, ok := data.(map[string]interface{}); ok {
			if val[sqlReviewCIFileRelativePathKeywordInGitLabCI] == SQLReviewCIFilePath {
				return val, true
			}
		}
	}

	return nil, false
}
