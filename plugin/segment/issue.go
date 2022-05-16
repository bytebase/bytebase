package segment

import (
	"context"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"github.com/segmentio/analytics-go"
)

// IssueReporter is the segment reporter for issue.
type IssueReporter struct {
}

var issueTypeList = []api.IssueType{
	api.IssueGeneral,
	api.IssueDatabaseCreate,
	api.IssueDatabaseGrant,
	api.IssueDatabaseSchemaUpdate,
	api.IssueDatabaseSchemaUpdate,
	api.IssueDatabaseSchemaUpdateGhost,
	api.IssueDatabaseDataUpdate,
	api.IssueDataSourceRequest,
}

// Report will exec the segment reporter for issue
func (t *IssueReporter) Report(ctx context.Context, store *store.Store, segment *segment) error {
	total, err := store.CountIssue(ctx, &api.IssueFind{})
	if err != nil {
		return err
	}

	properties := analytics.NewProperties().Set("count", total)

	for _, issueType := range issueTypeList {
		count, err := store.CountIssue(ctx, &api.IssueFind{
			Type: &issueType,
		})
		if err != nil {
			return err
		}
		properties.Set(string(issueType), count)
	}

	return segment.client.Enqueue(analytics.Page{
		UserId:     segment.identifier,
		Name:       string(IssueEventName),
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	})
}
