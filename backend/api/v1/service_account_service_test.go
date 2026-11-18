package v1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConvertMachineAccountsSetCreateTime(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)

	sa := convertToServiceAccount(&store.ServiceAccountMessage{
		Email: "deploy@service.bytebase.com", Name: "deploy", CreatedAt: created,
	})
	require.True(t, created.Equal(sa.GetCreateTime().AsTime()))

	wi := convertToWorkloadIdentity(&store.WorkloadIdentityMessage{
		Email: "deploy-ci@workload.bytebase.com", Name: "deploy-ci", CreatedAt: created,
		Config: &storepb.WorkloadIdentityConfig{ProviderType: storepb.WorkloadIdentityConfig_GITHUB},
	})
	require.True(t, created.Equal(wi.GetCreateTime().AsTime()))
}
