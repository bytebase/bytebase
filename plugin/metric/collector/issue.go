package collector

import (
	"context"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/store"
	"go.uber.org/zap"
)

// issueCollector is the metric data collector for issue.
type issueCollector struct {
	l     *zap.Logger
	store *store.Store
}

// issueEventName is the MetricEventName for issue
var issueEventName api.MetricEventName = "bb.issue"

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

// NewIssueCollector creates a new instance of issueCollector
func NewIssueCollector(l *zap.Logger, store *store.Store) api.MetricCollector {
	return &issueCollector{
		l:     l,
		store: store,
	}
}

// Collect will collect the netric for issue
func (c *issueCollector) Collect(ctx context.Context) ([]*api.Metric, error) {
	var res []*api.Metric

	for _, issueType := range issueTypeList {
		count, err := c.store.CountIssue(ctx, &api.IssueFind{
			Type: &issueType,
		})
		if err != nil {
			c.l.Debug("failed to count issue", zap.String("type", string(issueType)))
			continue
		}
		if count <= 0 {
			continue
		}

		res = append(res, &api.Metric{
			EventName: issueEventName,
			Properties: map[string]interface{}{
				"type":  string(issueType),
				"count": count,
			},
		})
	}

	return res, nil
}
