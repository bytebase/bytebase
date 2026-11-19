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
	batches, alone := packMergeSections([]*mergeSection{a, b, c, huge, d}, nil, 100)
	require.Equal(t, [][]*mergeSection{{a, b, c}, {d}}, batches)
	require.Equal(t, []*mergeSection{huge}, alone)
}

func TestParseMergeReply(t *testing.T) {
	t.Parallel()

	// Candidates 0 and 1 are on block 0, candidate 2 on block 1.
	members := []int{0, 1, 2}
	call := &mergeCall{blockOf: []int{0, 0, 1}, earlier: make([]bool, 3), standard: make([]bool, 3)}
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

			groups, same, problems := parseMergeReply(tc.reply, members, nil, call)
			require.Empty(t, same)
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

	_, _, problems := parseMergeReply(`{"groups": [["1"]]}`, members, nil, call)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "the reply is not a valid JSON object")

	// The subtests run after this, so these are new variables. Candidates 0 and
	// 1 are earlier findings.
	twoEarlier := &mergeCall{blockOf: []int{0, 0, 0}, earlier: []bool{true, true, false}, standard: make([]bool, 3)}
	groups, _, problems := parseMergeReply(`{"groups": [[1, 3], [2]]}`, members, nil, twoEarlier)
	require.Empty(t, problems)
	require.Equal(t, [][]int{{0, 2}, {1}}, groups)
	_, _, problems = parseMergeReply(`{"groups": [[1, 2, 3]]}`, members, nil, twoEarlier)
	require.Equal(t, []string{"groups[0] holds 2 earlier findings"}, problems)
}

func TestParseMergeReplyWithStandardFindings(t *testing.T) {
	t.Parallel()

	// Candidates 0 and 1 are findings of a review, 2 and 3 standard findings,
	// all on one block. Thread 0 is shown in the call, thread 1 is not.
	members := []int{0, 1, 2, 3}
	call := &mergeCall{blockOf: make([]int, 4), earlier: make([]bool, 4), standard: []bool{false, false, true, true}}
	for _, tc := range []struct {
		name     string
		reply    string
		want     [][]int
		same     []int
		problems []string
	}{
		{name: "one problem on the operation", reply: `{"groups": [[3, 1], [2]]}`, want: [][]int{{0, 2}, {1}}},
		{name: "two problems on one operation", reply: `{"groups": [[3, 1], [3, 2]]}`, want: [][]int{{0, 2}, {1, 2}}},
		{name: "in no list", reply: `{"groups": [[1], [2]]}`, want: [][]int{{0}, {1}}},
		{name: "alone, which says nothing", reply: `{"groups": [[1], [2], [3], [4]]}`, want: [][]int{{0}, {1}}},
		{name: "named twice in one list", reply: `{"groups": [[3, 1, 3], [2]]}`, want: [][]int{{0, 2}, {1}}},
		{name: "two standard findings", reply: `{"groups": [[3, 4, 1], [2]]}`, problems: []string{"groups[0] holds 2 standard findings"}},
		{name: "a finding of a review still goes in one list", reply: `{"groups": [[3, 1], [4, 1], [2]]}`, problems: []string{"finding 1 is in more than one list"}},
		{name: "a thread keeps its comment", reply: `{"groups": [[3, 1], [2]], "same": [1, 1]}`, want: [][]int{{0, 2}, {1}}, same: []int{0}},
		{name: "no thread named", reply: `{"groups": [[3, 1], [2]], "same": []}`, want: [][]int{{0, 2}, {1}}},
		{name: "a thread the call does not show", reply: `{"groups": [[3, 1], [2]], "same": [2]}`, problems: []string{"same holds 2, which is not a thread id"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			groups, same, problems := parseMergeReply(tc.reply, members, []int{0}, call)
			if tc.problems != nil {
				require.Equal(t, tc.problems, problems)
				return
			}
			require.Empty(t, problems)
			require.Equal(t, tc.want, groups)
			require.Equal(t, tc.same, same)
		})
	}
}

// standardRequest holds a review's finding that repeats a standard finding on
// the drop, another problem of the drop on a second database, and an index
// finding on a block with no standard finding.
func standardRequest() *MergeRequest {
	return &MergeRequest{
		Statement: mergeStatement,
		Blocks:    []LineRange{{Start: 1, End: 2}, {Start: 3, End: 3}},
		Candidates: []MergeCandidate{
			{Block: 0, Database: "instances/prod/databases/db1", Finding: dropFinding(2, "Dropping full_name loses its data")},
			{Block: 0, Database: "instances/prod/databases/db2", Finding: dropFinding(2, "DROP COLUMN full_name fails: v_user_directory reads it")},
			{Block: 1, Database: "instances/prod/databases/db1", Finding: dropFinding(3, "Create the index CONCURRENTLY")},
			{Block: 0, Standard: true, Finding: Finding{Title: "DISALLOW_DROP_OBJECT (DO101)", Severity: SeverityP1, Line: 2, Evidence: "Dropping column `full_name` loses its data."}},
		},
		Threads: []MergeThread{{Comment: "Dropping column `full_name` loses its data.\n\nusers holds 30,000 rows.", Candidates: []int{3}}},
	}
}

func TestMergeFindingsPutsFindingsWithAStandardFinding(t *testing.T) {
	t.Parallel()

	model := &scriptedModel{steps: []modelStep{finalReply(`{"groups": [[4, 1], [4, 2], [3]], "same": [1]}`)}}
	result, err := NewReviewer(model).MergeFindings(context.Background(), standardRequest())
	require.NoError(t, err)
	require.Equal(t, [][]int{{0, 3}, {1, 3}, {2}}, result.Groups, "the standard finding is in the list of each problem on its operation")
	require.Equal(t, []int{0}, result.Same)

	request := model.requests[0]
	require.Equal(t, strings.TrimSpace(mergePrompt)+"\n\n"+strings.TrimSpace(mergeStandardPrompt), request.Messages[0].GetContent())
	user := request.Messages[1].GetContent()
	require.Contains(t, user, `{"id":4,"block":1,"standard":true,"thread":1,"line":2,"severity":"P1","title":"DISALLOW_DROP_OBJECT (DO101)","rule":"","evidence":"Dropping column `+"`full_name`"+` loses its data.","fix":""}`, "a standard finding has no database")
	require.Contains(t, user, `{"id":2,"block":1,"database":2,`, "the databases are counted without it")
	require.Contains(t, user, "<threads-")
	require.Contains(t, user, `{"thread":1,"comment":"Dropping column `+"`full_name`"+` loses its data.\n\nusers holds 30,000 rows."}`)
}

func TestMergeFindingsShowsNoThreadTooLongToRead(t *testing.T) {
	t.Parallel()

	request := standardRequest()
	request.Threads[0].Comment = strings.Repeat("x", maxMergeThreadBytes+1)
	model := &scriptedModel{steps: []modelStep{
		finalReply(`{"groups": [[4, 1], [4, 2], [3]], "same": [1]}`),
		finalReply(`{"groups": [[4, 1], [4, 2], [3]]}`),
	}}
	result, err := NewReviewer(model).MergeFindings(context.Background(), request)
	require.NoError(t, err)
	require.Empty(t, result.Same)
	require.Equal(t, 2, result.Calls, "a thread that is not shown cannot be named")
	user := model.requests[0].Messages[1].GetContent()
	require.NotContains(t, user, "<threads-")
	require.NotContains(t, user, `"thread":`)
}

func TestMergeFindingsKeepsACommentOnlyWhenEveryCallSaysSo(t *testing.T) {
	t.Parallel()

	// The thread's two standard findings are on two blocks, and each block
	// fills a call of its own.
	request := standardRequest()
	request.Candidates = append(request.Candidates, MergeCandidate{Block: 1, Standard: true, Finding: Finding{Title: "DISALLOW_DROP_OBJECT (DO101)", Severity: SeverityP1, Line: 3}})
	request.Threads[0].Candidates = []int{3, 4}
	merge := func(replies ...string) *MergeResult {
		t.Helper()
		model := &scriptedModel{}
		for _, reply := range replies {
			model.steps = append(model.steps, finalReply(reply))
		}
		reviewer := NewReviewer(model)
		reviewer.mergePromptBytes = 1100
		result, err := reviewer.MergeFindings(context.Background(), request)
		require.NoError(t, err)
		require.Len(t, model.requests, 2)
		return result
	}

	require.Equal(t, []int{0}, merge(`{"groups": [[4, 1], [2]], "same": [1]}`, `{"groups": [[5, 3]], "same": [1]}`).Same)
	require.Empty(t, merge(`{"groups": [[4, 1], [2]], "same": [1]}`, `{"groups": [[5, 3]]}`).Same)
}

func TestMergeFindingsRejectsAThreadOfOtherCandidates(t *testing.T) {
	t.Parallel()

	request := standardRequest()
	request.Threads[0].Candidates = []int{0}
	_, err := NewReviewer(&scriptedModel{}).MergeFindings(context.Background(), request)
	require.ErrorContains(t, err, "thread 0 holds candidate 0, which is not a standard candidate")
}
