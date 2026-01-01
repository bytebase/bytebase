package store

import (
	"context"
	"database/sql"
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
	ProjectID    string
	CreatorEmail string
	Title        string
	Status       storepb.Issue_Status
	Type         storepb.Issue_Type
	Description  string
	Payload      *storepb.Issue
	PlanUID      *int64

	// The following fields are output only and not used for create().
	UID       int
	CreatedAt time.Time
	UpdatedAt time.Time
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
	UID             *int
	ProjectID       *string
	ProjectIDs      *[]string
	PlanUID         *int64
	CreatorID       *string
	CreatedAtBefore *time.Time
	CreatedAtAfter  *time.Time
	Types           *[]storepb.Issue_Type

	StatusList []storepb.Issue_Status
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit  *int
	Offset *int

	Query *string

	LabelList []string
}

// GetIssue gets issue by issue UID.
func (s *Store) GetIssue(ctx context.Context, find *FindIssueMessage) (*IssueMessage, error) {
	issues, err := s.ListIssues(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, nil
	}
	if len(issues) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d issues with find %#v, expect 1", len(issues), find)}
	}
	return issues[0], nil
}

// CreateIssue creates a new issue.
func (s *Store) CreateIssue(ctx context.Context, create *IssueMessage) (*IssueMessage, error) {
	create.Status = storepb.Issue_OPEN
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal issue payload")
	}
	tsVector := getTSVector(fmt.Sprintf("%s %s", create.Title, create.Description))

	q := qb.Q().Space(`
		INSERT INTO issue (
			creator,
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
		create.CreatorEmail,
		create.ProjectID,
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

	return s.GetIssue(ctx, &FindIssueMessage{UID: &create.UID})
}

// UpdateIssue updates an issue.
func (s *Store) UpdateIssue(ctx context.Context, uid int, patch *UpdateIssueMessage) (*IssueMessage, error) {
	oldIssue, err := s.GetIssue(ctx, &FindIssueMessage{UID: &uid})
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

	return s.GetIssue(ctx, &FindIssueMessage{UID: &uid})
}

// ListIssues returns the list of issues by find query.
func (s *Store) ListIssues(ctx context.Context, find *FindIssueMessage) ([]*IssueMessage, error) {
	orderByClause := "ORDER BY issue.id DESC"
	from := qb.Q().Space("issue")
	where := qb.Q()

	if v := find.UID; v != nil {
		where.And("issue.id = ?", *v)
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
	if v := find.CreatorID; v != nil {
		where.And("issue.creator = ?", *v)
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
		searchCondition := qb.Q()
		if tsQuery := getTSQuery(*v); tsQuery != "" {
			from.Space("LEFT JOIN CAST(? AS tsquery) AS query ON TRUE", tsQuery)
			searchCondition.Or("issue.ts_vector @@ query")
			orderByClause = "ORDER BY ts_rank(issue.ts_vector, query) DESC, issue.id DESC"
		}
		searchCondition.Or("issue.name ILIKE ?", "%"+*v+"%")
		where.And("(?)", searchCondition)
	}
	if len(find.StatusList) != 0 {
		statusStrings := make([]string, 0, len(find.StatusList))
		for _, status := range find.StatusList {
			statusStrings = append(statusStrings, status.String())
		}
		where.And("issue.status = ANY(?)", statusStrings)
	}
	if len(find.LabelList) != 0 {
		where.And("payload->'labels' ??& ?::TEXT[]", find.LabelList)
	}

	q := qb.Q().Space(`
		SELECT
			issue.id,
			issue.creator,
			issue.created_at,
			issue.updated_at,
			issue.project,
			issue.plan_id,
			issue.name,
			issue.status,
			issue.type,
			issue.description,
			issue.payload
		FROM ?
	`, from)
	if where.Len() > 0 {
		q.Space("WHERE ?", where)
	}
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
		var statusString string
		var typeString string
		if err := rows.Scan(
			&issue.UID,
			&issue.CreatorEmail,
			&issue.CreatedAt,
			&issue.UpdatedAt,
			&issue.ProjectID,
			&issue.PlanUID,
			&issue.Title,
			&statusString,
			&typeString,
			&issue.Description,
			&payload,
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
		issues = append(issues, &issue)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return issues, nil
}

// BatchUpdateIssueStatuses updates the status of multiple issues.
// Returns a map of issueUID -> old status for the updated issues.
func (s *Store) BatchUpdateIssueStatuses(ctx context.Context, projectID string, issueUIDs []int, newStatus storepb.Issue_Status) (map[int]storepb.Issue_Status, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Fetch current issues to validate project membership and get old statuses.
	fetchQuery := qb.Q().Space("SELECT id, status FROM issue WHERE id = ANY(?) AND project = ?", issueUIDs, projectID)
	fetchSQL, fetchArgs, err := fetchQuery.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build fetch sql")
	}

	rows, err := tx.QueryContext(ctx, fetchSQL, fetchArgs...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch issues")
	}
	defer rows.Close()

	oldStatuses := make(map[int]storepb.Issue_Status)
	for rows.Next() {
		var issueID int
		var statusString string
		if err := rows.Scan(&issueID, &statusString); err != nil {
			return nil, errors.Wrapf(err, "failed to scan issue")
		}
		statusValue, ok := storepb.Issue_Status_value[statusString]
		if !ok {
			return nil, errors.Errorf("invalid status string: %s", statusString)
		}
		issueStatus := storepb.Issue_Status(statusValue)

		// Prevent changing status from DONE to other statuses.
		if issueStatus == storepb.Issue_DONE && newStatus != storepb.Issue_DONE {
			return nil, &common.Error{Code: common.Invalid, Err: errors.Errorf("cannot change status from DONE to %s for issue %d", newStatus.String(), issueID)}
		}

		oldStatuses[issueID] = issueStatus
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "failed to iterate rows")
	}

	// Validate that all requested issues were found in the project.
	if len(oldStatuses) != len(issueUIDs) {
		return nil, &common.Error{Code: common.Invalid, Err: errors.Errorf("expected %d issues in project %s, found %d", len(issueUIDs), projectID, len(oldStatuses))}
	}

	// Update the statuses.
	updateQuery := qb.Q().Space("UPDATE issue SET status = ? WHERE id = ANY(?) AND project = ?", newStatus.String(), issueUIDs, projectID)
	updateSQL, updateArgs, err := updateQuery.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build update sql")
	}

	if _, err := tx.ExecContext(ctx, updateSQL, updateArgs...); err != nil {
		return nil, errors.Wrapf(err, "failed to update")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit")
	}
	return oldStatuses, nil
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
				issue.name,
				issue.description
			FROM issue
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
