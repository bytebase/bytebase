package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-ego/gse"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
	Project         *ProjectMessage
	Title           string
	Status          storepb.Issue_Status
	Type            storepb.Issue_Type
	Description     string
	Payload         *storepb.Issue
	PipelineUID     *int // Computed from plan.pipeline_id
	PlanUID         *int64
	TaskStatusCount map[string]int32

	// The following fields are output only and not used for create().
	UID       int
	Creator   *UserMessage
	CreatedAt time.Time
	UpdatedAt time.Time

	// Internal fields.
	projectID  string
	creatorUID int
}

// UpdateIssueMessage is the message for updating an issue.
type UpdateIssueMessage struct {
	Title       *string
	Status      *storepb.Issue_Status
	Description *string
	// PayloadUpsert upserts the presented top-level keys.
	PayloadUpsert *storepb.Issue
	RemoveLabels  bool
}

// FindIssueMessage is the message to find issues.
type FindIssueMessage struct {
	UID        *int
	ProjectID  *string
	ProjectIDs *[]string
	PlanUID    *int64
	PipelineID *int // Filter by pipeline_id (computed from plan)
	// To support pagination, we add into creator.
	// Only principleID or one of the following three fields can be set.
	CreatorID       *int
	CreatedAtBefore *time.Time
	CreatedAtAfter  *time.Time
	Types           *[]storepb.Issue_Type

	StatusList   []storepb.Issue_Status
	TaskTypes    *[]storepb.Task_Type
	MigrateTypes *[]storepb.MigrationType
	// Any of the task in the issue changes the instance with InstanceResourceID.
	InstanceResourceID *string
	// Any of the task in the issue changes the database with InstanceID and DatabaseName.
	InstanceID   *string
	DatabaseName *string
	// Should match the task environment.
	EnvironmentID *string
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit  *int
	Offset *int

	Query *string

	LabelList []string

	NoPipeline bool
}

// GetIssueV2 gets issue by issue UID.
func (s *Store) GetIssueV2(ctx context.Context, find *FindIssueMessage) (*IssueMessage, error) {
	if find.UID != nil {
		if v, ok := s.issueCache.Get(*find.UID); ok && s.enableCache {
			return v, nil
		}
	}
	if find.PipelineID != nil {
		if v, ok := s.issueByPipelineCache.Get(*find.PipelineID); ok && s.enableCache {
			return v, nil
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

	s.issueCache.Add(issue.UID, issue)
	if issue.PipelineUID != nil {
		s.issueByPipelineCache.Add(*issue.PipelineUID, issue)
	}
	return issue, nil
}

// CreateIssueV2 creates a new issue.
func (s *Store) CreateIssueV2(ctx context.Context, create *IssueMessage, creatorID int) (*IssueMessage, error) {
	create.Status = storepb.Issue_OPEN
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal issue payload")
	}
	tsVector := getTSVector(fmt.Sprintf("%s %s", create.Title, create.Description))

	q := qb.Q().Space(`
		INSERT INTO issue (
			creator_id,
			project,
			plan_id,
			name,
			status,
			type,
			description,
			payload,
			ts_vector
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id`,
		creatorID,
		create.Project.ResourceID,
		create.PlanUID,
		create.Title,
		create.Status.String(),
		create.Type.String(),
		create.Description,
		payload,
		tsVector,
	)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(ctx, query, args...).Scan(&create.UID); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetIssueV2(ctx, &FindIssueMessage{UID: &create.UID})
}

// UpdateIssueV2 updates an issue.
func (s *Store) UpdateIssueV2(ctx context.Context, uid int, patch *UpdateIssueMessage) (*IssueMessage, error) {
	oldIssue, err := s.GetIssueV2(ctx, &FindIssueMessage{UID: &uid})
	if err != nil {
		return nil, err
	}

	set := qb.Q()
	set.Comma("updated_at = ?", time.Now())

	if v := patch.Title; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Status; v != nil {
		set.Comma("status = ?", v.String())
	}
	if v := patch.Description; v != nil {
		set.Comma("description = ?", *v)
	}
	if v := patch.PayloadUpsert; v != nil {
		p, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal patch.PayloadUpsert")
		}
		set.Comma("payload = payload || ?", p)
	} else if patch.RemoveLabels {
		set.Comma("payload = payload || jsonb_build_object('labels', ?::JSONB)", nil)
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

		tsVector := getTSVector(fmt.Sprintf("%s %s", title, description))
		set.Comma("ts_vector = ?", tsVector)
	}

	q := qb.Q().Space("UPDATE issue SET ? WHERE id = ?", set, uid)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Invalid the cache and read the value again.
	s.issueCache.Remove(uid)
	if oldIssue.PipelineUID != nil {
		s.issueByPipelineCache.Remove(*oldIssue.PipelineUID)
	}
	return s.GetIssueV2(ctx, &FindIssueMessage{UID: &uid})
}

// ListIssueV2 returns the list of issues by find query.
func (s *Store) ListIssueV2(ctx context.Context, find *FindIssueMessage) ([]*IssueMessage, error) {
	orderByClause := "ORDER BY issue.id DESC"
	from := qb.Q().Space("issue")
	where := qb.Q()

	if v := find.UID; v != nil {
		where.And("issue.id = ?", *v)
	}
	if v := find.PipelineID; v != nil {
		where.And("plan.pipeline_id = ?", *v)
	}
	if v := find.PlanUID; v != nil {
		where.And("issue.plan_id = ?", *v)
	}
	if v := find.ProjectID; v != nil {
		where.And("issue.project = ?", *v)
	}
	if v := find.ProjectIDs; v != nil {
		where.And("issue.project = ANY(?)", *v)
	}
	if v := find.InstanceResourceID; v != nil {
		where.And("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = plan.pipeline_id AND task.instance = ?)", *v)
	}
	if find.InstanceID != nil && find.DatabaseName != nil {
		where.And("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = plan.pipeline_id AND task.instance = ? AND task.db_name = ?)", *find.InstanceID, *find.DatabaseName)
	}
	if v := find.CreatorID; v != nil {
		where.And("issue.creator_id = ?", *v)
	}
	if v := find.CreatedAtBefore; v != nil {
		where.And("issue.created_at < ?", *v)
	}
	if v := find.CreatedAtAfter; v != nil {
		where.And("issue.created_at > ?", *v)
	}
	if v := find.Types; v != nil {
		typeStrings := make([]string, 0, len(*v))
		for _, t := range *v {
			typeStrings = append(typeStrings, t.String())
		}
		where.And("issue.type = ANY(?)", typeStrings)
	}
	if v := find.Query; v != nil && *v != "" {
		if tsQuery := getTSQuery(*v); tsQuery != "" {
			from.Space("LEFT JOIN CAST(? AS tsquery) AS query ON TRUE", tsQuery)
			where.And("issue.ts_vector @@ query")
			orderByClause = "ORDER BY ts_rank(issue.ts_vector, query) DESC, issue.id DESC"
		}
	}
	if len(find.StatusList) != 0 {
		statusStrings := make([]string, 0, len(find.StatusList))
		for _, status := range find.StatusList {
			statusStrings = append(statusStrings, status.String())
		}
		where.And("issue.status = ANY(?)", statusStrings)
	}
	if v := find.TaskTypes; v != nil {
		taskTypeStrings := make([]string, 0, len(*v))
		for _, t := range *v {
			taskTypeStrings = append(taskTypeStrings, t.String())
		}
		if find.MigrateTypes != nil && len(*find.MigrateTypes) > 0 {
			migrateTypeStrings := make([]string, 0, len(*find.MigrateTypes))
			for _, mt := range *find.MigrateTypes {
				migrateTypeStrings = append(migrateTypeStrings, mt.String())
			}
			where.And("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = plan.pipeline_id AND task.type = ANY(?) AND task.payload->>'migrateType' = ANY(?))", taskTypeStrings, migrateTypeStrings)
		} else {
			where.And("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = plan.pipeline_id AND task.type = ANY(?))", taskTypeStrings)
		}
	}
	if v := find.EnvironmentID; v != nil {
		where.And("EXISTS (SELECT 1 FROM task WHERE task.pipeline_id = plan.pipeline_id AND task.environment = ?)", *v)
	}
	if len(find.LabelList) != 0 {
		where.And("payload->'labels' ??& ?::TEXT[]", find.LabelList)
	}
	if find.NoPipeline {
		where.And("plan.pipeline_id IS NULL")
	}

	q := qb.Q().Space(`
		SELECT
			issue.id,
			issue.creator_id,
			issue.created_at,
			issue.updated_at,
			issue.project,
			plan.pipeline_id,
			issue.plan_id,
			COALESCE(plan.name, issue.name) AS name,
			issue.status,
			issue.type,
			COALESCE(plan.description, issue.description) AS description,
			issue.payload,
			COALESCE(task_run_status_count.status_count, '{}'::jsonb)
		FROM ?
		LEFT JOIN plan ON issue.plan_id = plan.id
		LEFT JOIN LATERAL (
			SELECT
				jsonb_object_agg(t.status, t.count) AS status_count
			FROM (
				SELECT
					t.status,
					count(*) AS count
				FROM (
					SELECT
						CASE COALESCE((task.payload->>'skipped')::BOOLEAN, FALSE)
							WHEN TRUE THEN 'SKIPPED'
							ELSE latest_task_run.status
						END AS status
					FROM task
					LEFT JOIN LATERAL(
						SELECT COALESCE(
							(SELECT task_run.status FROM task_run WHERE task_run.task_id = task.id ORDER BY task_run.id DESC LIMIT 1), 'NOT_STARTED'
						) AS status
					) AS latest_task_run ON TRUE
					WHERE task.pipeline_id = plan.pipeline_id
				) AS t
				GROUP BY t.status
			) AS t
		) AS task_run_status_count ON TRUE
		WHERE ?
	`, from, where)
	q.Space(orderByClause)

	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var issues []*IssueMessage
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		issue := IssueMessage{
			Payload: &storepb.Issue{},
		}
		var payload []byte
		var taskRunStatusCount []byte
		var statusString string
		var typeString string
		if err := rows.Scan(
			&issue.UID,
			&issue.creatorUID,
			&issue.CreatedAt,
			&issue.UpdatedAt,
			&issue.projectID,
			&issue.PipelineUID,
			&issue.PlanUID,
			&issue.Title,
			&statusString,
			&typeString,
			&issue.Description,
			&payload,
			&taskRunStatusCount,
		); err != nil {
			return nil, err
		}
		if statusValue, ok := storepb.Issue_Status_value[statusString]; ok {
			issue.Status = storepb.Issue_Status(statusValue)
		} else {
			return nil, errors.Errorf("invalid status string: %s", statusString)
		}
		if typeValue, ok := storepb.Issue_Type_value[typeString]; ok {
			issue.Type = storepb.Issue_Type(typeValue)
		} else {
			return nil, errors.Errorf("invalid type string: %s", typeString)
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, issue.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal issue payload")
		}
		if err := json.Unmarshal(taskRunStatusCount, &issue.TaskStatusCount); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal task run status count")
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
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &issue.projectID})
		if err != nil {
			return nil, err
		}
		issue.Project = project
		creator, err := s.GetUserByID(ctx, issue.creatorUID)
		if err != nil {
			return nil, err
		}
		issue.Creator = creator

		s.issueCache.Add(issue.UID, issue)
		if issue.PipelineUID != nil {
			s.issueByPipelineCache.Add(*issue.PipelineUID, issue)
		}
	}

	return issues, nil
}

// BatchUpdateIssueStatuses updates the status of multiple issues.
func (s *Store) BatchUpdateIssueStatuses(ctx context.Context, issueUIDs []int, status storepb.Issue_Status) error {
	// First get the pipeline IDs for cache invalidation
	selectQuery := qb.Q().Space(`
		SELECT issue.id, plan.pipeline_id
		FROM issue
		LEFT JOIN plan ON issue.plan_id = plan.id
		WHERE issue.id = ANY(?)
	`, issueUIDs)
	selectSQL, selectArgs, err := selectQuery.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build select sql")
	}

	type issueInfo struct {
		id          int
		pipelineUID *int
	}
	var infos []issueInfo

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, selectSQL, selectArgs...)
	if err != nil {
		return errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()
	for rows.Next() {
		var info issueInfo
		if err := rows.Scan(&info.id, &info.pipelineUID); err != nil {
			return errors.Wrapf(err, "failed to scan")
		}
		infos = append(infos, info)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to scan issues")
	}

	// Update the statuses
	updateQuery := qb.Q().Space("UPDATE issue SET status = ? WHERE id = ANY(?)", status.String(), issueUIDs)
	updateSQL, updateArgs, err := updateQuery.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build update sql")
	}

	if _, err := tx.ExecContext(ctx, updateSQL, updateArgs...); err != nil {
		return errors.Wrapf(err, "failed to update")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	// Invalidate caches
	for _, info := range infos {
		s.issueCache.Remove(info.id)
		if info.pipelineUID != nil {
			s.issueByPipelineCache.Remove(*info.pipelineUID)
		}
	}

	return nil
}

func getTSVector(text string) string {
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

func getTSQuery(text string) string {
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

func (s *Store) BackfillIssueTSVector(ctx context.Context) error {
	chunkSize := 50
	offset := 0

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	for {
		selectQuery := qb.Q().Space(`
			SELECT
				issue.id,
				COALESCE(plan.name, issue.name) AS name,
				COALESCE(plan.description, issue.description) AS description
			FROM issue
			LEFT JOIN plan ON issue.plan_id = plan.id
			WHERE issue.ts_vector IS NULL
			ORDER BY issue.id
			LIMIT ?
			OFFSET ?`, chunkSize, offset)

		selectSQL, selectArgs, err := selectQuery.ToSQL()
		if err != nil {
			return errors.Wrapf(err, "failed to build select sql")
		}

		var issues []*IssueMessage
		if err := func() error {
			rows, err := tx.QueryContext(ctx, selectSQL, selectArgs...)
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
			tsVector := getTSVector(fmt.Sprintf("%s %s", issue.Title, issue.Description))
			updateQuery := qb.Q().Space("UPDATE issue SET ts_vector = ? WHERE id = ?", tsVector, issue.UID)
			updateSQL, updateArgs, err := updateQuery.ToSQL()
			if err != nil {
				return errors.Wrapf(err, "failed to build update sql")
			}
			if _, err := tx.ExecContext(ctx, updateSQL, updateArgs...); err != nil {
				return errors.Wrapf(err, "failed to update")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit")
	}

	return nil
}
