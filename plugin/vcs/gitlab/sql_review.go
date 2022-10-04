package gitlab

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	// sqlReviewCIVersion is the version for GitLab SQL review CI.
	sqlReviewCIVersion = "0.0.2"
	// CIFilePath is the local path for GitLab ci file.
	CIFilePath = ".gitlab-ci.yml"
)

// sqlReviewCIFilePath is the remote file path for GitLab SQL review CI.
// We can include the remote file path in the GitLab CI.
// So we can store the script in our own GitHub repo: https://github.com/bytebase/sql-review-gitlab-ci
var sqlReviewCIFilePath = fmt.Sprintf(
	"https://raw.githubusercontent.com/bytebase/sql-review-gitlab-ci/%s/sql-review.yml",
	sqlReviewCIVersion,
)

// SetupSQLReviewCI will update the GitLab CI content to add or update the SQL review CI.
func SetupSQLReviewCI(gitlabCI map[string]interface{}, sqlReviewEndpoint string) (string, error) {
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

		includeList = append(includeList, map[string]string{"remote": sqlReviewCIFilePath})
		gitlabCI["include"] = includeList
	}

	// Add SQL review endpoint.
	gitlabCI["sql-review"] = make(map[string]interface{})
	gitlabCI["sql-review"].(map[string]interface{})["variables"] = make(map[string]interface{})
	gitlabCI["sql-review"].(map[string]interface{})["variables"].(map[string]interface{})["VERSION"] = sqlReviewCIVersion
	gitlabCI["sql-review"].(map[string]interface{})["variables"].(map[string]interface{})["SQL_REVIEW_API"] = sqlReviewEndpoint

	newContent, err := yaml.Marshal(gitlabCI)
	if err != nil {
		return "", err
	}

	return string(newContent), nil
}
