package aireview

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// TestReviewLiveStandardRules checks with a real Claude model that the review
// leaves a standard rule's condition to the standard review or the project's
// switch, and still reports what the rule cannot see and what the policy
// names. It needs a key and network access that CI does not have:
//
//	AIREVIEW_LIVE_CLAUDE_API_KEY=... go test ./backend/plugin/aireview/ -run '^TestReviewLiveStandardRules$' -v -count=1
//
// claude-sonnet-5 still reports the missing table about half the time while
// WALK_THROUGH is off; claude-sonnet-5-5 and claude-haiku-5-5 do not.
func TestReviewLiveStandardRules(t *testing.T) {
	t.Parallel()
	apiKey := os.Getenv("AIREVIEW_LIVE_CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("AIREVIEW_LIVE_CLAUDE_API_KEY is not set")
	}
	modelName := os.Getenv("AIREVIEW_LIVE_CLAUDE_MODEL")
	if modelName == "" {
		modelName = "claude-sonnet-5-5"
	}

	everyRule := []StandardRule{
		{Type: storepb.ReviewRuleType_SYNTAX, Severity: SeverityP0},
		{Type: storepb.ReviewRuleType_WALK_THROUGH, Severity: SeverityP0},
		{Type: storepb.ReviewRuleType_REQUIRE_IS_NULL, Severity: SeverityP0},
		{Type: storepb.ReviewRuleType_REQUIRE_WHERE, Severity: SeverityP1},
		{Type: storepb.ReviewRuleType_DISALLOW_DROP_OBJECT, Severity: SeverityP1},
		{Type: storepb.ReviewRuleType_DISALLOW_TRUNCATE, Severity: SeverityP1},
		{Type: storepb.ReviewRuleType_DISALLOW_DROP_CONSTRAINT, Severity: SeverityP1},
		{Type: storepb.ReviewRuleType_DISALLOW_RENAME, Severity: SeverityP1},
		{Type: storepb.ReviewRuleType_REQUIRE_PRIMARY_KEY, Severity: SeverityP1},
	}
	// onPostgres is what the AI review sends for a PostgreSQL database with
	// every rule on: the standard review evaluates each one, and the model
	// checks WALK_THROUGH too, since the walk can stop early.
	onPostgres := make([]StandardRule, 0, len(everyRule))
	for _, rule := range everyRule {
		rule.Reviewed = rule.Type != storepb.ReviewRuleType_WALK_THROUGH
		onPostgres = append(onPostgres, rule)
	}
	walkOff := slices.DeleteFunc(slices.Clone(onPostgres), func(rule StandardRule) bool {
		return rule.Type == storepb.ReviewRuleType_WALK_THROUGH
	})
	const whereSentence = "Every UPDATE needs a WHERE clause."
	citesRule := func(rule string) func(Finding) bool {
		return func(finding Finding) bool { return strings.Contains(finding.Rule, rule) }
	}

	tests := []struct {
		name      string
		statement string
		policy    string
		rules     []StandardRule
		runs      bool
		// check judges the findings.
		check func(t *testing.T, findings []Finding)
	}{
		{
			name:      "an UPDATE with no WHERE is left to the standard review",
			statement: "UPDATE orders SET currency = 'EUR';",
			rules:     onPostgres,
			runs:      true,
			check:     func(t *testing.T, findings []Finding) { require.Empty(t, findings) },
		},
		{
			name:      "an always true WHERE is reported for the rule",
			statement: "UPDATE orders SET currency = 'EUR' WHERE 1=1;",
			rules:     onPostgres,
			runs:      true,
			check: func(t *testing.T, findings []Finding) {
				require.True(t, anyFinding(findings, citesRule("REQUIRE_WHERE")), "a finding cites REQUIRE_WHERE")
			},
		},
		{
			name:      "a dropped column reports the view that reads it, not the drop",
			statement: "ALTER TABLE orders DROP COLUMN legacy_status;",
			rules:     onPostgres,
			runs:      true,
			check: func(t *testing.T, findings []Finding) {
				require.NotEmpty(t, findings)
				for _, finding := range findings {
					require.Contains(t, finding.Title+finding.Evidence, "orders_summary", "every finding is about the view")
					require.NotContains(t, finding.Rule, "DISALLOW_DROP_OBJECT")
				}
			},
		},
		{
			name:      "a TRUNCATE the project turned off stays out under a general aim",
			statement: "TRUNCATE order_events;",
			policy:    "Protect production data.",
			rules:     []StandardRule{{Type: storepb.ReviewRuleType_REQUIRE_WHERE, Severity: SeverityP1, Reviewed: true}},
			runs:      true,
			check:     func(t *testing.T, findings []Finding) { require.Empty(t, findings) },
		},
		{
			name:      "a rule that is off comes back when the policy names it",
			statement: "UPDATE orders SET currency = 'EUR';",
			policy:    whereSentence,
			runs:      true,
			check: func(t *testing.T, findings []Finding) {
				require.True(t, anyFinding(findings, citesRule(whereSentence)), "a finding quotes the policy")
			},
		},
		{
			name:      "a rule that is reported is reported again when the policy names it",
			statement: "UPDATE orders SET currency = 'EUR';",
			policy:    whereSentence,
			rules:     onPostgres,
			runs:      true,
			check: func(t *testing.T, findings []Finding) {
				require.True(t, anyFinding(findings, citesRule(whereSentence)), "a finding quotes the policy")
			},
		},
		{
			name:      "a missing table is the model's while WALK_THROUGH is on",
			statement: "DELETE FROM invoices WHERE id = 1;",
			rules:     onPostgres,
			runs:      true,
			check: func(t *testing.T, findings []Finding) {
				require.True(t, anyFinding(findings, citesRule("WALK_THROUGH")), "a finding cites WALK_THROUGH")
			},
		},
		{
			name:      "a missing column is the model's while WALK_THROUGH is on",
			statement: "UPDATE orders SET nonexistent_column = 1 WHERE id = 1;",
			rules:     onPostgres,
			runs:      true,
			check: func(t *testing.T, findings []Finding) {
				require.True(t, anyFinding(findings, func(finding Finding) bool {
					return strings.Contains(finding.Title+finding.Evidence, "nonexistent_column")
				}), "a finding names the missing column")
			},
		},
		{
			name:      "a missing table stays out while WALK_THROUGH is off",
			statement: "DELETE FROM invoices WHERE id = 1;",
			rules:     walkOff,
			runs:      true,
			check:     func(t *testing.T, findings []Finding) { require.Empty(t, findings) },
		},
		{
			name:      "a rule that is on is the model's where no standard review runs",
			statement: "UPDATE orders SET currency = 'EUR';",
			rules:     everyRule,
			check: func(t *testing.T, findings []Finding) {
				require.True(t, anyFinding(findings, citesRule("REQUIRE_WHERE")), "a finding cites REQUIRE_WHERE")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			model := NewModel(&storepb.AISetting{
				Provider: storepb.AISetting_CLAUDE,
				Endpoint: "https://api.anthropic.com/v1/messages",
				Model:    modelName,
				ApiKey:   apiKey,
			})
			request := &Request{
				ProjectPolicy:      tc.policy,
				StandardRules:      tc.rules,
				StandardReviewRuns: tc.runs,
				Target: Target{
					Engine:      "POSTGRES",
					Version:     "16.2",
					Environment: "prod",
					Schemas:     []SchemaSummary{{Name: "public", ObjectCount: 3}},
				},
				Statement: tc.statement,
			}
			result, err := NewReviewer(model).Review(context.Background(), request, shopCatalog())
			require.NoError(t, err)
			t.Logf("model calls: %d, tokens: %d", result.Calls, result.TotalTokens)
			for _, finding := range result.Findings {
				t.Logf("%s line %d: %s\n  rule: %s\n  evidence: %s", finding.Severity, finding.Line, finding.Title, finding.Rule, finding.Evidence)
			}
			tc.check(t, result.Findings)
		})
	}
}

func anyFinding(findings []Finding, match func(Finding) bool) bool {
	for _, finding := range findings {
		if match(finding) {
			return true
		}
	}
	return false
}

// shopCatalog is a small schema, so that row counts raise no operational
// risk: a view reads orders.legacy_status, and nothing reads order_events.
func shopCatalog() Tools {
	return NewCatalogTools(storepb.Engine_POSTGRES, &metadatapb.DatabaseSchemaMetadata{
		Name: "shop",
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: "public",
			Tables: []*metadatapb.TableMetadata{
				{
					Name: "orders",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "bigint"},
						{Name: "currency", Type: "text"},
						{Name: "legacy_status", Type: "text", Nullable: true},
					},
					Indexes: []*metadatapb.IndexMetadata{
						{Name: "orders_pkey", Expressions: []string{"id"}, Type: "btree", Unique: true, Primary: true, IsConstraint: true},
					},
					RowCount: 1200,
					DataSize: 256 << 10,
				},
				{
					Name: "order_events",
					Columns: []*metadatapb.ColumnMetadata{
						{Name: "id", Type: "bigint"},
						{Name: "payload", Type: "jsonb", Nullable: true},
					},
					Indexes: []*metadatapb.IndexMetadata{
						{Name: "order_events_pkey", Expressions: []string{"id"}, Type: "btree", Unique: true, Primary: true, IsConstraint: true},
					},
					RowCount: 3000,
					DataSize: 512 << 10,
				},
			},
			Views: []*metadatapb.ViewMetadata{{
				Name:       "orders_summary",
				Definition: "SELECT legacy_status, count(*) AS order_count FROM public.orders GROUP BY legacy_status",
			}},
		}},
	})
}
