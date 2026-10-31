package aireview

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Rule is what a finding cites by ID: a line of a policy or one of the checks.
type Rule struct {
	ID   string
	Text string
}

// checks are what the review looks for besides the policy. Each holds one
// kind of problem with one kind of fix: findings that cite the same rule on
// the same line merge into one thread across databases, so a check that
// spans two fixes would merge two problems.
var checks = []Rule{
	// A statement fails on one database and not another because their schemas
	// or rows differ, and each cause has its own fix, so each is a check.
	{ID: "already-exists", Text: "The statement fails on this database because an object it creates already exists."},
	{ID: "missing-object", Text: "The statement fails on this database because an object it uses or changes does not exist."},
	{ID: "violates-constraint", Text: "The statement fails on this database because existing rows violate a constraint or a foreign key."},
	{ID: "fails", Text: "The statement fails on this database for another reason, such as syntax or a type the engine version does not support."},
	{ID: "blocks", Text: "The statement blocks reads or writes on a large table for a long time, for example with a lock held while the table is rewritten, scanned, or validated, or with one UPDATE or DELETE over many rows."},
	{ID: "full-scan", Text: "The statement reads every row of a large table without blocking it, for example an UPDATE or DELETE whose condition no index serves."},
	{ID: "drops-data", Text: "The statement drops a table, column, or partition that still holds data."},
	{ID: "too-many-rows", Text: "An UPDATE or DELETE changes more rows than the author means."},
	{ID: "cascades", Text: "The statement deletes or changes rows in other tables through foreign key actions or triggers."},
	{ID: "breaks-dependent", Text: "The statement breaks an object that depends on what it changes, such as a view, trigger, or function that reads a renamed or dropped column."},
}

// isCheck reports whether id names one of the checks.
func isCheck(id string) bool {
	return slices.ContainsFunc(checks, func(check Rule) bool { return check.ID == id })
}

var (
	// listMarker matches the bullet or number that starts a list item.
	listMarker = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+`)
	// headingMarker matches the hashes that start a markdown heading.
	headingMarker = regexp.MustCompile(`^#{1,6}(?:\s+|$)`)
)

// Rules returns every rule a review under the two policies can cite: the
// rules of the workspace policy, the rules of the project policy, then the
// checks.
func Rules(workspacePolicy, projectPolicy string) []Rule {
	rules := policyRules(parsePolicy("workspace", workspacePolicy))
	rules = append(rules, policyRules(parsePolicy("project", projectPolicy))...)
	return append(rules, checks...)
}

// policyPart is a heading of a policy, which the rules below it read under,
// or one of its rules.
type policyPart struct {
	Heading string
	Rule    Rule
}

// parsePolicy splits a policy into its headings and numbered rules. A rule is
// a list item or a line of text, with the lines that continue it: a line
// continues the rule above when that rule has not ended its sentence and the
// line starts in lower case, as a hard-wrapped sentence does. A blank line, a
// heading, or a list marker ends a rule. A policy that wraps a sentence
// before a capitalized word reads as two rules; the prompt still shows both.
func parsePolicy(level string, policy string) []policyPart {
	var parts []policyPart
	count := 0
	// open is the index of the rule the next line can continue, or -1.
	open := -1
	for _, line := range strings.Split(policy, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			open = -1
		case headingMarker.MatchString(line):
			if heading := strings.TrimSpace(headingMarker.ReplaceAllString(line, "")); heading != "" {
				parts = append(parts, policyPart{Heading: heading})
			}
			open = -1
		case open >= 0 && !listMarker.MatchString(line) && continuesRule(parts[open].Rule.Text, line):
			parts[open].Rule.Text += " " + line
		default:
			count++
			parts = append(parts, policyPart{Rule: Rule{ID: fmt.Sprintf("%s-%d", level, count), Text: listMarker.ReplaceAllString(line, "")}})
			open = len(parts) - 1
		}
	}
	return parts
}

// continuesRule reports whether line goes on with the sentence of rule.
func continuesRule(rule string, line string) bool {
	first, _ := utf8.DecodeRuneInString(line)
	if !unicode.IsLower(first) {
		return false
	}
	// Closing quotes, brackets, and emphasis can follow the end of a sentence.
	rule = strings.TrimRight(rule, ")]}\"'`*_\u201d\u2019")
	last, _ := utf8.DecodeLastRuneInString(rule)
	return !strings.ContainsRune(".!?;\u3002\uff01\uff1f\uff1b", last)
}

func policyRules(parts []policyPart) []Rule {
	var rules []Rule
	for _, part := range parts {
		if part.Heading == "" {
			rules = append(rules, part.Rule)
		}
	}
	return rules
}

// ruleList renders rules as the prompt lists them, one per line after its ID.
func ruleList(rules []Rule) string {
	lines := make([]string, 0, len(rules))
	for _, rule := range rules {
		lines = append(lines, fmt.Sprintf("- `%s`: %s", rule.ID, rule.Text))
	}
	return strings.Join(lines, "\n")
}
