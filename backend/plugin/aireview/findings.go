package aireview

import (
	"encoding/json"
	"fmt"
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
	Title    string
	Severity Severity
	// Line is the one-based line of the statements where the problem starts.
	Line     int
	Rule     string
	Evidence string
	Fix      string
}

type replyJSON struct {
	// Findings is a pointer so that a reply without the key is told apart from
	// an empty list, which means the change passes.
	Findings *[]findingJSON `json:"findings"`
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

// parseFindings reads the model's final reply. It returns the problems that
// make the reply invalid, worded for the model to correct. One invalid finding
// invalidates the whole reply: dropping it instead could empty the list, and an
// empty list passes the change.
func parseFindings(text string, lineCount int) ([]Finding, []string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, []string{"the reply is empty"}
	}

	reply, problem := decodeReply(text)
	if problem != "" {
		return nil, []string{problem}
	}
	if reply.Findings == nil {
		return nil, []string{`the JSON object has no "findings" key`}
	}

	var findings []Finding
	var problems []string
	for i, raw := range *reply.Findings {
		finding, findingProblems := validateFinding(raw, lineCount)
		for _, problem := range findingProblems {
			problems = append(problems, fmt.Sprintf("findings[%d].%s", i, problem))
		}
		findings = append(findings, finding)
	}
	if len(problems) > maxReportedProblems {
		more := len(problems) - maxReportedProblems
		problems = append(problems[:maxReportedProblems], fmt.Sprintf("and %d more problems", more))
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return findings, nil
}

// decodeReply accepts the bare object or the object inside a code fence. An
// object embedded in prose counts only when it holds findings: an empty list
// passes the change, and prose that mentions {"findings": []} is not a verdict.
func decodeReply(text string) (*replyJSON, string) {
	reply := &replyJSON{}
	if err := json.Unmarshal([]byte(text), reply); err == nil {
		return reply, ""
	}
	if unfenced, ok := stripCodeFence(text); ok {
		reply = &replyJSON{}
		if err := json.Unmarshal([]byte(unfenced), reply); err == nil {
			return reply, ""
		}
	}

	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return nil, "the reply is not a JSON object"
	}
	reply = &replyJSON{}
	if err := json.Unmarshal([]byte(text[start:end+1]), reply); err != nil {
		return nil, fmt.Sprintf("the reply is not a valid JSON object: %v", err)
	}
	if reply.Findings != nil && len(*reply.Findings) == 0 {
		return nil, "the reply has text around the JSON object; reply with only the JSON object"
	}
	return reply, ""
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
		Title:    strings.TrimSpace(raw.Title),
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
