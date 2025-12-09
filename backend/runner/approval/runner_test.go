package approval

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestRiskLevelToString(t *testing.T) {
	tests := []struct {
		name  string
		level storepb.RiskLevel
		want  string
	}{
		{
			name:  "LOW",
			level: storepb.RiskLevel_LOW,
			want:  "LOW",
		},
		{
			name:  "MODERATE",
			level: storepb.RiskLevel_MODERATE,
			want:  "MODERATE",
		},
		{
			name:  "HIGH",
			level: storepb.RiskLevel_HIGH,
			want:  "HIGH",
		},
		{
			name:  "UNSPECIFIED defaults to LOW",
			level: storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED,
			want:  "LOW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			got := riskLevelToString(tt.level)
			a.Equal(tt.want, got)
		})
	}
}

func TestInjectRiskLevelIntoCELVars(t *testing.T) {
	tests := []struct {
		name       string
		celVars    []map[string]any
		riskLevel  storepb.RiskLevel
		wantValue  string
		wantLength int
	}{
		{
			name: "inject HIGH into single map",
			celVars: []map[string]any{
				{"resource.environment_id": "prod"},
			},
			riskLevel:  storepb.RiskLevel_HIGH,
			wantValue:  "HIGH",
			wantLength: 1,
		},
		{
			name: "inject MODERATE into multiple maps",
			celVars: []map[string]any{
				{"resource.environment_id": "prod"},
				{"resource.environment_id": "test"},
			},
			riskLevel:  storepb.RiskLevel_MODERATE,
			wantValue:  "MODERATE",
			wantLength: 2,
		},
		{
			name:       "inject into empty list",
			celVars:    []map[string]any{},
			riskLevel:  storepb.RiskLevel_LOW,
			wantValue:  "LOW",
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)

			injectRiskLevelIntoCELVars(tt.celVars, tt.riskLevel)

			a.Len(tt.celVars, tt.wantLength)
			for _, vars := range tt.celVars {
				riskLevel, ok := vars[common.CELAttributeRiskLevel]
				a.True(ok, "risk.level should be present")
				a.Equal(tt.wantValue, riskLevel)
			}
		})
	}
}

func TestCalculateRiskLevelFromCELVars(t *testing.T) {
	tests := []struct {
		name    string
		celVars []map[string]any
		want    storepb.RiskLevel
	}{
		{
			name:    "nil returns LOW",
			celVars: nil,
			want:    storepb.RiskLevel_LOW,
		},
		{
			name:    "empty returns LOW",
			celVars: []map[string]any{},
			want:    storepb.RiskLevel_LOW,
		},
		{
			name: "SELECT returns LOW",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "SELECT"},
			},
			want: storepb.RiskLevel_LOW,
		},
		{
			name: "UPDATE returns MODERATE",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "UPDATE"},
			},
			want: storepb.RiskLevel_MODERATE,
		},
		{
			name: "DELETE returns MODERATE",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "DELETE"},
			},
			want: storepb.RiskLevel_MODERATE,
		},
		{
			name: "DROP_TABLE returns HIGH",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "DROP_TABLE"},
			},
			want: storepb.RiskLevel_HIGH,
		},
		{
			name: "TRUNCATE returns HIGH",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "TRUNCATE"},
			},
			want: storepb.RiskLevel_HIGH,
		},
		{
			name: "mixed SELECT and DROP_TABLE returns HIGH (highest)",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "SELECT"},
				{common.CELAttributeStatementSQLType: "DROP_TABLE"},
			},
			want: storepb.RiskLevel_HIGH,
		},
		{
			name: "mixed UPDATE and DELETE returns MODERATE",
			celVars: []map[string]any{
				{common.CELAttributeStatementSQLType: "UPDATE"},
				{common.CELAttributeStatementSQLType: "DELETE"},
			},
			want: storepb.RiskLevel_MODERATE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			got := calculateRiskLevelFromCELVars(tt.celVars)
			a.Equal(tt.want, got)
		})
	}
}
