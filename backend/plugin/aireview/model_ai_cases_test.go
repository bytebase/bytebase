package aireview

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"text/tabwriter"
	"unicode"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
)

const reviewCasesDir = "testdata/cases"

// caseProviders holds the endpoint and model each provider runs the cases
// with unless the environment names others.
var caseProviders = map[string]struct {
	provider storepb.AISetting_Provider
	endpoint string
	model    string
}{
	"gemini": {provider: storepb.AISetting_GEMINI, endpoint: "https://generativelanguage.googleapis.com/v1beta", model: "gemini-3.5-flash"},
	"openai": {provider: storepb.AISetting_OPEN_AI, endpoint: "https://api.openai.com/v1/responses", model: "gpt-5.6-terra"},
	"claude": {provider: storepb.AISetting_CLAUDE, endpoint: "https://api.anthropic.com/v1/messages", model: "claude-sonnet-5"},
}

// TestReviewCases runs every case under testdata/cases with a real model and
// logs how the review scored. testdata/cases/README.md has the commands. It
// fails when a review fails or misses a finding a case requires; a flag a
// case forbids is reported but does not fail it, since it costs noise rather
// than a missed problem.
func TestReviewCases(t *testing.T) {
	t.Parallel()
	setting := reviewCaseSetting(t)
	cases := loadReviewCases(t)
	model := redactingModel{Model: NewModel(setting), apiKey: setting.GetApiKey()}

	scores := make([]*caseScore, len(cases))
	// The parallel subtests run only after this function returns, so the
	// table is logged from a cleanup.
	t.Cleanup(func() {
		t.Logf("%s model %s at %s\n%s", setting.GetProvider(), setting.GetModel(), setting.GetEndpoint(), scoreTable(scores))
	})
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			scores[i] = runReviewCase(t, model, c)
		})
	}
}

// TestReviewCasesLoad keeps the case files valid without a model.
func TestReviewCasesLoad(t *testing.T) {
	t.Parallel()
	cases := loadReviewCases(t)
	engines := make(map[storepb.Engine]bool)
	for _, c := range cases {
		engines[c.engine] = true
	}
	require.True(t, engines[storepb.Engine_POSTGRES], "the cases cover PostgreSQL")
	require.True(t, engines[storepb.Engine_MYSQL], "the cases cover MySQL")
}

func TestScoreFindings(t *testing.T) {
	t.Parallel()

	expect := caseExpectation{
		Find:    []findingMatch{{Line: 3, Keywords: []string{"order_revenue", "view"}}},
		NotFlag: []findingMatch{{Line: 2}, {Keywords: []string{"rewrite"}}},
	}
	tests := []struct {
		name       string
		findings   []Finding
		missed     int
		falseFlags int
		others     int
	}{
		{
			name:     "a keyword in the evidence matches regardless of case",
			findings: []Finding{{Line: 3, Title: "Drop the column later", Evidence: "Public.Order_Revenue reads it"}},
		},
		{
			name:     "a keyword in the title matches the start of a word",
			findings: []Finding{{Line: 3, Title: "Update the dependent views first", Evidence: "it reads discount"}},
		},
		{
			name:     "a keyword inside a word does not match",
			findings: []Finding{{Line: 3, Title: "Have the drop reviewed", Evidence: "public.my_order_revenue reads it"}},
			missed:   1,
			others:   1,
		},
		{
			name:     "a second finding for a found entry is another finding",
			findings: []Finding{{Line: 3, Title: "Update the view first", Evidence: "order_revenue reads it"}, {Line: 3, Title: "Drop order_revenue first", Evidence: "it reads discount"}},
			others:   1,
		},
		{
			name:     "the right words on another line miss",
			findings: []Finding{{Line: 4, Title: "Update the view first", Evidence: "order_revenue reads it"}},
			missed:   1,
			others:   1,
		},
		{
			name:     "the right line without a keyword misses",
			findings: []Finding{{Line: 3, Title: "Back up the column", Evidence: "it holds data"}, {Line: 3, Title: "Batch the change", Evidence: "the table is large"}},
			missed:   1,
			others:   2,
		},
		{
			name:       "a forbidden line or word is a false flag",
			findings:   []Finding{{Line: 3, Title: "Update the view first", Evidence: "order_revenue reads it"}, {Line: 2, Title: "Set a lock timeout", Evidence: "the lock waits"}, {Line: 5, Title: "Avoid the table rewrite", Evidence: "it rewrites"}},
			falseFlags: 2,
		},
		{
			name:   "no findings miss what is required",
			missed: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			outcome := scoreFindings(expect, tc.findings)
			require.Len(t, outcome.missed, tc.missed)
			require.Len(t, outcome.falseFlags, tc.falseFlags)
			require.Len(t, outcome.others, tc.others)
		})
	}
}

func TestRedactingModelRemovesTheKeyFromAnErrorBody(t *testing.T) {
	t.Parallel()

	failing := failingModel{err: errors.Wrap(&ai.HTTPError{Provider: "OpenAI", StatusCode: 401, Body: `{"error": "invalid key sk-secret"}`}, "call failed")}
	_, err := redactingModel{Model: failing, apiKey: "sk-secret"}.Chat(context.Background(), &v1pb.AIChatRequest{})
	require.NotContains(t, err.Error(), "sk-secret")
	require.Contains(t, err.Error(), "invalid key [key]")
	var httpErr *ai.HTTPError
	require.ErrorAs(t, err, &httpErr, "the reviewer still tells a refusal of tools apart")
}

type failingModel struct{ err error }

func (m failingModel) Chat(context.Context, *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	return nil, m.err
}

func TestScoreFindingsCountsAFindingForOneEntry(t *testing.T) {
	t.Parallel()

	expect := caseExpectation{Find: []findingMatch{
		{Line: 2, Keywords: []string{"cascade", "orders"}},
		{Line: 2, Keywords: []string{"batch"}},
	}}
	both := Finding{Line: 2, Title: "Batch the delete", Evidence: "it cascades to orders"}
	cascade := Finding{Line: 2, Title: "Keep the order history", Evidence: "orders cascade"}

	outcome := scoreFindings(expect, []Finding{both})
	require.Len(t, outcome.missed, 1)

	// The first entry takes the first finding, and must give it up to the
	// second entry, which no other finding matches.
	outcome = scoreFindings(expect, []Finding{both, cascade})
	require.Empty(t, outcome.missed)
	require.Empty(t, outcome.others)
}

type reviewCase struct {
	// name is the file's path under testdata/cases without the extension,
	// which is also its subtest name.
	name    string
	engine  storepb.Engine
	schema  *metadatapb.DatabaseSchemaMetadata
	request *Request
	expect  caseExpectation
}

// testdata/cases/README.md describes the fields.
type caseFile struct {
	Description     string          `yaml:"description"`
	Engine          string          `yaml:"engine"`
	Version         string          `yaml:"version"`
	Environment     string          `yaml:"environment"`
	WorkspacePolicy string          `yaml:"workspace_policy"`
	ProjectPolicy   string          `yaml:"project_policy"`
	Statement       string          `yaml:"statement"`
	Schema          map[string]any  `yaml:"schema"`
	Expect          caseExpectation `yaml:"expect"`
}

type caseExpectation struct {
	Find    []findingMatch `yaml:"find"`
	NotFlag []findingMatch `yaml:"not_flag"`
}

// findingMatch matches a finding on its line and on any of its keywords. A
// keyword matches where a word of the title or the evidence starts, regardless
// of case: view matches views but not review. A line of 0 matches every line,
// and no keywords match every finding.
type findingMatch struct {
	Line     int      `yaml:"line"`
	Keywords []string `yaml:"keywords"`
}

func (m findingMatch) matches(finding Finding) bool {
	if m.Line != 0 && finding.Line != m.Line {
		return false
	}
	if len(m.Keywords) == 0 {
		return true
	}
	text := strings.ToLower(finding.Title + "\n" + finding.Evidence)
	for _, keyword := range m.Keywords {
		if startsWord(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func startsWord(text string, word string) bool {
	for offset := 0; ; {
		i := strings.Index(text[offset:], word)
		if i < 0 {
			return false
		}
		i += offset
		before, _ := utf8.DecodeLastRuneInString(text[:i])
		if i == 0 || (before != '_' && !unicode.IsLetter(before) && !unicode.IsDigit(before)) {
			return true
		}
		offset = i + 1
	}
}

func (m findingMatch) String() string {
	where := "any line"
	if m.Line != 0 {
		where = fmt.Sprintf("line %d", m.Line)
	}
	if len(m.Keywords) == 0 {
		return where
	}
	return fmt.Sprintf("%s about %s", where, strings.Join(m.Keywords, " or "))
}

type caseOutcome struct {
	missed     []findingMatch
	falseFlags []Finding
	// others are the findings neither paired with a find entry nor forbidden.
	// They are logged for a reader to judge and do not decide the result.
	others []Finding
}

// scoreFindings pairs each find entry with a finding of its own, so that one
// finding cannot count for two entries. The pairing is Kuhn's maximum
// matching, in which an entry left unpaired in its turn stays unpaired.
func scoreFindings(expect caseExpectation, findings []Finding) caseOutcome {
	var outcome caseOutcome
	pairedWith := make([]int, len(findings))
	for j := range pairedWith {
		pairedWith[j] = -1
	}
	var pair func(i int, visited []bool) bool
	pair = func(i int, visited []bool) bool {
		for j, finding := range findings {
			if visited[j] || !expect.Find[i].matches(finding) {
				continue
			}
			visited[j] = true
			if pairedWith[j] < 0 || pair(pairedWith[j], visited) {
				pairedWith[j] = i
				return true
			}
		}
		return false
	}
	for i, want := range expect.Find {
		if !pair(i, make([]bool, len(findings))) {
			outcome.missed = append(outcome.missed, want)
		}
	}
	for j, finding := range findings {
		switch {
		case slices.ContainsFunc(expect.NotFlag, func(m findingMatch) bool { return m.matches(finding) }):
			outcome.falseFlags = append(outcome.falseFlags, finding)
		case pairedWith[j] < 0:
			outcome.others = append(outcome.others, finding)
		default:
		}
	}
	return outcome
}

type caseScore struct {
	name      string
	wanted    int
	outcome   caseOutcome
	calls     int
	toolCalls int
	tokens    int
	err       error
}

func (s *caseScore) result() string {
	switch {
	case s.err != nil:
		return "error"
	case len(s.outcome.missed) > 0:
		return "miss"
	case len(s.outcome.falseFlags) > 0:
		return "noisy"
	default:
		return "pass"
	}
}

// redactingModel removes the key from a provider's error body, which can
// quote the key it was sent, before the reviewer logs or returns the error.
type redactingModel struct {
	Model
	apiKey string
}

func (m redactingModel) Chat(ctx context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	response, err := m.Model.Chat(ctx, request)
	var httpErr *ai.HTTPError
	if errors.As(err, &httpErr) {
		httpErr.Body = strings.ReplaceAll(httpErr.Body, m.apiKey, "[key]")
	}
	return response, err
}

func runReviewCase(t *testing.T, model Model, c *reviewCase) *caseScore {
	tools := &recordingTools{Tools: NewCatalogTools(c.engine, c.schema, WithPromptOf(c.request))}
	result, err := NewReviewer(model).Review(context.Background(), c.request, tools)
	score := &caseScore{name: c.name, wanted: len(c.expect.Find)}
	if err != nil {
		score.err = err
		t.Errorf("the review failed: %v", err)
		return score
	}
	score.calls, score.toolCalls, score.tokens = result.Calls, len(tools.calls), result.TotalTokens
	score.outcome = scoreFindings(c.expect, result.Findings)

	t.Logf("tool calls: %v", tools.calls)
	for _, finding := range result.Findings {
		t.Logf("%s line %d: %s\n  evidence: %s\n  fix: %s", finding.Severity, finding.Line, finding.Title, finding.Evidence, finding.Fix)
	}
	for _, note := range result.Notes {
		t.Logf("note: %s", note)
	}
	for _, finding := range score.outcome.falseFlags {
		t.Logf("false flag: line %d: %s", finding.Line, finding.Title)
	}
	for _, missed := range score.outcome.missed {
		t.Errorf("missed the finding on %s", missed)
	}
	return score
}

func scoreTable(scores []*caseScore) string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "case\tfound\tfalse flags\tother findings\tmodel calls\ttool calls\ttokens\tresult")
	var found, wanted, falseFlags, others, calls, toolCalls, tokens int
	results := make(map[string]int)
	for _, s := range scores {
		// A case that -run filtered out has no score.
		if s == nil {
			continue
		}
		results[s.result()]++
		// The findings an errored review never reported count as not found.
		wanted += s.wanted
		if s.err != nil {
			_, _ = fmt.Fprintf(w, "%s\t-/%d\t-\t-\t-\t-\t-\terror\n", s.name, s.wanted)
			continue
		}
		caseFound := s.wanted - len(s.outcome.missed)
		_, _ = fmt.Fprintf(w, "%s\t%d/%d\t%d\t%d\t%d\t%d\t%d\t%s\n", s.name, caseFound, s.wanted, len(s.outcome.falseFlags), len(s.outcome.others), s.calls, s.toolCalls, s.tokens, s.result())
		found += caseFound
		falseFlags += len(s.outcome.falseFlags)
		others += len(s.outcome.others)
		calls += s.calls
		toolCalls += s.toolCalls
		tokens += s.tokens
	}
	_, _ = fmt.Fprintf(w, "total\t%d/%d\t%d\t%d\t%d\t%d\t%d\t%d pass, %d noisy, %d miss, %d error\n", found, wanted, falseFlags, others, calls, toolCalls, tokens, results["pass"], results["noisy"], results["miss"], results["error"])
	_ = w.Flush()
	return b.String()
}

// reviewCaseSetting skips the test unless the environment names a provider
// and a key.
func reviewCaseSetting(t *testing.T) *storepb.AISetting {
	t.Helper()
	name := os.Getenv("AIREVIEW_CASES_PROVIDER")
	apiKey := os.Getenv("AIREVIEW_CASES_API_KEY")
	if name == "" || apiKey == "" {
		t.Skip("AIREVIEW_CASES_PROVIDER and AIREVIEW_CASES_API_KEY are not both set")
	}
	provider, ok := caseProviders[name]
	if !ok {
		t.Fatalf("AIREVIEW_CASES_PROVIDER is %q, not one of gemini, openai, or claude", name)
	}
	setting := &storepb.AISetting{Provider: provider.provider, Endpoint: provider.endpoint, Model: provider.model, ApiKey: apiKey}
	if endpoint := os.Getenv("AIREVIEW_CASES_ENDPOINT"); endpoint != "" {
		setting.Endpoint = endpoint
	}
	if model := os.Getenv("AIREVIEW_CASES_MODEL"); model != "" {
		setting.Model = model
	}
	return setting
}

func loadReviewCases(t *testing.T) []*reviewCase {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(reviewCasesDir, "*", "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	cases := make([]*reviewCase, 0, len(paths))
	for _, path := range paths {
		c, err := loadReviewCase(path)
		require.NoError(t, err, path)
		cases = append(cases, c)
	}
	return cases
}

func loadReviewCase(path string) (*reviewCase, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file caseFile
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return nil, errors.Wrap(err, "failed to decode the case")
	}

	value, ok := storepb.Engine_value[file.Engine]
	if !ok || value == 0 {
		return nil, errors.Errorf("engine %q is not a storepb.Engine", file.Engine)
	}
	engine := storepb.Engine(value)
	if dir := filepath.Base(filepath.Dir(path)); dir != strings.ToLower(file.Engine) {
		return nil, errors.Errorf("a %s case belongs in %s/, not %s/", file.Engine, strings.ToLower(file.Engine), dir)
	}

	// The schema is written in the field names of the metadata proto. An
	// unknown one fails the case rather than drop out of the fixture, which is
	// why this is not common.ProtojsonUnmarshaler.
	schemaJSON, err := json.Marshal(file.Schema)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode the schema")
	}
	schema := &metadatapb.DatabaseSchemaMetadata{}
	//nolint:forbidigo
	if err := (protojson.UnmarshalOptions{}).Unmarshal(schemaJSON, schema); err != nil {
		return nil, errors.Wrap(err, "the schema is not a DatabaseSchemaMetadata")
	}
	if len(schema.GetSchemas()) == 0 {
		return nil, errors.New("the schema has no schemas")
	}

	if strings.TrimSpace(file.Description) == "" {
		return nil, errors.New("the description is empty")
	}
	if err := checkExpectation(file.Statement, file.Expect); err != nil {
		return nil, err
	}

	relative, err := filepath.Rel(reviewCasesDir, path)
	if err != nil {
		return nil, err
	}
	return &reviewCase{
		name:   strings.TrimSuffix(filepath.ToSlash(relative), ".yaml"),
		engine: engine,
		schema: schema,
		request: &Request{
			WorkspacePolicy: file.WorkspacePolicy,
			ProjectPolicy:   file.ProjectPolicy,
			Target:          caseTarget(file, schema),
			Statement:       file.Statement,
		},
		expect: file.Expect,
	}, nil
}

// checkExpectation rejects a line an expectation names that is outside the
// statement, blank, or a -- comment, which is where lines shifted by an edit
// to the statement usually land.
func checkExpectation(statement string, expect caseExpectation) error {
	if strings.TrimSpace(statement) == "" {
		return errors.New("the statement is empty")
	}
	if len(expect.Find)+len(expect.NotFlag) == 0 {
		return errors.New("the case expects nothing")
	}
	numbered, lineCount := numberLines(statement)
	statementLines := strings.Split(numbered, "\n")
	checkLine := func(line int) error {
		if line < 1 || line > lineCount {
			return errors.Errorf("line %d is outside the statement, which has lines 1 to %d", line, lineCount)
		}
		text := strings.TrimSpace(strings.SplitN(statementLines[line-1], "\t", 2)[1])
		if text == "" || strings.HasPrefix(text, "--") {
			return errors.Errorf("line %d is blank or a comment", line)
		}
		return nil
	}
	checkKeywords := func(keywords []string) error {
		for _, keyword := range keywords {
			if strings.TrimSpace(keyword) == "" {
				return errors.New("a keyword is empty")
			}
		}
		return nil
	}
	for _, want := range expect.Find {
		if err := checkLine(want.Line); err != nil {
			return errors.Wrap(err, "find")
		}
		if len(want.Keywords) == 0 {
			return errors.Errorf("find on line %d has no keywords", want.Line)
		}
		if err := checkKeywords(want.Keywords); err != nil {
			return errors.Wrap(err, "find")
		}
	}
	for _, flag := range expect.NotFlag {
		if flag.Line == 0 && len(flag.Keywords) == 0 {
			return errors.New("a not_flag entry needs a line, keywords, or both")
		}
		if flag.Line != 0 {
			if err := checkLine(flag.Line); err != nil {
				return errors.Wrap(err, "not_flag")
			}
		}
		if err := checkKeywords(flag.Keywords); err != nil {
			return errors.Wrap(err, "not_flag")
		}
	}
	return nil
}

// caseTarget states what reviewrun's aiReviewTarget states of a synced database.
func caseTarget(file caseFile, schema *metadatapb.DatabaseSchemaMetadata) Target {
	target := Target{
		Engine:       file.Engine,
		Version:      file.Version,
		Environment:  file.Environment,
		CharacterSet: schema.GetCharacterSet(),
		Collation:    schema.GetCollation(),
		SearchPath:   schema.GetSearchPath(),
		Owner:        schema.GetOwner(),
	}
	for _, s := range schema.GetSchemas() {
		count := len(s.GetTables()) + len(s.GetExternalTables()) + len(s.GetViews()) + len(s.GetMaterializedViews()) +
			len(s.GetFunctions()) + len(s.GetProcedures()) + len(s.GetSequences()) + len(s.GetPackages()) +
			len(s.GetEvents()) + len(s.GetStreams()) + len(s.GetTasks()) + len(s.GetEnumTypes()) + len(s.GetCompositeTypes())
		target.Schemas = append(target.Schemas, SchemaSummary{Name: s.GetName(), ObjectCount: count})
	}
	return target
}
