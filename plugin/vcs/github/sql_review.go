package github

import (
	_ "embed"
	"fmt"
)

// SQLReviewAction is the GitHub action for SQL review in VCS workflow.
//
//go:embed sql-review.yml
var SQLReviewAction string

const (
	// SQLReviewActionFilePath is the SQL review action file path.
	SQLReviewActionFilePath = ".github/workflows/sql-review.yml"
)
