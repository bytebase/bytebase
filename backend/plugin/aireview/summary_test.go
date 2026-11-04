package aireview

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const summaryStatement = "ALTER TABLE events\n  ALTER COLUMN payload TYPE json;\nALTER TABLE orders DROP COLUMN legacy_ref;\n"

func rewriteFinding(severity Severity, evidence string) Finding {
	return Finding{Title: "Avoid a full table rewrite", Severity: severity, Line: 1, Rule: "Flag a schema change that blocks writes on a table with more than 1 million rows.", Evidence: evidence, Fix: "Add a json column and backfill it."}
}

// rewriteRequest holds one problem on three databases, the first of which
// reports it twice.
func rewriteRequest() *SummaryRequest {
	return &SummaryRequest{
		Statement: summaryStatement,
		Block:     LineRange{Start: 1, End: 2},
		Findings: []SummaryFinding{
			{Database: "instances/prod/databases/big", Finding: rewriteFinding(SeverityP1, "events has 10,000,000 rows.")},
			{Database: "instances/prod/databases/big", Finding: rewriteFinding(SeverityP1, "payload is read by v_events.")},
			{Database: "instances/prod/databases/small", Finding: rewriteFinding(SeverityP2, "events has 24,000 rows.")},
			{Database: "instances/prod/databases/medium", Finding: rewriteFinding(SeverityP2, "events has 116,000 rows.")},
		},
	}
}

// testDatabaseIDs are the ids summaryReviewer gives the databases, in the
// order they first appear in the request.
var testDatabaseIDs = []string{"kq7rzm", "p3n8ra", "w2x5tc"}

func summaryReviewer(model Model) *Reviewer {
	reviewer := NewReviewer(model)
	reviewer.databaseIDs = func(count int, _ string) []string { return testDatabaseIDs[:count] }
	return reviewer
}

const rewriteReply = `{"title": "Avoid rewriting events to change payload", "evidence": "jsonb to json rewrites the table under an ACCESS EXCLUSIVE lock.", "groups": [{"databases": ["kq7rzm"], "facts": "events has 10M rows."}, {"databases": ["p3n8ra", "w2x5tc"], "facts": "events has 24k to 116k rows."}], "fix": "Add a json column and backfill it."}`

func TestSummarizeReturnsTheModelsGroups(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(rewriteReply)}}
	summary, err := summaryReviewer(model).Summarize(context.Background(), rewriteRequest())
	require.NoError(t, err)
	require.Equal(t, "Avoid rewriting events to change payload", summary.Title)
	require.Equal(t, "jsonb to json rewrites the table under an ACCESS EXCLUSIVE lock.", summary.Evidence)
	require.Equal(t, []SummaryGroup{
		{Databases: []string{"instances/prod/databases/big"}, Facts: "events has 10M rows."},
		{Databases: []string{"instances/prod/databases/small", "instances/prod/databases/medium"}, Facts: "events has 24k to 116k rows."},
	}, summary.Groups)
	require.Equal(t, "Add a json column and backfill it.", summary.Fix)
	require.Equal(t, 1, summary.Calls)
	require.Equal(t, 100, summary.TotalTokens)

	require.Len(t, model.requests, 1)
	request := model.requests[0]
	require.Empty(t, request.ToolDefinitions)
	require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser}, roles(request.Messages))
	require.Equal(t, strings.TrimSpace(summaryPrompt), request.Messages[0].GetContent())
	user := request.Messages[1].GetContent()
	require.Contains(t, user, ">\n1\tALTER TABLE events\n2\t  ALTER COLUMN payload TYPE json;\n</block-", "the prompt shows the problem's statement")
	require.NotContains(t, user, "legacy_ref", "and no other statement")
	require.Contains(t, user, `{"database":"kq7rzm","line":1,"severity":"P1","title":"Avoid a full table rewrite","rule":"Flag a schema change that blocks writes on a table with more than 1 million rows.","evidence":"events has 10,000,000 rows.","fix":"Add a json column and backfill it."}`)
	require.Contains(t, user, `{"database":"kq7rzm","line":1,"severity":"P1","title":"Avoid a full table rewrite","rule":"Flag a schema change that blocks writes on a table with more than 1 million rows.","evidence":"payload is read by v_events."`, "a database keeps its id across its findings")
	require.Contains(t, user, `{"database":"w2x5tc","line":1,"severity":"P2",`)
	require.NotContains(t, user, "instances/prod", "the prompt names no database")
}

func TestSummarizeCorrectsAnInvalidReply(t *testing.T) {
	t.Parallel()

	invalid := `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm", "p3n8ra"], "facts": ""}], "fix": "f"}`
	model := &scriptedModel{steps: []modelStep{finalReply(invalid), finalReply(rewriteReply)}}
	summary, err := summaryReviewer(model).Summarize(context.Background(), rewriteRequest())
	require.NoError(t, err)
	require.Len(t, summary.Groups, 2)
	require.Equal(t, 2, summary.Calls)
	require.Equal(t, 200, summary.TotalTokens)
	second := model.requests[1].Messages
	require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser, roleAssistant, roleUser}, roles(second))
	require.Equal(t, invalid, second[2].GetContent())
	require.Contains(t, second[3].GetContent(), "Your reply is not a valid summary:\n- databases \"w2x5tc\" are in no group\n")
}

func TestSummarizeFailsWhenTheCorrectionIsInvalidToo(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(`{"title": "t", "evidence": "e", "groups": [], "fix": "f"}`)}}
	_, err := summaryReviewer(model).Summarize(context.Background(), rewriteRequest())
	require.ErrorIs(t, err, ErrInvalidReply)
	require.ErrorContains(t, err, "groups: is empty")
	require.Len(t, model.requests, 2)
}

func TestSummarizeFailsOnAnUnfinishedReply(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		step modelStep
		want error
	}{
		{name: "token limit", step: finalReply(`{"title": "Avoid`).stoppedFor(stopTokenLimit), want: ErrTruncatedReply},
		{name: "blocked", step: finalReply("").stoppedFor(stopBlocked), want: ErrBlockedReply},
		{name: "tool call", step: toolCallReply(&v1pb.AIChatToolCall{Id: "c1", Name: "read", Arguments: "{}"}), want: ErrInvalidReply},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := &scriptedModel{steps: []modelStep{tc.step}}
			_, err := summaryReviewer(model).Summarize(context.Background(), rewriteRequest())
			require.ErrorIs(t, err, tc.want)
			require.Len(t, model.requests, 1, "no correction round")
		})
	}
}

func TestSummarizeRefusesAPromptOverTheLimit(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(rewriteReply)}}
	reviewer := summaryReviewer(model)
	reviewer.summaryPromptBytes = 100
	_, err := reviewer.Summarize(context.Background(), rewriteRequest())
	require.ErrorContains(t, err, "over the 100 byte limit")
	require.Empty(t, model.requests)
}

func TestSummarizeCanceledParentIsNotADeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	model := &scriptedModel{steps: []modelStep{{block: true, cancelParent: cancel}}}
	_, err := summaryReviewer(model).Summarize(ctx, rewriteRequest())
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrDeadline)
}

func TestParseSummaryReply(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		reply    string
		want     *summaryReplyJSON
		problems []string
	}{
		{
			name:  "one group needs no facts",
			reply: `{"title": "Keep events append-only", "evidence": "The table comment says the rows are never updated.", "groups": [{"databases": ["w2x5tc", "kq7rzm", "p3n8ra"], "facts": ""}], "fix": "Insert a corrected event."}`,
			want:  &summaryReplyJSON{Title: "Keep events append-only", Evidence: "The table comment says the rows are never updated.", Groups: &[]summaryGroupJSON{{Databases: []string{"w2x5tc", "kq7rzm", "p3n8ra"}}}, Fix: "Insert a corrected event."},
		},
		{
			name:  "fenced, with the title and facts on one line",
			reply: "```json\n{\"title\": \"Drop\\nthe view first\", \"evidence\": \" e \", \"groups\": [{\"databases\": [\"kq7rzm\"], \"facts\": \"a\\n b\"}, {\"databases\": [\"p3n8ra\", \"w2x5tc\"], \"facts\": \"c\"}], \"fix\": \"f\"}\n```",
			want:  &summaryReplyJSON{Title: "Drop the view first", Evidence: "e", Groups: &[]summaryGroupJSON{{Databases: []string{"kq7rzm"}, Facts: "a b"}, {Databases: []string{"p3n8ra", "w2x5tc"}, Facts: "c"}}, Fix: "f"},
		},
		{
			name:  "one group without facts among several",
			reply: `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm", "p3n8ra"], "facts": ""}, {"databases": ["w2x5tc"], "facts": "events has 10M rows."}], "fix": "f"}`,
			want:  &summaryReplyJSON{Title: "t", Evidence: "e", Groups: &[]summaryGroupJSON{{Databases: []string{"kq7rzm", "p3n8ra"}}, {Databases: []string{"w2x5tc"}, Facts: "events has 10M rows."}}, Fix: "f"},
		},
		{name: "empty texts", reply: `{"title": " ", "evidence": "", "groups": [{"databases": ["kq7rzm", "p3n8ra", "w2x5tc"]}], "fix": ""}`, problems: []string{"title: is empty", "evidence: is empty", "fix: is empty"}},
		{name: "no groups", reply: `{"title": "t", "evidence": "e", "fix": "f"}`, problems: []string{"groups: is empty"}},
		{name: "two groups without facts", reply: `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm"], "facts": ""}, {"databases": ["p3n8ra"], "facts": "a"}, {"databases": ["w2x5tc"], "facts": " "}], "fix": "f"}`, problems: []string{"groups[2].facts: is empty, and at most one group"}},
		{
			name:     "database ids in the texts, however they are written",
			reply:    `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm", "p3n8ra"], "facts": "unlike DB #w2x5tc, a"}, {"databases": ["w2x5tc"], "facts": "b"}], "fix": "For database number KQ7RZM, archive first."}`,
			problems: []string{"fix: holds a database id", "groups[0].facts: holds a database id"},
		},
		{name: "unknown database", reply: `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm", "p3n8ra", "w2x5tc", "1"]}], "fix": "f"}`, problems: []string{`groups[0].databases holds "1", which is not a database id`}},
		{name: "twice", reply: `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm", "p3n8ra"], "facts": "a"}, {"databases": ["p3n8ra", "w2x5tc"], "facts": "b"}], "fix": "f"}`, problems: []string{`database "p3n8ra" is in more than one group`}},
		{name: "empty group", reply: `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm", "p3n8ra", "w2x5tc"], "facts": "a"}, {"databases": [], "facts": "b"}], "fix": "f"}`, problems: []string{"groups[1].databases: is empty"}},
		{name: "missing", reply: `{"title": "t", "evidence": "e", "groups": [{"databases": ["p3n8ra"]}], "fix": "f"}`, problems: []string{`databases "kq7rzm", "w2x5tc" are in no group`}},
		{name: "numbers for ids", reply: `{"title": "t", "evidence": "e", "groups": [{"databases": [1, 2, 3]}], "fix": "f"}`, problems: []string{"the reply is not a valid JSON object"}},
		{name: "unknown key", reply: `{"title": "t", "evidence": "e", "groups": [{"databases": ["kq7rzm", "p3n8ra", "w2x5tc"]}], "fix": "f", "severity": "P1"}`, problems: []string{"the reply is not a valid JSON object"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reply, problems := parseSummaryReply(tc.reply, testDatabaseIDs)
			if tc.problems != nil {
				require.Nil(t, reply)
				require.Len(t, problems, len(tc.problems))
				for i, problem := range tc.problems {
					require.Contains(t, problems[i], problem)
				}
				return
			}
			require.Empty(t, problems)
			require.Equal(t, tc.want, reply)
		})
	}
}

func TestNewDatabaseIDs(t *testing.T) {
	t.Parallel()

	ids := newDatabaseIDs(50, "some finding text")
	require.Len(t, ids, 50)
	seen := make(map[string]bool)
	for _, id := range ids {
		require.Regexp(t, `^[a-z2-7]{6}$`, id)
		require.False(t, seen[id], "ids are distinct")
		seen[id] = true
	}

	// Summarize hands the generator the statement and the findings in lower
	// case, which no id may occur in.
	reviewer := NewReviewer(&scriptedModel{steps: []modelStep{finalReply(rewriteReply)}})
	var texts []string
	reviewer.databaseIDs = func(count int, text string) []string {
		texts = append(texts, text)
		return testDatabaseIDs[:count]
	}
	request := rewriteRequest()
	request.Findings[0].Finding.Evidence = "Events HAS 10,000,000 rows."
	_, err := reviewer.Summarize(context.Background(), request)
	require.NoError(t, err)
	require.Len(t, texts, 1)
	require.Contains(t, texts[0], "events has 10,000,000 rows.", "the ids avoid the findings' text, compared in lower case")
	require.Contains(t, texts[0], "alter column payload type json", "and the statement's")
}
