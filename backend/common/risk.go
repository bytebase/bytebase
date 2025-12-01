//nolint:revive
package common

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// High risk statement types - destructive, irreversible operations.
var highRiskStatementTypes = map[string]bool{
	"DROP_DATABASE": true,
	"DROP_TABLE":    true,
	"DROP_SCHEMA":   true,
	"TRUNCATE":      true,
}

// Moderate risk statement types - data modification, potentially wide impact.
var moderateRiskStatementTypes = map[string]bool{
	"DELETE":      true,
	"UPDATE":      true,
	"ALTER_TABLE": true,
	"DROP_INDEX":  true,
}

// GetRiskLevelFromStatementTypes returns the highest risk level for the given statement types.
func GetRiskLevelFromStatementTypes(statementTypes []string) storepb.RiskLevel {
	for _, t := range statementTypes {
		if highRiskStatementTypes[t] {
			return storepb.RiskLevel_HIGH
		}
	}
	for _, t := range statementTypes {
		if moderateRiskStatementTypes[t] {
			return storepb.RiskLevel_MODERATE
		}
	}
	return storepb.RiskLevel_LOW
}
