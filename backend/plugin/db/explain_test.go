package db

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestPlanFormat(t *testing.T) {
	explain := Explain{DefaultFormat: v1pb.QueryOption_TEXT}
	require.Equal(t, v1pb.QueryOption_TEXT, explain.PlanFormat(v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED))
	require.Equal(t, v1pb.QueryOption_JSON, explain.PlanFormat(v1pb.QueryOption_JSON))
}

func TestExplainStatementNeedsAStatementExplain(t *testing.T) {
	// No driver is registered in this package's tests.
	_, err := ExplainStatement(storepb.Engine_POSTGRES, "SELECT 1", v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED)
	require.Error(t, err)
}
