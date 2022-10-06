package gitlab

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

const (
	// CIFilePath is the local path for GitLab ci file.
	CIFilePath = ".gitlab-ci.yml"
	// SQLReviewCIFilePath is the local path for SQL review CI in GitLab repo.
	SQLReviewCIFilePath = ".gitlab/sql-review.yml"
	// SQLReviewScriptFilePath is the local path for SQL review script in GitLab repo.
	SQLReviewScriptFilePath = ".gitlab/sql-review.sh"
)

// SQLReviewCI is the GitLab CI for SQL review in VCS workflow.
//
//go:embed .gitlab/sql-review.yml
var SQLReviewCI string

// SQLReviewScript is the SQL review script used in SQL review CI.
//
//go:embed .gitlab/sql-review.sh
var SQLReviewScript string

type sqlReviewCIConfig struct {
	Variables sqlReviewCIVariables `yaml:"variables"`
}

type sqlReviewCIVariables struct {
	API string `yaml:"API"`
}

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

		includeList = append(includeList, map[string]string{"local": SQLReviewCIFilePath})
		gitlabCI["include"] = includeList
	}

	// Add SQL review endpoint.
	gitlabCI["sql-review"] = sqlReviewCIConfig{
		Variables: sqlReviewCIVariables{
			API: sqlReviewEndpoint,
		},
	}

	newContent, err := yaml.Marshal(gitlabCI)
	if err != nil {
		return "", err
	}

	return string(newContent), nil
}
