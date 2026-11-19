package aireview

import (
	jsonv2 "encoding/json/v2"
	"fmt"
	"slices"
	"strings"
)

// Severity is how serious a finding is.
type Severity string

const (
	SeverityP0 Severity = "P0"
	SeverityP1 Severity = "P1"
	SeverityP2 Severity = "P2"
)

// Finding is one problem the review found in the statements.
type Finding struct {
	// Title is one line, since it heads the finding's comment.
	Title    string
	Severity Severity
	// Line is the one-based line of the statements where the problem starts.
	Line     int
	Rule     string
	Evidence string
	Fix      string
}

// StandardFinding is a finding the standard review posted on the database
// under review.
type StandardFinding struct {
	// ID names the finding in the prompt and in the reply, such as "S1".
	ID   string
	Line int
	Rule string
	// Codes are the finding codes of the conditions it reports.
	Codes []string
	// Text is the comment the finding posts.
	Text string
}

// Fact is what a review adds to a standard finding: something the database
// shows about the finding's statement that the finding's text lacks.
type Fact struct {
	// Standard is the ID of the standard finding.
	Standard string
	Text     string
}

// maxFactBytes bounds the text of one fact, which is one or two sentences.
const maxFactBytes = 400

// reply is the model's final answer once parsed and validated.
type reply struct {
	Findings []Finding
	Facts    []Fact
	// Notes name what the model could not check and why. They are for the
	// backend, never for the user, and they never count as findings.
	Notes []string
}

type replyJSON struct {
	// Findings is a pointer so that a reply without the key is told apart from
	// an empty list, which means the change passes.
	Findings *[]findingJSON `json:"findings"`
	Facts    []factJSON     `json:"facts"`
	Notes    []string       `json:"notes"`
}

type factJSON struct {
	Standard string `json:"standard"`
	Text     string `json:"text"`
}

type findingJSON struct {
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Line     int    `json:"line"`
	Rule     string `json:"rule"`
	Evidence string `json:"evidence"`
	Fix      string `json:"fix"`
}

const maxReportedProblems = 10

// parseReply reads the model's final reply. It returns the problems that make
// the reply invalid, worded for the model to correct. One invalid finding
// invalidates the whole reply: dropping it instead could empty the list, and an
// empty list passes the change. A fact changes no verdict, so one that names
// none of standard, or says nothing, is dropped.
func parseReply(text string, lineCount int, standard []StandardFinding) (*reply, []string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, []string{"the reply is empty"}
	}

	decoded, problem := decodeReply(text)
	if problem != "" {
		return nil, []string{problem}
	}
	if decoded.Findings == nil {
		return nil, []string{`the JSON object has no "findings" key`}
	}

	result := &reply{}
	var problems []string
	for i, raw := range *decoded.Findings {
		finding, findingProblems := validateFinding(raw, lineCount)
		for _, problem := range findingProblems {
			problems = append(problems, fmt.Sprintf("findings[%d].%s", i, problem))
		}
		result.Findings = append(result.Findings, finding)
	}
	if len(problems) > maxReportedProblems {
		more := len(problems) - maxReportedProblems
		problems = append(problems[:maxReportedProblems], fmt.Sprintf("and %d more problems", more))
	}
	if len(problems) > 0 {
		return nil, problems
	}
	for _, note := range decoded.Notes {
		if note = strings.TrimSpace(note); note != "" {
			result.Notes = append(result.Notes, note)
		}
	}
	for _, raw := range decoded.Facts {
		id := strings.TrimSpace(raw.Standard)
		// A fact is one paragraph, since it sits inside a comment's list.
		fact := strings.Join(strings.Fields(raw.Text), " ")
		if fact == "" || !slices.ContainsFunc(standard, func(finding StandardFinding) bool { return finding.ID == id }) {
			continue
		}
		fact = clip(fact, maxFactBytes)
		if i := slices.IndexFunc(result.Facts, func(f Fact) bool { return f.Standard == id }); i >= 0 {
			result.Facts[i].Text += " " + fact
			continue
		}
		result.Facts = append(result.Facts, Fact{Standard: id, Text: fact})
	}
	return result, nil
}

// decodeReply accepts the bare object or the object inside a code fence. An
// object embedded in prose counts only when it holds findings: an empty list
// passes the change, and prose that mentions {"findings": []} is not a verdict.
func decodeReply(text string) (*replyJSON, string) {
	if decoded, err := decodeStrict(text); err == nil {
		return decoded, ""
	}
	if unfenced, ok := stripCodeFence(text); ok {
		if decoded, err := decodeStrict(unfenced); err == nil {
			return decoded, ""
		}
	}

	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return nil, "the reply is not a JSON object"
	}
	decoded, err := decodeStrict(text[start : end+1])
	if err != nil {
		return nil, fmt.Sprintf("the reply is not a valid JSON object: %v", err)
	}
	if decoded.Findings != nil && len(*decoded.Findings) == 0 {
		return nil, "the reply has text around the JSON object; reply with only the JSON object"
	}
	return decoded, ""
}

// decodeStrict accepts only the answer format. A model that cannot review
// sometimes says so in a key of its own, such as "error", and dropping that
// key would turn its report into a pass. json/v2 also rejects a duplicate key,
// which in v1 would let a second "findings" erase the first, and trailing text.
func decodeStrict(text string) (*replyJSON, error) {
	decoded := &replyJSON{}
	if err := jsonv2.Unmarshal([]byte(text), decoded, jsonv2.RejectUnknownMembers(true)); err != nil {
		return nil, err
	}
	return decoded, nil
}

func stripCodeFence(text string) (string, bool) {
	if !strings.HasPrefix(text, "```") || !strings.HasSuffix(text, "```") {
		return "", false
	}
	body := strings.TrimSuffix(text, "```")
	newline := strings.Index(body, "\n")
	if newline < 0 {
		return "", false
	}
	// The opening line can name the language, as in ```json. Any other text is
	// prose outside the object, which must not ride along into a pass.
	switch strings.ToLower(strings.TrimSpace(strings.TrimPrefix(body[:newline], "```"))) {
	case "", "json", "jsonc", "json5":
		return strings.TrimSpace(body[newline+1:]), true
	default:
		return "", false
	}
}

func validateFinding(raw findingJSON, lineCount int) (Finding, []string) {
	finding := Finding{
		Title:    strings.Join(strings.Fields(raw.Title), " "),
		Severity: Severity(strings.ToUpper(strings.TrimSpace(raw.Severity))),
		Line:     raw.Line,
		Rule:     strings.TrimSpace(raw.Rule),
		Evidence: strings.TrimSpace(raw.Evidence),
		Fix:      strings.TrimSpace(raw.Fix),
	}

	var problems []string
	switch finding.Severity {
	case SeverityP0, SeverityP1, SeverityP2:
	default:
		problems = append(problems, fmt.Sprintf("severity: %q is not one of P0, P1, P2", raw.Severity))
	}
	if finding.Line < 1 || finding.Line > lineCount {
		problems = append(problems, fmt.Sprintf("line: %d is outside the statements, which have lines 1 to %d", finding.Line, lineCount))
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"title", finding.Title},
		{"rule", finding.Rule},
		{"evidence", finding.Evidence},
		{"fix", finding.Fix},
	} {
		if field.value == "" {
			problems = append(problems, field.name+": is empty")
		}
	}
	return finding, problems
}
