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

	properties := analytics.NewProperties().Set("total", total)

	for _, issueType := range issueTypeList {
		count, err := store.CountIssue(ctx, &api.IssueFind{
			Type: &issueType,
		})
		if err != nil {
			return err
		}
		// We need to skip the empty value, otherwise the Segment cannot sync the event to Google Analytics 4
		if count <= 0 {
			continue
		}

		key := getIssueType(issueType)
		if key == "" {
			continue
		}

		properties.Set(key, count)
	}

	return segment.client.Enqueue(analytics.Page{
		UserId:     segment.identifier,
		Name:       string(IssueEventName),
		Properties: properties,
		Timestamp:  time.Now().UTC(),
	})
}

// getIssueType returns the issue type.
// We need to convert the type manually instead of using the string(issueType) because
// 1. NOT support dot in the key name.
// 2. Key length is limited.
func getIssueType(issueType api.IssueType) string {
	switch issueType {
	case api.IssueGeneral:
		return "general"
	case api.IssueDatabaseCreate:
		return "database_create"
	case api.IssueDatabaseGrant:
		return "database_grant"
	case api.IssueDatabaseSchemaUpdate:
		return "schema_update"
	case api.IssueDatabaseSchemaUpdateGhost:
		return "schema_update_ghost"
	case api.IssueDatabaseDataUpdate:
		return "data_update"
	case api.IssueDataSourceRequest:
		return "datasource_request"
	}

	return ""
}
