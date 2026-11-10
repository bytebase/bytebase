package reviewrun

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/bytebase/omni/review"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// standardRuleCatalogPath is the catalog the SQL Review settings pages and
// the review result cards read the engines and finding codes from. The path
// is relative to this package directory, where `go test` runs.
const standardRuleCatalogPath = "../../../frontend/src/components/sql-review/standardRuleCatalog.json"

// To regenerate the catalog, set recordCatalog to true, run
// TestFrontendStandardRuleCatalogInSync, and set it back.
var recordCatalog = false

type standardRuleCatalog struct {
	// Engines the standard rule review covers, in enum order.
	Engines []string `json:"engines"`
	// By rule name.
	Rules map[string]catalogRule `json:"rules"`
}

type catalogRule struct {
	// Engines that check the rule, in enum order. An engine that only
	// reports that it lacks the rule's feature, such as online migration,
	// is left out.
	Engines []string      `json:"engines"`
	Codes   []catalogCode `json:"codes"`
}

type catalogCode struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

// TestFrontendStandardRuleCatalogInSync fails when the frontend catalog drifts
// from what omni and reviewFuncs define: the engines reviewed, which of them
// check each rule, and each rule's finding codes.
func TestFrontendStandardRuleCatalogInSync(t *testing.T) {
	t.Parallel()
	want := buildStandardRuleCatalog(t)
	if recordCatalog {
		data, err := json.MarshalIndent(want, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(standardRuleCatalogPath, append(data, '\n'), 0o644))
		t.Fatal("recorded the catalog; set recordCatalog back to false")
	}
	data, err := os.ReadFile(standardRuleCatalogPath)
	require.NoError(t, err)
	var got standardRuleCatalog
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, want, got, "set recordCatalog to true in this test and run it to regenerate %s", standardRuleCatalogPath)
}

func buildStandardRuleCatalog(t *testing.T) standardRuleCatalog {
	t.Helper()
	engines := make([]storepb.Engine, 0, len(reviewFuncs))
	for engine := range reviewFuncs {
		require.Contains(t, engineRules, engine, "engineRules is missing a reviewed engine")
		engines = append(engines, engine)
	}
	require.Len(t, engineRules, len(reviewFuncs), "engineRules lists an engine reviewFuncs does not review")
	slices.Sort(engines)

	catalog := standardRuleCatalog{Rules: make(map[string]catalogRule)}
	for _, engine := range engines {
		catalog.Engines = append(catalog.Engines, engine.String())
		lacking := lackingFeatureRules(t, engine)
		for _, rule := range engineRules[engine] {
			if lacking[rule] {
				continue
			}
			entry := catalog.Rules[string(rule)]
			entry.Engines = append(entry.Engines, engine.String())
			catalog.Rules[string(rule)] = entry
		}
	}
	for _, info := range review.Codes() {
		entry, ok := catalog.Rules[string(info.Rule)]
		if !ok {
			continue
		}
		entry.Codes = append(entry.Codes, catalogCode{Code: string(info.Code), Title: info.Title})
		catalog.Rules[string(info.Rule)] = entry
	}
	for rule := range catalog.Rules {
		require.NotEqual(t, storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED, storepb.ReviewRuleType(storepb.ReviewRuleType_value[rule]), "omni rule %s is unknown to this server", rule)
	}
	return catalog
}

// lackingFeatureRules reports the rules an engine lists only to say it lacks
// their feature: asked for online migration or prior backup, it reports
// OM101 or PB101 whatever the SQL.
func lackingFeatureRules(t *testing.T, engine storepb.Engine) map[review.Rule]bool {
	t.Helper()
	result, err := reviewFuncs[engine](context.Background(), "UPDATE t SET a = 1 WHERE id = 1;", review.Options{
		Rules:  []review.Rule{review.OnlineMigration, review.PriorBackup},
		Change: review.Change{OnlineMigration: true, PriorBackup: true},
	}, []review.Target{{}})
	require.NoError(t, err, engine.String())
	lacking := make(map[review.Rule]bool)
	for _, finding := range result.Findings {
		for _, code := range finding.Codes {
			switch code {
			case review.OnlineMigrationUnsupported:
				lacking[review.OnlineMigration] = true
			case review.PriorBackupUnsupported:
				lacking[review.PriorBackup] = true
			default:
			}
		}
	}
	return lacking
}
