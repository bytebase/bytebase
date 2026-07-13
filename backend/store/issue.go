package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
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

// ErrIssueUpdateSkipped indicates that a guarded issue update did not match the current row state.
var ErrIssueUpdateSkipped = errors.New("issue update skipped")

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

// CanonicalizeIssueLabels returns the deterministic representation stored in issue payloads.
func CanonicalizeIssueLabels(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}

	canonicalLabels := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		canonicalLabels = append(canonicalLabels, label)
	}
	if len(canonicalLabels) == 0 {
		return nil
	}

	slices.Sort(canonicalLabels)
	return slices.Compact(canonicalLabels)
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
	UID       int64
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
	// ConditionalPayloadUpsert upserts the presented top-level keys only if
	// ConditionalPlanApprovalInputVersion matches. The main update still applies
	// if it does not.
	ConditionalPayloadUpsert *storepb.Issue
	RemoveLabels             bool

	RequirePlanApprovalInputVersion  *int64
	RequireIssueApprovalInputVersion *int64
	RequireApprovalFindingDone       *bool
	RequireLabels                    *[]string
	RequireNoRollout                 bool
	// SkipIfCurrentApprovalFindingDone skips when approval finding is already done
	// for the same approval input version.
	SkipIfCurrentApprovalFindingDone *int64

	ConditionalPlanApprovalInputVersion *int64
	ConditionalRequireNoRollout         bool
}

// FindIssueMessage is the message to find issues.
type FindIssueMessage struct {
	// Workspace filters issues by the parent project's workspace.
	// Empty string skips filtering (for cross-workspace queries like runners).
	Workspace string
	// Required field
	ProjectIDs []string

	UID       *int64
	PlanUID   *int64
	PlanUIDs  *[]int64
	CreatorID *string

	CreatedAtBefore *time.Time
	CreatedAtAfter  *time.Time
	Types           *[]storepb.Issue_Type

	StatusList []storepb.Issue_Status
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit  *int
	Offset *int

	Query        *string
	ExcludeDraft bool

	LabelList     []string
	RiskLevelList []storepb.RiskLevel
	OrderByKeys   []*OrderByKey
}

// GetIssueOrders parses the order_by string and returns the corresponding OrderByKeys.
func GetIssueOrders(orderBy string) ([]*OrderByKey, error) {
	keys, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, err
	}

	orderByKeys := []*OrderByKey{}
	for _, key := range keys {
		switch key.Key {
		case "create_time":
			orderByKeys = append(orderByKeys, &OrderByKey{
				Key:       "issue.created_at",
				SortOrder: key.SortOrder,
			})
		case "update_time":
			orderByKeys = append(orderByKeys, &OrderByKey{
				Key:       "issue.updated_at",
				SortOrder: key.SortOrder,
			})
		default:
			return nil, errors.Errorf(`invalid order key "%v"`, key.Key)
		}
	}
	return orderByKeys, nil
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
	if create.Payload != nil {
		create.Payload.Labels = CanonicalizeIssueLabels(create.Payload.Labels)
	}
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal issue payload")
	}
	tsVector := getTSVector(fmt.Sprintf("%s %s", create.Title, create.Description))

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	if create.PlanUID != nil {
		if err := acquirePlanIssueRolloutAdvisoryLock(ctx, tx, create.ProjectID, *create.PlanUID); err != nil {
			return nil, errors.Wrap(err, "failed to acquire plan issue-rollout lock")
		}

		var hasRollout bool
		if err := tx.QueryRowContext(ctx, `
			SELECT COALESCE((config->>'hasRollout')::boolean, false)
			FROM plan
			WHERE project = $1
			  AND id = $2`,
			create.ProjectID, *create.PlanUID).Scan(&hasRollout); err != nil {
			return nil, errors.Wrapf(err, "failed to get plan %d", *create.PlanUID)
		}
		if create.Payload.GetDraft() && hasRollout {
			return nil, ErrPlanHasRollout
		}
	}

	nextID, err := nextProjectID(ctx, tx, "issue", create.ProjectID)
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO issue (
			id,
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nextID,
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

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	create.UID = nextID
	return s.GetIssue(ctx, &FindIssueMessage{ProjectIDs: []string{create.ProjectID}, UID: &create.UID})
}

// UpdateIssue updates an issue.
func (s *Store) UpdateIssue(ctx context.Context, projectID string, uid int64, patch *UpdateIssueMessage) (*IssueMessage, error) {
	oldIssue, err := s.GetIssue(ctx, &FindIssueMessage{ProjectIDs: []string{projectID}, UID: &uid})
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
	payloadSet := qb.Q().Space("payload")
	if v := patch.PayloadUpsert; v != nil {
		v.Labels = CanonicalizeIssueLabels(v.Labels)
		p, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal patch.PayloadUpsert")
		}
		payloadSet.Space("|| ?::jsonb", string(p))
	}
	if patch.RemoveLabels {
		payloadSet.Space("|| jsonb_build_object('labels', ?::JSONB)", nil)
	}
	if v := patch.ConditionalPayloadUpsert; v != nil {
		if patch.ConditionalPlanApprovalInputVersion == nil {
			return nil, errors.New("ConditionalPayloadUpsert requires ConditionalPlanApprovalInputVersion")
		}
		v.Labels = CanonicalizeIssueLabels(v.Labels)
		p, err := protojson.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal patch.ConditionalPayloadUpsert")
		}
		condition := issuePlanApprovalInputVersionCondition(*patch.ConditionalPlanApprovalInputVersion, patch.ConditionalRequireNoRollout)
		payloadSet.Space("|| CASE WHEN ? THEN ?::jsonb ELSE '{}'::jsonb END", condition, string(p))
	}
	if payloadSet.Len() > 1 {
		set.Comma("payload = ?", payloadSet)
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

	where, hasGuard, err := buildUpdateIssueGuard(patch, projectID, uid)
	if err != nil {
		return nil, err
	}
	q := qb.Q().Space("UPDATE issue SET ? WHERE ?", set, where)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if hasGuard {
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to inspect issue update")
		}
		if rowsAffected == 0 {
			return nil, ErrIssueUpdateSkipped
		}
	}

	return s.GetIssue(ctx, &FindIssueMessage{ProjectIDs: []string{projectID}, UID: &uid})
}

// UpdateIssueLabelsAndMaybeResetApproval updates issue labels and, while the linked plan
// has no rollout, resets approval in the same transaction. This remains separate from
// generic UpdateIssue because labels and approval reset are one domain invariant here:
// composing a label write with a conditional approval reset can persist labels while
// silently skipping the reset during plan-version or rollout races.
func (s *Store) UpdateIssueLabelsAndMaybeResetApproval(ctx context.Context, projectID string, uid int64, labels []string) (*IssueMessage, bool, error) {
	labels = CanonicalizeIssueLabels(labels)

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	var planUID sql.NullInt64
	currentPayload := &storepb.Issue{}
	var payload []byte
	if err := tx.QueryRowContext(ctx, `
		SELECT plan_id, payload
		FROM issue
		WHERE project = $1
		  AND id = $2
		FOR UPDATE`,
		projectID, uid).Scan(&planUID, &payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "failed to lock issue")
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, currentPayload); err != nil {
		return nil, false, errors.Wrap(err, "failed to unmarshal issue payload")
	}
	if slices.Equal(CanonicalizeIssueLabels(currentPayload.GetLabels()), labels) {
		if err := tx.Commit(); err != nil {
			return nil, false, errors.Wrapf(err, "failed to commit tx")
		}
		issue, err := s.GetIssue(ctx, &FindIssueMessage{ProjectIDs: []string{projectID}, UID: &uid})
		if err != nil {
			return nil, false, err
		}
		return issue, false, nil
	}

	approvalResetApplied := false
	var approvalInputVersion int64
	if planUID.Valid {
		planConfig := &storepb.PlanConfig{}
		var config []byte
		if err := tx.QueryRowContext(ctx, `
			SELECT config
			FROM plan
			WHERE project = $1
			  AND id = $2
			FOR UPDATE`,
			projectID, planUID.Int64).Scan(&config); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, false, errors.Errorf("plan %d not found", planUID.Int64)
			}
			return nil, false, errors.Wrapf(err, "failed to lock plan")
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal(config, planConfig); err != nil {
			return nil, false, errors.Wrap(err, "failed to unmarshal plan config")
		}
		if !planConfig.GetHasRollout() {
			approvalResetApplied = true
			approvalInputVersion = planConfig.GetApprovalInputVersion()
		}
	}

	payloadSet := qb.Q().Space("payload")
	if len(labels) == 0 {
		payloadSet.Space("|| jsonb_build_object('labels', ?::JSONB)", nil)
	} else {
		p, err := protojson.Marshal(&storepb.Issue{Labels: labels})
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to marshal issue labels")
		}
		payloadSet.Space("|| ?::jsonb", string(p))
	}
	if approvalResetApplied {
		p, err := protojson.Marshal(&storepb.Issue{
			Approval: &storepb.IssuePayloadApproval{
				ApprovalFindingDone:  false,
				ApprovalInputVersion: approvalInputVersion,
			},
		})
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to marshal issue approval reset")
		}
		payloadSet.Space("|| ?::jsonb", string(p))
	}

	q := qb.Q().Space(`
		UPDATE issue
		SET
			updated_at = ?,
			payload = ?
		WHERE project = ?
		  AND id = ?`,
		time.Now(), payloadSet, projectID, uid)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, false, err
	}

	if err := tx.Commit(); err != nil {
		return nil, false, errors.Wrapf(err, "failed to commit tx")
	}

	issue, err := s.GetIssue(ctx, &FindIssueMessage{ProjectIDs: []string{projectID}, UID: &uid})
	if err != nil {
		return nil, false, err
	}
	return issue, approvalResetApplied, nil
}

func marshalCanonicalIssueLabels(labels []string) ([]byte, error) {
	canonicalLabels := CanonicalizeIssueLabels(labels)
	if canonicalLabels == nil {
		canonicalLabels = []string{}
	}
	labelBytes, err := json.Marshal(canonicalLabels)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal issue labels")
	}
	return labelBytes, nil
}

func buildUpdateIssueGuard(patch *UpdateIssueMessage, projectID string, uid int64) (*qb.Query, bool, error) {
	where := qb.Q().Space("project = ? AND id = ?", projectID, uid)
	hasGuard := false
	if version := patch.RequirePlanApprovalInputVersion; version != nil {
		where.Space("AND ?", issuePlanApprovalInputVersionCondition(*version, patch.RequireNoRollout))
		hasGuard = true
	} else if patch.RequireNoRollout {
		return nil, false, errors.New("RequireNoRollout requires RequirePlanApprovalInputVersion")
	}
	if version := patch.RequireIssueApprovalInputVersion; version != nil {
		where.Space("AND COALESCE((payload->'approval'->>'approvalInputVersion')::bigint, 0) = ?", *version)
		hasGuard = true
	}
	if done := patch.RequireApprovalFindingDone; done != nil {
		where.Space("AND COALESCE((payload->'approval'->>'approvalFindingDone')::boolean, false) = ?", *done)
		hasGuard = true
	}
	if labels := patch.RequireLabels; labels != nil {
		labelBytes, err := marshalCanonicalIssueLabels(*labels)
		if err != nil {
			return nil, false, err
		}
		where.Space("AND COALESCE(NULLIF(payload->'labels', 'null'::jsonb), '[]'::jsonb) = ?::jsonb", string(labelBytes))
		hasGuard = true
	}
	if version := patch.SkipIfCurrentApprovalFindingDone; version != nil {
		where.Space(`
			AND NOT (
				COALESCE((payload->'approval'->>'approvalFindingDone')::boolean, false)
				AND COALESCE((payload->'approval'->>'approvalInputVersion')::bigint, 0) = ?
			)`, *version)
		hasGuard = true
	}
	return where, hasGuard, nil
}

func issuePlanApprovalInputVersionCondition(approvalInputVersion int64, requireNoRollout bool) *qb.Query {
	planWhere := qb.Q().Space(`
		plan.project = issue.project
		  AND plan.id = issue.plan_id
		  AND COALESCE((plan.config->>'approvalInputVersion')::bigint, 0) = ?`, approvalInputVersion)
	if requireNoRollout {
		planWhere.Space("AND COALESCE((plan.config->>'hasRollout')::boolean, false) = false")
	}
	return qb.Q().Space(`
		EXISTS (
			SELECT 1
			FROM plan
			WHERE ?
		)`, planWhere)
}

// ListIssues returns the list of issues by find query.
func (s *Store) ListIssues(ctx context.Context, find *FindIssueMessage) ([]*IssueMessage, error) {
	orderByClause := "ORDER BY issue.id DESC"
	from := qb.Q().Space("issue")
	where := qb.Q()

	// `ProjectIDs` is required: empty slice intentionally returns zero
	// rows, which `SearchIssues` relies on as its IAM-derived "this user
	// has access to no projects" guard. Skipping the filter on empty
	// would leak cross-project issues. Cross-project runner scans must
	// pre-collect the project ID set (e.g. via `ListProjects`) and pass
	// the explicit list here.
	if len(find.ProjectIDs) == 1 {
		where.And("issue.project = ?", find.ProjectIDs[0])
	} else {
		where.And("issue.project = ANY(?)", find.ProjectIDs)
	}
	if find.Workspace != "" {
		from.Space("JOIN project ON issue.project = project.resource_id")
		where.And("project.workspace = ?", find.Workspace)
	}

	if v := find.UID; v != nil {
		where.And("issue.id = ?", *v)
	}
	if v := find.PlanUID; v != nil {
		where.And("issue.plan_id = ?", *v)
	}
	if v := find.PlanUIDs; v != nil {
		where.And("issue.plan_id = ANY(?)", *v)
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
	if len(find.RiskLevelList) != 0 {
		riskLevelStrings := make([]string, 0, len(find.RiskLevelList))
		for _, rl := range find.RiskLevelList {
			riskLevelStrings = append(riskLevelStrings, rl.String())
		}
		where.And("payload->>'riskLevel' = ANY(?)", riskLevelStrings)
	}
	if find.ExcludeDraft {
		where.And("COALESCE(issue.payload->>'draft', 'false') = 'false'")
	}

	if len(find.OrderByKeys) > 0 && orderByClause == "ORDER BY issue.id DESC" {
		parts := make([]string, 0, len(find.OrderByKeys)+1)
		for _, v := range find.OrderByKeys {
			parts = append(parts, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		parts = append(parts, "issue.id DESC")
		orderByClause = fmt.Sprintf("ORDER BY %s", strings.Join(parts, ", "))
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

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
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

	return issues, nil
}

// BatchUpdateIssueStatuses updates the status of multiple issues.
// Returns a map of issueUID -> old status for the updated issues.
func (s *Store) BatchUpdateIssueStatuses(ctx context.Context, projectID string, issueUIDs []int64, newStatus storepb.Issue_Status) (map[int64]storepb.Issue_Status, error) {
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

	oldStatuses := make(map[int64]storepb.Issue_Status)
	for rows.Next() {
		var issueID int64
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
		_, _ = fmt.Fprintf(&tsVector, "%s:%d", part, i+1)
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
		_, _ = fmt.Fprintf(&tsQuery, "%s:*", part)
	}
	return tsQuery.String()
}
