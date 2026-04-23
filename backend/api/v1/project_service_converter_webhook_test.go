package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestConvertWebhookTypeGoogleChat(t *testing.T) {
	a := require.New(t)

	storeType, err := convertToStoreWebhookType(v1pb.WebhookType_GOOGLE_CHAT)
	a.NoError(err)
	a.Equal(storepb.WebhookType_GOOGLE_CHAT, storeType)

	v1Type := convertToV1WebhookType(storepb.WebhookType_GOOGLE_CHAT)
	a.Equal(v1pb.WebhookType_GOOGLE_CHAT, v1Type)
}
