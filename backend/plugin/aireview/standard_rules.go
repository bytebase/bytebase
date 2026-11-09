package aireview

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

//go:embed standard_rules_prompt.md
var standardRulesPrompt string

// StandardRule is a standard review rule that is on for the project, with the
// severity the standard review gives its findings.
type StandardRule struct {
	Type     storepb.ReviewRuleType
	Severity Severity
	// Reviewed is true when the standard review evaluates the rule on the
	// target's engine, so the model leaves the rule's condition to it.
	// Otherwise the model checks it.
	Reviewed bool
}

// standardRuleSentences names the condition of each rule the prompt states,
// and the risk that comes with it, in rule order. The model quotes a sentence
// as the rule of a finding it reports for the rule, so each is one sentence.
// ONLINE_MIGRATION and PRIOR_BACKUP are left out: they judge the change's
// options, which the review does not see, so no finding of the model repeats
// theirs.
var standardRuleSentences = []struct {
	rule     storepb.ReviewRuleType
	sentence string
}{
	{storepb.ReviewRuleType_SYNTAX, "A statement that does not parse for this engine fails when it runs."},
	{storepb.ReviewRuleType_WALK_THROUGH, "A statement fails when a name it uses does not fit the schema: an object that does not exist or is of another kind, a name that is already taken or used twice, a name the engine rejects, or a schema to drop that still holds objects."},
	{storepb.ReviewRuleType_REQUIRE_IS_NULL, "A comparison with NULL by = or <> is never true, so its predicate matches no row."},
	{storepb.ReviewRuleType_REQUIRE_WHERE, "An UPDATE or DELETE with no WHERE clause changes every row of its table."},
	{storepb.ReviewRuleType_DISALLOW_DROP_OBJECT, "Dropping an object, such as a table, column, view, index, sequence, schema, function, user, or role, loses the data, definition, or access it held."},
	{storepb.ReviewRuleType_DISALLOW_TRUNCATE, "A TRUNCATE removes every row of its table."},
	{storepb.ReviewRuleType_DISALLOW_DROP_CONSTRAINT, "Dropping a primary key, foreign key, unique, or check constraint stops the database from enforcing it."},
	{storepb.ReviewRuleType_DISALLOW_RENAME, "Renaming a table, view, or column breaks the application queries that use the old name."},
	{storepb.ReviewRuleType_REQUIRE_PRIMARY_KEY, "A table created without a primary key, or whose primary key is dropped and not added back, has nothing that identifies its rows."},
}

// writeStandardRules states the standard rules section and then every rule
// with its state for this review: left to the standard review where the rule
// is on and the standard review evaluates it on the target's engine, yours
// where it is on and the standard review does not evaluate it there, and off
// otherwise. A state reads as a fact, not an order: "reported" read as one to
// report.
func writeStandardRules(b *strings.Builder, request *Request) {
	_, _ = b.WriteString("\n\n" + strings.TrimSpace(standardRulesPrompt) + "\n\n## Rules for this review\n\n")
	// The model is told outright, since the section's general sentences
	// cannot say which case this review is.
	if request.StandardReviewRuns {
		_, _ = b.WriteString("The standard review runs on this database's engine.\n\n")
	} else {
		_, _ = b.WriteString("No standard review runs on this database's engine, so the rules that are on are yours.\n\n")
	}
	for _, rule := range standardRuleSentences {
		i := slices.IndexFunc(request.StandardRules, func(on StandardRule) bool { return on.Type == rule.rule })
		switch {
		case i < 0:
			_, _ = fmt.Fprintf(b, "- %s (off): %s\n", rule.rule, rule.sentence)
		case request.StandardRules[i].Reviewed:
			_, _ = fmt.Fprintf(b, "- %s (left to the standard review, %s): %s\n", rule.rule, request.StandardRules[i].Severity, rule.sentence)
		default:
			_, _ = fmt.Fprintf(b, "- %s (yours, %s): %s\n", rule.rule, request.StandardRules[i].Severity, rule.sentence)
		}
	}
}
