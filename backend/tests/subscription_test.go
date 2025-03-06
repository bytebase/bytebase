package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func TestSubscription(t *testing.T) {
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir: dataDir,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	err = ctl.removeLicense(ctx)
	a.NoError(err)
	subscription, err := ctl.getSubscription(ctx)
	a.NoError(err)
	a.Equal(v1pb.PlanType_FREE, subscription.Plan)
}
