package segment

import (
	"context"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"github.com/segmentio/analytics-go"
	"go.uber.org/zap"
)

// IssueReporter is the segment reporter for issue.
type IssueReporter struct {
	l *zap.Logger
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
func (r *IssueReporter) Report(ctx context.Context, store *store.Store, segment *segment) error {
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

		properties := analytics.NewProperties().
			Set("type", key).
			Set("count", count)
		if err := segment.client.Enqueue(analytics.Track{
			UserId:     segment.identifier,
			Event:      string(IssueEventName),
			Properties: properties,
			Timestamp:  time.Now().UTC(),
		}); err != nil {
			r.l.Debug("failed to enqueue report event for issue", zap.String("type", string(issueType)))
		}
	}

	return nil
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
