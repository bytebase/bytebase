package advisor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var mockConfigOverrideYAMLStr = `
template: bb.sql-review.prod # Provide the template id, then we can extend rules from the specific template.
ruleList:
  - type: statement.select.no-select-all
    level: DISABLED
  - type: table.drop-naming-convention
    level: WARNING
  - type: table.require-pk
    level: TEST
  - type: naming.table
    payload:
      format: "^table_[a-z]+(_[a-z]+)*$"
  - type: naming.column
    payload:
      maxLength: 24
  - type: column.required
    level: ERROR
    payload:
      list:
        - name
`

func TestConfigOverride(t *testing.T) {
	override := &SQLReviewConfigOverride{}
	err := yaml.Unmarshal([]byte(mockConfigOverrideYAMLStr), override)
	require.NoError(t, err)

	ruleList, err := MergeSQLReviewRules(override)
	require.NoError(t, err)

	for _, rule := range ruleList {
		switch rule.Type {
		case "statement.select.no-select-all":
			assert.Equal(t, storepb.SQLReviewRuleLevel_DISABLED, rule.Level)
		case "table.drop-naming-convention":
			assert.Equal(t, storepb.SQLReviewRuleLevel_WARNING, rule.Level)
		case "table.require-pk":
			assert.Equal(t, storepb.SQLReviewRuleLevel_ERROR, rule.Level)
		case "naming.table":
			assert.Equal(t, storepb.SQLReviewRuleLevel_WARNING, rule.Level)

			var nr NamingRulePayload
			err := json.Unmarshal([]byte(rule.Payload), &nr)
			require.NoError(t, err)

			assert.Equal(t, "^table_[a-z]+(_[a-z]+)*$", nr.Format)
			assert.Equal(t, 63, nr.MaxLength)
		case "naming.column":
			assert.Equal(t, storepb.SQLReviewRuleLevel_WARNING, rule.Level)

			var nr NamingRulePayload
			err := json.Unmarshal([]byte(rule.Payload), &nr)
			require.NoError(t, err)

			assert.Equal(t, "^[a-z]+(_[a-z]+)*$", nr.Format)
			assert.Equal(t, 24, nr.MaxLength)
		case "column.required":
			assert.Equal(t, storepb.SQLReviewRuleLevel_ERROR, rule.Level)

			var payload StringArrayTypeRulePayload
			err := json.Unmarshal([]byte(rule.Payload), &payload)
			require.NoError(t, err)

			assert.Equal(t, 1, len(payload.List))
			assert.Equal(t, "name", payload.List[0])
		}
	}
}
