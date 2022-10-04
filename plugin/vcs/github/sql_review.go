package github

import (
	_ "embed"
	"fmt"
)

// sqlReviewAction is the GitHub action for SQL review in VCS workflow.
//
//go:embed sql-review-action.yml
var sqlReviewAction string

const (
	// SQLReviewActionFilePath is the SQL review action file path.
	SQLReviewActionFilePath = ".github/workflows/sql-review.yml"
)

// SetupSQLReviewCI will update the GitHub action content for SQL review.
func SetupSQLReviewCI(sqlReviewEndpoint string) string {
	return fmt.Sprintf(sqlReviewAction, sqlReviewEndpoint)
}
