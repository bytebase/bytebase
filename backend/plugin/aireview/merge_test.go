package aireview

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

const mergeStatement = "ALTER TABLE users\n  DROP COLUMN full_name;\nCREATE INDEX orders_user_id_idx ON orders (user_id);\n"

func dropFinding(line int, title string) Finding {
	return Finding{Title: title, Severity: SeverityP1, Line: line, Rule: "Data risk: a dropped column that still holds data.", Evidence: "users holds 30,000 rows.", Fix: "Archive full_name first."}
}

// threeDatabaseRequest holds the same drop on two databases, a different
// problem on the drop's block of a third, and an index finding on the second
// block.
func threeDatabaseRequest() *MergeRequest {
	return &MergeRequest{
		Statement: mergeStatement,
		Blocks:    []LineRange{{Start: 1, End: 2}, {Start: 3, End: 3}},
		Candidates: []MergeCandidate{
			{Block: 0, Database: "instances/prod/databases/db1", Finding: dropFinding(1, "Do not drop full_name while it holds data")},
			{Block: 0, Database: "instances/prod/databases/db2", Finding: dropFinding(2, "Archive full_name before dropping it")},
			{Block: 0, Database: "instances/prod/databases/db3", Finding: dropFinding(1, "Drop v_user_directory before full_name")},
			{Block: 1, Database: "instances/prod/databases/db1", Finding: dropFinding(3, "Create the index CONCURRENTLY")},
		},
	}
}

func TestMergeFindingsReturnsTheModelsGroups(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": [[4], [1, 2], [3]]}`)}}
	result, err := NewReviewer(model).MergeFindings(context.Background(), threeDatabaseRequest())
	require.NoError(t, err)
	require.Equal(t, [][]int{{0, 1}, {2}, {3}}, result.Groups)
	require.Equal(t, 1, result.Calls)
	require.Equal(t, 100, result.TotalTokens)

	require.Len(t, model.requests, 1)
	request := model.requests[0]
	require.Empty(t, request.ToolDefinitions)
	require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser}, roles(request.Messages))
	require.Equal(t, strings.TrimSpace(mergePrompt), request.Messages[0].GetContent())
	user := request.Messages[1].GetContent()
	require.Contains(t, user, "1\tALTER TABLE users\n2\t  DROP COLUMN full_name;\n</block-")
	require.Contains(t, user, "\">\n3\tCREATE INDEX orders_user_id_idx ON orders (user_id);\n</block-")
	require.Contains(t, user, `{"id":2,"block":1,"database":2,"line":2,"severity":"P1","title":"Archive full_name before dropping it","rule":"Data risk: a dropped column that still holds data.","evidence":"users holds 30,000 rows.","fix":"Archive full_name first."}`)
	require.Contains(t, user, `{"id":4,"block":2,"database":1,"line":3,`)
	require.NotContains(t, user, "instances/prod", "the prompt names no database")
}

func TestMergeFindingsShowsTheLinesAroundTheFindingsOfALongBlock(t *testing.T) {
	t.Parallel()

	var lines []string
	for i := range 100 {
		lines = append(lines, fmt.Sprintf("  -- line %d", i+1))
	}
	request := &MergeRequest{
		Statement: strings.Join(lines, "\n"),
		Blocks:    []LineRange{{Start: 1, End: 100}},
		Candidates: []MergeCandidate{
			{Block: 0, Database: "a", Finding: dropFinding(50, "x")},
			{Block: 0, Database: "b", Finding: dropFinding(52, "y")},
		},
	}
	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": [[1, 2]]}`)}}
	_, err := NewReviewer(model).MergeFindings(context.Background(), request)
	require.NoError(t, err)
	user := model.requests[0].Messages[1].GetContent()
	require.Contains(t, user, "\">\n1\t  -- line 1\n...\n45\t  -- line 45\n")
	require.Contains(t, user, "\n57\t  -- line 57\n...\n</block-")
	require.NotContains(t, user, "\n44\t")
	require.NotContains(t, user, "\n58\t")
}

func TestMergeFindingsShowsAnEarlierFindingWithoutADatabase(t *testing.T) {
	t.Parallel()

	request := threeDatabaseRequest()
	request.Candidates = append([]MergeCandidate{{Block: 0, Earlier: true, Finding: Finding{Title: "Keep full_name", Severity: SeverityP1, Line: 2, Rule: "r", Fix: "f"}}}, request.Candidates...)
	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": [[1, 2, 3], [4], [5]]}`)}}
	result, err := NewReviewer(model).MergeFindings(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, [][]int{{0, 1, 2}, {3}, {4}}, result.Groups)
	user := model.requests[0].Messages[1].GetContent()
	require.Contains(t, user, `{"id":1,"block":1,"earlier":true,"line":2,"severity":"P1","title":"Keep full_name","rule":"r","evidence":"","fix":"f"}`)
	require.Contains(t, user, `{"id":2,"block":1,"database":1,"line":1,`, "an earlier finding takes no database ordinal")
}

func TestMergeFindingsClipsLongTexts(t *testing.T) {
	t.Parallel()

	request := threeDatabaseRequest()
	request.Candidates[0].Finding.Evidence = strings.Repeat("é", 1000)
	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": [[1, 2, 3], [4]]}`)}}
	_, err := NewReviewer(model).MergeFindings(context.Background(), request)
	require.NoError(t, err)
	user := model.requests[0].Messages[1].GetContent()
	require.Contains(t, user, `"evidence":"`+strings.Repeat("é", maxMergeFieldBytes/2)+`…"`)

	require.Equal(t, "é…", clip("ééé", 3), "a cut never splits a character")
	require.True(t, utf8.ValidString(clip("ééé", 3)))
	require.Equal(t, "short", clip("short", 5))
}

func TestMergeFindingsCorrectsAnInvalidReply(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{
		finalReply(`{"groups": [[1, 2], [4]]}`),
		finalReply(`{"groups": [[1, 2], [3], [4]]}`),
	}}
	result, err := NewReviewer(model).MergeFindings(context.Background(), threeDatabaseRequest())
	require.NoError(t, err)
	require.Equal(t, [][]int{{0, 1}, {2}, {3}}, result.Groups)
	require.Equal(t, 2, result.Calls)
	second := model.requests[1].Messages
	require.Equal(t, []v1pb.AIChatMessageRole{roleSystem, roleUser, roleAssistant, roleUser}, roles(second))
	require.Equal(t, `{"groups": [[1, 2], [4]]}`, second[2].GetContent())
	require.Contains(t, second[3].GetContent(), "- findings 3 are in no list\n")
}

func TestMergeFindingsFailsWhenTheCorrectionIsInvalidToo(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": [[1, 2, 3]]}`)}}
	_, err := NewReviewer(model).MergeFindings(context.Background(), threeDatabaseRequest())
	require.ErrorIs(t, err, ErrInvalidReply)
	require.ErrorContains(t, err, "findings 4 are in no list")
	require.Len(t, model.requests, 2)
}

func TestMergeFindingsFailsOnAnUnfinishedReply(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		step modelStep
		want error
	}{
		{name: "token limit", step: finalReply(`{"groups": [[1, 2]`).stoppedFor(stopTokenLimit), want: ErrTruncatedReply},
		{name: "blocked", step: finalReply("").stoppedFor(stopBlocked), want: ErrBlockedReply},
		{name: "other", step: finalReply(`{"groups": []}`).stoppedFor(stopOther), want: ErrUnfinishedReply},
		{name: "malformed tool call", step: finalReply("").stoppedFor(stopMalformedToolCall), want: ErrUnfinishedReply},
		{name: "empty", step: finalReply("  "), want: ErrEmptyReply},
		{name: "tool call", step: toolCallReply(&v1pb.AIChatToolCall{Id: "c1", Name: "read", Arguments: "{}"}), want: ErrInvalidReply},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := &scriptedModel{steps: []modelStep{tc.step}}
			_, err := NewReviewer(model).MergeFindings(context.Background(), threeDatabaseRequest())
			require.ErrorIs(t, err, tc.want)
			require.Len(t, model.requests, 1, "no correction round")
		})
	}
}

func TestMergeFindingsLeavesABlockTooLargeForOneCallUnmerged(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": []}`)}}
	reviewer := NewReviewer(model)
	reviewer.mergePromptBytes = 1
	result, err := reviewer.MergeFindings(context.Background(), threeDatabaseRequest())
	require.NoError(t, err)
	require.Equal(t, [][]int{{0}, {1}, {2}, {3}}, result.Groups)
	require.Zero(t, result.Calls)
	require.Empty(t, model.requests)
}

func TestMergeFindingsRejectsACandidateOffTheBlocks(t *testing.T) {
	t.Parallel()

	request := threeDatabaseRequest()
	request.Candidates[1].Block = 2
	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": []}`)}}
	_, err := NewReviewer(model).MergeFindings(context.Background(), request)
	require.ErrorContains(t, err, "candidate 1 is on block 2, and the request has 2 blocks")
	require.Empty(t, model.requests)
}

func TestMergeFindingsCanceledParentIsNotADeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	model := &scriptedModel{steps: []modelStep{{block: true, cancelParent: cancel}}}
	_, err := NewReviewer(model).MergeFindings(ctx, threeDatabaseRequest())
	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, ErrDeadline)
}

func TestPackMergeSections(t *testing.T) {
	t.Parallel()

	section := func(size int) *mergeSection {
		return &mergeSection{block: strings.Repeat("b", size/2), findings: strings.Repeat("f", size-size/2)}
	}
	a, b, c, huge, d := section(40), section(50), section(10), section(101), section(60)
	batches, alone := packMergeSections([]*mergeSection{a, b, c, huge, d}, 100)
	require.Equal(t, [][]*mergeSection{{a, b, c}, {d}}, batches)
	require.Equal(t, []*mergeSection{huge}, alone)
}

func TestParseMergeReply(t *testing.T) {
	t.Parallel()

	// Candidates 0 and 1 are on block 0, candidate 2 on block 1.
	members := []int{0, 1, 2}
	blockOf := []int{0, 0, 1}
	earlier := []bool{false, false, false}
	for _, tc := range []struct {
		name     string
		reply    string
		want     [][]int
		problems []string
	}{
		{name: "bare", reply: `{"groups": [[2, 1], [3]]}`, want: [][]int{{0, 1}, {2}}},
		{name: "fenced", reply: "```json\n{\"groups\": [[1], [2], [3]]}\n```", want: [][]int{{0}, {1}, {2}}},
		{name: "inside prose", reply: "Here it is: {\"groups\": [[1, 2], [3]]} Done.", want: [][]int{{0, 1}, {2}}},
		{name: "unknown id", reply: `{"groups": [[1, 2, 9], [3]]}`, problems: []string{"groups[0] holds 9, which is not a finding id"}},
		{name: "twice", reply: `{"groups": [[1, 2], [2], [3]]}`, problems: []string{"finding 2 is in more than one list"}},
		{name: "empty list", reply: `{"groups": [[1, 2], [], [3]]}`, problems: []string{"groups[1] is empty"}},
		{name: "two blocks", reply: `{"groups": [[1, 3], [2]]}`, problems: []string{"groups[0] holds findings of more than one block"}},
		{name: "missing", reply: `{"groups": [[1]]}`, problems: []string{"findings 2, 3 are in no list"}},
		{name: "no groups key", reply: `{"lists": [[1, 2, 3]]}`, problems: []string{`"lists"`}},
		{name: "null groups", reply: `{"groups": null}`, problems: []string{`the JSON object has no "groups" key`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			groups, problems := parseMergeReply(tc.reply, members, blockOf, earlier)
			if tc.problems != nil {
				require.Len(t, problems, len(tc.problems))
				for i, problem := range tc.problems {
					require.Contains(t, problems[i], problem)
				}
				return
			}
			require.Empty(t, problems)
			require.Equal(t, tc.want, groups)
		})
	}

	_, problems := parseMergeReply(`{"groups": [["1"]]}`, members, blockOf, earlier)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "the reply is not a valid JSON object")

	// The subtests run after this, so these are new variables. Candidates 0 and
	// 1 are earlier findings.
	twoEarlier := []bool{true, true, false}
	oneBlock := []int{0, 0, 0}
	groups, problems := parseMergeReply(`{"groups": [[1, 3], [2]]}`, members, oneBlock, twoEarlier)
	require.Empty(t, problems)
	require.Equal(t, [][]int{{0, 2}, {1}}, groups)
	_, problems = parseMergeReply(`{"groups": [[1, 2, 3]]}`, members, oneBlock, twoEarlier)
	require.Equal(t, []string{"groups[0] holds 2 earlier findings"}, problems)
}
