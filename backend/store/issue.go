package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-ego/gse"
	"github.com/jackc/pgtype"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var getSegmenter func() *gse.Segmenter

func init() {
	var segmenterDic gse.Segmenter
	if err := segmenterDic.LoadDictEmbed("zh"); err != nil {
		panic(errors.Wrapf(err, "failed to load segmenter dictionary"))
	}
	getSegmenter = func() *gse.Segmenter {
		var segmenter gse.Segmenter
		segmenter.Dict = segmenterDic.Dict
		return &segmenter
	}
}

// IssueMessage is the mssage for issues.
type IssueMessage struct {
	Project     *ProjectMessage
	Title       string
	Status      api.IssueStatus
	Type        api.IssueType
	Description string
	Assignee    *UserMessage
	Payload     *storepb.IssuePayload
	Subscribers []*UserMessage
	PipelineUID *int
	PlanUID     *int64

	// The following fields are output only and not used for create().
	UID         int
	Creator     *UserMessage
	CreatedTime time.Time
	Updater     *UserMessage
	UpdatedTime time.Time

	// Internal fields.
	projectUID     int
	assigneeUID    *int
	subscriberUIDs []int
	creatorUID     int
	createdTs      int64
	updaterUID     int
	updatedTs      int64
}

// UpdateIssueMessage is the message for updating an issue.
type UpdateIssueMessage struct {
	Title          *string
	Status         *api.IssueStatus
	Description    *string
	UpdateAssignee bool
	Assignee       *UserMessage
	// PayloadUpsert upserts the presented top-level keys.
	PayloadUpsert *storepb.IssuePayload
	Subscribers   *[]*UserMessage

	PipelineUID *int
}

// FindIssueMessage is the message to find issues.
type FindIssueMessage struct {
	UID        *int
	ProjectIDs *[]string
	PlanUID    *int64
	PipelineID *int
	// To support pagination, we add into creator, assignee and subscriber.
	// Only principleID or one of the following three fields can be set.
	CreatorID       *int
	AssigneeID      *int
	SubscriberID    *int
	CreatedTsBefore *int64
	CreatedTsAfter  *int64

	StatusList []api.IssueStatus
	TaskTypes  *[]api.TaskType
	// Any of the task in the issue changes the instance with InstanceResourceID.
	InstanceResourceID *string
	// Any of the task in the issue changes the database with DatabaseUID.
	DatabaseUID *int
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit  *int
	Offset *int

	Query *string
}

// GetIssueV2 gets issue by issue UID.
func (s *Store) GetIssueV2(ctx context.Context, find *FindIssueMessage) (*IssueMessage, error) {
	if find.UID != nil {
		if issue, ok := s.issueCache.Load(*find.UID); ok {
			if v, ok := issue.(*IssueMessage); ok {
				return v, nil
			}
		}
	}
	if find.PipelineID != nil {
		if issue, ok := s.issueByPipelineCache.Load(*find.PipelineID); ok {
			if v, ok := issue.(*IssueMessage); ok {
				return v, nil
			}
		}
	}

	issues, err := s.ListIssueV2(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, nil
	}
	if len(issues) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d issues with find %#v, expect 1", len(issues), find)}
	}
	issue := issues[0]

	s.issueCache.Store(issue.UID, issue)
	s.issueByPipelineCache.Store(issue.PipelineUID, issue)
	return issue, nil
}

// CreateIssueV2 creates a new issue.
func (s *Store) CreateIssueV2(ctx context.Context, create *IssueMessage, creatorID int) (*IssueMessage, error) {
	create.Status = api.IssueOpen

	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal issue payload")
	}
	creator, err := s.GetUserByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	tsVector := getTsVector(fmt.Sprintf("%s %s", create.Title, create.Description))
	var assigneeID *int
	if create.Assignee != nil {
		assigneeID = &create.Assignee.ID
	}
	query := `
		INSERT INTO issue (
			creator_id,
			updater_id,
			project_id,
			pipeline_id,
			plan_id,
			name,
			status,
			type,
			description,
			assignee_id,
			payload,
			ts_vector
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_ts, updated_ts
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query,
		creatorID,
		creatorID,
		create.Project.UID,
		create.PipelineUID,
		create.PlanUID,
		create.Title,
		create.Status,
		create.Type,
		create.Description,
		assigneeID,
		payload,
		tsVector,
	).Scan(
		&create.UID,
		&create.createdTs,
		&create.updatedTs,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	create.CreatedTime = time.Unix(create.createdTs, 0)
	create.UpdatedTime = time.Unix(create.updatedTs, 0)
	create.Creator = creator
	create.Updater = creator

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.issueCache.Store(create.UID, create)
	s.issueByPipelineCache.Store(create.PipelineUID, create)
	return create, nil
}

// UpdateIssueV2 updates an issue.
func (s *Store) UpdateIssueV2(ctx context.Context, uid int, patch *UpdateIssueMessage, updaterID int) (*IssueMessage, error) {
	oldIssue, err := s.GetIssueV2(ctx, &FindIssueMessage{UID: &uid})
	if err != nil {
		return nil, err
	}

	set, args := []string{"updater_id = $1"}, []any{updaterID}

	if v := patch.PipelineUID; v != nil {
		set, args = append(set, fmt.Sprintf("pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Status; v != nil {
		set, args = append(set, fmt.Sprintf("status = $%d", len(args)+1)), append(args, api.IssueStatus(*v))
	}
	if v := patch.Description; v != nil {
		set, args = append(set, fmt.Sprintf("description = $%d", len(args)+1)), append(args, *v)
	}
	if patch.UpdateAssignee {
		if v := patch.Assignee; v != nil {
			set, args = append(set, fmt.Sprintf("assignee_id = $%d", len(args)+1)), append(args, v.ID)
		} else {
			set = append(set, "assignee_id = NULL")
		}
	}
	if v := patch.PayloadUpsert; v != nil {
		p, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal patch.PayloadUpsert")
		}
		set, args = append(set, fmt.Sprintf("payload = payload || $%d", len(args)+1)), append(args, p)
	}
	if patch.Title != nil || patch.Description != nil {
		title := oldIssue.Title
		if patch.Title != nil {
			title = *patch.Title
		}
		description := oldIssue.Description
		if patch.Description != nil {
			description = *patch.Description
		}

		tsVector := getTsVector(fmt.Sprintf("%s %s", title, description))
		set = append(set, fmt.Sprintf("ts_vector = $%d", len(args)+1))
		args = append(args, tsVector)
	}

	args = append(args, uid)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE issue
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d`, len(args)),
		args...,
	); err != nil {
		return nil, err
	}

	if patch.Subscribers != nil {
		if err := setSubscribers(ctx, tx, uid, *patch.Subscribers); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalid the cache and read the value again.
	s.issueCache.Delete(uid)
	s.issueByPipelineCache.Delete(oldIssue.PipelineUID)
	return s.GetIssueV2(ctx, &FindIssueMessage{UID: &uid})
}

func setSubscribers(ctx context.Context, tx *Tx, issueUID int, subscribers []*UserMessage) error {
	subscriberIDs := make(map[int]bool)
	for _, subscriber := range subscribers {
		subscriberIDs[subscriber.ID] = true
	}

	oldSubscriberIDs := make(map[int]bool)
	rows, err := tx.QueryContext(ctx, `
		SELECT
			subscriber_id
		FROM issue_subscriber
		WHERE issue_id = $1`,
		issueUID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var subscriberID int
		if err := rows.Scan(
			&subscriberID,
		); err != nil {
			return err
		}

		oldSubscriberIDs[subscriberID] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	var adds, deletes []int
	for v := range oldSubscriberIDs {
		if _, ok := subscriberIDs[v]; !ok {
			deletes = append(deletes, v)
		}
	}
	for v := range subscriberIDs {
		if _, ok := oldSubscriberIDs[v]; !ok {
			adds = append(adds, v)
		}
	}
	if len(adds) > 0 {
		var tokens []string
		var args []any
		for i, v := range adds {
			tokens = append(tokens, fmt.Sprintf("($%d, $%d)", 2*i+1, 2*i+2))
			args = append(args, issueUID, v)
		}
		query := fmt.Sprintf(`INSERT INTO issue_subscriber (issue_id, subscriber_id) VALUES %s`, strings.Join(tokens, ", "))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	if len(deletes) > 0 {
		var tokens []string
		var args []any
		args = append(args, issueUID)
		for i, v := range deletes {
			tokens = append(tokens, fmt.Sprintf("$%d", i+2))
			args = append(args, v)
		}
		query := fmt.Sprintf(`DELETE FROM issue_subscriber WHERE issue_id = $1 AND subscriber_id IN (%s)`, strings.Join(tokens, ", "))
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

// ListIssueV2 returns the list of issues by find query.
func (s *Store) ListIssueV2(ctx context.Context, find *FindIssueMessage) ([]*IssueMessage, error) {
	orderByClause := "ORDER BY issue.id DESC"
	from := "issue"
	where, args := []string{"TRUE"}, []any{}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PipelineID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.pipeline_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PlanUID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.plan_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectIDs; v != nil {
		where, args = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM project WHERE project.id = issue.project_id AND project.resource_id = ANY($%d))", len(args)+1)), append(args, *v)
	}
	if v := find.InstanceResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM task LEFT JOIN instance ON instance.id = task.instance_id WHERE task.pipeline_id = issue.pipeline_id AND instance.resource_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = issue.pipeline_id AND task.database_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.AssigneeID; v != nil {
		where, args = append(where, fmt.Sprintf("issue.assignee_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatedTsBefore; v != nil {
		where, args = append(where, fmt.Sprintf("issue.created_ts < $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatedTsAfter; v != nil {
		where, args = append(where, fmt.Sprintf("issue.created_ts > $%d", len(args)+1)), append(args, *v)
	}
	if v := find.SubscriberID; v != nil {
		where, args = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM issue_subscriber WHERE issue_subscriber.issue_id = issue.id AND issue_subscriber.subscriber_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.Query; v != nil && *v != "" {
		if tsQuery := getTsQuery(*v); tsQuery != "" {
			from += fmt.Sprintf(`, CAST($%d AS tsquery) AS query`, len(args)+1)
			args = append(args, tsQuery)
			where = append(where, "issue.ts_vector @@ query")
			orderByClause = "ORDER BY ts_rank(issue.ts_vector, query) DESC, issue.id DESC"
		}
	}
	if len(find.StatusList) != 0 {
		var list []string
		for _, status := range find.StatusList {
			list = append(list, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, status)
		}
		where = append(where, fmt.Sprintf("issue.status IN (%s)", strings.Join(list, ", ")))
	}
	if v := find.TaskTypes; v != nil {
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = issue.pipeline_id AND task.type = ANY($%d))", len(args)+1))
		args = append(args, *v)
	}
	limitOffsetClause := ""
	if v := find.Limit; v != nil {
		limitOffsetClause = fmt.Sprintf(" LIMIT %d", *v)
	}
	if v := find.Offset; v != nil {
		limitOffsetClause += fmt.Sprintf(" OFFSET %d", *v)
	}

	var issues []*IssueMessage
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
	SELECT
		issue.id,
		issue.creator_id,
		issue.created_ts,
		issue.updater_id,
		issue.updated_ts,
		issue.project_id,
		issue.pipeline_id,
		issue.plan_id,
		issue.name,
		issue.status,
		issue.type,
		issue.description,
		issue.assignee_id,
		issue.payload,
		(SELECT ARRAY_AGG (issue_subscriber.subscriber_id) FROM issue_subscriber WHERE issue_subscriber.issue_id = issue.id) subscribers
	FROM %s
	WHERE %s
	%s
	%s`, from, strings.Join(where, " AND "), orderByClause, limitOffsetClause)

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		issue := IssueMessage{
			Payload: &storepb.IssuePayload{},
		}
		var payload []byte
		var pipelineUID sql.NullInt32
		var subscriberUIDs pgtype.Int4Array
		if err := rows.Scan(
			&issue.UID,
			&issue.creatorUID,
			&issue.createdTs,
			&issue.updaterUID,
			&issue.updatedTs,
			&issue.projectUID,
			&pipelineUID,
			&issue.PlanUID,
			&issue.Title,
			&issue.Status,
			&issue.Type,
			&issue.Description,
			&issue.assigneeUID,
			&payload,
			&subscriberUIDs,
		); err != nil {
			return nil, err
		}
		if err := subscriberUIDs.AssignTo(&issue.subscriberUIDs); err != nil {
			return nil, err
		}
		if pipelineUID.Valid {
			v := int(pipelineUID.Int32)
			issue.PipelineUID = &v
		}
		if err := protojson.Unmarshal(payload, issue.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal issue payload")
		}
		issues = append(issues, &issue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Populate from internal fields.
	for _, issue := range issues {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{UID: &issue.projectUID})
		if err != nil {
			return nil, err
		}
		issue.Project = project
		if issue.assigneeUID != nil {
			assignee, err := s.GetUserByID(ctx, *issue.assigneeUID)
			if err != nil {
				return nil, err
			}
			issue.Assignee = assignee
		}
		creator, err := s.GetUserByID(ctx, issue.creatorUID)
		if err != nil {
			return nil, err
		}
		issue.Creator = creator
		updater, err := s.GetUserByID(ctx, issue.updaterUID)
		if err != nil {
			return nil, err
		}
		issue.Updater = updater
		for _, subscriberUID := range issue.subscriberUIDs {
			subscriber, err := s.GetUserByID(ctx, subscriberUID)
			if err != nil {
				return nil, err
			}
			issue.Subscribers = append(issue.Subscribers, subscriber)
		}
		issue.CreatedTime = time.Unix(issue.createdTs, 0)
		issue.UpdatedTime = time.Unix(issue.updatedTs, 0)

		s.issueCache.Store(issue.UID, issue)
		s.issueByPipelineCache.Store(issue.PipelineUID, issue)
	}

	return issues, nil
}

// BatchUpdateIssueStatuses updates the status of multiple issues.
func (s *Store) BatchUpdateIssueStatuses(ctx context.Context, issueUIDs []int, status api.IssueStatus, updaterID int) error {
	var ids []string
	for _, id := range issueUIDs {
		ids = append(ids, fmt.Sprintf("%d", id))
	}
	query := fmt.Sprintf(`
		UPDATE issue
		SET status = $1, updater_id = $2
		WHERE id IN (%s)
		RETURNING id, pipeline_id;
	`, strings.Join(ids, ","))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, status, updaterID)
	if err != nil {
		return errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	var issueIDs []int
	var pipelineIDs []int
	for rows.Next() {
		var issueID int
		var pipelineID sql.NullInt32
		if err := rows.Scan(&issueID, &pipelineID); err != nil {
			return errors.Wrapf(err, "failed to scan")
		}
		issueIDs = append(issueIDs, issueID)
		if pipelineID.Valid {
			pipelineIDs = append(pipelineIDs, int(pipelineID.Int32))
		}
	}
	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to scan")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	for _, issueID := range issueIDs {
		s.issueCache.Delete(issueID)
	}
	for _, pipelineID := range pipelineIDs {
		s.issueByPipelineCache.Delete(pipelineID)
	}

	return nil
}

func getTsVector(text string) string {
	seg := getSegmenter()
	parts := seg.CutTrim(text)
	var tsVector strings.Builder
	for i, part := range parts {
		if i != 0 {
			_, _ = tsVector.WriteString(" ")
		}
		_, _ = tsVector.WriteString(fmt.Sprintf("%s:%d", part, i+1))
	}
	return tsVector.String()
}

func getTsQuery(text string) string {
	seg := getSegmenter()
	parts := seg.Trim(seg.CutSearch(text))
	// CutSearch returns empty for a single word.
	if len(parts) == 0 {
		parts = seg.CutTrim(text)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%s:*", text)
	}
	var tsQuery strings.Builder
	for i, part := range parts {
		if i != 0 {
			_, _ = tsQuery.WriteString("|")
		}
		_, _ = tsQuery.WriteString(fmt.Sprintf("%s:*", part))
	}
	return tsQuery.String()
}

func (s *Store) BackfillIssueTsVector(ctx context.Context) error {
	chunkSize := 50
	offset := 0
	selectQuery := `
		SELECT id, name, description
		FROM issue
		WHERE ts_vector IS NULL
		ORDER BY id
		LIMIT $1
		OFFSET $2
	`
	updateStatement := `
		UPDATE issue
		SET ts_vector = $1
		WHERE id = $2
	`
	disableTriggerStatement := "ALTER TABLE issue DISABLE TRIGGER update_issue_updated_ts"
	enableTriggerStatement := "ALTER TABLE issue ENABLE TRIGGER update_issue_updated_ts"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, disableTriggerStatement); err != nil {
		return errors.Wrapf(err, "failed to disable trigger")
	}

	for {
		var issues []*IssueMessage
		if err := func() error {
			rows, err := tx.QueryContext(ctx, selectQuery, chunkSize, offset)
			if err != nil {
				return errors.Wrapf(err, "failed to query")
			}
			defer rows.Close()
			for rows.Next() {
				var issue IssueMessage
				if err := rows.Scan(&issue.UID, &issue.Title, &issue.Description); err != nil {
					return errors.Wrapf(err, "failed to scan")
				}
				issues = append(issues, &issue)
			}
			if err := rows.Err(); err != nil {
				return errors.Wrapf(err, "failed to scan")
			}
			return nil
		}(); err != nil {
			return err
		}

		if len(issues) == 0 {
			break
		}
		offset += len(issues)

		for _, issue := range issues {
			tsVector := getTsVector(fmt.Sprintf("%s %s", issue.Title, issue.Description))
			if _, err := tx.ExecContext(ctx, updateStatement, tsVector, issue.UID); err != nil {
				return errors.Wrapf(err, "failed to update")
			}
		}
	}

	if _, err := tx.ExecContext(ctx, enableTriggerStatement); err != nil {
		return errors.Wrapf(err, "failed to enable trigger")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	return nil
}
