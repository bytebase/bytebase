package aireview

import (
	"fmt"
	"regexp"
	"strings"
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
	{ID: "drops-data", Text: "The statement drops a table, column, or partition that still holds data."},
	{ID: "too-many-rows", Text: "An UPDATE or DELETE changes more rows than the author means."},
	{ID: "cascades", Text: "The statement deletes or changes rows in other tables through foreign key actions or triggers."},
	{ID: "breaks-dependent", Text: "The statement breaks an object that depends on what it changes, such as a view, trigger, or function that reads a renamed or dropped column."},
}

// listMarker matches the bullet or number that starts a list item.
var listMarker = regexp.MustCompile(`^(?:[-*+]|\d+[.)])\s+`)

// Rules returns every rule a review under the two policies can cite: the
// lines of the workspace policy, the lines of the project policy, then the
// checks.
func Rules(workspacePolicy, projectPolicy string) []Rule {
	rules := policyRules("workspace", workspacePolicy)
	rules = append(rules, policyRules("project", projectPolicy)...)
	return append(rules, checks...)
}

// policyRules numbers the lines of a policy that hold text: one line is one
// rule.
func policyRules(level string, policy string) []Rule {
	var rules []Rule
	for _, line := range strings.Split(policy, "\n") {
		line = listMarker.ReplaceAllString(strings.TrimSpace(line), "")
		if line == "" {
			continue
		}
		rules = append(rules, Rule{ID: fmt.Sprintf("%s-%d", level, len(rules)+1), Text: line})
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
