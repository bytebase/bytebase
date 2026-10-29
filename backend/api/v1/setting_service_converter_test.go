package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestConvertAISetting(t *testing.T) {
	t.Parallel()

	stored := convertAISetting(&v1pb.AISetting{
		Enabled:          true,
		Provider:         v1pb.AISetting_CLAUDE,
		Endpoint:         "https://api.anthropic.com/v1/messages",
		ApiKey:           "key",
		Model:            "claude-sonnet-5-5",
		Version:          "2023-06-01",
		SqlReviewEnabled: true,
	})
	require.Equal(t, &storepb.AISetting{
		Enabled:          true,
		Provider:         storepb.AISetting_CLAUDE,
		Endpoint:         "https://api.anthropic.com/v1/messages",
		ApiKey:           "key",
		Model:            "claude-sonnet-5-5",
		Version:          "2023-06-01",
		SqlReviewEnabled: true,
	}, stored)

	returned := convertToAISetting(stored)
	require.Empty(t, returned.ApiKey, "the key is never returned")
	require.True(t, returned.SqlReviewEnabled)
	require.True(t, returned.Enabled)

	require.False(t, convertToAISetting(&storepb.AISetting{Enabled: true}).SqlReviewEnabled, "the review is off until it is switched on")
}
