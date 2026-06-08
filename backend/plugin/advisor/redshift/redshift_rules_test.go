package redshift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func TestRedshiftTableDropNamingConventionUsesOmniAST(t *testing.T) {
	sm := sheet.NewManager()
	rule := &storepb.SQLReviewRule{
		Type:  storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		Level: storepb.SQLReviewRule_WARNING,
		Payload: &storepb.SQLReviewRule_NamingPayload{
			NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{
				Format: "_delete$",
			},
		},
	}

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sm, "DROP TABLE active;", []*storepb.SQLReviewRule{rule}, advisor.Context{
		DBType:          storepb.Engine_REDSHIFT,
		NoAppendBuiltin: true,
	})

	require.NoError(t, err)
	require.Len(t, adviceList, 1)
	require.Equal(t, storepb.Advice_WARNING, adviceList[0].Status)
	require.Equal(t, code.TableDropNamingConventionMismatch.Int32(), adviceList[0].Code)
	require.Contains(t, adviceList[0].Content, "`active` mismatches drop table naming convention")
}
