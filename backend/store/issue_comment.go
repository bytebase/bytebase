package store

import (
	"context"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ThreadState is the resolvable state carried by a thread's root comment.
// Events and replies have no state (NULL thread_state).
type ThreadState string

const (
	ThreadStateOpen     ThreadState = "OPEN"
	ThreadStateResolved ThreadState = "RESOLVED"
)

type IssueCommentMessage struct {
	ProjectID    string
	ResourceID   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	IssueUID     int64
	Payload      *storepb.IssueCommentPayload
	CreatorEmail string
	// ParentID names the thread's root comment on a reply; nil on root
	// comments and events.
	ParentID *string
	// ThreadState is set on thread roots; nil on replies and events.
	ThreadState *ThreadState
}

type FindIssueCommentMessage struct {
	ProjectID  string
	ResourceID *string
	IssueUID   *int64
	// ParentIDs limits the result to replies of these root comments.
	ParentIDs *[]string
	// TopLevelOnly limits the result to events and root comments — the
	// activity timeline entries.
	TopLevelOnly bool
	// ThreadState limits the result to thread roots in this state.
	ThreadState *ThreadState

	Limit  *int
	Offset *int
}

type UpdateIssueCommentMessage struct {
	ProjectID  string
	ResourceID string

	Comment *string
	// ThreadState resolves or reopens a thread; the target must be a root
	// comment. Replying never changes it.
	ThreadState *ThreadState
}

func (s ThreadState) valid() bool {
	return s == ThreadStateOpen || s == ThreadStateResolved
}

// threadStatePredicate appends a literal thread_state predicate. The literal
// (never a bind parameter) is what lets the planner match the
// idx_issue_comment_open_thread partial index; valid() makes the
// interpolation safe.
func threadStatePredicate(q *qb.Query, state ThreadState) error {
	if !state.valid() {
		return common.Errorf(common.Invalid, "invalid thread state %q", state)
	}
	q.And("thread_state = '" + string(state) + "'")
	return nil
}

// validateIssueCommentPayload pins the statement anchor invariants shared by
// both writers. The anchor is write-once, so a malformed one is permanent.
func validateIssueCommentPayload(payload *storepb.IssueCommentPayload) error {
	anchor := payload.GetStatementAnchor()
	if anchor == nil {
		return nil
	}
	if payload.GetEvent() != nil {
		return common.Errorf(common.Invalid, "an event cannot carry a statement anchor")
	}
	if anchor.SpecId == "" {
		return common.Errorf(common.Invalid, "a statement anchor requires a spec id")
	}
	if len(anchor.SheetSha256) != 64 || strings.ToLower(anchor.SheetSha256) != anchor.SheetSha256 {
		return common.Errorf(common.Invalid, "a statement anchor requires the sheet sha256 as 64 lowercase hex characters")
	}
	if _, err := hex.DecodeString(anchor.SheetSha256); err != nil {
		return common.Errorf(common.Invalid, "a statement anchor requires the sheet sha256 as 64 lowercase hex characters")
	}
	start, end := anchor.StartPosition, anchor.EndPosition
	if start == nil || end == nil {
		return common.Errorf(common.Invalid, "a statement anchor requires start and end positions")
	}
	switch {
	case start.Column == 0 && end.Column == 0:
		// Whole-line anchor; the end line is inclusive.
		if start.Line < 1 || end.Line < start.Line {
			return common.Errorf(common.Invalid, "invalid whole-line anchor range %d-%d", start.Line, end.Line)
		}
	case start.Column > 0 && end.Column > 0:
		// Code-point range; the end position is exclusive.
		if start.Line < 1 || end.Line < start.Line || (end.Line == start.Line && end.Column <= start.Column) {
			return common.Errorf(common.Invalid, "invalid anchor range %d:%d-%d:%d", start.Line, start.Column, end.Line, end.Column)
		}
	default:
		return common.Errorf(common.Invalid, "an anchor range must set both columns or neither")
	}
	return nil
}

func (s *Store) GetIssueComment(ctx context.Context, find *FindIssueCommentMessage) (*IssueCommentMessage, error) {
	list, err := s.ListIssueComment(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	if len(list) > 1 {
		return nil, errors.Errorf("found %d issue comment, expected 1", len(list))
	}
	return list[0], nil
}

func (s *Store) ListIssueComment(ctx context.Context, find *FindIssueCommentMessage) ([]*IssueCommentMessage, error) {
	q := qb.Q().Space(`
		SELECT
			project,
			resource_id,
			creator,
			created_at,
			updated_at,
			issue_id,
			payload,
			parent_id,
			thread_state
		FROM
			issue_comment
		WHERE project = ?
	`, find.ProjectID)

	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
	}
	if v := find.IssueUID; v != nil {
		q.And("issue_id = ?", *v)
	}
	if v := find.ParentIDs; v != nil {
		q.And("parent_id = ANY(?)", *v)
	}
	if find.TopLevelOnly {
		q.And("parent_id IS NULL")
	}
	if v := find.ThreadState; v != nil {
		if err := threadStatePredicate(q, *v); err != nil {
			return nil, err
		}
	}

	// resource_id is the primary key. created_at alone would not be total —
	// it defaults to now(), the transaction timestamp — so CreateIssueComments
	// offsets each row of a batch by its ordinal to keep insertion order; see
	// the comment there.
	q.Space("ORDER BY issue_comment.created_at ASC, issue_comment.resource_id ASC")
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

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query context")
	}
	defer rows.Close()

	var issueComments []*IssueCommentMessage
	for rows.Next() {
		ic := IssueCommentMessage{
			Payload: &storepb.IssueCommentPayload{},
		}
		var p []byte
		var parentID, threadState sql.NullString
		if err := rows.Scan(
			&ic.ProjectID,
			&ic.ResourceID,
			&ic.CreatorEmail,
			&ic.CreatedAt,
			&ic.UpdatedAt,
			&ic.IssueUID,
			&p,
			&parentID,
			&threadState,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		if parentID.Valid {
			ic.ParentID = &parentID.String
		}
		if threadState.Valid {
			state := ThreadState(threadState.String)
			ic.ThreadState = &state
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal(p, ic.Payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal")
		}
		issueComments = append(issueComments, &ic)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	return issueComments, nil
}

// CreateIssueComments creates one or more root comments or events.
// Replies go through CreateIssueCommentReply.
// For a single comment, it returns the created comment with UID, CreatedAt, and UpdatedAt filled in.
// For multiple comments, it performs a batch insert and returns nil.
func (s *Store) CreateIssueComments(ctx context.Context, creator string, creates ...*IssueCommentMessage) (*IssueCommentMessage, error) {
	if len(creates) == 0 {
		return nil, nil
	}

	// Prepare all payloads.
	projectIDs := make([]string, 0, len(creates))
	issueIDs := make([]int64, 0, len(creates))
	payloads := make([][]byte, 0, len(creates))
	threadRoots := make([]bool, 0, len(creates))
	for _, create := range creates {
		if create.ParentID != nil {
			return nil, common.Errorf(common.Invalid, "replies must be created through CreateIssueCommentReply")
		}
		if create.ThreadState != nil {
			return nil, common.Errorf(common.Invalid, "thread state is derived on create; a new root starts OPEN")
		}
		if err := validateIssueCommentPayload(create.Payload); err != nil {
			return nil, err
		}
		payload, err := protojson.Marshal(create.Payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal payload")
		}
		projectIDs = append(projectIDs, create.ProjectID)
		issueIDs = append(issueIDs, create.IssueUID)
		payloads = append(payloads, payload)
		// Event-less writes are roots. Event and hybrid rows remain outside threads.
		threadRoots = append(threadRoots, create.Payload.GetEvent() == nil)
	}

	// Use UNNEST to insert all comments in one query.
	//
	// created_at defaults to now(), which is the transaction timestamp, so a
	// batch would otherwise land on one instant and ListIssueComment could only
	// fall back to its resource_id tiebreak — a random UUID, which scrambles the
	// activity feed of a multi-field UpdateIssue. Offsetting each row by its
	// ordinal keeps the batch in insertion order and makes created_at unique
	// within it, at a cost of microseconds.
	q := qb.Q().Space(`
		INSERT INTO issue_comment (creator, project, issue_id, payload, thread_state, created_at, updated_at)
		SELECT ?, c.project, c.issue_id, c.payload, CASE WHEN c.is_thread_root THEN 'OPEN' END, t.at, t.at
		FROM unnest(?::TEXT[], ?::INT[], ?::JSONB[], ?::BOOLEAN[])
		     WITH ORDINALITY AS c(project, issue_id, payload, is_thread_root, ordinality),
		     LATERAL (SELECT now() + ((c.ordinality - 1) * interval '1 microsecond')) AS t(at)
	`, creator, projectIDs, issueIDs, payloads, threadRoots)

	// For single comment, use RETURNING to get the created comment details.
	if len(creates) == 1 {
		q.Space("RETURNING resource_id, created_at, updated_at")
		query, args, err := q.ToSQL()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build sql")
		}

		create := creates[0]
		if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.ResourceID, &create.CreatedAt, &create.UpdatedAt); err != nil {
			return nil, errors.Wrapf(err, "failed to insert")
		}
		if threadRoots[0] {
			state := ThreadStateOpen
			create.ThreadState = &state
		}
		create.CreatorEmail = creator
		return create, nil
	}

	// For multiple comments, just execute without RETURNING.
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to batch insert comments")
	}

	return nil, nil
}

// CreateIssueCommentReply creates a reply in the thread rooted at
// create.ParentID and returns it with ResourceID, CreatedAt, and UpdatedAt
// filled in. The insert-select pins every reply invariant in one statement:
// the parent is a root comment (thread_state set, so never an event or a
// reply) in the same project and issue, and replying never changes the
// thread state. Like CreateIssueComments, it requires only the referenced
// rows to exist, regardless of project lifecycle.
func (s *Store) CreateIssueCommentReply(ctx context.Context, creator string, create *IssueCommentMessage) (*IssueCommentMessage, error) {
	if create.ParentID == nil {
		return nil, common.Errorf(common.Invalid, "a reply must name its thread root")
	}
	if create.ThreadState != nil {
		return nil, common.Errorf(common.Invalid, "replies carry no thread state")
	}
	if create.Payload.GetEvent() != nil {
		return nil, common.Errorf(common.Invalid, "a reply cannot carry an event")
	}
	if err := validateIssueCommentPayload(create.Payload); err != nil {
		return nil, err
	}
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal payload")
	}

	q := qb.Q().Space(`
		INSERT INTO issue_comment (creator, project, issue_id, payload, parent_id)
		SELECT ?, root.project, root.issue_id, ?, root.resource_id
		FROM issue_comment AS root
		WHERE root.project = ? AND root.resource_id = ? AND root.issue_id = ?
		  AND root.parent_id IS NULL AND root.thread_state IS NOT NULL
		RETURNING resource_id, created_at, updated_at
	`, creator, payload, create.ProjectID, *create.ParentID, create.IssueUID)
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&create.ResourceID, &create.CreatedAt, &create.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, s.replyTargetError(ctx, create)
		}
		return nil, errors.Wrapf(err, "failed to insert reply")
	}
	create.CreatorEmail = creator
	return create, nil
}

// replyTargetError explains why the reply insert matched no thread root.
func (s *Store) replyTargetError(ctx context.Context, create *IssueCommentMessage) error {
	var issueUID int64
	var parentID, threadState sql.NullString
	err := s.GetDB().QueryRowContext(ctx,
		"SELECT issue_id, parent_id, thread_state FROM issue_comment WHERE project = $1 AND resource_id = $2",
		create.ProjectID, *create.ParentID).Scan(&issueUID, &parentID, &threadState)
	if errors.Is(err, sql.ErrNoRows) {
		return common.Errorf(common.NotFound, "comment %s not found in project %s", *create.ParentID, create.ProjectID)
	}
	if err != nil {
		return errors.Wrapf(err, "failed to read reply target %s", *create.ParentID)
	}
	switch {
	case issueUID != create.IssueUID:
		return common.Errorf(common.NotFound, "comment %s not found in issue %d", *create.ParentID, create.IssueUID)
	case parentID.Valid:
		return common.Errorf(common.Invalid, "comment %s is a reply; a reply must name the thread root", *create.ParentID)
	case threadState.Valid:
		// The row satisfies every insert predicate on re-read, so a
		// concurrent write landed between the insert and this diagnosis.
		return common.Errorf(common.Conflict, "comment %s changed concurrently; retry the reply", *create.ParentID)
	default:
		// Events, hybrid comment+event rows, and unclassified legacy rows
		// all carry NULL thread_state.
		return common.Errorf(common.Invalid, "comment %s is not a thread root and cannot be replied to", *create.ParentID)
	}
}

func (s *Store) UpdateIssueComment(ctx context.Context, patch *UpdateIssueCommentMessage) error {
	if patch.Comment == nil && patch.ThreadState == nil {
		return common.Errorf(common.Invalid, "no fields to update")
	}

	set := qb.Q()
	if v := patch.Comment; v != nil {
		// updated_at marks content edits; a thread state change alone must
		// not make the comment render as edited.
		set.Comma("updated_at = ?", time.Now())
		set.Comma("payload = payload || jsonb_build_object('comment',?::TEXT)", *v)
	}
	if v := patch.ThreadState; v != nil {
		if !v.valid() {
			return common.Errorf(common.Invalid, "invalid thread state %q", *v)
		}
		set.Comma("thread_state = ?", string(*v))
	}

	q := qb.Q().Space("UPDATE issue_comment SET ?", set)
	q.Space("WHERE project = ? AND resource_id = ?", patch.ProjectID, patch.ResourceID)
	// Only thread roots carry a state; events and replies must keep NULL.
	if patch.ThreadState != nil {
		q.And("thread_state IS NOT NULL")
	}
	// Text edits apply to rows that already render text: roots, replies, and
	// hybrid event+comment rows — never to pure events.
	if patch.Comment != nil {
		q.And("(payload ?? 'comment' OR thread_state IS NOT NULL OR parent_id IS NOT NULL)")
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	result, err := s.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to update issue comment")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "failed to get affected rows")
	}
	if rows == 0 {
		if patch.ThreadState != nil {
			return common.Errorf(common.NotFound, "comment %s in project %s is missing or not a thread root; nothing was updated", patch.ResourceID, patch.ProjectID)
		}
		return common.Errorf(common.NotFound, "comment %s in project %s is missing or a pure event; nothing was updated", patch.ResourceID, patch.ProjectID)
	}
	return nil
}
