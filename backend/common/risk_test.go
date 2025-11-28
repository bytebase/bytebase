package common // nolint:revive

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestGetRiskLevelFromStatementTypes(t *testing.T) {
	tests := []struct {
		name           string
		statementTypes []string
		want           storepb.RiskLevel
	}{
		{
			name:           "empty returns LOW",
			statementTypes: []string{},
			want:           storepb.RiskLevel_LOW,
		},
		{
			name:           "DROP_TABLE is HIGH",
			statementTypes: []string{"DROP_TABLE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "DROP_DATABASE is HIGH",
			statementTypes: []string{"DROP_DATABASE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "TRUNCATE is HIGH",
			statementTypes: []string{"TRUNCATE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "DROP_SCHEMA is HIGH",
			statementTypes: []string{"DROP_SCHEMA"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "DELETE is MODERATE",
			statementTypes: []string{"DELETE"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "UPDATE is MODERATE",
			statementTypes: []string{"UPDATE"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "ALTER_TABLE is MODERATE",
			statementTypes: []string{"ALTER_TABLE"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "DROP_INDEX is MODERATE",
			statementTypes: []string{"DROP_INDEX"},
			want:           storepb.RiskLevel_MODERATE,
		},
		{
			name:           "CREATE_TABLE is LOW",
			statementTypes: []string{"CREATE_TABLE"},
			want:           storepb.RiskLevel_LOW,
		},
		{
			name:           "INSERT is LOW",
			statementTypes: []string{"INSERT"},
			want:           storepb.RiskLevel_LOW,
		},
		{
			name:           "mixed returns highest - HIGH wins",
			statementTypes: []string{"INSERT", "DELETE", "DROP_TABLE"},
			want:           storepb.RiskLevel_HIGH,
		},
		{
			name:           "mixed returns highest - MODERATE wins over LOW",
			statementTypes: []string{"INSERT", "UPDATE", "CREATE_TABLE"},
			want:           storepb.RiskLevel_MODERATE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRiskLevelFromStatementTypes(tt.statementTypes)
			require.Equal(t, tt.want, got)
		})
	}
}
