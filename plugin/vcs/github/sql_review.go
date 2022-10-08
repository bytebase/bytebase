package github

import (
	_ "embed"
	"fmt"

	"github.com/bytebase/bytebase/plugin/vcs"
)

// sqlReviewAction is the GitHub action for SQL review in VCS workflow.
//
//go:embed sql-review.yml
var sqlReviewAction string

const (
	// SQLReviewActionFilePath is the SQL review action file path.
	SQLReviewActionFilePath = ".github/workflows/sql-review.yml"
)

// SetupSQLReviewCI will setup the SQL review CI content with SQL review endpoint.
func SetupSQLReviewCI(endpoint string) string {
	return fmt.Sprintf(sqlReviewAction, endpoint, vcs.SQLReviewAPISecretName)
}
